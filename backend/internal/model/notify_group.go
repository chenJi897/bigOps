package model

import "gorm.io/gorm"

// NotifyGroup 发送组：将通知对象（Webhook 群 + 通知人 + 升级链 + 重复策略）打包管理。
type NotifyGroup struct {
	ID                     int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                   string         `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Description            string         `gorm:"size:500" json:"description"`
	WebhooksJSON           string         `gorm:"type:text" json:"webhooks_json"`            // [{channel_type,label,webhook_url,secret}]
	NotifyUserIDs          string         `gorm:"type:json" json:"notify_user_ids"`           // [userID1, userID2]
	RepeatEnabled          int8           `gorm:"default:0" json:"repeat_enabled"`
	RepeatIntervalSeconds  int            `gorm:"default:300" json:"repeat_interval_seconds"`
	SendResolved           int8           `gorm:"default:1" json:"send_resolved"`
	EscalationEnabled      int8           `gorm:"default:0" json:"escalation_enabled"`
	EscalationMinutes      int            `gorm:"default:20" json:"escalation_minutes"`
	EscalationUserIDs      string         `gorm:"type:json" json:"escalation_user_ids"`
	EscalationWebhooksJSON string         `gorm:"type:text" json:"escalation_webhooks_json"`
	Status                 int8           `gorm:"default:1;index" json:"status"`
	CreatedBy              int64          `gorm:"index" json:"created_by"`
	CreatedAt              LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt              LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"-"`
}

func (NotifyGroup) TableName() string {
	return "notify_groups"
}

func (g *NotifyGroup) BeforeSave(tx *gorm.DB) error {
	if g.WebhooksJSON == "" {
		g.WebhooksJSON = "[]"
	}
	if g.NotifyUserIDs == "" {
		g.NotifyUserIDs = "[]"
	}
	if g.EscalationUserIDs == "" {
		g.EscalationUserIDs = "[]"
	}
	if g.EscalationWebhooksJSON == "" {
		g.EscalationWebhooksJSON = "[]"
	}
	return nil
}

// GroupWebhook 发送组中的单个 Webhook 配置。
type GroupWebhook struct {
	ChannelType string `json:"channel_type"` // lark/dingtalk/wecom/webhook
	Label       string `json:"label"`        // "SRE飞书群"
	WebhookURL  string `json:"webhook_url"`
	Secret      string `json:"secret"`
}
