package model

import "gorm.io/gorm"

// Department 部门模型，对应 departments 表。
type Department struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Code        string         `gorm:"size:50;index" json:"code"`                      // 部门编码，如 dev/ops/qa
	Description string         `gorm:"size:500" json:"description"`
	ManagerID   int64          `gorm:"default:0" json:"manager_id"`                    // 部门负责人 user_id
	ManagerName string         `gorm:"-" json:"manager_name,omitempty"`                // 关联查询，不入库
	Sort        int            `gorm:"default:0;not null" json:"sort"`
	Status      int8           `gorm:"default:1;not null" json:"status"`               // 1=启用 0=禁用
	UserCount   int64          `gorm:"-" json:"user_count,omitempty"`                  // 关联统计，不入库
	CreatedAt   LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt   LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Department) TableName() string {
	return "departments"
}
