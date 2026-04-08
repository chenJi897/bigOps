package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
	"github.com/bigops/platform/internal/repository"
	"gorm.io/gorm"
)

// 状态转换规则
var ticketTransitions = map[string][]string{
	"open":       {"processing"},
	"processing": {"resolved", "rejected"},
	"resolved":   {"closed", "processing"},
	"rejected":   {"closed", "processing"},
	"closed":     {"processing"},
}

type TicketService struct {
	repo         *repository.TicketRepository
	typeRepo     *repository.TicketTypeRepository
	activityRepo *repository.TicketActivityRepository
	userRepo     *repository.UserRepository
	deptRepo     *repository.DepartmentRepository
	assetRepo    *repository.AssetRepository
	accountRepo  *repository.CloudAccountRepository
	treeRepo     *repository.ServiceTreeRepository
	requestRepo  *repository.RequestTemplateRepository
	policyRepo   *repository.ApprovalPolicyRepository
	approvalRepo *repository.ApprovalInstanceRepository
	roleRepo     *repository.RoleRepository
	notifySvc    *NotificationService
}

func NewTicketService() *TicketService {
	return &TicketService{
		repo:         repository.NewTicketRepository(),
		typeRepo:     repository.NewTicketTypeRepository(),
		activityRepo: repository.NewTicketActivityRepository(),
		userRepo:     repository.NewUserRepository(),
		deptRepo:     repository.NewDepartmentRepository(),
		assetRepo:    repository.NewAssetRepository(),
		accountRepo:  repository.NewCloudAccountRepository(),
		treeRepo:     repository.NewServiceTreeRepository(),
		requestRepo:  repository.NewRequestTemplateRepository(),
		policyRepo:   repository.NewApprovalPolicyRepository(),
		approvalRepo: repository.NewApprovalInstanceRepository(),
		roleRepo:     repository.NewRoleRepository(),
		notifySvc:    NewNotificationService(),
	}
}

type ticketExtraFields struct {
	ResourceIDs   []int64              `json:"resource_ids,omitempty"`
	ResourceItems []ticketResourceItem `json:"resource_items,omitempty"`
}

type ticketResourceItem struct {
	ID              int64  `json:"id"`
	Hostname        string `json:"hostname"`
	IP              string `json:"ip"`
	Status          string `json:"status"`
	OS              string `json:"os"`
	ServiceTreeID   int64  `json:"service_tree_id"`
	ServiceTreeName string `json:"service_tree_name"`
	ServiceTreePath string `json:"service_tree_path"`
}

type approvalBootstrap struct {
	template    *model.RequestTemplate
	policy      *model.ApprovalPolicy
	firstStage  *model.ApprovalPolicyStage
	approverIDs []int64
}

