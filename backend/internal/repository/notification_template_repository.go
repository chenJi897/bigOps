package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type NotificationTemplateRepository struct{}

func NewNotificationTemplateRepository() *NotificationTemplateRepository {
	return &NotificationTemplateRepository{}
}

func (r *NotificationTemplateRepository) List() ([]*model.NotificationTemplate, error) {
	var items []*model.NotificationTemplate
	if err := database.GetDB().Order("event_type ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *NotificationTemplateRepository) GetByEventType(eventType string) (*model.NotificationTemplate, error) {
	var item model.NotificationTemplate
	if err := database.GetDB().Where("event_type = ?", eventType).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *NotificationTemplateRepository) GetByID(id int64) (*model.NotificationTemplate, error) {
	var item model.NotificationTemplate
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *NotificationTemplateRepository) Upsert(item *model.NotificationTemplate) error {
	var existing model.NotificationTemplate
	err := database.GetDB().Where("event_type = ?", item.EventType).First(&existing).Error
	if err == nil {
		existing.Title = item.Title
		existing.Content = item.Content
		existing.Variables = item.Variables
		return database.GetDB().Save(&existing).Error
	}
	return database.GetDB().Create(item).Error
}

func (r *NotificationTemplateRepository) Update(item *model.NotificationTemplate) error {
	return database.GetDB().Save(item).Error
}
