package model

// TaskHostResult 单主机执行结果，对应 task_host_results 表。
type TaskHostResult struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	ExecutionID int64      `gorm:"index;not null" json:"execution_id"`
	AgentID     string     `gorm:"size:100;index" json:"agent_id"`
	HostIP      string     `gorm:"size:50" json:"host_ip"`
	Hostname    string     `gorm:"size:200" json:"hostname"`
	Status      string     `gorm:"size:20;index;default:pending" json:"status"` // pending/running/success/failed/timeout
	ExitCode    int        `json:"exit_code"`
	Stdout      string     `gorm:"type:longtext" json:"stdout"`
	Stderr      string     `gorm:"type:longtext" json:"stderr"`
	StartedAt   *LocalTime `json:"started_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	FinishedAt  *LocalTime `json:"finished_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	CreatedAt   LocalTime  `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt   LocalTime  `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
}

func (TaskHostResult) TableName() string {
	return "task_host_results"
}
