package model

import "gorm.io/gorm"

// NotificationEvent 通知事件头。
type NotificationEvent struct {
	ID            int64                  `gorm:"primaryKey;autoIncrement" json:"id"`
	EventType     string                 `gorm:"size:50;index;not null" json:"event_type"` // approval_pending/execution_failed/...
	BizType       string                 `gorm:"size:30;index;not null" json:"biz_type"`   // ticket/approval/execution
	BizID         int64                  `gorm:"index;not null" json:"biz_id"`
	Title         string                 `gorm:"size:200" json:"title"`
	Payload       string                 `gorm:"type:json" json:"payload"`
	Status        string                 `gorm:"size:20;index;default:pending" json:"status"`
	StatusSummary string                 `gorm:"-" json:"status_summary,omitempty"`
	CanRetry      bool                   `gorm:"-" json:"can_retry"`
	Deliveries    []NotificationDelivery `gorm:"-" json:"deliveries,omitempty"`
	CreatedAt     LocalTime              `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt     LocalTime              `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt     gorm.DeletedAt         `gorm:"index" json:"-"`
}

func (NotificationEvent) TableName() string {
	return "notification_events"
}

// NotificationDelivery 渠道投递记录。
type NotificationDelivery struct {
	ID            int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	EventID       int64          `gorm:"index;not null" json:"event_id"`
	Channel       string         `gorm:"size:20;index;not null" json:"channel"` // in_app/email/webhook/im
	Recipient     string         `gorm:"size:200;index" json:"recipient"`
	Status        string         `gorm:"size:20;index;default:pending" json:"status"`
	StatusSummary string         `gorm:"-" json:"status_summary,omitempty"`
	CanRetry      bool           `gorm:"-" json:"can_retry"`
	Response      string         `gorm:"type:text" json:"response"`
	RetryCount    int            `gorm:"default:0" json:"retry_count"`
	LastAttemptAt *LocalTime     `json:"last_attempt_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	NextRetryAt   *LocalTime     `json:"next_retry_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	SentAt        *LocalTime     `json:"sent_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	CreatedAt     LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt     LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (NotificationDelivery) TableName() string {
	return "notification_deliveries"
}

// InAppNotification 站内通知。
type InAppNotification struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64          `gorm:"index;not null" json:"user_id"`
	Title     string         `gorm:"size:200;not null" json:"title"`
	Content   string         `gorm:"type:text" json:"content"`
	Level     string         `gorm:"size:20;default:info" json:"level"` // info/success/warning/error
	BizType   string         `gorm:"size:30;index" json:"biz_type"`
	BizID     int64          `gorm:"index" json:"biz_id"`
	ReadAt    *LocalTime     `json:"read_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	CreatedAt LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (InAppNotification) TableName() string {
	return "in_app_notifications"
}
