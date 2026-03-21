// backend/internal/model/service_tree.go
package model

import "gorm.io/gorm"

// ServiceTree 服务树节点模型，对应 service_trees 表。
type ServiceTree struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Code        string         `gorm:"size:100;uniqueIndex" json:"code"`
	ParentID    int64          `gorm:"default:0;not null;index" json:"parent_id"`
	Level       int            `gorm:"not null" json:"level"`       // 1=业务线 2=产品 3=模块（不硬限制，支持N层）
	Sort        int            `gorm:"default:0;not null" json:"sort"`
	Description string         `gorm:"size:500" json:"description"`
	OwnerID     int64          `gorm:"default:0;index" json:"owner_id"`
	Children    []ServiceTree  `gorm:"-" json:"children,omitempty"`
	CreatedAt   LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt   LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ServiceTree) TableName() string {
	return "service_trees"
}
