package model

// AuditLog 操作审计日志模型，对应 audit_logs 表。
type AuditLog struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"index;not null;default:0" json:"user_id"`
	Username   string    `gorm:"size:50;index;not null;default:''" json:"username"`
	Action     string    `gorm:"size:50;index;not null" json:"action"`     // create/update/delete/login/logout
	Resource   string    `gorm:"size:50;index;not null" json:"resource"`   // user/role/menu
	ResourceID int64     `gorm:"default:0" json:"resource_id"`
	Detail     string    `gorm:"size:500" json:"detail"`
	IP         string    `gorm:"size:50" json:"ip"`
	StatusCode int       `gorm:"default:0" json:"status_code"`
	CreatedAt  LocalTime `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