func (s *TicketService) Create(ticket *model.Ticket, operatorID int64, operatorName string) error {
	if ticket.Title == "" {
		return errors.New("工单标题不能为空")
	}

	// 从模板获取默认值
	var template *model.RequestTemplate
	if ticket.RequestTemplateID > 0 {
		tpl, err := s.requestRepo.GetByID(ticket.RequestTemplateID)
		if err != nil {
			return errors.New("请求模板不存在")
		}
		template = tpl
		if ticket.TypeID == 0 {
			ticket.TypeID = template.TypeID
		}
		if ticket.TicketKind == "" {
			ticket.TicketKind = template.TicketKind
		}
	}

	// TypeID 不再强制要求（模板用 category 分类）
	// 如果有 TypeID 则校验存在性
	var tt *model.TicketType
	if ticket.TypeID > 0 {
		found, err := s.typeRepo.GetByID(ticket.TypeID)
		if err != nil {
			return errors.New("工单类型不存在")
		}
		tt = found
	}

	ticket.Status = "open"
	// 优先级取值优先级：请求传入 > 模板默认 > 工单类型默认 > medium
	if ticket.Priority == "" {
		if template != nil && template.Priority != "" {
			ticket.Priority = template.Priority
		} else if tt != nil {
			ticket.Priority = tt.Priority
		} else {
			ticket.Priority = "medium"
		}
	}
	if ticket.Source == "" {
		ticket.Source = "manual"
	}
	if ticket.TicketKind == "" {
		ticket.TicketKind = "incident"
	}
	ticket.ApprovalStatus = "not_required"
	ticket.ExecutionStatus = "not_started"

	// 自动填充部门
	ticket.CreatorID = operatorID
	if ticket.SubmitDeptID == 0 {
		if user, err := s.userRepo.GetByID(operatorID); err == nil {
			ticket.SubmitDeptID = user.DepartmentID
		}
	}
	// 处理部门：请求传入 > 模板默认 > 工单类型默认
	if ticket.HandleDeptID == 0 {
		if template != nil && template.HandleDeptID > 0 {
			ticket.HandleDeptID = template.HandleDeptID
		} else if tt != nil {
			ticket.HandleDeptID = tt.HandleDeptID
		}
	}
	if ticket.ResourceType == "asset" && ticket.ResourceID == 0 {
		extra := parseTicketExtraFields(ticket.ExtraFields)
		if len(extra.ResourceIDs) > 0 {
			ticket.ResourceID = extra.ResourceIDs[0]
		}
	}

	// 自动填充资源名称
	s.autoFillFromResource(ticket)

	approvalPlan, err := s.prepareApprovalBootstrap(ticket, operatorID)
	if err != nil {
		return err
	}
	if approvalPlan == nil {
		s.autoAssign(ticket, template, tt)
	}

	var approvalEventID int64
	err = database.GetDB().Transaction(func(tx *gorm.DB) error {
		// 重试循环：并发时 ticket_no 可能冲突，最多重试 3 次
		const maxRetries = 3
		var createErr error
		for attempt := 0; attempt < maxRetries; attempt++ {
			ticket.TicketNo = s.repo.GenerateTicketNo()
			ticket.ID = 0 // 重置 ID，避免重试时 GORM 误认为是更新
			createErr = tx.Create(ticket).Error
			if createErr == nil {
				break
			}
			if !isDuplicateTicketNo(createErr) {
				return fmt.Errorf("创建工单失败: %w", createErr)
			}
			// ticket_no 冲突，继续重试
		}
		if createErr != nil {
			return fmt.Errorf("创建工单失败（编号冲突重试耗尽）: %w", createErr)
		}

		if err := tx.Create(&model.TicketActivity{
			TicketID: ticket.ID,
			UserID:   operatorID,
			Type:     "create",
			Content:  "创建工单",
		}).Error; err != nil {
			return err
		}

		if approvalPlan != nil {
			eventID, err := s.startApprovalFlowTx(tx, ticket, approvalPlan, operatorID, operatorName)
			if err != nil {
				return err
			}
			approvalEventID = eventID
			return nil
		}

		if ticket.AssigneeID > 0 && ticket.Status == "processing" {
			assigneeName := s.getUserName(ticket.AssigneeID)
			rule := "auto"
			if tt != nil {
				rule = tt.AutoAssignRule
			} else if template != nil {
				rule = template.AutoAssignRule
			}
			if err := tx.Create(&model.TicketActivity{
				TicketID: ticket.ID,
				UserID:   0,
				Type:     "assign",
				Content:  fmt.Sprintf("自动分派给 %s（%s）", assigneeName, rule),
				NewValue: assigneeName,
				IsSystem: true,
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	if approvalEventID > 0 {
		s.notifySvc.DispatchEventAsync(approvalEventID)
	}
	return nil
}

func (s *TicketService) Assign(id, assigneeID, operatorID int64) error {
	ticket, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("工单不存在")
	}
	if err := validateTransition(ticket.Status, "processing"); err != nil {
		return err
	}
	assignee, err := s.userRepo.GetByID(assigneeID)
	if err != nil {
		return errors.New("指定的处理人不存在")
	}

	oldAssignee := s.getUserName(ticket.AssigneeID)
	newAssignee := assignee.RealName
	if newAssignee == "" {
		newAssignee = assignee.Username
	}

	ticket.AssigneeID = assigneeID
	ticket.Status = "processing"
	if err := s.repo.Update(ticket); err != nil {
		return err
	}

	if err := s.activityRepo.Create(&model.TicketActivity{
		TicketID: ticket.ID,
		UserID:   operatorID,
		Type:     "assign",
		Content:  fmt.Sprintf("分配处理人: %s", newAssignee),
		OldValue: oldAssignee,
		NewValue: newAssignee,
	}); err != nil {
		return fmt.Errorf("记录活动失败: %w", err)
	}
	return nil
}

func (s *TicketService) Process(id int64, action, content string, operatorID int64) error {
	ticket, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("工单不存在")
	}

	var targetStatus string
	switch action {
	case "resolve":
		targetStatus = "resolved"
	case "reject":
		targetStatus = "rejected"
	default:
		return errors.New("无效的处理操作")
	}

	if err := validateTransition(ticket.Status, targetStatus); err != nil {
		return err
	}

	oldStatus := ticket.Status
	ticket.Status = targetStatus
	if targetStatus == "resolved" {
		now := model.LocalTime(time.Now())
		ticket.ResolvedAt = &now
	}
	if err := s.repo.Update(ticket); err != nil {
		return err
	}

	if err := s.activityRepo.Create(&model.TicketActivity{
		TicketID: ticket.ID,
		UserID:   operatorID,
		Type:     action,
		Content:  content,
		OldValue: oldStatus,
		NewValue: targetStatus,
	}); err != nil {
		return fmt.Errorf("记录活动失败: %w", err)
	}
	return nil
}

func (s *TicketService) Close(id int64, resolution, note string, operatorID int64) error {
	ticket, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("工单不存在")
	}
	if err := validateTransition(ticket.Status, "closed"); err != nil {
		return err
	}
	if resolution == "" {
		return errors.New("请选择处理结果")
	}

	oldStatus := ticket.Status
	ticket.Status = "closed"
	ticket.Resolution = resolution
	ticket.ResolutionNote = note
	now := model.LocalTime(time.Now())
	ticket.ClosedAt = &now
	if err := s.repo.Update(ticket); err != nil {
		return err
	}

	if err := s.activityRepo.Create(&model.TicketActivity{
		TicketID: ticket.ID,
		UserID:   operatorID,
		Type:     "close",
		Content:  fmt.Sprintf("关闭工单，处理结果: %s", resolution),
		OldValue: oldStatus,
		NewValue: "closed",
	}); err != nil {
		return fmt.Errorf("记录活动失败: %w", err)
	}
	return nil
}

func (s *TicketService) Reopen(id int64, content string, operatorID int64) error {
	ticket, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("工单不存在")
	}
	if err := validateTransition(ticket.Status, "processing"); err != nil {
		return err
	}

	oldStatus := ticket.Status
	ticket.Status = "processing"
	ticket.Resolution = ""
	ticket.ResolutionNote = ""
	ticket.ClosedAt = nil
	ticket.ResolvedAt = nil
	if err := s.repo.Update(ticket); err != nil {
		return err
	}

	if err := s.activityRepo.Create(&model.TicketActivity{
		TicketID: ticket.ID,
		UserID:   operatorID,
		Type:     "reopen",
		Content:  content,
		OldValue: oldStatus,
		NewValue: "processing",
	}); err != nil {
		return fmt.Errorf("记录活动失败: %w", err)
	}
	return nil
}

func (s *TicketService) Transfer(id, newAssigneeID int64, content string, operatorID int64) error {
	ticket, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("工单不存在")
	}
	if ticket.Status == "closed" {
		return errors.New("已关闭的工单不能转交")
	}
	if newAssigneeID == 0 {
		return errors.New("请选择转交人")
	}
	if newAssigneeID == ticket.AssigneeID {
		return errors.New("不能转交给当前处理人")
	}
	assignee, err := s.userRepo.GetByID(newAssigneeID)
	if err != nil {
		return errors.New("指定的转交人不存在")
	}

	oldAssignee := s.getUserName(ticket.AssigneeID)
	newAssignee := assignee.RealName
	if newAssignee == "" {
		newAssignee = assignee.Username
	}

	ticket.AssigneeID = newAssigneeID
	if err := s.repo.Update(ticket); err != nil {
		return err
	}

	if err := s.activityRepo.Create(&model.TicketActivity{
		TicketID: ticket.ID,
		UserID:   operatorID,
		Type:     "transfer",
		Content:  content,
		OldValue: oldAssignee,
		NewValue: newAssignee,
	}); err != nil {
		return fmt.Errorf("记录活动失败: %w", err)
	}
	return nil
}

