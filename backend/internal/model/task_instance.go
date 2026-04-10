package model

import "gorm.io/gorm"

// TaskInstance 任务执行实例，对应 task_instances 表。
// 注意：当前生产主链路使用 model.TaskExecution（task_executions）承载执行记录；
// 本表为方案预留，与 TaskHostResult 的外键关联未接入 gRPC 下发路径。
type TaskInstance struct {
	ID               int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskDefinitionID int64          `gorm:"index;not null" json:"task_definition_id"`
	TaskName         string         `gorm:"-" json:"task_name,omitempty"`
	ExecutionID      string         `gorm:"size:64;uniqueIndex" json:"execution_id"` // 全局唯一执行ID
	Status           string         `gorm:"size:20;index;default:pending" json:"status"` // pending/running/success/failed/partial_failed/canceled/timeout
	Parameters       string         `gorm:"type:json" json:"parameters"`               // 本次执行的实际参数值
	TargetType       string         `gorm:"size:20;default:all" json:"target_type"`    // all/specific/group/service_tree
	TargetHosts      string         `gorm:"type:json" json:"target_hosts"`             // 执行目标列表
	TotalCount       int            `json:"total_count"`
	SuccessCount     int            `json:"success_count"`
	FailCount        int            `json:"fail_count"`
	Timeout          int            `gorm:"default:300" json:"timeout"`
	OperatorID       int64          `gorm:"index" json:"operator_id"`
	OperatorName     string         `gorm:"-" json:"operator_name,omitempty"`
	StartedAt        *LocalTime     `json:"started_at" swaggertype:"string" example:"2026-04-09 10:00:00"`
	FinishedAt       *LocalTime     `json:"finished_at" swaggertype:"string" example:"2026-04-09 10:05:00"`
	Duration         int64          `json:"duration"` // 执行耗时（毫秒）
	CreatedAt        LocalTime      `json:"created_at" swaggertype:"string" example:"2026-04-09 10:00:00"`
	UpdatedAt        LocalTime      `json:"updated_at" swaggertype:"string" example:"2026-04-09 10:05:00"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系
	TaskDefinition TaskDefinition   `gorm:"foreignKey:TaskDefinitionID" json:"task_definition,omitempty"`
	Steps          []TaskStep       `gorm:"foreignKey:InstanceID" json:"steps,omitempty"`
	HostResults    []TaskHostResult `gorm:"foreignKey:ExecutionID;references:ExecutionID" json:"host_results,omitempty"`
}

func (TaskInstance) TableName() string {
	return "task_instances"
}

func (t *TaskInstance) BeforeSave(tx *gorm.DB) error {
	if t.Parameters == "" {
		t.Parameters = "{}"
	}
	if t.TargetHosts == "" {
		t.TargetHosts = "[]"
	}
	return nil
}