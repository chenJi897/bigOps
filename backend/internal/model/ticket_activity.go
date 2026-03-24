package model

// TicketActivity 工单活动流，对应 ticket_activities 表。
// 记录工单生命周期内的所有操作和事件（人工+系统）。
type TicketActivity struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TicketID  int64     `gorm:"index;not null" json:"ticket_id"`
	UserID    int64     `gorm:"not null" json:"user_id"`
	UserName  string    `gorm:"-" json:"user_name,omitempty"`
	Type      string    `gorm:"size:20;not null;index" json:"type"` // create/assign/comment/resolve/close/reject/reopen/transfer/sla_warn/auto_create
	Content   string    `gorm:"type:text" json:"content"`
	OldValue  string    `gorm:"size:100" json:"old_value"`
	NewValue  string    `gorm:"size:100" json:"new_value"`
	IsSystem  bool      `gorm:"default:false" json:"is_system"`
	CreatedAt LocalTime `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
}

func (TicketActivity) TableName() string {
	return "ticket_activities"
}
