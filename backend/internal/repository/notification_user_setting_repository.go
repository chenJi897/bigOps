package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type NotificationUserSettingRepository struct{}

func NewNotificationUserSettingRepository() *NotificationUserSettingRepository {
	return &NotificationUserSettingRepository{}
}

func (r *NotificationUserSettingRepository) GetByUserID(userID int64) (*model.NotificationUserSetting, error) {
	var item model.NotificationUserSetting
	if err := database.GetDB().Where("user_id = ?", userID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *NotificationUserSettingRepository) Upsert(item *model.NotificationUserSetting) error {
	var existing model.NotificationUserSetting
	err := database.GetDB().Where("user_id = ?", item.UserID).First(&existing).Error
	if err == nil {
		existing.EnabledChannels = item.EnabledChannels
		existing.SubscribedBizTypes = item.SubscribedBizTypes
		existing.Enabled = item.Enabled
		return database.GetDB().Save(&existing).Error
	}
	return database.GetDB().Create(item).Error
}

func (r *NotificationUserSettingRepository) ListByUserIDs(userIDs []int64) (map[int64]*model.NotificationUserSetting, error) {
	result := make(map[int64]*model.NotificationUserSetting)
	if len(userIDs) == 0 {
		return result, nil
	}
	var items []*model.NotificationUserSetting
	if err := database.GetDB().Where("user_id IN ?", userIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.UserID] = item
	}
	return result, nil
}
