package model

import "gorm.io/gorm"

// CloudAccount 云账号模型，对应 cloud_accounts 表。
type CloudAccount struct {
	ID              int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"size:100;not null" json:"name"`
	Provider        string         `gorm:"size:50;not null;index" json:"provider"`            // aliyun/tencent/aws
	AccessKey       string         `gorm:"size:500;not null" json:"-"`                        // AES加密存储
	SecretKey       string         `gorm:"size:500;not null" json:"-"`                        // AES加密存储
	Region          string         `gorm:"size:500" json:"region"`                            // 逗号分隔的地域列表
	Status          int8           `gorm:"default:1;not null" json:"status"`                  // 1=启用 0=禁用
	SyncEnabled     bool           `gorm:"default:false;not null" json:"sync_enabled"`        // 是否启用定时同步
	SyncInterval    int            `gorm:"default:0;not null" json:"sync_interval"`           // 同步周期（分钟），0=不同步，10/30/60/1440
	LastSyncAt      *LocalTime     `json:"last_sync_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	LastSyncStatus  string         `gorm:"size:20" json:"last_sync_status"`                   // success/failed/syncing
	LastSyncMessage string         `gorm:"type:text" json:"last_sync_message"`
	CreatedAt       LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt       LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (CloudAccount) TableName() string {
	return "cloud_accounts"
}