func (s *TicketService) Comment(id int64, content string, operatorID int64) error {
	ticket, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("工单不存在")
	}
	_ = ticket

	if err := s.activityRepo.Create(&model.TicketActivity{
		TicketID: id,
		UserID:   operatorID,
		Type:     "comment",
		Content:  content,
	}); err != nil {
		return fmt.Errorf("记录评论失败: %w", err)
	}
	return nil
}

func (s *TicketService) GetByID(id int64) (*model.Ticket, error) {
	ticket, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if ticket.ResourceType != "" {
		s.autoFillFromResource(ticket)
	}
	s.fillNames(ticket)
	return ticket, nil
}

func (s *TicketService) List(q repository.TicketListQuery) ([]*model.Ticket, int64, error) {
	items, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	s.fillNamesBatch(items)
	return items, total, nil
}

func (s *TicketService) GetActivities(ticketID int64, page, size int) ([]*model.TicketActivity, int64, error) {
	items, total, err := s.activityRepo.ListByTicketID(ticketID, page, size)
	if err != nil {
		return nil, 0, err
	}
	nameMap := make(map[int64]string)
	for _, a := range items {
		if a.UserID > 0 {
			if name, ok := nameMap[a.UserID]; ok {
				a.UserName = name
			} else {
				name := s.getUserName(a.UserID)
				nameMap[a.UserID] = name
				a.UserName = name
			}
		}
		if a.IsSystem {
			a.UserName = "系统"
		}
	}
	return items, total, nil
}

