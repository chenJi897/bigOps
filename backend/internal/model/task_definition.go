package model

import "gorm.io/gorm"

// TaskDefinition 任务定义（模板），对应 task_definitions 表。
// 注意：当前生产主链路统一使用 model.Task（tasks 表）+ model.TaskExecution；
// 本结构为方案预留/历史迁移用，业务代码请勿与 Task双写；新功能以 Task 为准。
type TaskDefinition struct {
	ID              int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"size:200;not null;uniqueIndex" json:"name"`
	Version         string         `gorm:"size:50;not null;default:'v1.0.0'" json:"version"` // 语义化版本
	TaskType        string         `gorm:"size:30;index;not null;default:shell" json:"task_type"` // shell/python/file_transfer/approval
	ScriptType      string         `gorm:"size:20;default:bash" json:"script_type"`
	ScriptContent   string         `gorm:"type:longtext" json:"script_content"`
	Timeout         int            `gorm:"default:300" json:"timeout"` // 单位：秒
	RunAsUser       string         `gorm:"size:50" json:"run_as_user"`
	Description     string         `gorm:"type:text" json:"description"`
	Parameters      string         `gorm:"type:json" json:"parameters"`     // 变量定义 JSON
	Tags            string         `gorm:"size:500" json:"tags"`            // 逗号分隔标签
	Category        string         `gorm:"size:50;index" json:"category"`   // 运维/安全/云管/业务
	IsTemplate      bool           `gorm:"default:true" json:"is_template"` // 是否为模板
	CreatorID       int64          `gorm:"index" json:"creator_id"`
	CreatorName     string         `gorm:"-" json:"creator_name,omitempty"`
	Status          int            `gorm:"default:1;index" json:"status"` // 1=启用 0=禁用
	CreatedAt       LocalTime      `json:"created_at" swaggertype:"string" example:"2026-04-09 10:00:00"`
	UpdatedAt       LocalTime      `json:"updated_at" swaggertype:"string" example:"2026-04-09 10:00:00"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	// 兼容现有执行模型（task_executions 使用 task_id 字段），暂不做 GORM 关联映射
	Executions []TaskExecution `gorm:"-" json:"executions,omitempty"`
}

func (TaskDefinition) TableName() string {
	return "task_definitions"
}

func (t *TaskDefinition) BeforeSave(tx *gorm.DB) error {
	if t.Parameters == "" {
		t.Parameters = "{}"
	}
	if t.Tags == "" {
		t.Tags = ""
	}
	return nil
}