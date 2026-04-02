package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type NotifyGroupRepository struct{}

func NewNotifyGroupRepository() *NotifyGroupRepository {
	return &NotifyGroupRepository{}
}

func (r *NotifyGroupRepository) List(page, size int, keyword string) ([]*model.NotifyGroup, int64, error) {
	db := database.GetDB().Model(&model.NotifyGroup{})
	if keyword != "" {
		db = db.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []*model.NotifyGroup
	if err := db.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *NotifyGroupRepository) ListAll() ([]*model.NotifyGroup, error) {
	var items []*model.NotifyGroup
	if err := database.GetDB().Where("status = 1").Order("name ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *NotifyGroupRepository) GetByID(id int64) (*model.NotifyGroup, error) {
	var item model.NotifyGroup
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *NotifyGroupRepository) Create(item *model.NotifyGroup) error {
	return database.GetDB().Create(item).Error
}

func (r *NotifyGroupRepository) Update(item *model.NotifyGroup) error {
	return database.GetDB().Save(item).Error
}

func (r *NotifyGroupRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.NotifyGroup{}, id).Error
}
