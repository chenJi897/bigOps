package model

import (
	"gorm.io/gorm"
)

// Role 角色模型，对应 roles 表。
type Role struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"size:50;uniqueIndex;not null" json:"name"`
	DisplayName string         `gorm:"size:100;not null" json:"display_name"`
	Description string         `gorm:"size:255" json:"description"`
	Sort        int            `gorm:"default:0;not null" json:"sort"`
	Status      int8           `gorm:"default:1;not null" json:"status"`
	Menus       []Menu         `gorm:"many2many:role_menus;" json:"menus,omitempty"`
	CreatedAt   LocalTime      `json:"created_at"`
	UpdatedAt   LocalTime      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Role) TableName() string {
	return "roles"
}