// --- 内部方法 ---

func validateTransition(current, target string) error {
	allowed, ok := ticketTransitions[current]
	if !ok {
		return fmt.Errorf("当前状态 %s 不允许操作", current)
	}
	for _, s := range allowed {
		if s == target {
			return nil
		}
	}
	return fmt.Errorf("不允许从 %s 转换为 %s", current, target)
}

func (s *TicketService) autoFillFromResource(ticket *model.Ticket) {
	switch ticket.ResourceType {
	case "asset":
		extra := parseTicketExtraFields(ticket.ExtraFields)
		ids := extra.ResourceIDs
		if len(ids) == 0 && ticket.ResourceID > 0 {
			ids = []int64{ticket.ResourceID}
		}
		if len(ids) == 0 {
			return
		}

		assets, err := s.assetRepo.GetByIDs(ids)
		if err != nil || len(assets) == 0 {
			return
		}
		assetMap := make(map[int64]*model.Asset, len(assets))
		for _, asset := range assets {
			assetMap[asset.ID] = asset
		}

		pathMap := s.buildServiceTreePathMap()
		items := make([]ticketResourceItem, 0, len(ids))
		for _, id := range ids {
			asset, ok := assetMap[id]
			if !ok {
				continue
			}
			items = append(items, ticketResourceItem{
				ID:              asset.ID,
				Hostname:        asset.Hostname,
				IP:              asset.IP,
				Status:          asset.Status,
				OS:              asset.OS,
				ServiceTreeID:   asset.ServiceTreeID,
				ServiceTreeName: asset.ServiceTreeName,
				ServiceTreePath: pathMap[asset.ServiceTreeID],
			})
		}
		if len(items) == 0 {
			return
		}

		first := assetMap[ids[0]]
		if first == nil {
			first = assetMap[items[0].ID]
		}
		ticket.ResourceID = first.ID
		if len(items) == 1 {
			ticket.ResourceName = fmt.Sprintf("%s (%s)", first.Hostname, first.IP)
		} else {
			ticket.ResourceName = fmt.Sprintf("%d 台主机资产", len(items))
		}
		if ticket.ServiceTreeID == 0 {
			ticket.ServiceTreeID = first.ServiceTreeID
		}
		extra.ResourceIDs = ids
		extra.ResourceItems = items
		ticket.ExtraFields = mustMarshalTicketExtraFields(extra)
	case "cloud_account":
		if account, err := s.accountRepo.GetByID(ticket.ResourceID); err == nil {
			ticket.ResourceName = account.Name
			if ticket.ServiceTreeID == 0 {
				ticket.ServiceTreeID = account.ServiceTreeID
			}
		}
	case "service_tree":
		if node, err := s.treeRepo.GetByID(ticket.ResourceID); err == nil {
			ticket.ResourceName = node.Name
			ticket.ServiceTreeID = node.ID
		}
	}
}

