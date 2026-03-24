package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type TicketTypeService struct {
	repo     *repository.TicketTypeRepository
	deptRepo *repository.DepartmentRepository
}

func NewTicketTypeService() *TicketTypeService {
	return &TicketTypeService{
		repo:     repository.NewTicketTypeRepository(),
		deptRepo: repository.NewDepartmentRepository(),
	}
}

func (s *TicketTypeService) Create(tt *model.TicketType) error {
	if tt.Name == "" {
		return errors.New("工单类型名称不能为空")
	}
	_, err := s.repo.GetByName(tt.Name)
	if err == nil {
		return errors.New("工单类型名称已存在")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询失败: %w", err)
	}
	return s.repo.Create(tt)
}

func (s *TicketTypeService) Update(id int64, tt *model.TicketType) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("工单类型不存在")
	}
	if tt.Name != existing.Name {
		dup, err := s.repo.GetByName(tt.Name)
		if err == nil && dup.ID != id {
			return errors.New("工单类型名称已存在")
		}
	}
	existing.Name = tt.Name
	existing.Code = tt.Code
	existing.Icon = tt.Icon
	existing.Description = tt.Description
	existing.HandleDeptID = tt.HandleDeptID
	existing.DefaultAssignee = tt.DefaultAssignee
	existing.Priority = tt.Priority
	existing.AutoAssignRule = tt.AutoAssignRule
	existing.Sort = tt.Sort
	if tt.Status != 0 {
		existing.Status = tt.Status
	}
	return s.repo.Update(existing)
}

func (s *TicketTypeService) Delete(id int64) error {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("工单类型不存在")
	}
	return s.repo.Delete(id)
}

func (s *TicketTypeService) GetByID(id int64) (*model.TicketType, error) {
	tt, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	s.fillExtra(tt)
	return tt, nil
}

func (s *TicketTypeService) List(page, size int) ([]*model.TicketType, int64, error) {
	items, total, err := s.repo.List(page, size)
	if err != nil {
		return nil, 0, err
	}
	for _, tt := range items {
		s.fillExtra(tt)
	}
	return items, total, nil
}

func (s *TicketTypeService) GetAll() ([]*model.TicketType, error) {
	return s.repo.GetAll()
}

func (s *TicketTypeService) fillExtra(tt *model.TicketType) {
	if tt.HandleDeptID > 0 {
		if dept, err := s.deptRepo.GetByID(tt.HandleDeptID); err == nil {
			tt.HandleDeptName = dept.Name
		}
	}
}
