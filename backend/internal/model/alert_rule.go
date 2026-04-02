package model

import "gorm.io/gorm"

// AlertRule 告警规则定义。
type AlertRule struct {
	ID            int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string         `gorm:"size:100;uniqueIndex;not null" json:"name"`
	MetricType    string         `gorm:"size:50;index;not null" json:"metric_type"`
	Operator      string         `gorm:"size:10;not null;default:gt" json:"operator"` // gt/gte/lt/lte/eq/neq
	Threshold     float64        `gorm:"type:decimal(18,4);not null" json:"threshold"`
	Severity      string         `gorm:"size:20;index;default:warning" json:"severity"` // info/warning/critical
	Enabled       int8           `gorm:"index;default:1" json:"enabled"`
	Description   string         `gorm:"size:500" json:"description"`
	NotifyUserIDs string         `gorm:"type:json" json:"notify_user_ids"` // JSON 数组
	NotifyChannels string        `gorm:"type:json" json:"notify_channels"` // JSON 数组
	NotifyConfig   string        `gorm:"type:text" json:"notify_config"`   // JSON: 渠道→webhook 配置
	NotifyGroupID  int64         `gorm:"index" json:"notify_group_id"`     // 发送组ID，>0 则使用发送组模式
	NotifyTemplate string        `gorm:"type:text" json:"notify_template"`
	Action        string         `gorm:"size:32;index;default:notify_only" json:"action"`
	RepairTaskID  int64          `gorm:"index" json:"repair_task_id"`
	TicketTypeID  int64          `gorm:"index" json:"ticket_type_id"`
	OnCallScheduleID int64       `gorm:"index" json:"oncall_schedule_id"`
	ServiceTreeID int64          `gorm:"index" json:"service_tree_id"`
	OwnerID       int64          `gorm:"index" json:"owner_id"`
	CreatedBy     int64          `gorm:"index;default:0" json:"created_by"`
	UpdatedBy     int64          `gorm:"index;default:0" json:"updated_by"`
	CreatedAt     LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt     LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

// BeforeSave 保证 NotifyUserIDs 字段总是合法 JSON。
func (r *AlertRule) BeforeSave(tx *gorm.DB) error {
	if r.NotifyUserIDs == "" {
		r.NotifyUserIDs = "[]"
	}
	if r.NotifyChannels == "" {
		r.NotifyChannels = "[]"
	}
	if r.Action == "" {
		r.Action = "notify_only"
	}
	return nil
}

const (
	AlertRuleActionNotifyOnly   = "notify_only"
	AlertRuleActionCreateTicket = "create_ticket"
	AlertRuleActionExecuteTask  = "execute_task"
)
