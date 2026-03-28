package model

import "gorm.io/gorm"

// AlertSilence 告警静默/抑制规则。
type AlertSilence struct {
	ID            int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string         `gorm:"size:120;not null;uniqueIndex" json:"name"`
	RuleID        int64          `gorm:"index;default:0" json:"rule_id"`
	AgentID       string         `gorm:"size:100;index" json:"agent_id"`
	ServiceTreeID int64          `gorm:"index;default:0" json:"service_tree_id"`
	OwnerID       int64          `gorm:"index;default:0" json:"owner_id"`
	Reason        string         `gorm:"size:500" json:"reason"`
	Enabled       int8           `gorm:"default:1;index" json:"enabled"`
	StartsAt      LocalTime      `json:"starts_at"`
	EndsAt        LocalTime      `json:"ends_at"`
	CreatedBy     int64          `gorm:"index;default:0" json:"created_by"`
	UpdatedBy     int64          `gorm:"index;default:0" json:"updated_by"`
	CreatedAt     LocalTime      `json:"created_at"`
	UpdatedAt     LocalTime      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AlertSilence) TableName() string {
	return "alert_silences"
}
