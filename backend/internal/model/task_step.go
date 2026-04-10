package model

import "gorm.io/gorm"

// TaskStep 任务执行步骤，对应 task_steps 表。
// 支持复杂任务的步骤编排（串行/并行），用于可视化工作流。
type TaskStep struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	InstanceID   int64      `gorm:"index;not null" json:"instance_id"`
	StepOrder    int        `gorm:"not null" json:"step_order"`      // 执行顺序
	StepName     string     `gorm:"size:100;not null" json:"step_name"`
	StepType     string     `gorm:"size:30;not null" json:"step_type"` // shell/approval/wait/parallel
	ScriptType   string     `gorm:"size:20" json:"script_type"`
	ScriptContent string    `gorm:"type:longtext" json:"script_content"`
	Parameters   string     `gorm:"type:json" json:"parameters"`
	Status       string     `gorm:"size:20;default:pending" json:"status"` // pending/running/success/failed/skipped
	ExitCode     int        `json:"exit_code"`
	Output       string     `gorm:"type:longtext" json:"output"`
	Error        string     `gorm:"type:text" json:"error"`
	StartedAt    *LocalTime `json:"started_at" swaggertype:"string" example:"2026-04-09 10:00:01"`
	FinishedAt   *LocalTime `json:"finished_at" swaggertype:"string" example:"2026-04-09 10:00:05"`
	Duration     int64      `json:"duration"` // 毫秒
	CreatedAt    LocalTime  `json:"created_at" swaggertype:"string" example:"2026-04-09 10:00:00"`
	UpdatedAt    LocalTime  `json:"updated_at" swaggertype:"string" example:"2026-04-09 10:00:05"`

	// 关联
	Instance TaskInstance `gorm:"foreignKey:InstanceID" json:"-"`
}

func (TaskStep) TableName() string {
	return "task_steps"
}

func (t *TaskStep) BeforeSave(tx *gorm.DB) error {
	if t.Parameters == "" {
		t.Parameters = "{}"
	}
	return nil
}