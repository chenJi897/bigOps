package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	casbinPkg "github.com/bigops/platform/internal/pkg/casbin"
	"github.com/bigops/platform/internal/repository"
)

// RoleService 角色管理服务。
type RoleService struct {
	roleRepo *repository.RoleRepository
	menuRepo *repository.MenuRepository
}

func NewRoleService() *RoleService {
	return &RoleService{
		roleRepo: repository.NewRoleRepository(),
		menuRepo: repository.NewMenuRepository(),
	}
}

// Create 创建角色。
func (s *RoleService) Create(name, displayName, description string, sort int) error {
	_, err := s.roleRepo.GetByName(name)
	if err == nil {
		return errors.New("角色标识已存在")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询角色失败: %w", err)
	}

	role := &model.Role{
		Name:        name,
		DisplayName: displayName,
		Description: description,
		Sort:        sort,
		Status:      1,
	}
	return s.roleRepo.Create(role)
}

// Update 更新角色信息。
func (s *RoleService) Update(id int64, displayName, description string, sort int, status int8) error {
	role, err := s.roleRepo.GetByID(id)
	if err != nil {
		return errors.New("角色不存在")
	}
	role.DisplayName = displayName
	role.Description = description
	role.Sort = sort
	role.Status = status
	return s.roleRepo.Update(role)
}

// Delete 删除角色。
func (s *RoleService) Delete(id int64) error {
	role, err := s.roleRepo.GetByID(id)
	if err != nil {
		return errors.New("角色不存在")
	}
	if role.Name == "admin" {
		return errors.New("不允许删除管理员角色")
	}
	return s.roleRepo.Delete(id)
}

// GetByID 获取角色详情（含菜单列表）。
func (s *RoleService) GetByID(id int64) (*model.Role, error) {
	return s.roleRepo.GetByID(id)
}

// List 分页查询角色列表。
func (s *RoleService) List(page, size int) ([]*model.Role, int64, error) {
	return s.roleRepo.List(page, size)
}

// SetMenus 设置角色的菜单权限，同时同步 Casbin 策略。
func (s *RoleService) SetMenus(roleID int64, menuIDs []int64) error {
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		return errors.New("角色不存在")
	}

	// 更新数据库关联
	if err := s.roleRepo.SetMenus(roleID, menuIDs); err != nil {
		return fmt.Errorf("设置菜单失败: %w", err)
	}

	// 同步 Casbin 策略：先删除该角色的旧策略，再添加新策略
	enforcer := casbinPkg.GetEnforcer()
	enforcer.RemoveFilteredPolicy(0, role.Name)

	menus, err := s.menuRepo.GetByIDs(menuIDs)
	if err != nil {
		return fmt.Errorf("查询菜单失败: %w", err)
	}

	for _, menu := range menus {
		if menu.APIPath != "" && menu.APIMethod != "" {
			enforcer.AddPolicy(role.Name, menu.APIPath, menu.APIMethod)
		}
	}

	return nil
}

// GetUserRoles 获取用户的角色列表。
func (s *RoleService) GetUserRoles(userID int64) ([]*model.Role, error) {
	return s.roleRepo.GetRolesByUserID(userID)
}

// SetUserRoles 设置用户的角色，同时同步 Casbin 用户-角色映射。
func (s *RoleService) SetUserRoles(userID int64, roleIDs []int64, username string) error {
	// 更新数据库关联
	if err := s.roleRepo.SetUserRoles(userID, roleIDs); err != nil {
		return fmt.Errorf("设置用户角色失败: %w", err)
	}

	// 同步 Casbin 用户-角色映射
	enforcer := casbinPkg.GetEnforcer()
	enforcer.DeleteRolesForUser(username)

	for _, roleID := range roleIDs {
		role, err := s.roleRepo.GetByID(roleID)
		if err != nil {
			continue
		}
		enforcer.AddRoleForUser(username, role.Name)
	}

	return nil
}

// GetMenuIDsByUserID 获取用户通过所有角色关联到的菜单 ID 列表（已去重）。
// 返回值：菜单ID列表, 是否是管理员, 错误。
func (s *RoleService) GetMenuIDsByUserID(userID int64) ([]int64, bool, error) {
	roles, err := s.roleRepo.GetRolesByUserID(userID)
	if err != nil {
		return nil, false, err
	}

	for _, role := range roles {
		if role.Name == "admin" {
			return nil, true, nil
		}
	}

	unique := make(map[int64]bool)
	var ids []int64
	for _, role := range roles {
		menuIDs, err := s.roleRepo.GetMenusByRoleID(role.ID)
		if err != nil {
			continue
		}
		for _, id := range menuIDs {
			if !unique[id] {
				unique[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids, false, nil
}
