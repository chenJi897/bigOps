package model

import "gorm.io/gorm"

// TaskExecution 任务执行记录，对应 task_executions 表。
type TaskExecution struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID       int64      `gorm:"index;not null" json:"task_id"`
	TaskName     string     `gorm:"-" json:"task_name,omitempty"`
	Status       string     `gorm:"size:20;index;default:pending" json:"status"` // 见 task_execution_fsm.go：pending/running/success/partial_fail/failed/canceled
	TargetHosts  string     `gorm:"type:json" json:"target_hosts"`               // ["1.2.3.4","5.6.7.8"]
	TotalCount   int        `json:"total_count"`
	SuccessCount int        `json:"success_count"`
	FailCount    int        `json:"fail_count"`
	OperatorID   int64      `gorm:"index" json:"operator_id"`
	OperatorName string     `gorm:"-" json:"operator_name,omitempty"`
	StartedAt    *LocalTime `json:"started_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	FinishedAt   *LocalTime `json:"finished_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	CreatedAt    LocalTime  `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt    LocalTime  `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`

	// 关联加载
	HostResults []TaskHostResult `gorm:"-" json:"host_results,omitempty"`
}

func (TaskExecution) TableName() string {
	return "task_executions"
}

// BeforeSave 确保 JSON 字段合法。
func (e *TaskExecution) BeforeSave(tx *gorm.DB) error {
	if e.TargetHosts == "" {
		e.TargetHosts = "[]"
	}
	return nil
}
