package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

// RoleRepository 角色数据访问对象。
type RoleRepository struct{}

func NewRoleRepository() *RoleRepository {
	return &RoleRepository{}
}

// Create 创建角色。
func (r *RoleRepository) Create(role *model.Role) error {
	return database.GetDB().Create(role).Error
}

// GetByID 根据 ID 查询角色（预加载菜单列表）。
func (r *RoleRepository) GetByID(id int64) (*model.Role, error) {
	var role model.Role
	if err := database.GetDB().Preload("Menus").First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByName 根据角色标识查询。
func (r *RoleRepository) GetByName(name string) (*model.Role, error) {
	var role model.Role
	if err := database.GetDB().Where("name = ?", name).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// Update 更新角色。
func (r *RoleRepository) Update(role *model.Role) error {
	return database.GetDB().Save(role).Error
}

// Delete 软删除角色。
func (r *RoleRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.Role{}, id).Error
}

// List 分页查询角色列表。
func (r *RoleRepository) List(page, size int) ([]*model.Role, int64, error) {
	var roles []*model.Role
	var total int64
	db := database.GetDB().Model(&model.Role{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("sort ASC, id ASC").Find(&roles).Error; err != nil {
		return nil, 0, err
	}
	return roles, total, nil
}

// SetMenus 设置角色的菜单权限（替换关联）。
func (r *RoleRepository) SetMenus(roleID int64, menuIDs []int64) error {
	role := &model.Role{ID: roleID}
	var menus []model.Menu
	for _, id := range menuIDs {
		menus = append(menus, model.Menu{ID: id})
	}
	return database.GetDB().Model(role).Association("Menus").Replace(menus)
}

// GetMenusByRoleID 查询角色拥有的菜单 ID 列表。
func (r *RoleRepository) GetMenusByRoleID(roleID int64) ([]int64, error) {
	var ids []int64
	err := database.GetDB().Table("role_menus").
		Where("role_id = ?", roleID).
		Pluck("menu_id", &ids).Error
	return ids, err
}

// GetRolesByUserID 查询用户拥有的角色列表。
func (r *RoleRepository) GetRolesByUserID(userID int64) ([]*model.Role, error) {
	var roles []*model.Role
	err := database.GetDB().
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	return roles, err
}

// SetUserRoles 设置用户的角色（替换关联）。
func (r *RoleRepository) SetUserRoles(userID int64, roleIDs []int64) error {
	// 先删除旧关联
	if err := database.GetDB().Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
		return err
	}
	// 插入新关联
	for _, roleID := range roleIDs {
		ur := model.UserRole{UserID: userID, RoleID: roleID}
		if err := database.GetDB().Create(&ur).Error; err != nil {
			return err
		}
	}
	return nil
}
