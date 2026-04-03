package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
	"github.com/bigops/platform/internal/repository"
)

type ApprovalTodoItem struct {
	InstanceID   int64           `json:"instance_id"`
	TicketID     int64           `json:"ticket_id"`
	TicketNo     string          `json:"ticket_no"`
	TicketTitle  string          `json:"ticket_title"`
	TicketKind   string          `json:"ticket_kind"`
	StageNo      int             `json:"stage_no"`
	StageName    string          `json:"stage_name"`
	PolicyID     int64           `json:"policy_id"`
	PolicyName   string          `json:"policy_name"`
	ApproverID   int64           `json:"approver_id"`
	ApproverName string          `json:"approver_name"`
	CreatedAt    model.LocalTime `json:"created_at"`
}

type ApprovalService struct {
	instanceRepo *repository.ApprovalInstanceRepository
	policyRepo   *repository.ApprovalPolicyRepository
	ticketRepo   *repository.TicketRepository
	requestRepo  *repository.RequestTemplateRepository
	userRepo     *repository.UserRepository
	roleRepo     *repository.RoleRepository
	deptRepo     *repository.DepartmentRepository
	treeRepo     *repository.ServiceTreeRepository
	notifySvc    *NotificationService
}

type approvalApproverConfig struct {
	UserIDs   []int64  `json:"user_ids"`
	RoleNames []string `json:"role_names"`
}

func NewApprovalService() *ApprovalService {
	return &ApprovalService{
		instanceRepo: repository.NewApprovalInstanceRepository(),
		policyRepo:   repository.NewApprovalPolicyRepository(),
		ticketRepo:   repository.NewTicketRepository(),
		requestRepo:  repository.NewRequestTemplateRepository(),
		userRepo:     repository.NewUserRepository(),
		roleRepo:     repository.NewRoleRepository(),
		deptRepo:     repository.NewDepartmentRepository(),
		treeRepo:     repository.NewServiceTreeRepository(),
		notifySvc:    NewNotificationService(),
	}
}

func (s *ApprovalService) ticketNotifyChannels(ticket *model.Ticket) []string {
	if ticket == nil || ticket.RequestTemplateID <= 0 {
		return nil
	}
	template, err := s.requestRepo.GetByID(ticket.RequestTemplateID)
	if err != nil || template == nil {
		return nil
	}
	return parseNotifyChannels(template.NotifyChannels)
}

func (s *ApprovalService) ticketNotifyConfig(ticket *model.Ticket) map[string]WebhookTarget {
	if ticket == nil || ticket.RequestTemplateID <= 0 {
		return nil
	}
	template, err := s.requestRepo.GetByID(ticket.RequestTemplateID)
	if err != nil || template == nil {
		return nil
	}
	return ParseNotifyConfig(template.NotifyConfig)
}

func (s *ApprovalService) ListPendingByApproverID(approverID int64) ([]*ApprovalTodoItem, error) {
	records, err := s.instanceRepo.ListPendingRecordsByApproverID(approverID)
	if err != nil {
		return nil, err
	}
	items := make([]*ApprovalTodoItem, 0, len(records))
	for _, record := range records {
		instance, err := s.instanceRepo.GetByID(record.InstanceID)
		if err != nil || instance.Status == "approved" || instance.Status == "rejected" || instance.Status == "canceled" {
			continue
		}
		ticket, err := s.ticketRepo.GetByID(instance.TicketID)
		if err != nil {
			continue
		}
		policy, stages, err := s.resolveApprovalStages(instance, ticket)
		if err != nil {
			continue
		}
		stageName := ""
		for _, stage := range stages {
			if stage.StageNo == record.StageNo {
				stageName = stage.Name
				break
			}
		}
		items = append(items, &ApprovalTodoItem{
			InstanceID:   instance.ID,
			TicketID:     ticket.ID,
			TicketNo:     ticket.TicketNo,
			TicketTitle:  ticket.Title,
			TicketKind:   ticket.TicketKind,
			StageNo:      record.StageNo,
			StageName:    stageName,
			PolicyID:     policy.ID,
			PolicyName:   policy.Name,
			ApproverID:   record.ApproverID,
			ApproverName: record.ApproverName,
			CreatedAt:    record.CreatedAt,
		})
	}
	return items, nil
}

