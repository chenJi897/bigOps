package model

import "gorm.io/gorm"

// TaskApproval 高危任务审批记录。
type TaskApproval struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID      int64          `gorm:"index;not null" json:"task_id"`
	TaskName    string         `gorm:"-" json:"task_name,omitempty"`
	RequestorID int64          `gorm:"index;not null" json:"requestor_id"`
	ApproverID  int64          `gorm:"index;default:0" json:"approver_id"`
	Status      string         `gorm:"size:20;index;default:pending" json:"status"` // pending/approved/rejected
	Comment     string         `gorm:"size:500" json:"comment"`
	HostIPs     string         `gorm:"type:text" json:"host_ips"`
	CreatedAt   LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt   LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

const (
	TaskApprovalStatusPending  = "pending"
	TaskApprovalStatusApproved = "approved"
	TaskApprovalStatusRejected = "rejected"
)

func (TaskApproval) TableName() string {
	return "task_approvals"
}
