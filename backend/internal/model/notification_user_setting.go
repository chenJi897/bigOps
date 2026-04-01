package model

import "gorm.io/gorm"

// NotificationUserSetting 用户个人通知偏好。
type NotificationUserSetting struct {
	ID                 int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID             int64          `gorm:"uniqueIndex;not null" json:"user_id"`
	EnabledChannels    string         `gorm:"type:json" json:"enabled_channels"`
	SubscribedBizTypes string         `gorm:"type:json" json:"subscribed_biz_types"`
	// ChannelTargets 表示“个人偏好 -> Message Pusher 的通道名”映射
	// 例如：{"dingtalk":"ding-robot-001","wecom":"wecom-bot-xyz"}
	ChannelTargets string `gorm:"type:json" json:"channel_targets"`
	Enabled            int8           `gorm:"default:1;index" json:"enabled"`
	CreatedAt          LocalTime      `json:"created_at"`
	UpdatedAt          LocalTime      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (NotificationUserSetting) TableName() string {
	return "notification_user_settings"
}

func (s *NotificationUserSetting) BeforeSave(tx *gorm.DB) error {
	if s.EnabledChannels == "" {
		s.EnabledChannels = "[]"
	}
	if s.SubscribedBizTypes == "" {
		s.SubscribedBizTypes = "[]"
	}
	if s.ChannelTargets == "" || s.ChannelTargets == "null" {
		s.ChannelTargets = "{}"
	}
	return nil
}
