// backend/internal/model/asset.go
package model

import "gorm.io/gorm"

// Asset 主机资产模型，对应 assets 表。
type Asset struct {
	ID              int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Hostname        string         `gorm:"size:100;uniqueIndex;not null" json:"hostname"`
	IP              string         `gorm:"size:50;index;not null" json:"ip"`
	InnerIP         string         `gorm:"size:50" json:"inner_ip"`
	OS              string         `gorm:"size:50" json:"os"`
	OSVersion       string         `gorm:"size:50" json:"os_version"`
	CPUCores        int            `gorm:"default:0" json:"cpu_cores"`
	MemoryMB        int            `gorm:"default:0" json:"memory_mb"`
	DiskGB          int            `gorm:"default:0" json:"disk_gb"`
	Status          string         `gorm:"size:20;default:online;index" json:"status"`    // online/offline
	AssetType       string         `gorm:"size:50;default:server;index" json:"asset_type"` // server/network
	Source          string         `gorm:"size:20;default:manual;index" json:"source"`     // manual/aliyun/tencent/aws
	ServiceTreeID   int64          `gorm:"default:0;index" json:"service_tree_id"`
	ServiceTreeName string         `gorm:"-" json:"service_tree_name,omitempty"`           // 关联查询，不入库
	ServiceTreePath string         `gorm:"-" json:"service_tree_path,omitempty"`           // 完整路径，如 "阿里云 / 北京 / xxx"
	CloudAccountID  int64          `gorm:"default:0;index" json:"cloud_account_id"`
	CloudInstanceID string         `gorm:"size:100;index" json:"cloud_instance_id"`       // 云实例ID，用于同步 upsert
	IDC             string         `gorm:"column:idc;size:100" json:"idc"`
	SN              string         `gorm:"size:100" json:"sn"`
	OwnerIDs        string         `gorm:"size:500;default:[]" json:"owner_ids"`              // 负责人ID列表 JSON [1,5]
	OwnerNames      []string       `gorm:"-" json:"owner_names,omitempty"`                    // 关联查询
	Tags            string         `gorm:"type:json;default:null" json:"tags"`               // JSON 数组 ["tag1","tag2"]
	Remark          string         `gorm:"size:500" json:"remark"`
	LastSyncAt      *LocalTime     `json:"last_sync_at" swaggertype:"string" example:"2024-01-01 00:00:00"`   // 最后同步时间
	LastSeenAt      *LocalTime     `json:"last_seen_at" swaggertype:"string" example:"2024-01-01 00:00:00"`   // 最后在云端被发现的时间
	OfflineAt       *LocalTime     `json:"offline_at" swaggertype:"string" example:"2024-01-01 00:00:00"`     // 标记离线的时间
	CreatedAt       LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt       LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Asset) TableName() string {
	return "assets"
}

// BeforeSave 确保 tags 字段为合法 JSON（空值转为 "[]"）。
func (a *Asset) BeforeSave(tx *gorm.DB) error {
	if a.Tags == "" {
		a.Tags = "[]"
	}
	if a.OwnerIDs == "" {
		a.OwnerIDs = "[]"
	}
	return nil
}
