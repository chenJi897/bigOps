package model

import "gorm.io/gorm"

// ExecutionOrder 执行单。
type ExecutionOrder struct {
	ID                int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	TicketID          int64          `gorm:"index;not null" json:"ticket_id"`
	RequestTemplateID int64          `gorm:"index" json:"request_template_id"`
	OrderType         string         `gorm:"size:40;index;not null" json:"order_type"` // provision/change/access/recycle
	ExecutorType      string         `gorm:"size:30;not null" json:"executor_type"`    // manual/task_center/webhook
	ExecutorRef       string         `gorm:"size:100" json:"executor_ref"`
	Status            string         `gorm:"size:20;index;default:pending" json:"status"`
	Payload           string         `gorm:"type:json" json:"payload"`
	Result            string         `gorm:"type:json" json:"result"`
	ErrorMessage      string         `gorm:"type:text" json:"error_message"`
	StartedAt         *LocalTime     `json:"started_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	FinishedAt        *LocalTime     `json:"finished_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	CreatedAt         LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt         LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ExecutionOrder) TableName() string {
	return "execution_orders"
}
