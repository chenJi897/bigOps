package service

import (
	"errors"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

// MenuService 菜单管理服务。
type MenuService struct {
	menuRepo *repository.MenuRepository
}

func NewMenuService() *MenuService {
	return &MenuService{
		menuRepo: repository.NewMenuRepository(),
	}
}

// Create 创建菜单。
func (s *MenuService) Create(menu *model.Menu) error {
	return s.menuRepo.Create(menu)
}

// Update 更新菜单。
func (s *MenuService) Update(menu *model.Menu) error {
	existing, err := s.menuRepo.GetByID(menu.ID)
	if err != nil {
		return errors.New("菜单不存在")
	}
	existing.ParentID = menu.ParentID
	existing.Name = menu.Name
	existing.Title = menu.Title
	existing.Icon = menu.Icon
	existing.Path = menu.Path
	existing.Component = menu.Component
	existing.APIPath = menu.APIPath
	existing.APIMethod = menu.APIMethod
	existing.Type = menu.Type
	existing.Sort = menu.Sort
	existing.Visible = menu.Visible
	if menu.Status != 0 {
		existing.Status = menu.Status
	}
	return s.menuRepo.Update(existing)
}

// Delete 删除菜单。
func (s *MenuService) Delete(id int64) error {
	return s.menuRepo.Delete(id)
}

// GetByID 获取菜单详情。
func (s *MenuService) GetByID(id int64) (*model.Menu, error) {
	return s.menuRepo.GetByID(id)
}

// GetTree 获取完整菜单树。
func (s *MenuService) GetTree() ([]*model.Menu, error) {
	menus, err := s.menuRepo.GetAll()
	if err != nil {
		return nil, err
	}
	return buildMenuTree(menus, 0), nil
}

// GetTreeByIDs 根据菜单 ID 列表构建菜单树（用于角色的菜单树）。
func (s *MenuService) GetTreeByIDs(ids []int64) ([]*model.Menu, error) {
	menus, err := s.menuRepo.GetByIDs(ids)
	if err != nil {
		return nil, err
	}
	return buildMenuTree(menus, 0), nil
}

// buildMenuTree 将平铺的菜单列表构建为树形结构。
func buildMenuTree(menus []*model.Menu, parentID int64) []*model.Menu {
	var tree []*model.Menu
	for _, menu := range menus {
		if menu.ParentID == parentID {
			children := buildMenuTree(menus, menu.ID)
			if len(children) > 0 {
				// 将 []*Menu 转为 []Menu 以匹配模型字段类型
				childSlice := make([]model.Menu, len(children))
				for i, c := range children {
					childSlice[i] = *c
				}
				menu.Children = childSlice
			}
			tree = append(tree, menu)
		}
	}
	return tree
}
