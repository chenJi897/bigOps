package model

import "gorm.io/gorm"

// CICDPipeline 表示某个项目下的流水线定义，包含触发配置与执行策略。
type CICDPipeline struct {
	ID                  int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID           int64          `gorm:"index;not null" json:"project_id"`
	ProjectName         string         `gorm:"-" json:"project_name,omitempty"`
	Code                string         `gorm:"size:80;not null;index:idx_pipeline_code,unique" json:"code"`
	Name                string         `gorm:"size:120;not null" json:"name"`
	Description         string         `gorm:"size:500" json:"description"`
	Environment         string         `gorm:"size:50;default:test" json:"environment"`
	TriggerType         string         `gorm:"size:30;default:commit" json:"trigger_type"` // commit/manual/schedule
	TriggerRef          string         `gorm:"size:200" json:"trigger_ref"`                // 默认分支或 tag
	Branch              string         `gorm:"size:200" json:"branch"`                     // 运行的默认分支
	Schedule            string         `gorm:"size:100" json:"schedule"`                   // cron 表达式
	BuildTaskID         int64          `gorm:"index" json:"build_task_id"`
	BuildTaskName       string         `gorm:"-" json:"build_task_name,omitempty"`
	DeployTaskID        int64          `gorm:"index" json:"deploy_task_id"`
	DeployTaskName      string         `gorm:"-" json:"deploy_task_name,omitempty"`
	RequestTemplateID   int64          `gorm:"index" json:"request_template_id"`
	RequestTemplateName string         `gorm:"-" json:"request_template_name,omitempty"`
	TargetHosts         string         `gorm:"type:json" json:"target_hosts"`
	VariablesJSON       string         `gorm:"type:json" json:"variables_json"`
	TemplateYAML        string         `gorm:"type:text" json:"template_yaml"`
	TimeoutSeconds      int            `gorm:"default:1800" json:"timeout_seconds"`
	Concurrency         int            `gorm:"default:1" json:"concurrency"`
	Status              int8           `gorm:"default:1;index" json:"status"` // 1=启用 0=禁用
	ConfigJSON          string         `gorm:"type:json" json:"config_json"`
	NotifyTemplate      string         `gorm:"type:text" json:"notify_template"`
	CreatedAt           LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt           LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

func (CICDPipeline) TableName() string {
	return "cicd_pipelines"
}

func (p *CICDPipeline) BeforeSave(tx *gorm.DB) error {
	if p.TargetHosts == "" {
		p.TargetHosts = "[]"
	}
	if p.VariablesJSON == "" {
		p.VariablesJSON = "{}"
	}
	if p.ConfigJSON == "" {
		p.ConfigJSON = "{}"
	}
	return nil
}
