package model

import "gorm.io/gorm"

// AlertEvent 告警事件记录。
type AlertEvent struct {
	ID                  int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	RuleID              int64          `gorm:"index;not null" json:"rule_id"`
	RuleName            string         `gorm:"size:100" json:"rule_name"`
	AgentID             string         `gorm:"size:100;index;not null" json:"agent_id"`
	Hostname            string         `gorm:"size:200" json:"hostname"`
	IP                  string         `gorm:"size:50;index" json:"ip"`
	MetricType          string         `gorm:"size:50;index;not null" json:"metric_type"`
	MetricValue         float64        `gorm:"type:decimal(18,4);not null" json:"metric_value"`
	Threshold           float64        `gorm:"type:decimal(18,4);not null" json:"threshold"`
	Operator            string         `gorm:"size:10;not null" json:"operator"`
	Severity            string         `gorm:"size:20;index;default:warning" json:"severity"`
	Action              string         `gorm:"size:32;index;default:notify_only" json:"action"`
	Status              string         `gorm:"size:20;index;default:firing" json:"status"` // firing/acknowledged/resolved
	Description         string         `gorm:"size:500" json:"description"`
	NotificationEventID int64          `gorm:"index;default:0" json:"notification_event_id"`
	TicketID            int64          `gorm:"index;default:0" json:"ticket_id"`
	TaskExecutionID     int64          `gorm:"index;default:0" json:"task_execution_id"`
	InspectionRecordID  int64          `gorm:"index;default:0" json:"inspection_record_id"`
	ServiceTreeID       int64          `gorm:"index;default:0" json:"service_tree_id"`
	OwnerID             int64          `gorm:"index;default:0" json:"owner_id"`
	TriggeredAt         LocalTime      `gorm:"index" json:"triggered_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	ResolvedAt          *LocalTime     `json:"resolved_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	LastNotifyAt        *LocalTime     `json:"last_notify_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	Escalated           int8           `gorm:"default:0" json:"escalated"`
	AssigneeID          int64          `gorm:"index;default:0" json:"assignee_id"`
	AssignedAt          *LocalTime     `json:"assigned_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	SLADeadlineAt       *LocalTime     `json:"sla_deadline_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	AcknowledgedBy      int64          `gorm:"index;default:0" json:"acknowledged_by"`
	AcknowledgedAt      *LocalTime     `json:"acknowledged_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	AcknowledgementNote string         `gorm:"size:500" json:"acknowledgement_note"`
	ResolvedBy          int64          `gorm:"index;default:0" json:"resolved_by"`
	ResolutionNote      string         `gorm:"size:500" json:"resolution_note"`
	CreatedAt           LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt           LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

const (
	AlertEventStatusFiring       = "firing"
	AlertEventStatusAcknowledged = "acknowledged"
	AlertEventStatusResolved     = "resolved"
	AlertEventStatusSuppressed   = "suppressed"
)

func (AlertEvent) TableName() string {
	return "alert_events"
}
