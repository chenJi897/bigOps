package model

import (
	"time"

	"gorm.io/gorm"
)

// Menu 菜单/权限模型，对应 menus 表。
// 使用树形结构（parent_id）表示目录-菜单-按钮三级层级。
type Menu struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID  int64          `gorm:"default:0;not null;index" json:"parent_id"`           // 父菜单ID，0=顶级
	Name      string         `gorm:"size:50;uniqueIndex;not null" json:"name"`            // 菜单标识
	Title     string         `gorm:"size:100;not null" json:"title"`                      // 菜单显示名
	Icon      string         `gorm:"size:100" json:"icon"`                                // 菜单图标
	Path      string         `gorm:"size:255" json:"path"`                                // 前端路由路径
	Component string         `gorm:"size:255" json:"component"`                           // 前端组件路径
	APIPath   string         `gorm:"column:api_path;size:255" json:"api_path"`            // API 路径
	APIMethod string         `gorm:"column:api_method;size:10" json:"api_method"`         // HTTP 方法
	Type      int8           `gorm:"default:1;not null" json:"type"`                      // 1=目录 2=菜单 3=按钮/API
	Sort      int            `gorm:"default:0;not null" json:"sort"`                      // 排序值
	Visible   int8           `gorm:"default:1;not null" json:"visible"`                   // 1=可见 0=隐藏
	Status    int8           `gorm:"default:1;not null" json:"status"`                    // 1=启用 0=禁用
	Children  []Menu         `gorm:"-" json:"children,omitempty"`                          // 子菜单（非数据库字段）
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
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
