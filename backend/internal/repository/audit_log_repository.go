package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

// AuditLogRepository 审计日志数据访问对象。
type AuditLogRepository struct{}

// NewAuditLogRepository 创建 AuditLogRepository 实例。
func NewAuditLogRepository() *AuditLogRepository {
	return &AuditLogRepository{}
}

// Create 创建审计日志记录。
func (r *AuditLogRepository) Create(log *model.AuditLog) error {
	return database.GetDB().Create(log).Error
}

// List 分页查询审计日志，支持按用户名、操作类型、资源类型过滤。
func (r *AuditLogRepository) List(page, size int, username, action, resource string) ([]*model.AuditLog, int64, error) {
	var logs []*model.AuditLog
	var total int64

	db := database.GetDB().Model(&model.AuditLog{})

	if username != "" {
		db = db.Where("username = ?", username)
	}
	if action != "" {
		db = db.Where("action = ?", action)
	}
	if resource != "" {
		db = db.Where("resource = ?", resource)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
