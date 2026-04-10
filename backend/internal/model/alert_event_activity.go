package model

type AlertEventActivity struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	EventID    int64     `gorm:"index;not null" json:"event_id"`
	FromStatus string    `gorm:"size:20;not null" json:"from_status"`
	ToStatus   string    `gorm:"size:20;not null" json:"to_status"`
	OperatorID int64     `gorm:"index;default:0" json:"operator_id"`
	Note       string    `gorm:"size:500" json:"note"`
	CreatedAt  LocalTime `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
}

func (AlertEventActivity) TableName() string {
	return "alert_event_activities"
}
