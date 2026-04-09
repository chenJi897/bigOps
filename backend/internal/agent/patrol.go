// Package agent patrol 提供 Agent 进程自身的资源巡检，
// 参考 didi/falcon-log-agent 的实现思路。
package agent

import (
	"log"
	"math"
	"os"
	"runtime"
	"time"
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
func StartMemoryPatrol(maxMemMB int, interval time.Duration) {
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
				log.Printf("[patrol] WARNING: agent heap memory usage %dMB, rate %d%% (limit %dMB)", usedMB, rate, maxMemMB)
			}
			if rate > 100 {
				log.Printf("[patrol] FATAL: agent heap memory over limit, exiting. used=%dMB limit=%dMB rate=%d%%", usedMB, maxMemMB, rate)
				os.Exit(1)
			}
		}
	}()
}
