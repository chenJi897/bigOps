package cloudsync

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/service"
)

// Scheduler 定时同步调度器。
// 每分钟检查一次哪些云账号需要同步，按各自的 sync_interval 判断是否到期。
type Scheduler struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	// lastRun 记录每个账号上次同步时间，用于判断是否到期
	lastRun   map[int64]time.Time
	lastRunMu sync.Mutex
}

// NewScheduler 创建调度器实例。
func NewScheduler() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		ctx:     ctx,
		cancel:  cancel,
		lastRun: make(map[int64]time.Time),
	}
}

// Start 启动调度器，每 60 秒巡检一次。
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		logger.Info("云同步调度器已启动，巡检间隔 60s")
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				logger.Info("云同步调度器已停止")
				return
			case <-ticker.C:
				s.check()
			}
		}
	}()
}

// Stop 优雅停止调度器。
func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
}

// check 巡检所有启用定时同步的云账号。
func (s *Scheduler) check() {
	accountSvc := service.NewCloudAccountService()
	accounts, err := accountSvc.ListEnabled()
	if err != nil {
		logger.Error("调度器: 查询云账号失败", zap.Error(err))
		return
	}

	for _, account := range accounts {
		if account.SyncInterval <= 0 {
			continue
		}

		s.lastRunMu.Lock()
		last, ok := s.lastRun[account.ID]
		s.lastRunMu.Unlock()

		interval := time.Duration(account.SyncInterval) * time.Minute
		if ok && time.Since(last) < interval {
			continue // 还没到期
		}

		// 到期了，异步执行同步
		accountID := account.ID
		accountName := account.Name
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			logger.Info("调度器: 触发定时同步",
				zap.Int64("account_id", accountID),
				zap.String("account_name", accountName),
			)
			_, err := RunSync(accountID, "schedule", 0, "system")
			if err != nil {
				logger.Error("调度器: 同步失败",
					zap.Int64("account_id", accountID),
					zap.Error(err),
				)
			}
			// 记录本次执行时间（无论成功失败都记录，避免失败时疯狂重试）
			s.lastRunMu.Lock()
			s.lastRun[accountID] = time.Now()
			s.lastRunMu.Unlock()
		}()
	}
}
