package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type RequestTemplateService struct {
	repo       *repository.RequestTemplateRepository
	policyRepo *repository.ApprovalPolicyRepository
	typeRepo   *repository.TicketTypeRepository
}

func NewRequestTemplateService() *RequestTemplateService {
	return &RequestTemplateService{
		repo:       repository.NewRequestTemplateRepository(),
		policyRepo: repository.NewApprovalPolicyRepository(),
		typeRepo:   repository.NewTicketTypeRepository(),
	}
}

func (s *RequestTemplateService) Create(item *model.RequestTemplate) error {
	if item.Name == "" {
		return errors.New("请求模板名称不能为空")
	}
	if item.Code == "" {
		return errors.New("请求模板编码不能为空")
	}
	if item.TypeID == 0 {
		return errors.New("请求模板必须绑定工单类型")
	}
	if item.Category == "" {
		item.Category = "other"
	}
	if item.TicketKind == "" {
		item.TicketKind = "request"
	}
	if item.ApprovalPolicyID > 0 {
		if _, err := s.policyRepo.GetByID(item.ApprovalPolicyID); err != nil {
			return errors.New("审批策略不存在")
		}
	}
	if _, err := s.typeRepo.GetByID(item.TypeID); err != nil {
		return errors.New("工单类型不存在")
	}
	if _, err := s.repo.GetByName(item.Name); err == nil {
		return errors.New("请求模板名称已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询失败: %w", err)
	}
	if _, err := s.repo.GetByCode(item.Code); err == nil {
		return errors.New("请求模板编码已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询失败: %w", err)
	}
	return s.repo.Create(item)
}

func (s *RequestTemplateService) Update(id int64, item *model.RequestTemplate) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("请求模板不存在")
	}
	if item.Name == "" {
		return errors.New("请求模板名称不能为空")
	}
	if item.Code == "" {
		return errors.New("请求模板编码不能为空")
	}
	if item.TypeID == 0 {
		return errors.New("请求模板必须绑定工单类型")
	}
	if item.ApprovalPolicyID > 0 {
		if _, err := s.policyRepo.GetByID(item.ApprovalPolicyID); err != nil {
			return errors.New("审批策略不存在")
		}
	}
	if _, err := s.typeRepo.GetByID(item.TypeID); err != nil {
		return errors.New("工单类型不存在")
	}
	if item.Name != existing.Name {
		if dup, err := s.repo.GetByName(item.Name); err == nil && dup.ID != id {
			return errors.New("请求模板名称已存在")
		}
	}
	if item.Code != existing.Code {
		if dup, err := s.repo.GetByCode(item.Code); err == nil && dup.ID != id {
			return errors.New("请求模板编码已存在")
		}
	}
	existing.Name = item.Name
	existing.Code = item.Code
	existing.Category = item.Category
	existing.Description = item.Description
	existing.Icon = item.Icon
	existing.TypeID = item.TypeID
	existing.FormSchema = item.FormSchema
	existing.ApprovalPolicyID = item.ApprovalPolicyID
	existing.ExecutionTemplate = item.ExecutionTemplate
	existing.TicketKind = item.TicketKind
	existing.AutoCreateOrder = item.AutoCreateOrder
	existing.NotifyApplicant = item.NotifyApplicant
	existing.Sort = item.Sort
	if item.Status != 0 {
		existing.Status = item.Status
	}
	return s.repo.Update(existing)
}

func (s *RequestTemplateService) Delete(id int64) error {
	if _, err := s.repo.GetByID(id); err != nil {
		return errors.New("请求模板不存在")
	}
	return s.repo.Delete(id)
}

func (s *RequestTemplateService) GetByID(id int64) (*model.RequestTemplate, error) {
	item, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	s.fillExtra(item)
	return item, nil
}

func (s *RequestTemplateService) List(enabledOnly bool) ([]*model.RequestTemplate, error) {
	items, err := s.repo.List(enabledOnly)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		s.fillExtra(item)
	}
	return items, nil
}

func (s *RequestTemplateService) fillExtra(item *model.RequestTemplate) {
	if item.ApprovalPolicyID > 0 {
		if policy, err := s.policyRepo.GetByID(item.ApprovalPolicyID); err == nil {
			item.ApprovalPolicyName = policy.Name
		}
	}
	if item.TypeID > 0 {
		if tt, err := s.typeRepo.GetByID(item.TypeID); err == nil {
			item.TypeName = tt.Name
		}
	}
}
