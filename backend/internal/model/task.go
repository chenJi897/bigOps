package model

import "gorm.io/gorm"

// Task 任务模型，对应 tasks 表。
type Task struct {
	ID            int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string         `gorm:"size:200;not null" json:"name"`
	TaskType      string         `gorm:"size:30;index;not null;default:shell" json:"task_type"` // script/file_transfer/api_call
	ScriptType    string         `gorm:"size:20;default:bash" json:"script_type"`               // bash/python/powershell
	ScriptContent string         `gorm:"type:text" json:"script_content"`
	Timeout       int            `gorm:"default:60" json:"timeout"`
	RunAsUser     string         `gorm:"size:50" json:"run_as_user"`
	Description   string         `gorm:"type:text" json:"description"`
	RiskLevel     string         `gorm:"size:20;default:low;index" json:"risk_level"` // low/medium/high/critical
	RequireApproval int8         `gorm:"default:0" json:"require_approval"`           // 1=执行前需审批
	CreatorID     int64          `gorm:"index" json:"creator_id"`
	CreatorName   string         `gorm:"-" json:"creator_name,omitempty"`
	Status        int            `gorm:"default:1;index" json:"status"`
	CreatedAt     LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt     LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

var DangerousCommandPatterns = []string{
	"rm -rf /",
	"mkfs",
	"dd if=",
	"> /dev/sd",
	"shutdown",
	"reboot",
	"init 0",
	"halt",
}

func (Task) TableName() string {
	return "tasks"
}
