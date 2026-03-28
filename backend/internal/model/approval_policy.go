package model

import "gorm.io/gorm"

// ApprovalPolicy 审批策略头。
type ApprovalPolicy struct {
	ID          int64                 `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string                `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Code        string                `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Description string                `gorm:"size:500" json:"description"`
	Scope       string                `gorm:"size:30;default:request" json:"scope"` // request/change
	Enabled     int8                  `gorm:"default:1;index" json:"enabled"`
	Stages      []ApprovalPolicyStage `gorm:"foreignKey:PolicyID" json:"stages,omitempty"`
	CreatedAt   LocalTime             `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt   LocalTime             `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt   gorm.DeletedAt        `gorm:"index" json:"-"`
}

func (ApprovalPolicy) TableName() string {
	return "approval_policies"
}

// ApprovalPolicyStage 审批阶段定义。
type ApprovalPolicyStage struct {
	ID             int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	PolicyID       int64          `gorm:"index;not null" json:"policy_id"`
	StageNo        int            `gorm:"not null" json:"stage_no"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	StageType      string         `gorm:"size:20;default:serial" json:"stage_type"`       // serial/parallel
	ApproverType   string         `gorm:"size:40;not null" json:"approver_type"`          // direct_manager/fixed_user/fixed_role/dept_leader/service_owner/resource_pool_owner/security_group
	ApproverConfig string         `gorm:"type:json" json:"approver_config"`                // JSON rules
	PassRule       string         `gorm:"size:20;default:all" json:"pass_rule"`            // all/any
	TimeoutHours   int            `gorm:"default:24" json:"timeout_hours"`
	Required       int8           `gorm:"default:1" json:"required"`
	Sort           int            `gorm:"default:0" json:"sort"`
	CreatedAt      LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt      LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ApprovalPolicyStage) TableName() string {
	return "approval_policy_stages"
}
