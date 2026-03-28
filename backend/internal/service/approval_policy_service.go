package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type ApprovalPolicyService struct {
	repo *repository.ApprovalPolicyRepository
}

func NewApprovalPolicyService() *ApprovalPolicyService {
	return &ApprovalPolicyService{repo: repository.NewApprovalPolicyRepository()}
}

func (s *ApprovalPolicyService) Create(item *model.ApprovalPolicy, stages []model.ApprovalPolicyStage) error {
	if item.Name == "" {
		return errors.New("审批策略名称不能为空")
	}
	if item.Code == "" {
		return errors.New("审批策略编码不能为空")
	}
	if len(stages) == 0 {
		return errors.New("审批策略至少需要一个审批阶段")
	}
	if _, err := s.repo.GetByName(item.Name); err == nil {
		return errors.New("审批策略名称已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询失败: %w", err)
	}
	if _, err := s.repo.GetByCode(item.Code); err == nil {
		return errors.New("审批策略编码已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询失败: %w", err)
	}
	if item.Scope == "" {
		item.Scope = "request"
	}
	if err := s.repo.Create(item); err != nil {
		return err
	}
	return s.repo.ReplaceStages(item.ID, stages)
}

func (s *ApprovalPolicyService) Update(id int64, item *model.ApprovalPolicy, stages []model.ApprovalPolicyStage) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("审批策略不存在")
	}
	if item.Name == "" {
		return errors.New("审批策略名称不能为空")
	}
	if item.Code == "" {
		return errors.New("审批策略编码不能为空")
	}
	if len(stages) == 0 {
		return errors.New("审批策略至少需要一个审批阶段")
	}
	if item.Name != existing.Name {
		if dup, err := s.repo.GetByName(item.Name); err == nil && dup.ID != id {
			return errors.New("审批策略名称已存在")
		}
	}
	if item.Code != existing.Code {
		if dup, err := s.repo.GetByCode(item.Code); err == nil && dup.ID != id {
			return errors.New("审批策略编码已存在")
		}
	}
	existing.Name = item.Name
	existing.Code = item.Code
	existing.Description = item.Description
	existing.Scope = item.Scope
	if item.Enabled != 0 {
		existing.Enabled = item.Enabled
	}
	if err := s.repo.Update(existing); err != nil {
		return err
	}
	return s.repo.ReplaceStages(existing.ID, stages)
}

func (s *ApprovalPolicyService) Delete(id int64) error {
	if _, err := s.repo.GetByID(id); err != nil {
		return errors.New("审批策略不存在")
	}
	return s.repo.Delete(id)
}

func (s *ApprovalPolicyService) GetByID(id int64) (*model.ApprovalPolicy, error) {
	return s.repo.GetByID(id)
}

func (s *ApprovalPolicyService) List() ([]*model.ApprovalPolicy, error) {
	return s.repo.List()
}
