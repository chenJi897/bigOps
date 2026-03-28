package model

import "gorm.io/gorm"

// MonitorDatasource 表示 Prometheus 等监控数据源。
type MonitorDatasource struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Type        string         `gorm:"size:32;not null" json:"type"` // prometheus
	BaseURL     string         `gorm:"size:512;not null" json:"base_url"`
	AccessType  string         `gorm:"size:32;default:proxy" json:"access_type"`
	AuthType    string         `gorm:"size:32" json:"auth_type"` // none/basic
	Username    string         `gorm:"size:128" json:"username"`
	Password    string         `gorm:"size:256" json:"password"`
	HeadersJSON string         `gorm:"type:json" json:"headers_json"`
	Status      string         `gorm:"size:32;default:active" json:"status"`
	CreatedAt   LocalTime      `json:"created_at"`
	UpdatedAt   LocalTime      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (MonitorDatasource) TableName() string {
	return "monitor_datasources"
}

func (m *MonitorDatasource) BeforeSave(tx *gorm.DB) error {
	if m.HeadersJSON == "" {
		m.HeadersJSON = "{}"
	}
	if m.Status == "" {
		m.Status = "active"
	}
	return nil
}
