// Package model 定义数据库模型（与数据表一一对应的结构体）。
package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型，对应 users 表。
type User struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Password  string         `gorm:"size:255;not null" json:"-"`            // json:"-" 确保密码不会出现在响应中
	Email     *string        `gorm:"size:100;uniqueIndex" json:"email"`
	Phone     string         `gorm:"size:20" json:"phone"`
	RealName  string         `gorm:"size:50" json:"real_name"`
	Avatar    string         `gorm:"size:255" json:"avatar"`
	Status    int8           `gorm:"default:1;not null;index" json:"status"` // 1=启用 0=禁用
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`                        // 软删除
}

// TableName 指定表名。
func (User) TableName() string {
	return "users"
}
