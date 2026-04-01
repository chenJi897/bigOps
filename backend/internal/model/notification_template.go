package model

import "gorm.io/gorm"

type NotificationTemplate struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	EventType string         `gorm:"size:50;uniqueIndex;not null" json:"event_type"`
	Title     string         `gorm:"size:500;not null" json:"title"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	Variables string         `gorm:"type:text" json:"variables"`
	IsDefault int8           `gorm:"default:1;not null" json:"is_default"`
	CreatedAt LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (NotificationTemplate) TableName() string {
	return "notification_templates"
}
