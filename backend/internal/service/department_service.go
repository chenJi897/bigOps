package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type DepartmentService struct {
	repo     *repository.DepartmentRepository
	userRepo *repository.UserRepository
}

func NewDepartmentService() *DepartmentService {
	return &DepartmentService{
		repo:     repository.NewDepartmentRepository(),
		userRepo: repository.NewUserRepository(),
	}
}

func (s *DepartmentService) Create(dept *model.Department) error {
	if dept.Name == "" {
		return errors.New("部门名称不能为空")
	}
	_, err := s.repo.GetByName(dept.Name)
	if err == nil {
		return errors.New("部门名称已存在")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询失败: %w", err)
	}
	return s.repo.Create(dept)
}

func (s *DepartmentService) Update(id int64, dept *model.Department) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("部门不存在")
	}
	// 名称唯一校验（排除自身）
	if dept.Name != existing.Name {
		dup, err := s.repo.GetByName(dept.Name)
		if err == nil && dup.ID != id {
			return errors.New("部门名称已存在")
		}
	}
	existing.Name = dept.Name
	existing.Code = dept.Code
	existing.Description = dept.Description
	existing.ManagerID = dept.ManagerID
	existing.Sort = dept.Sort
	if dept.Status != 0 {
		existing.Status = dept.Status
	}
	return s.repo.Update(existing)
}

func (s *DepartmentService) Delete(id int64) error {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("部门不存在")
	}
	count, err := s.repo.CountUsersByDepartmentID(id)
	if err != nil {
		return fmt.Errorf("查询关联用户失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("该部门下有 %d 个用户，请先转移后再删除", count)
	}
	return s.repo.Delete(id)
}

func (s *DepartmentService) GetByID(id int64) (*model.Department, error) {
	dept, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	s.fillExtra(dept)
	return dept, nil
}

func (s *DepartmentService) List(page, size int) ([]*model.Department, int64, error) {
	departments, total, err := s.repo.List(page, size)
	if err != nil {
		return nil, 0, err
	}
	for _, d := range departments {
		s.fillExtra(d)
	}
	return departments, total, nil
}

func (s *DepartmentService) GetAll() ([]*model.Department, error) {
	return s.repo.GetAll()
}

func (s *DepartmentService) fillExtra(dept *model.Department) {
	// 填充负责人名称
	if dept.ManagerID > 0 {
		if user, err := s.userRepo.GetByID(dept.ManagerID); err == nil {
			dept.ManagerName = user.RealName
			if dept.ManagerName == "" {
				dept.ManagerName = user.Username
			}
		}
	}
	// 填充用户数
	if count, err := s.repo.CountUsersByDepartmentID(dept.ID); err == nil {
		dept.UserCount = count
	}
}
