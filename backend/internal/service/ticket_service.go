package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
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
	}
}

func (s *TicketService) Create(ticket *model.Ticket, operatorID int64, operatorName string) error {
	if ticket.Title == "" {
		return errors.New("工单标题不能为空")
	}
	if ticket.TypeID == 0 {
		return errors.New("请选择工单类型")
	}

	// 查类型
	tt, err := s.typeRepo.GetByID(ticket.TypeID)
	if err != nil {
		return errors.New("工单类型不存在")
	}

	// 生成编号
	ticket.TicketNo = s.repo.GenerateTicketNo()
	ticket.Status = "open"
	if ticket.Priority == "" {
		ticket.Priority = tt.Priority
	}
	if ticket.Source == "" {
		ticket.Source = "manual"
	}

	// 自动填充部门
	ticket.CreatorID = operatorID
	if ticket.SubmitDeptID == 0 {
		if user, err := s.userRepo.GetByID(operatorID); err == nil {
			ticket.SubmitDeptID = user.DepartmentID
		}
	}
	if ticket.HandleDeptID == 0 {
		ticket.HandleDeptID = tt.HandleDeptID
	}

	// 自动填充资源名称
	s.autoFillFromResource(ticket)

	// 自动分派
	s.autoAssign(ticket, tt)

	// 创建工单
	if err := s.repo.Create(ticket); err != nil {
		return fmt.Errorf("创建工单失败: %w", err)
	}

	// 写创建活动
	s.activityRepo.Create(&model.TicketActivity{
		TicketID: ticket.ID,
		UserID:   operatorID,
		Type:     "create",
		Content:  "创建工单",
	})

	// 如果自动分派了，再写一条分派活动
	if ticket.AssigneeID > 0 && ticket.Status == "processing" {
		assigneeName := s.getUserName(ticket.AssigneeID)
		s.activityRepo.Create(&model.TicketActivity{
			TicketID: ticket.ID,
			UserID:   0,
			Type:     "assign",
			Content:  fmt.Sprintf("自动分派给 %s（%s）", assigneeName, tt.AutoAssignRule),
			NewValue: assigneeName,
			IsSystem: true,
		})
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

	oldAssignee := s.getUserName(ticket.AssigneeID)
	newAssignee := s.getUserName(assigneeID)

	ticket.AssigneeID = assigneeID
	ticket.Status = "processing"
	if err := s.repo.Update(ticket); err != nil {
		return err
	}

	s.activityRepo.Create(&model.TicketActivity{
		TicketID: ticket.ID,
		UserID:   operatorID,
		Type:     "assign",
		Content:  fmt.Sprintf("分配处理人: %s", newAssignee),
		OldValue: oldAssignee,
		NewValue: newAssignee,
	})
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

	s.activityRepo.Create(&model.TicketActivity{
		TicketID: ticket.ID,
		UserID:   operatorID,
		Type:     action,
		Content:  content,
		OldValue: oldStatus,
		NewValue: targetStatus,
	})
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

	s.activityRepo.Create(&model.TicketActivity{
		TicketID: ticket.ID,
		UserID:   operatorID,
		Type:     "close",
		Content:  fmt.Sprintf("关闭工单，处理结果: %s", resolution),
		OldValue: oldStatus,
		NewValue: "closed",
	})
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

	s.activityRepo.Create(&model.TicketActivity{
		TicketID: ticket.ID,
		UserID:   operatorID,
		Type:     "reopen",
		Content:  content,
		OldValue: oldStatus,
		NewValue: "processing",
	})
	return nil
}

func (s *TicketService) Transfer(id, newAssigneeID int64, content string, operatorID int64) error {
	ticket, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("工单不存在")
	}

	oldAssignee := s.getUserName(ticket.AssigneeID)
	newAssignee := s.getUserName(newAssigneeID)

	ticket.AssigneeID = newAssigneeID
	if err := s.repo.Update(ticket); err != nil {
		return err
	}

	s.activityRepo.Create(&model.TicketActivity{
		TicketID: ticket.ID,
		UserID:   operatorID,
		Type:     "transfer",
		Content:  content,
		OldValue: oldAssignee,
		NewValue: newAssignee,
	})
	return nil
}

func (s *TicketService) Comment(id int64, content string, operatorID int64) error {
	ticket, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("工单不存在")
	}
	_ = ticket

	s.activityRepo.Create(&model.TicketActivity{
		TicketID: id,
		UserID:   operatorID,
		Type:     "comment",
		Content:  content,
	})
	return nil
}

func (s *TicketService) GetByID(id int64) (*model.Ticket, error) {
	ticket, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	s.fillNames(ticket)
	return ticket, nil
}

func (s *TicketService) List(q repository.TicketListQuery) ([]*model.Ticket, int64, error) {
	items, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	for _, t := range items {
		s.fillNames(t)
	}
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
		if asset, err := s.assetRepo.GetByID(ticket.ResourceID); err == nil {
			ticket.ResourceName = fmt.Sprintf("%s (%s)", asset.Hostname, asset.IP)
			if ticket.ServiceTreeID == 0 {
				ticket.ServiceTreeID = asset.ServiceTreeID
			}
		}
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

func (s *TicketService) autoAssign(ticket *model.Ticket, tt *model.TicketType) {
	switch tt.AutoAssignRule {
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
		if tt.DefaultAssignee > 0 {
			ticket.AssigneeID = tt.DefaultAssignee
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
}
