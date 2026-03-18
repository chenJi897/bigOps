package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

// MenuRepository 菜单数据访问对象。
type MenuRepository struct{}

func NewMenuRepository() *MenuRepository {
	return &MenuRepository{}
}

// Create 创建菜单。
func (r *MenuRepository) Create(menu *model.Menu) error {
	return database.GetDB().Create(menu).Error
}

// GetByID 根据 ID 查询菜单。
func (r *MenuRepository) GetByID(id int64) (*model.Menu, error) {
	var menu model.Menu
	if err := database.GetDB().First(&menu, id).Error; err != nil {
		return nil, err
	}
	return &menu, nil
}

// Update 更新菜单。
func (r *MenuRepository) Update(menu *model.Menu) error {
	return database.GetDB().Save(menu).Error
}

// Delete 删除菜单。
func (r *MenuRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.Menu{}, id).Error
}

// GetAll 查询全部菜单（按排序），用于构建菜单树。
func (r *MenuRepository) GetAll() ([]*model.Menu, error) {
	var menus []*model.Menu
	err := database.GetDB().Order("sort ASC, id ASC").Find(&menus).Error
	return menus, err
}

// GetByIDs 根据 ID 列表批量查询菜单。
func (r *MenuRepository) GetByIDs(ids []int64) ([]*model.Menu, error) {
	var menus []*model.Menu
	err := database.GetDB().Where("id IN ?", ids).Order("sort ASC, id ASC").Find(&menus).Error
	return menus, err
}
