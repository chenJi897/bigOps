package model

import (
	"time"

	"gorm.io/gorm"
)

// Role 角色模型，对应 roles 表。
type Role struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"size:50;uniqueIndex;not null" json:"name"`          // 角色标识（英文）
	DisplayName string         `gorm:"size:100;not null" json:"display_name"`             // 角色显示名
	Description string         `gorm:"size:255" json:"description"`                       // 角色描述
	Sort        int            `gorm:"default:0;not null" json:"sort"`                    // 排序值
	Status      int8           `gorm:"default:1;not null" json:"status"`                  // 1=启用 0=禁用
	Menus       []Menu         `gorm:"many2many:role_menus;" json:"menus,omitempty"`       // 角色拥有的菜单
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Role) TableName() string {
	return "roles"
}
