package model

import "gorm.io/gorm"

// RequestTemplate 请求模板/服务目录项。
type RequestTemplate struct {
	ID                 int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name               string         `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Code               string         `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Category           string         `gorm:"size:30;index;not null" json:"category"` // release/access/db_release/repo/other
	ProjectName        string         `gorm:"size:100" json:"project_name"`
	EnvironmentName    string         `gorm:"size:100" json:"environment_name"`
	Description        string         `gorm:"size:500" json:"description"`
	Icon               string         `gorm:"size:50" json:"icon"`
	TypeID             int64          `gorm:"index" json:"type_id"` // 兼容保留，新流程不再必填
	TypeName           string         `gorm:"-" json:"type_name,omitempty"`
	FormSchema         string         `gorm:"type:json" json:"form_schema"`
	ApprovalPolicyID   int64          `gorm:"index" json:"approval_policy_id"`
	ApprovalPolicyName string         `gorm:"-" json:"approval_policy_name,omitempty"`
	NodesJSON          string         `gorm:"type:json" json:"nodes_json"`
	ExecutionTemplate  string         `gorm:"size:100" json:"execution_template"`
	TicketKind         string         `gorm:"size:20;default:request" json:"ticket_kind"` // request/change/incident
	Priority           string         `gorm:"size:20;default:medium" json:"priority"`     // low/medium/high/urgent
	HandleDeptID       int64          `gorm:"index" json:"handle_dept_id"`
	AutoAssignRule     string         `gorm:"size:30;default:manual" json:"auto_assign_rule"` // manual/resource_owner/service_owner/dept_default
	DefaultAssignee    int64          `json:"default_assignee"`
	AutoCreateOrder    int8           `gorm:"default:1" json:"auto_create_order"`
	NotifyApplicant    int8           `gorm:"default:1" json:"notify_applicant"`
	Status             int8           `gorm:"default:1;index" json:"status"`
	Sort               int            `gorm:"default:0" json:"sort"`
	CreatedAt          LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt          LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (RequestTemplate) TableName() string {
	return "request_templates"
}