func (s *ApprovalService) GetByTicketID(ticketID int64) (*model.ApprovalInstance, error) {
	instance, err := s.instanceRepo.GetByTicketID(ticketID)
	if err != nil {
		return nil, err
	}
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err == nil {
		policy, stages, resolveErr := s.resolveApprovalStages(instance, ticket)
		if resolveErr == nil {
			instance.PolicyName = policy.Name
			for _, stage := range stages {
				if stage.StageNo == instance.CurrentStageNo {
					instance.CurrentStageName = stage.Name
					break
				}
			}
		}
	}
	return instance, nil
}

func (s *ApprovalService) Approve(instanceID, approverID int64, comment string) error {
	return s.act(instanceID, approverID, "approve", comment)
}

func (s *ApprovalService) Reject(instanceID, approverID int64, comment string) error {
	return s.act(instanceID, approverID, "reject", comment)
}

func (s *ApprovalService) act(instanceID, approverID int64, action, comment string) error {
	var eventID int64
	var approvedTicketID int64
	var rejectedTicketID int64
	var rejectedReason string
	err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		instance, err := s.instanceRepo.GetByID(instanceID)
		if err != nil {
			return errors.New("审批实例不存在")
		}
		if instance.Status == "approved" || instance.Status == "rejected" || instance.Status == "canceled" {
			return errors.New("审批实例已结束")
		}
		record, err := s.instanceRepo.GetRecordByInstanceStageApprover(instance.ID, instance.CurrentStageNo, approverID)
		if err != nil {
			return errors.New("当前用户不是该阶段审批人")
		}
		if record.Status != "pending" {
			return errors.New("该审批已处理")
		}
		now := model.LocalTime(time.Now())
		record.Action = action
		record.Comment = comment
		record.Status = action
		record.ActedAt = &now
		if err := tx.Save(record).Error; err != nil {
			return err
		}

		ticket, err := s.ticketRepo.GetByID(instance.TicketID)
		if err != nil {
			return errors.New("工单不存在")
		}
		_, stages, err := s.resolveApprovalStages(instance, ticket)
		if err != nil {
			return err
		}
		stage := findStageByNo(stages, instance.CurrentStageNo)
		if stage == nil {
			return errors.New("审批阶段不存在")
		}

		if action == "reject" {
			instance.Status = "rejected"
			instance.FinishedAt = &now
			ticket.ApprovalStatus = "rejected"
			ticket.Status = "rejected"
			if err := tx.Save(instance).Error; err != nil {
				return err
			}
			if err := tx.Save(ticket).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.TicketActivity{
				TicketID: ticket.ID,
				UserID:   approverID,
				Type:     "approval_rejected",
				Content:  comment,
				IsSystem: false,
			}).Error; err != nil {
				return err
			}
			var notifyErr error
			channels := s.ticketNotifyChannels(ticket)
			notifyConfig := s.ticketNotifyConfig(ticket)
			eventID, notifyErr = s.notifySvc.PublishTx(tx, NotificationPublishRequest{
				EventType: "approval_rejected",
				BizType:   "ticket",
				BizID:     ticket.ID,
				Title:     fmt.Sprintf("审批已拒绝：%s", ticket.Title),
				Content:   fmt.Sprintf("工单 %s 在审批阶段 %s 被拒绝", ticket.TicketNo, stage.Name),
				Level:     "warning",
				UserIDs:   []int64{ticket.CreatorID},
				Channels:     channels,
				NotifyConfig: notifyConfig,
				Payload: map[string]interface{}{
					"ticket_id":   ticket.ID,
					"ticket_no":   ticket.TicketNo,
					"instance_id": instance.ID,
					"stage_no":    stage.StageNo,
					"stage_name":  stage.Name,
					"approver_id": approverID,
					"action":      action,
					"comment":     comment,
				},
			})
			rejectedTicketID = ticket.ID
			rejectedReason = comment
			return notifyErr
		}

		var currentStageRecords []*model.ApprovalRecord
		if err := tx.
			Where("instance_id = ? AND stage_no = ?", instance.ID, instance.CurrentStageNo).
			Order("id ASC").
			Find(&currentStageRecords).Error; err != nil {
			return err
		}
		if !stageApproved(stage.PassRule, currentStageRecords) {
			if err := tx.Create(&model.TicketActivity{
				TicketID: ticket.ID,
				UserID:   approverID,
				Type:     "approval_recorded",
				Content:  fmt.Sprintf("审批通过，等待同阶段其他审批人：%s", stage.Name),
				IsSystem: false,
			}).Error; err != nil {
				return err
			}
			return nil
		}

		nextStage := findNextStage(stages, instance.CurrentStageNo)
		if nextStage == nil {
			instance.Status = "approved"
			instance.FinishedAt = &now
			ticket.ApprovalStatus = "approved"
			ticket.Status = "processing" // 审批通过 → 推进到处理中

			// 自动指派处理人
			assigneeID := s.resolveAssigneeAfterApproval(ticket)
			if assigneeID > 0 {
				ticket.AssigneeID = assigneeID
			}

			if err := tx.Save(instance).Error; err != nil {
				return err
			}
			if err := tx.Save(ticket).Error; err != nil {
				return err
			}

			// 审批完成活动
			if err := tx.Create(&model.TicketActivity{
				TicketID: ticket.ID,
				UserID:   approverID,
				Type:     "approval_approved",
				Content:  "审批流程已完成",
				IsSystem: false,
			}).Error; err != nil {
				return err
			}

			// 自动指派活动
			if assigneeID > 0 {
				assigneeName := s.userRepo.GetNamesByIDs([]int64{assigneeID})[assigneeID]
				if err := tx.Create(&model.TicketActivity{
					TicketID: ticket.ID,
					UserID:   0,
					Type:     "assign",
					Content:  fmt.Sprintf("审批通过后自动分配给 %s", assigneeName),
					NewValue: assigneeName,
					IsSystem: true,
				}).Error; err != nil {
					return err
				}
			}
			var notifyErr error
			channels := s.ticketNotifyChannels(ticket)
			notifyConfig := s.ticketNotifyConfig(ticket)
			eventID, notifyErr = s.notifySvc.PublishTx(tx, NotificationPublishRequest{
				EventType: "approval_approved",
				BizType:   "ticket",
				BizID:     ticket.ID,
				Title:     fmt.Sprintf("审批已通过：%s", ticket.Title),
				Content:   fmt.Sprintf("工单 %s 审批流程已完成", ticket.TicketNo),
				Level:     "success",
				UserIDs:   []int64{ticket.CreatorID},
				Channels:     channels,
				NotifyConfig: notifyConfig,
				Payload: map[string]interface{}{
					"ticket_id":   ticket.ID,
					"ticket_no":   ticket.TicketNo,
					"instance_id": instance.ID,
				},
			})
			approvedTicketID = ticket.ID
			return notifyErr
		}

		nextApproverIDs, err := s.resolveApproverIDs(nextStage, ticket, ticket.CreatorID)
		if err != nil {
			return err
		}
		if len(nextApproverIDs) == 0 {
			return errors.New("下一审批阶段未解析到审批人")
		}
		instance.CurrentStageNo = nextStage.StageNo
		instance.Status = "pending"
		if err := tx.Save(instance).Error; err != nil {
			return err
		}

		nameMap := s.userRepo.GetNamesByIDs(nextApproverIDs)
		for _, nextApproverID := range nextApproverIDs {
			newRecord := &model.ApprovalRecord{
				InstanceID:   instance.ID,
				StageNo:      nextStage.StageNo,
				ApproverID:   nextApproverID,
				ApproverName: nameMap[nextApproverID],
				Status:       "pending",
			}
			if err := tx.Create(newRecord).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(&model.TicketActivity{
			TicketID: ticket.ID,
			UserID:   approverID,
			Type:     "approval_pending",
			Content:  fmt.Sprintf("进入下一审批阶段：%s", nextStage.Name),
			IsSystem: false,
		}).Error; err != nil {
			return err
		}
		var notifyErr error
		channels := s.ticketNotifyChannels(ticket)
		notifyConfig := s.ticketNotifyConfig(ticket)
		eventID, notifyErr = s.notifySvc.PublishTx(tx, NotificationPublishRequest{
			EventType: "approval_pending",
			BizType:   "ticket",
			BizID:     ticket.ID,
			Title:     fmt.Sprintf("审批待处理：%s", ticket.Title),
			Content:   fmt.Sprintf("工单 %s 进入审批阶段：%s", ticket.TicketNo, nextStage.Name),
			Level:     "info",
			UserIDs:   nextApproverIDs,
			Channels:     channels,
			NotifyConfig: notifyConfig,
			Payload: map[string]interface{}{
				"ticket_id":    ticket.ID,
				"ticket_no":    ticket.TicketNo,
				"ticket_title": ticket.Title,
				"instance_id":  instance.ID,
				"stage_no":     nextStage.StageNo,
				"stage_name":   nextStage.Name,
			},
		})
		return notifyErr
	})
	if err != nil {
		return err
	}
	if eventID > 0 {
		s.notifySvc.DispatchEventAsync(eventID)
	}
	if approvedTicketID > 0 {
		_ = NewCICDService().StartApprovedRunsByTicketID(approvedTicketID)
	}
	if rejectedTicketID > 0 {
		_ = NewCICDService().MarkRejectedRunsByTicketID(rejectedTicketID, rejectedReason)
	}
	return nil
}

