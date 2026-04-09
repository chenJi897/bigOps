package service

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/safego"
	"github.com/bigops/platform/internal/repository"
	"go.uber.org/zap"
)

// AuditLogService 审计日志业务逻辑。
type AuditLogService struct {
	repo *repository.AuditLogRepository
}

// NewAuditLogService 创建 AuditLogService 实例。
func NewAuditLogService() *AuditLogService {
	return &AuditLogService{repo: repository.NewAuditLogRepository()}
}

// Record 异步记录审计日志。
func (s *AuditLogService) Record(log *model.AuditLog) {
	safego.Go(func() {
		if err := s.repo.Create(log); err != nil {
			logger.Error("写入审计日志失败", zap.Error(err), zap.String("action", log.Action), zap.String("resource", log.Resource))
		}
	})
}

// List 分页查询审计日志。
func (s *AuditLogService) List(page, size int, username, action, resource string) ([]*model.AuditLog, int64, error) {
	return s.repo.List(page, size, username, action, resource)
}