func (s *TicketService) buildServiceTreePathMap() map[int64]string {
	nodes, err := s.treeRepo.GetAll()
	if err != nil {
		return map[int64]string{}
	}
	nodeMap := make(map[int64]*model.ServiceTree, len(nodes))
	for _, node := range nodes {
		n := node
		nodeMap[n.ID] = n
	}
	pathMap := make(map[int64]string, len(nodes))
	for _, node := range nodes {
		pathMap[node.ID] = buildServiceTreePath(node.ID, nodeMap)
	}
	return pathMap
}

func buildServiceTreePath(id int64, nodeMap map[int64]*model.ServiceTree) string {
	node, ok := nodeMap[id]
	if !ok || node == nil {
		return ""
	}
	names := []string{}
	current := node
	for current != nil {
		names = append([]string{current.Name}, names...)
		if current.ParentID == 0 {
			break
		}
		current = nodeMap[current.ParentID]
	}
	var path string
	for i, name := range names {
		if i == 0 {
			path = name
		} else {
			path += " / " + name
		}
	}
	return path
}

func parseTicketExtraFields(raw string) ticketExtraFields {
	var extra ticketExtraFields
	if raw == "" {
		return extra
	}
	_ = json.Unmarshal([]byte(raw), &extra)
	return extra
}

