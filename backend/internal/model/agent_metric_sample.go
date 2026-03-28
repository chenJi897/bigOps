package model

import "gorm.io/gorm"

// AgentMetricSample Agent 指标采样点。
type AgentMetricSample struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID     string         `gorm:"size:100;index;not null" json:"agent_id"`
	Hostname    string         `gorm:"size:200" json:"hostname"`
	IP          string         `gorm:"size:50;index" json:"ip"`
	MetricType  string         `gorm:"size:50;index;not null" json:"metric_type"` // cpu_usage/memory_usage/disk_usage/load1...
	MetricValue float64        `gorm:"type:decimal(18,4);not null" json:"metric_value"`
	Unit        string         `gorm:"size:20" json:"unit"`
	Labels      string         `gorm:"type:json" json:"labels"` // JSON 对象
	CollectedAt LocalTime      `gorm:"index" json:"collected_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	CreatedAt   LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt   LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AgentMetricSample) TableName() string {
	return "agent_metric_samples"
}

// BeforeSave 保证 Labels 字段总是合法 JSON。
func (s *AgentMetricSample) BeforeSave(tx *gorm.DB) error {
	if s.Labels == "" {
		s.Labels = "{}"
	}
	return nil
}
