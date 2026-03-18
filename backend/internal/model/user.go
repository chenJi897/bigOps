// Package model 定义数据库模型（与数据表一一对应的结构体）。
package model

import (
	"gorm.io/gorm"
)

// User 用户模型，对应 users 表。
type User struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Password  string         `gorm:"size:255;not null" json:"-"`
	Email     *string        `gorm:"size:100;uniqueIndex" json:"email"`
	Phone     string         `gorm:"size:20" json:"phone"`
	RealName  string         `gorm:"size:50" json:"real_name"`
	Avatar    string         `gorm:"size:255" json:"avatar"`
	Status    int8           `gorm:"default:1;not null;index" json:"status"`
	CreatedAt LocalTime      `json:"created_at"`
	UpdatedAt LocalTime      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