func (s *ApprovalService) resolveApprovalStages(instance *model.ApprovalInstance, ticket *model.Ticket) (*model.ApprovalPolicy, []model.ApprovalPolicyStage, error) {
	if instance.PolicyID > 0 {
		policy, err := s.policyRepo.GetByID(instance.PolicyID)
		if err != nil {
			return nil, nil, errors.New("审批策略不存在")
		}
		return policy, policy.Stages, nil
	}
	if ticket != nil && ticket.RequestTemplateID > 0 {
		template, err := s.requestRepo.GetByID(ticket.RequestTemplateID)
		if err == nil {
			stages := buildApprovalStagesFromTemplate(template)
			if len(stages) > 0 {
				return &model.ApprovalPolicy{
					ID:     0,
					Name:   template.Name,
					Scope:  template.TicketKind,
					Stages: stages,
				}, stages, nil
			}
		}
	}
	return nil, nil, errors.New("审批策略不存在")
}

func findStageByNo(stages []model.ApprovalPolicyStage, stageNo int) *model.ApprovalPolicyStage {
	for i := range stages {
		if stages[i].StageNo == stageNo {
			stage := stages[i]
			return &stage
		}
	}
	return nil
}

func findNextStage(stages []model.ApprovalPolicyStage, currentStageNo int) *model.ApprovalPolicyStage {
	filtered := make([]model.ApprovalPolicyStage, 0, len(stages))
	for _, stage := range stages {
		if stage.StageNo > currentStageNo {
			filtered = append(filtered, stage)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].StageNo == filtered[j].StageNo {
			return filtered[i].Sort < filtered[j].Sort
		}
		return filtered[i].StageNo < filtered[j].StageNo
	})
	stage := filtered[0]
	return &stage
}

