package model

import "gorm.io/gorm"

// AgentInfo Agent 信息，对应 agent_infos 表。
type AgentInfo struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID        string     `gorm:"size:100;uniqueIndex;not null" json:"agent_id"`
	Hostname       string     `gorm:"size:200" json:"hostname"`
	IP             string     `gorm:"size:50" json:"ip"`
	PrivateIP      string     `gorm:"column:private_ip;size:50" json:"private_ip"`
	PublicIP       string     `gorm:"column:public_ip;size:50" json:"public_ip"`
	Version        string     `gorm:"size:50" json:"version"`
	OS             string     `gorm:"column:os;size:100" json:"os"`
	Status         string     `gorm:"size:20;index;default:offline" json:"status"` // online/offline
	Labels         string     `gorm:"type:json" json:"labels"`
	CPUCount       int        `gorm:"column:cpu_count" json:"cpu_count"`
	CPUUsagePct    float64    `gorm:"column:cpu_usage_pct;type:decimal(5,2);default:0" json:"cpu_usage_pct"`
	MemoryTotal    int64      `json:"memory_total"`
	MemoryUsed     int64      `json:"memory_used"`
	MemoryUsagePct float64    `gorm:"column:memory_usage_pct;type:decimal(5,2);default:0" json:"memory_usage_pct"`
	DiskTotal      int64      `json:"disk_total"`
	DiskUsed       int64      `json:"disk_used"`
	DiskUsagePct   float64    `gorm:"column:disk_usage_pct;type:decimal(5,2);default:0" json:"disk_usage_pct"`
	LastHeartbeat  *LocalTime `json:"last_heartbeat" swaggertype:"string" example:"2024-01-01 00:00:00"`
	CreatedAt      LocalTime  `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt      LocalTime  `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
}

func (AgentInfo) TableName() string {
	return "agent_infos"
}

// BeforeSave 确保 JSON 字段合法。
func (a *AgentInfo) BeforeSave(tx *gorm.DB) error {
	if a.Labels == "" {
		a.Labels = "{}"
	}
	return nil
}
