package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type OnCallScheduleRepository struct{}

func NewOnCallScheduleRepository() *OnCallScheduleRepository {
	return &OnCallScheduleRepository{}
}

func (r *OnCallScheduleRepository) Create(item *model.OnCallSchedule) error {
	return database.GetDB().Create(item).Error
}

func (r *OnCallScheduleRepository) Update(item *model.OnCallSchedule) error {
	return database.GetDB().Save(item).Error
}

func (r *OnCallScheduleRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.OnCallSchedule{}, id).Error
}

func (r *OnCallScheduleRepository) GetByID(id int64) (*model.OnCallSchedule, error) {
	var item model.OnCallSchedule
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *OnCallScheduleRepository) List() ([]*model.OnCallSchedule, error) {
	var items []*model.OnCallSchedule
	if err := database.GetDB().Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
