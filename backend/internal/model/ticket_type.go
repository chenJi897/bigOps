package model

import "gorm.io/gorm"

// TicketType 工单类型模型，对应 ticket_types 表。
type TicketType struct {
	ID              int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"size:100;uniqueIndex;not null" json:"name"` // 故障报修/权限申请/资源申请
	Code            string         `gorm:"size:50;index" json:"code"`                 // incident/access/resource/change
	Icon            string         `gorm:"size:50" json:"icon"`
	Description     string         `gorm:"size:500" json:"description"`
	HandleDeptID    int64          `gorm:"default:0" json:"handle_dept_id"`
	HandleDeptName  string         `gorm:"-" json:"handle_dept_name,omitempty"`
	DefaultAssignee int64          `gorm:"default:0" json:"default_assignee"`
	Priority        string         `gorm:"size:20;default:medium" json:"priority"`
	AutoAssignRule  string         `gorm:"size:50;default:manual" json:"auto_assign_rule"` // manual/resource_owner/service_owner/dept_default
	Sort            int            `gorm:"default:0" json:"sort"`
	Status          int8           `gorm:"default:1" json:"status"`
	CreatedAt       LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt       LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (TicketType) TableName() string {
	return "ticket_types"
}
