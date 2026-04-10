package model

import "gorm.io/gorm"

type InspectionTemplate struct {
	ID                int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name              string         `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Description       string         `gorm:"size:512" json:"description"`
	TaskID            int64          `gorm:"not null" json:"task_id"`
	RemediationTaskID int64          `gorm:"default:0" json:"remediation_task_id"`
	DefaultHosts      string         `gorm:"type:json" json:"default_hosts"`
	MaxRetries        int            `gorm:"default:0" json:"max_retries"`
	Enabled           int8           `gorm:"default:1" json:"enabled"`
	CreatedBy    int64          `gorm:"not null;default:0" json:"created_by"`
	UpdatedBy    int64          `gorm:"not null;default:0" json:"updated_by"`
	CreatedAt    LocalTime      `json:"created_at"`
	UpdatedAt    LocalTime      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (InspectionTemplate) TableName() string {
	return "inspection_templates"
}

type InspectionPlan struct {
	ID         int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string         `gorm:"size:128;not null;uniqueIndex" json:"name"`
	TemplateID int64          `gorm:"not null;index" json:"template_id"`
	CronExpr   string         `gorm:"size:64;not null" json:"cron_expr"`
	Enabled    int8           `gorm:"default:1" json:"enabled"`
	LastRunAt  *LocalTime     `json:"last_run_at,omitempty"`
	NextRunAt  *LocalTime     `json:"next_run_at,omitempty"`
	CreatedBy  int64          `gorm:"not null;default:0" json:"created_by"`
	UpdatedBy  int64          `gorm:"not null;default:0" json:"updated_by"`
	CreatedAt  LocalTime      `json:"created_at"`
	UpdatedAt  LocalTime      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (InspectionPlan) TableName() string {
	return "inspection_plans"
}

type InspectionRecord struct {
	ID              int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	PlanID          int64          `gorm:"index" json:"plan_id"`
	TemplateID      int64          `gorm:"index" json:"template_id"`
	TaskExecutionID int64          `gorm:"index" json:"task_execution_id"`
	Status          string         `gorm:"size:32;index;default:running" json:"status"`
	RetryCount      int            `gorm:"default:0" json:"retry_count"`
	StartedAt       *LocalTime     `json:"started_at,omitempty"`
	FinishedAt      *LocalTime     `json:"finished_at,omitempty"`
	ReportJSON      string         `gorm:"type:json" json:"report_json"`
	CreatedAt       LocalTime      `json:"created_at"`
	UpdatedAt       LocalTime      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (InspectionRecord) TableName() string {
	return "inspection_records"
}
