// backend/internal/repository/service_tree_repository.go
package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type ServiceTreeRepository struct{}

func NewServiceTreeRepository() *ServiceTreeRepository {
	return &ServiceTreeRepository{}
}

func (r *ServiceTreeRepository) Create(node *model.ServiceTree) error {
	return database.GetDB().Create(node).Error
}

func (r *ServiceTreeRepository) GetByID(id int64) (*model.ServiceTree, error) {
	var node model.ServiceTree
	if err := database.GetDB().First(&node, id).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *ServiceTreeRepository) GetByCode(code string) (*model.ServiceTree, error) {
	var node model.ServiceTree
	if err := database.GetDB().Where("code = ?", code).First(&node).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *ServiceTreeRepository) Update(node *model.ServiceTree) error {
	return database.GetDB().Save(node).Error
}

func (r *ServiceTreeRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.ServiceTree{}, id).Error
}

func (r *ServiceTreeRepository) GetAll() ([]*model.ServiceTree, error) {
	var nodes []*model.ServiceTree
	err := database.GetDB().Order("sort ASC, id ASC").Find(&nodes).Error
	return nodes, err
}

func (r *ServiceTreeRepository) GetChildren(parentID int64) ([]*model.ServiceTree, error) {
	var nodes []*model.ServiceTree
	err := database.GetDB().Where("parent_id = ?", parentID).Order("sort ASC, id ASC").Find(&nodes).Error
	return nodes, err
}

func (r *ServiceTreeRepository) HasChildren(id int64) (bool, error) {
	var count int64
	err := database.GetDB().Model(&model.ServiceTree{}).Where("parent_id = ?", id).Count(&count).Error
	return count > 0, err
}