func mustMarshalTicketExtraFields(extra ticketExtraFields) string {
	data, err := json.Marshal(extra)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (s *TicketService) autoAssign(ticket *model.Ticket, tpl *model.RequestTemplate, tt *model.TicketType) {
	// 分派规则优先取模板，兜底取工单类型
	assignRule := ""
	var defaultAssignee int64
	if tpl != nil && tpl.AutoAssignRule != "" && tpl.AutoAssignRule != "manual" {
		assignRule = tpl.AutoAssignRule
		defaultAssignee = tpl.DefaultAssignee
	} else if tt != nil {
		assignRule = tt.AutoAssignRule
		defaultAssignee = tt.DefaultAssignee
	}

	switch assignRule {
	case "resource_owner":
		ownerID := s.getResourceOwner(ticket.ResourceType, ticket.ResourceID)
		if ownerID > 0 {
			ticket.AssigneeID = ownerID
		}
	case "service_owner":
		if ticket.ServiceTreeID > 0 {
			if node, err := s.treeRepo.GetByID(ticket.ServiceTreeID); err == nil && node.OwnerID > 0 {
				ticket.AssigneeID = node.OwnerID
			}
		}
	case "dept_default":
		if defaultAssignee > 0 {
			ticket.AssigneeID = defaultAssignee
		}
	}
	if ticket.AssigneeID > 0 {
		ticket.Status = "processing"
	}
}

func (s *TicketService) getResourceOwner(resourceType string, resourceID int64) int64 {
	var ownerIDsStr string
	switch resourceType {
	case "asset":
		if asset, err := s.assetRepo.GetByID(resourceID); err == nil {
			ownerIDsStr = asset.OwnerIDs
		}
	case "cloud_account":
		if account, err := s.accountRepo.GetByID(resourceID); err == nil {
			ownerIDsStr = account.OwnerIDs
		}
	}
	if ownerIDsStr == "" || ownerIDsStr == "[]" {
		return 0
	}
	var ids []int64
	json.Unmarshal([]byte(ownerIDsStr), &ids)
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

func (s *TicketService) getUserName(id int64) string {
	if id == 0 {
		return ""
	}
	if user, err := s.userRepo.GetByID(id); err == nil {
		if user.RealName != "" {
			return user.RealName
		}
		return user.Username
	}
	return ""
}

func (s *TicketService) fillNames(ticket *model.Ticket) {
	// 类型名
	if ticket.TypeID > 0 {
		if tt, err := s.typeRepo.GetByID(ticket.TypeID); err == nil {
			ticket.TypeName = tt.Name
		}
	}
	// 人员名
	ticket.CreatorName = s.getUserName(ticket.CreatorID)
	ticket.AssigneeName = s.getUserName(ticket.AssigneeID)
	// 部门名
	if ticket.SubmitDeptID > 0 {
		if dept, err := s.deptRepo.GetByID(ticket.SubmitDeptID); err == nil {
			ticket.SubmitDeptName = dept.Name
		}
	}
	if ticket.HandleDeptID > 0 {
		if dept, err := s.deptRepo.GetByID(ticket.HandleDeptID); err == nil {
			ticket.HandleDeptName = dept.Name
		}
	}
	if ticket.RequestTemplateID > 0 {
		if tpl, err := s.requestRepo.GetByID(ticket.RequestTemplateID); err == nil {
			ticket.RequestTemplateName = tpl.Name
		}
	}
}

// fillNamesBatch 批量填充名称字段，避免 N+1 查询。
func (s *TicketService) fillNamesBatch(items []*model.Ticket) {
	if len(items) == 0 {
		return
	}

	// 收集所有需要查询的 ID
	typeIDSet := make(map[int64]struct{})
	userIDSet := make(map[int64]struct{})
	deptIDSet := make(map[int64]struct{})
	tplIDSet := make(map[int64]struct{})

	for _, t := range items {
		if t.TypeID > 0 {
			typeIDSet[t.TypeID] = struct{}{}
		}
		if t.CreatorID > 0 {
			userIDSet[t.CreatorID] = struct{}{}
		}
		if t.AssigneeID > 0 {
			userIDSet[t.AssigneeID] = struct{}{}
		}
		if t.SubmitDeptID > 0 {
			deptIDSet[t.SubmitDeptID] = struct{}{}
		}
		if t.HandleDeptID > 0 {
			deptIDSet[t.HandleDeptID] = struct{}{}
		}
		if t.RequestTemplateID > 0 {
			tplIDSet[t.RequestTemplateID] = struct{}{}
		}
	}

	// 批量查询
	typeIDs := setToSlice(typeIDSet)
	userIDs := setToSlice(userIDSet)
	deptIDs := setToSlice(deptIDSet)
	tplIDs := setToSlice(tplIDSet)

	typeMap, _ := s.typeRepo.GetByIDs(typeIDs)
	userMap := s.userRepo.GetNamesByIDs(userIDs)
	deptMap, _ := s.deptRepo.GetByIDs(deptIDs)
	tplMap, _ := s.requestRepo.GetByIDs(tplIDs)

	if typeMap == nil {
		typeMap = make(map[int64]*model.TicketType)
	}
	if deptMap == nil {
		deptMap = make(map[int64]*model.Department)
	}
	if tplMap == nil {
		tplMap = make(map[int64]*model.RequestTemplate)
	}

	// 填充
	for _, t := range items {
		if tt, ok := typeMap[t.TypeID]; ok {
			t.TypeName = tt.Name
		}
		t.CreatorName = userMap[t.CreatorID]
		t.AssigneeName = userMap[t.AssigneeID]
		if dept, ok := deptMap[t.SubmitDeptID]; ok {
			t.SubmitDeptName = dept.Name
		}
		if dept, ok := deptMap[t.HandleDeptID]; ok {
			t.HandleDeptName = dept.Name
		}
		if tpl, ok := tplMap[t.RequestTemplateID]; ok {
			t.RequestTemplateName = tpl.Name
		}
	}
}

func setToSlice(m map[int64]struct{}) []int64 {
	s := make([]int64, 0, len(m))
	for id := range m {
		s = append(s, id)
	}
	return s
}

func (s *TicketService) prepareApprovalBootstrap(ticket *model.Ticket, operatorID int64) (*approvalBootstrap, error) {
	if ticket.RequestTemplateID == 0 {
		return nil, nil
	}
	template, err := s.requestRepo.GetByID(ticket.RequestTemplateID)
	if err != nil {
		return nil, errors.New("请求模板不存在")
	}
	if template.Status != 1 {
		return nil, errors.New("请求模板已禁用")
	}
	if template.TicketKind != "" {
		ticket.TicketKind = template.TicketKind
	}
	templateStages := buildApprovalStagesFromTemplate(template)
	if len(templateStages) > 0 {
		firstStage := pickFirstApprovalStage(templateStages)
		if firstStage == nil {
			return nil, errors.New("工单模板缺少审批节点")
		}
		approverIDs, err := s.resolveApproverIDs(firstStage, ticket, operatorID)
		if err != nil {
			return nil, err
		}
		if len(approverIDs) == 0 {
			return nil, errors.New("审批节点未解析到处理成员")
		}
		ticket.ApprovalStatus = "pending"
		return &approvalBootstrap{
			template: template,
			policy: &model.ApprovalPolicy{
				ID:     0,
				Name:   template.Name,
				Scope:  template.TicketKind,
				Stages: templateStages,
			},
			firstStage:  firstStage,
			approverIDs: approverIDs,
		}, nil
	}
	if template.ApprovalPolicyID == 0 {
		ticket.ApprovalStatus = "not_required"
		return nil, nil
	}
	policy, err := s.policyRepo.GetByID(template.ApprovalPolicyID)
	if err != nil {
		return nil, errors.New("审批策略不存在")
	}
	if policy.Enabled != 1 {
		return nil, errors.New("审批策略已禁用")
	}
	firstStage := pickFirstApprovalStage(policy.Stages)
	if firstStage == nil {
		return nil, errors.New("审批策略缺少审批阶段")
	}
	approverIDs, err := s.resolveApproverIDs(firstStage, ticket, operatorID)
	if err != nil {
		return nil, err
	}
	if len(approverIDs) == 0 {
		return nil, errors.New("审批阶段未解析到审批人")
	}
	ticket.ApprovalStatus = "pending"
	return &approvalBootstrap{
		template:    template,
		policy:      policy,
		firstStage:  firstStage,
		approverIDs: approverIDs,
	}, nil
}

func pickFirstApprovalStage(stages []model.ApprovalPolicyStage) *model.ApprovalPolicyStage {
	if len(stages) == 0 {
		return nil
	}
	var picked *model.ApprovalPolicyStage
	for i := range stages {
		stage := stages[i]
		if picked == nil || stage.StageNo < picked.StageNo || (stage.StageNo == picked.StageNo && stage.Sort < picked.Sort) {
			s := stage
			picked = &s
		}
	}
	return picked
}

type approverConfig struct {
	UserIDs   []int64  `json:"user_ids"`
	RoleNames []string `json:"role_names"`
}

func (s *TicketService) resolveApproverIDs(stage *model.ApprovalPolicyStage, ticket *model.Ticket, operatorID int64) ([]int64, error) {
	var cfg approverConfig
	if stage.ApproverConfig != "" {
		_ = json.Unmarshal([]byte(stage.ApproverConfig), &cfg)
	}
	var ids []int64
	switch stage.ApproverType {
	case "fixed_user":
		ids = append(ids, cfg.UserIDs...)
	case "fixed_role":
		for _, roleName := range cfg.RoleNames {
			roleUserIDs, err := s.roleRepo.GetUserIDsByRoleName(roleName)
			if err != nil {
				return nil, err
			}
			ids = append(ids, roleUserIDs...)
		}
	case "dept_leader":
		user, err := s.userRepo.GetByID(operatorID)
		if err != nil {
			return nil, errors.New("提交人不存在")
		}
		if user.DepartmentID == 0 {
			return nil, errors.New("提交人未绑定部门，无法解析部门负责人")
		}
		dept, err := s.deptRepo.GetByID(user.DepartmentID)
		if err != nil {
			return nil, errors.New("提交人部门不存在")
		}
		if dept.ManagerID == 0 {
			return nil, errors.New("提交人部门未配置负责人")
		}
		ids = append(ids, dept.ManagerID)
	case "service_owner":
		if ticket.ServiceTreeID == 0 {
			return nil, errors.New("工单未关联服务树，无法解析服务负责人")
		}
		node, err := s.treeRepo.GetByID(ticket.ServiceTreeID)
		if err != nil {
			return nil, errors.New("服务树节点不存在")
		}
		if node.OwnerID == 0 {
			return nil, errors.New("服务树未配置负责人")
		}
		ids = append(ids, node.OwnerID)
	default:
		return nil, fmt.Errorf("暂不支持的审批人类型: %s", stage.ApproverType)
	}
	return dedupeInt64(ids), nil
}

// isDuplicateTicketNo 判断是否为 ticket_no 唯一索引冲突错误。
func isDuplicateTicketNo(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") && strings.Contains(msg, "ticket_no")
}

func dedupeInt64(items []int64) []int64 {
	result := make([]int64, 0, len(items))
	seen := make(map[int64]bool, len(items))
	for _, item := range items {
		if item <= 0 || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}

func (s *TicketService) startApprovalFlowTx(
	tx *gorm.DB,
	ticket *model.Ticket,
	plan *approvalBootstrap,
	operatorID int64,
	operatorName string,
) (int64, error) {
	now := model.LocalTime(time.Now())
	instance := &model.ApprovalInstance{
		TicketID:       ticket.ID,
		PolicyID:       plan.policy.ID,
		CurrentStageNo: plan.firstStage.StageNo,
		Status:         "pending",
		StartedAt:      &now,
	}
	if err := tx.Create(instance).Error; err != nil {
		return 0, err
	}

	nameMap := s.userRepo.GetNamesByIDs(plan.approverIDs)
	for _, approverID := range plan.approverIDs {
		record := &model.ApprovalRecord{
			InstanceID:   instance.ID,
			StageNo:      plan.firstStage.StageNo,
			ApproverID:   approverID,
			ApproverName: nameMap[approverID],
			Status:       "pending",
		}
		if err := tx.Create(record).Error; err != nil {
			return 0, err
		}
	}

	ticket.ApprovalInstanceID = instance.ID
	ticket.ApprovalStatus = "pending"
	ticket.ExecutionStatus = "not_started"
	ticket.RequestTemplateID = plan.template.ID
	if err := tx.Save(ticket).Error; err != nil {
		return 0, err
	}

	activity := &model.TicketActivity{
		TicketID: ticket.ID,
		UserID:   0,
		Type:     "approval_pending",
		Content:  fmt.Sprintf("已发起审批流程，当前阶段：%s", plan.firstStage.Name),
		IsSystem: true,
	}
	if err := tx.Create(activity).Error; err != nil {
		return 0, err
	}

	title := fmt.Sprintf("审批待处理：%s", ticket.Title)
	content := fmt.Sprintf("%s 提交了请求单 %s，当前审批阶段：%s", operatorName, ticket.TicketNo, plan.firstStage.Name)
	channels := parseNotifyChannels(plan.template.NotifyChannels)
	payload := map[string]interface{}{
		"ticket_id":            ticket.ID,
		"ticket_no":            ticket.TicketNo,
		"ticket_title":         ticket.Title,
		"approval_instance_id": instance.ID,
		"stage_no":             plan.firstStage.StageNo,
		"stage_name":           plan.firstStage.Name,
	}
	eventID, err := s.notifySvc.PublishTx(tx, NotificationPublishRequest{
		EventType: "approval_pending",
		BizType:   "ticket",
		BizID:     ticket.ID,
		Title:     title,
		Content:   content,
		Level:     "info",
		UserIDs:   plan.approverIDs,
		Channels:  channels,
		Payload:   payload,
	})
	if err != nil {
		return 0, err
	}
	return eventID, nil
}
