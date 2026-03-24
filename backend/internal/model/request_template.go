package model

import "gorm.io/gorm"

// RequestTemplate 请求模板/服务目录项。
type RequestTemplate struct {
	ID                  int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                string         `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Code                string         `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Category            string         `gorm:"size:30;index;not null" json:"category"` // resource/access/change/other
	Description         string         `gorm:"size:500" json:"description"`
	Icon                string         `gorm:"size:50" json:"icon"`
	TypeID              int64          `gorm:"index;not null" json:"type_id"`
	TypeName            string         `gorm:"-" json:"type_name,omitempty"`
	FormSchema          string         `gorm:"type:json" json:"form_schema"`
	ApprovalPolicyID    int64          `gorm:"index" json:"approval_policy_id"`
	ApprovalPolicyName  string         `gorm:"-" json:"approval_policy_name,omitempty"`
	ExecutionTemplate   string         `gorm:"size:100" json:"execution_template"`
	TicketKind          string         `gorm:"size:20;default:request" json:"ticket_kind"` // request/change
	AutoCreateOrder     int8           `gorm:"default:1" json:"auto_create_order"`
	NotifyApplicant     int8           `gorm:"default:1" json:"notify_applicant"`
	Status              int8           `gorm:"default:1;index" json:"status"`
	Sort                int            `gorm:"default:0" json:"sort"`
	CreatedAt           LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt           LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

func (RequestTemplate) TableName() string {
	return "request_templates"
}
