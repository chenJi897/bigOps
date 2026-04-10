package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type AlertEventActivityRepository struct{}

func NewAlertEventActivityRepository() *AlertEventActivityRepository {
	return &AlertEventActivityRepository{}
}

func (r *AlertEventActivityRepository) Create(item *model.AlertEventActivity) error {
	return database.GetDB().Create(item).Error
}

func (r *AlertEventActivityRepository) ListByEventID(eventID int64) ([]*model.AlertEventActivity, error) {
	var items []*model.AlertEventActivity
	if err := database.GetDB().
		Where("event_id = ?", eventID).
		Order("id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
