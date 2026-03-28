package model

import "gorm.io/gorm"

// OnCallSchedule 值班表。
type OnCallSchedule struct {
	ID                   int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                 string         `gorm:"size:120;not null;uniqueIndex" json:"name"`
	Description          string         `gorm:"size:500" json:"description"`
	Timezone             string         `gorm:"size:64;default:Asia/Shanghai" json:"timezone"`
	UsersJSON            string         `gorm:"type:json" json:"users_json"`
	RotationDays         int            `gorm:"default:1" json:"rotation_days"`
	NotifyChannelsJSON   string         `gorm:"type:json" json:"notify_channels_json"`
	EscalationMinutes    int            `gorm:"default:0" json:"escalation_minutes"`
	Enabled              int8           `gorm:"default:1;index" json:"enabled"`
	CreatedBy            int64          `gorm:"index;default:0" json:"created_by"`
	UpdatedBy            int64          `gorm:"index;default:0" json:"updated_by"`
	CreatedAt            LocalTime      `json:"created_at"`
	UpdatedAt            LocalTime      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

func (OnCallSchedule) TableName() string {
	return "oncall_schedules"
}

func (s *OnCallSchedule) BeforeSave(tx *gorm.DB) error {
	if s.UsersJSON == "" {
		s.UsersJSON = "[]"
	}
	if s.NotifyChannelsJSON == "" {
		s.NotifyChannelsJSON = "[]"
	}
	if s.Timezone == "" {
		s.Timezone = "Asia/Shanghai"
	}
	if s.RotationDays <= 0 {
		s.RotationDays = 1
	}
	return nil
}
