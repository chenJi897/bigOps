// Package agent patrol 提供 Agent 进程自身的资源巡检，
// 参考 didi/falcon-log-agent 的实现思路。
package agent

import (
	"math"
	"os"
	"runtime"
	"time"

	"github.com/bigops/platform/internal/pkg/logger"
	"go.uber.org/zap"
)

// ResourceConfig Agent 资源限制配置。
type ResourceConfig struct {
	MaxCPURate float64 // 最大 CPU 使用率（0.0~1.0），如 0.5 表示最多用 50% 的核
	MaxMemMB   int     // 最大堆内存（MB），超过则自杀退出
}

// ApplyCPULimit 根据 maxCPURate 限制 Go 调度器可用的 CPU 核数。
// 返回实际设置的核数。
func ApplyCPULimit(maxCPURate float64) int {
	if maxCPURate <= 0 || maxCPURate > 1.0 {
		maxCPURate = 1.0
	}
	numCPU := runtime.NumCPU()
	limit := int(math.Ceil(float64(numCPU) * maxCPURate))
	if limit < 1 {
		limit = 1
	}
	runtime.GOMAXPROCS(limit)
	return limit
}

// StartMemoryPatrol 启动内存巡检协程，每 interval 检查一次堆内存。
// 超过 50% 打 warning，超过 100% 退出进程。
func StartMemoryPatrol(agentID string, maxMemMB int, interval time.Duration) {
	if maxMemMB <= 0 {
		maxMemMB = 512 // 默认 512MB
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}

	go func() {
		for {
			time.Sleep(interval)

			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)
			usedMB := stats.HeapAlloc / 1024 / 1024
			rate := (usedMB * 100) / uint64(maxMemMB)

			if rate > 50 {
				logger.Warn("Agent memory patrol warning",
					zap.String("agent_id", agentID),
					zap.Int("used_mb", int(usedMB)),
					zap.Int("limit_mb", maxMemMB),
					zap.Int("rate", int(rate)),
				)
			}
			if rate > 100 {
				logger.Error("Agent memory over limit, exiting",
					zap.String("agent_id", agentID),
					zap.Int("used_mb", int(usedMB)),
					zap.Int("limit_mb", maxMemMB),
					zap.Int("rate", int(rate)),
				)
				os.Exit(1)
			}
		}
	}()
}
