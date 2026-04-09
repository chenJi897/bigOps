// Package safego 提供带 panic 恢复的 goroutine 启动函数，防止裸 goroutine panic 导致进程崩溃。
package safego

import (
	"github.com/bigops/platform/internal/pkg/logger"
	"go.uber.org/zap"
)

// Go 启动一个带 recover 保护的 goroutine。
// panic 时记录错误日志而非崩溃进程。
func Go(fn func()) {
	go func() {
		defer Recover("anonymous")
		fn()
	}()
}

// Recover 用于 defer 调用，捕获 panic 并记录日志。
// 适用于已有 goroutine 但需要加 recover 保护的场景。
func Recover(name string) {
	if r := recover(); r != nil {
		logger.Error("goroutine panic recovered", zap.String("goroutine", name), zap.Any("panic", r))
	}
}
