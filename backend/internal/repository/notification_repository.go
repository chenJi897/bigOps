package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type NotificationRepository struct{}

func NewNotificationRepository() *NotificationRepository {
	return &NotificationRepository{}
}

func (r *NotificationRepository) CreateEvent(item *model.NotificationEvent) error {
	return database.GetDB().Create(item).Error
}

func (r *NotificationRepository) CreateDelivery(item *model.NotificationDelivery) error {
	return database.GetDB().Create(item).Error
}

func (r *NotificationRepository) CreateInApp(item *model.InAppNotification) error {
	return database.GetDB().Create(item).Error
}

func (r *NotificationRepository) ListInAppByUserID(userID int64, unreadOnly bool) ([]*model.InAppNotification, error) {
	var items []*model.InAppNotification
	db := database.GetDB().Where("user_id = ?", userID)
	if unreadOnly {
		db = db.Where("read_at IS NULL")
	}
	if err := db.Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *NotificationRepository) CountUnreadByUserID(userID int64) (int64, error) {
	var count int64
	err := database.GetDB().Model(&model.InAppNotification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Count(&count).Error
	return count, err
}

func (r *NotificationRepository) GetInAppByID(id int64) (*model.InAppNotification, error) {
	var item model.InAppNotification
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *NotificationRepository) MarkRead(id int64, readAt model.LocalTime) error {
	return database.GetDB().Model(&model.InAppNotification{}).
		Where("id = ? AND read_at IS NULL", id).
		Update("read_at", readAt).Error
}

func (r *NotificationRepository) GetEventByID(id int64) (*model.NotificationEvent, error) {
	var item model.NotificationEvent
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *NotificationRepository) ListEvents(limit int) ([]*model.NotificationEvent, error) {
	var items []*model.NotificationEvent
	db := database.GetDB().Model(&model.NotificationEvent{}).Order("id DESC")
	if limit > 0 {
		db = db.Limit(limit)
	}
	if err := db.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *NotificationRepository) ListDeliveriesByEventID(eventID int64, pendingOnly bool) ([]*model.NotificationDelivery, error) {
	var items []*model.NotificationDelivery
	db := database.GetDB().Where("event_id = ?", eventID)
	if pendingOnly {
		db = db.Where("status = ?", "pending")
	}
	if err := db.Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *NotificationRepository) ListDeliveriesByEventIDAndStatuses(eventID int64, statuses []string) ([]*model.NotificationDelivery, error) {
	if len(statuses) == 0 {
		return r.ListDeliveriesByEventID(eventID, false)
	}
	var items []*model.NotificationDelivery
	if err := database.GetDB().
		Where("event_id = ? AND status IN ?", eventID, statuses).
		Order("id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *NotificationRepository) ListRetryableDeliveries(now model.LocalTime, limit int) ([]*model.NotificationDelivery, error) {
	var items []*model.NotificationDelivery
	db := database.GetDB().Where(
		"status IN ? AND (next_retry_at IS NULL OR next_retry_at <= ?)",
		[]string{"pending", "failed"},
		now,
	).Order("id ASC")
	if limit > 0 {
		db = db.Limit(limit)
	}
	if err := db.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *NotificationRepository) UpdateDelivery(item *model.NotificationDelivery) error {
	return database.GetDB().Save(item).Error
}

func (r *NotificationRepository) UpdateEvent(item *model.NotificationEvent) error {
	return database.GetDB().Save(item).Error
}
