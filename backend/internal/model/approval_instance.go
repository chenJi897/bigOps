package model

import "gorm.io/gorm"

// ApprovalInstance 审批实例。
type ApprovalInstance struct {
	ID             int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	TicketID       int64            `gorm:"index;not null" json:"ticket_id"`
	PolicyID       int64            `gorm:"index;not null" json:"policy_id"`
	PolicyName     string           `gorm:"-" json:"policy_name,omitempty"`
	CurrentStageNo int              `gorm:"default:1" json:"current_stage_no"`
	CurrentStageName string         `gorm:"-" json:"current_stage_name,omitempty"`
	Status         string           `gorm:"size:20;index;default:pending" json:"status"` // pending/in_progress/approved/rejected/canceled/timeout
	StartedAt      *LocalTime       `json:"started_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	FinishedAt     *LocalTime       `json:"finished_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	CreatedAt      LocalTime        `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt      LocalTime        `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt      gorm.DeletedAt   `gorm:"index" json:"-"`
	Records        []ApprovalRecord `gorm:"foreignKey:InstanceID" json:"records,omitempty"`
}

func (ApprovalInstance) TableName() string {
	return "approval_instances"
}

// ApprovalRecord 审批动作记录。
type ApprovalRecord struct {
	ID           int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	InstanceID   int64          `gorm:"index;not null" json:"instance_id"`
	StageNo      int            `gorm:"not null" json:"stage_no"`
	ApproverID   int64          `gorm:"index;not null" json:"approver_id"`
	ApproverName string         `gorm:"size:100" json:"approver_name"`
	Action       string         `gorm:"size:20;index" json:"action"` // approve/reject/return/transfer/add_sign/timeout
	Comment      string         `gorm:"type:text" json:"comment"`
	Status       string         `gorm:"size:20;index;default:pending" json:"status"`
	ActedAt      *LocalTime     `json:"acted_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	CreatedAt    LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt    LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ApprovalRecord) TableName() string {
	return "approval_records"
}
