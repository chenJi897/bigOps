package service

import (
	"strings"
	"sync"

	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/repository"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type InspectionScheduler struct {
	repo       *repository.InspectionRepository
	svc        *InspectionService
	cronEngine *cron.Cron
	mu         sync.Mutex
	planEntry  map[int64]cron.EntryID
}

func NewInspectionScheduler() *InspectionScheduler {
	return &InspectionScheduler{
		repo:       repository.NewInspectionRepository(),
		svc:        NewInspectionService(),
		cronEngine: cron.New(cron.WithSeconds()),
		planEntry:  make(map[int64]cron.EntryID),
	}
}

func (s *InspectionScheduler) Start() {
	s.reloadPlans()
	_, _ = s.cronEngine.AddFunc("@every 60s", s.reloadPlans)
	_, _ = s.cronEngine.AddFunc("@every 30s", s.syncRunningRecords)
	s.cronEngine.Start()
	logger.Info("inspection scheduler started")
}

func (s *InspectionScheduler) syncRunningRecords() {
	records, err := s.repo.ListRunningRecords()
	if err != nil || len(records) == 0 {
		return
	}
	for _, rec := range records {
		if rec.TaskExecutionID > 0 {
			s.svc.SyncRecordStatus(rec.TaskExecutionID)
		}
	}
}

func (s *InspectionScheduler) Stop() {
	ctx := s.cronEngine.Stop()
	<-ctx.Done()
	logger.Info("inspection scheduler stopped")
}

func (s *InspectionScheduler) reloadPlans() {
	plans, err := s.repo.ListEnabledPlans()
	if err != nil {
		logger.Warn("inspection scheduler load plans failed", zap.Error(err))
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for planID, entryID := range s.planEntry {
		s.cronEngine.Remove(entryID)
		delete(s.planEntry, planID)
	}
	for _, p := range plans {
		if p == nil || strings.TrimSpace(p.CronExpr) == "" {
			continue
		}
		planID := p.ID
		expr := ensureCronWithSeconds(p.CronExpr)
		entryID, addErr := s.cronEngine.AddFunc(expr, func() {
			if _, runErr := s.svc.ExecutePlan(planID); runErr != nil {
				logger.Warn("inspection plan execute failed", zap.Int64("plan_id", planID), zap.Error(runErr))
				return
			}
			logger.Info("inspection plan executed", zap.Int64("plan_id", planID))
		})
		if addErr != nil {
			logger.Warn("inspection plan cron invalid", zap.Int64("plan_id", planID), zap.String("cron", p.CronExpr), zap.Error(addErr))
			continue
		}
		s.planEntry[planID] = entryID
	}
	logger.Info("inspection scheduler plans reloaded", zap.Int("count", len(s.planEntry)))
}

func ensureCronWithSeconds(expr string) string {
	parts := strings.Fields(expr)
	if len(parts) >= 6 {
		return expr
	}
	return "0 " + expr
}
