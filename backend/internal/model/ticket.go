package model

import "gorm.io/gorm"

// Ticket 工单模型，对应 tickets 表。
type Ticket struct {
	ID             int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Title          string         `gorm:"size:200;not null" json:"title"`
	TicketNo       string         `gorm:"size:50;uniqueIndex;not null" json:"ticket_no"` // TK-20260324-0001
	TypeID         int64          `gorm:"index;not null" json:"type_id"`
	TypeName       string         `gorm:"-" json:"type_name,omitempty"`
	TicketKind     string         `gorm:"size:20;index;default:incident" json:"ticket_kind"` // incident/request/change
	RequestTemplateID   int64     `gorm:"index" json:"request_template_id"`
	RequestTemplateName string    `gorm:"-" json:"request_template_name,omitempty"`

	// 状态
	Status   string `gorm:"size:20;not null;index;default:open" json:"status"`  // open/processing/resolved/closed/rejected
	Priority string `gorm:"size:20;not null;default:medium" json:"priority"`    // low/medium/high/urgent
	ApprovalStatus    string `gorm:"size:20;index;default:not_required" json:"approval_status"` // not_required/pending/in_progress/approved/rejected/canceled
	ApprovalInstanceID int64 `gorm:"index" json:"approval_instance_id"`
	ExecutionStatus   string `gorm:"size:20;index;default:not_started" json:"execution_status"` // not_started/pending/running/succeeded/failed/canceled

	// 内容
	Description string `gorm:"type:text" json:"description"`
	ExtraFields string `gorm:"type:json" json:"extra_fields"`

	// 来源
	Source          string `gorm:"size:20;not null;default:manual;index" json:"source"` // manual/monitor/sync/system/cicd
	SourceEventType string `gorm:"size:50" json:"source_event_type"`                   // alert/sync_failed/pipeline_failed/k8s_event
	SourceEventID   string `gorm:"size:100;index" json:"source_event_id"`
	DedupeKey       string `gorm:"size:200;index" json:"dedupe_key"`

	// 人员
	CreatorID    int64  `gorm:"index;not null" json:"creator_id"`
	CreatorName  string `gorm:"-" json:"creator_name,omitempty"`
	AssigneeID   int64  `gorm:"index" json:"assignee_id"`
	AssigneeName string `gorm:"-" json:"assignee_name,omitempty"`

	// 部门
	SubmitDeptID   int64  `gorm:"index" json:"submit_dept_id"`
	SubmitDeptName string `gorm:"-" json:"submit_dept_name,omitempty"`
	HandleDeptID   int64  `gorm:"index" json:"handle_dept_id"`
	HandleDeptName string `gorm:"-" json:"handle_dept_name,omitempty"`

	// 通用资源关联
	ResourceType string `gorm:"size:50;index" json:"resource_type"` // asset/cloud_account/service_tree/k8s_pod/database/pipeline
	ResourceID   int64  `gorm:"index" json:"resource_id"`
	ResourceName string `gorm:"size:200" json:"resource_name"`

	// 快捷访问
	ServiceTreeID int64 `gorm:"index" json:"service_tree_id"`

	// 处理结果
	Resolution     string `gorm:"size:20" json:"resolution"`       // fixed/wontfix/duplicate/invalid/workaround
	ResolutionNote string `gorm:"type:text" json:"resolution_note"`

	// SLA 预留
	SLADeadline *LocalTime `json:"sla_deadline" swaggertype:"string" example:"2024-01-01 00:00:00"`
	RespondedAt *LocalTime `json:"responded_at" swaggertype:"string" example:"2024-01-01 00:00:00"`

	// 时间
	ResolvedAt *LocalTime     `json:"resolved_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	ClosedAt   *LocalTime     `json:"closed_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	CreatedAt  LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt  LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Ticket) TableName() string {
	return "tickets"
}

// BeforeSave 确保 JSON 字段合法。
func (t *Ticket) BeforeSave(tx *gorm.DB) error {
	if t.ExtraFields == "" {
		t.ExtraFields = "{}"
	}
	return nil
}
