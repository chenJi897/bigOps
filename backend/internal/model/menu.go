package model

import (
	"gorm.io/gorm"
)

// Menu 菜单/权限模型，对应 menus 表。
type Menu struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID  int64          `gorm:"default:0;not null;index" json:"parent_id"`
	Name      string         `gorm:"size:50;uniqueIndex;not null" json:"name"`
	Title     string         `gorm:"size:100;not null" json:"title"`
	Icon      string         `gorm:"size:100" json:"icon"`
	Path      string         `gorm:"size:255" json:"path"`
	Component string         `gorm:"size:255" json:"component"`
	APIPath   string         `gorm:"column:api_path;size:255" json:"api_path"`
	APIMethod string         `gorm:"column:api_method;size:10" json:"api_method"`
	Type      int8           `gorm:"default:1;not null" json:"type"`
	Sort      int            `gorm:"default:0;not null" json:"sort"`
	Visible   int8           `gorm:"default:1;not null" json:"visible"`
	Status    int8           `gorm:"default:1;not null" json:"status"`
	Children  []Menu         `gorm:"-" json:"children,omitempty"`
	CreatedAt LocalTime      `json:"created_at"`
	UpdatedAt LocalTime      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Menu) TableName() string {
	return "menus"
}

// UserRole 用户-角色关联模型，对应 user_roles 表。
type UserRole struct {
	UserID int64 `gorm:"primaryKey" json:"user_id"`
	RoleID int64 `gorm:"primaryKey" json:"role_id"`
}

func (UserRole) TableName() string {
	return "user_roles"
}