func stageApproved(passRule string, records []*model.ApprovalRecord) bool {
	if len(records) == 0 {
		return false
	}
	switch passRule {
	case "any":
		for _, record := range records {
			if record.Action == "approve" {
				return true
			}
		}
		return false
	default:
		for _, record := range records {
			if record.Action != "approve" {
				return false
			}
		}
		return true
	}
}

func (s *ApprovalService) resolveApproverIDs(stage *model.ApprovalPolicyStage, ticket *model.Ticket, operatorID int64) ([]int64, error) {
	var cfg approvalApproverConfig
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
	return dedupeApprovalIDs(ids), nil
}

func dedupeApprovalIDs(items []int64) []int64 {
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

// resolveAssigneeAfterApproval 审批通过后自动指派处理人。
// 优先级：模板 autoAssignRule → 工单类型 autoAssignRule → 不指派
func (s *ApprovalService) resolveAssigneeAfterApproval(ticket *model.Ticket) int64 {
	// 从模板获取指派规则
	if ticket.RequestTemplateID > 0 {
		tpl, err := s.requestRepo.GetByID(ticket.RequestTemplateID)
		if err == nil && tpl.AutoAssignRule != "" && tpl.AutoAssignRule != "manual" {
			if id := s.resolveAssignee(tpl.AutoAssignRule, tpl.DefaultAssignee, ticket); id > 0 {
				return id
			}
		}
	}
	return 0
}

func (s *ApprovalService) resolveAssignee(rule string, defaultAssignee int64, ticket *model.Ticket) int64 {
	switch rule {
	case "resource_owner":
		if ticket.ResourceType == "asset" && ticket.ResourceID > 0 {
			assetRepo := repository.NewAssetRepository()
			if asset, err := assetRepo.GetByID(ticket.ResourceID); err == nil {
				var ownerIDs []int64
				_ = json.Unmarshal([]byte(asset.OwnerIDs), &ownerIDs)
				if len(ownerIDs) > 0 {
					return ownerIDs[0]
				}
			}
		}
	case "service_owner":
		if ticket.ServiceTreeID > 0 {
			if node, err := s.treeRepo.GetByID(ticket.ServiceTreeID); err == nil && node.OwnerID > 0 {
				return node.OwnerID
			}
		}
	case "dept_default":
		if defaultAssignee > 0 {
			return defaultAssignee
		}
	}
	return 0
}
