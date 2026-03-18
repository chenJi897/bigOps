package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/pkg/logger"
)

// GinLogger 基于 Zap 的 Gin 请求日志中间件。
// 替代 gin.Logger()，将请求日志写入 Zap（文件+控制台）。
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", statusCode),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()))
		}

		if statusCode >= 500 {
			logger.Error("HTTP request", fields...)
		} else if statusCode >= 400 {
			logger.Warn("HTTP request", fields...)
		} else {
			logger.Info("HTTP request", fields...)
		}
	}
}
