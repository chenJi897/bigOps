package model

import "gorm.io/gorm"

// CICDPipelineRun 保存流水线每次执行的快照与状态。
type CICDPipelineRun struct {
	ID               int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	PipelineID       int64          `gorm:"index;not null" json:"pipeline_id"`
	PipelineCode     string         `gorm:"-" json:"pipeline_code,omitempty"`
	PipelineName     string         `gorm:"-" json:"pipeline_name,omitempty"`
	ProjectID        int64          `gorm:"index;not null" json:"project_id"`
	ProjectName      string         `gorm:"-" json:"project_name,omitempty"`
	RunNumber        int64          `gorm:"index" json:"run_number"`
	TriggerType      string         `gorm:"size:30;index" json:"trigger_type"` // manual/commit/schedule/webhook
	TriggerRef       string         `gorm:"size:200" json:"trigger_ref"`       // 分支/标签/事件
	Branch           string         `gorm:"size:200" json:"branch"`
	CommitID         string         `gorm:"size:100" json:"commit_id"`
	CommitMessage    string         `gorm:"size:500" json:"commit_message"`
	Status           string         `gorm:"size:30;index" json:"status"` // pending/running/success/failed/canceled
	Result           string         `gorm:"size:30" json:"result"`
	TaskExecutionID  int64          `gorm:"index" json:"task_execution_id"`
	ApprovalTicketID int64          `gorm:"index" json:"approval_ticket_id"`
	StartedAt        LocalTime      `json:"started_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	FinishedAt       LocalTime      `json:"finished_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DurationSeconds  int            `json:"duration_seconds"`
	QueuedSeconds    int            `json:"queued_seconds"`
	RetryCount       int            `gorm:"default:0" json:"retry_count"`
	TriggeredBy      int64          `gorm:"index" json:"triggered_by"`
	TriggeredByName  string         `gorm:"-" json:"triggered_by_name,omitempty"`
	TargetHosts      string         `gorm:"type:json" json:"target_hosts"`
	VariablesJSON    string         `gorm:"type:json" json:"variables_json"`
	MetadataJSON     string         `gorm:"type:json" json:"metadata_json"`
	LogSnippet       string         `gorm:"type:text" json:"log_snippet"`
	Summary          string         `gorm:"size:500" json:"summary"`
	ArtifactSummary  string         `gorm:"type:json" json:"artifact_summary"`
	ErrorMessage     string         `gorm:"size:500" json:"error_message"`
	CreatedAt        LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt        LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (CICDPipelineRun) TableName() string {
	return "cicd_pipeline_runs"
}

func (r *CICDPipelineRun) BeforeSave(tx *gorm.DB) error {
	if r.TargetHosts == "" {
		r.TargetHosts = "[]"
	}
	if r.VariablesJSON == "" {
		r.VariablesJSON = "{}"
	}
	if r.MetadataJSON == "" {
		r.MetadataJSON = "{}"
	}
	if r.ArtifactSummary == "" {
		r.ArtifactSummary = "{}"
	}
	return nil
}
