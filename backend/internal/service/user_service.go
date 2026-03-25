package service

import (
	"errors"
	"fmt"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type UserService struct {
	userRepo *repository.UserRepository
	deptRepo *repository.DepartmentRepository
}

func NewUserService() *UserService {
	return &UserService{
		userRepo: repository.NewUserRepository(),
		deptRepo: repository.NewDepartmentRepository(),
	}
}

// List 用户列表，批量填充部门名称（消除 N+1 查询）
func (s *UserService) List(page, size int, keyword string) ([]*model.User, int64, error) {
	users, total, err := s.userRepo.List(page, size, keyword)
	if err != nil {
		return nil, 0, err
	}
	// 批量收集 department IDs
	ids := make([]int64, 0)
	seen := make(map[int64]bool)
	for _, u := range users {
		if u.DepartmentID > 0 && !seen[u.DepartmentID] {
			ids = append(ids, u.DepartmentID)
			seen[u.DepartmentID] = true
		}
	}
	// 单次批量查询
	if len(ids) > 0 {
		deptMap, _ := s.deptRepo.GetByIDs(ids)
		for _, u := range users {
			if dept, ok := deptMap[u.DepartmentID]; ok {
				u.DepartmentName = dept.Name
			}
		}
	}
	return users, total, nil
}

// UpdateUserParams 更新用户参数
type UpdateUserParams struct {
	RealName     string
	Phone        string
	Email        string
	DepartmentID int64
}

func (s *UserService) Update(id int64, params UpdateUserParams) error {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return errors.New("用户不存在")
	}
	user.RealName = params.RealName
	user.Phone = params.Phone
	if params.Email != "" {
		user.Email = &params.Email
	}
	user.DepartmentID = params.DepartmentID
	return s.userRepo.Update(user)
}

func (s *UserService) UpdateStatus(id int64, status int8) (username string, err error) {
	if id == 1 {
		return "", errors.New("不允许禁用管理员")
	}
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return "", errors.New("用户不存在")
	}
	user.Status = status
	if err := s.userRepo.Update(user); err != nil {
		return "", err
	}
	return user.Username, nil
}

func (s *UserService) Delete(id int64) (username string, err error) {
	if id == 1 {
		return "", errors.New("不允许删除管理员")
	}
	user, _ := s.userRepo.GetByID(id)
	if user != nil {
		username = user.Username
	}
	if err := s.userRepo.Delete(id); err != nil {
		return "", err
	}
	return username, nil
}

func (s *UserService) SetDepartment(id int64, deptID int64) (username string, err error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return "", errors.New("用户不存在")
	}
	user.DepartmentID = deptID
	if err := s.userRepo.Update(user); err != nil {
		return "", fmt.Errorf("更新失败: %w", err)
	}
	return user.Username, nil
}
