package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/pkg/database"
)

const (
	loginMaxAttempts  = 5               // 最大失败次数
	loginLockDuration = 15 * time.Minute // 锁定时长
	rateLimitWindow   = 1 * time.Minute  // 限流窗口
	rateLimitMax      = 10               // 窗口内最大请求数
)

// RateLimit 基于 IP 的速率限制中间件（使用 Redis）。
// 每个 IP 在 window 内最多 maxRequests 次请求。
func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("ratelimit:%s:%s", c.Request.URL.Path, ip)

		rdb := database.GetRedis()
		ctx := context.Background()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// Redis 不可用时放行，不因限流组件故障阻断服务
			c.Next()
			return
		}
		if count == 1 {
			rdb.Expire(ctx, key, window)
		}
		if count > int64(maxRequests) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
			})
			return
		}
		c.Next()
	}
}

// LoginRateLimit 登录专用限流：IP 限流 + 账号失败次数锁定。
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		// 1. IP 级别限流
		ipKey := fmt.Sprintf("ratelimit:login:ip:%s", ip)
		rdb := database.GetRedis()
		ctx := context.Background()

		count, err := rdb.Incr(ctx, ipKey).Result()
		if err == nil {
			if count == 1 {
				rdb.Expire(ctx, ipKey, rateLimitWindow)
			}
			if count > int64(rateLimitMax) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"code":    429,
					"message": "登录请求过于频繁，请稍后再试",
				})
				return
			}
		}

		c.Next()

		// 2. 登录失败后检查并递增失败计数（在 handler 执行后）
		// 通过 c.GetInt("login_status") 判断登录结果
		// handler 中需 c.Set("login_username", username) 和 c.Set("login_failed", true)
		username, _ := c.Get("login_username")
		failed, _ := c.Get("login_failed")
		if username == nil || username == "" {
			return
		}
		usernameStr := username.(string)

		if failed != nil && failed.(bool) {
			// 递增失败次数
			failKey := fmt.Sprintf("login:fail:%s", usernameStr)
			failCount, _ := rdb.Incr(ctx, failKey).Result()
			if failCount == 1 {
				rdb.Expire(ctx, failKey, loginLockDuration)
			}
		} else {
			// 登录成功，清除失败计数
			failKey := fmt.Sprintf("login:fail:%s", usernameStr)
			rdb.Del(ctx, failKey)
		}
	}
}

// IsAccountLocked 检查账号是否被锁定。
func IsAccountLocked(username string) bool {
	rdb := database.GetRedis()
	ctx := context.Background()
	failKey := fmt.Sprintf("login:fail:%s", username)
	count, err := rdb.Get(ctx, failKey).Int64()
	if err != nil {
		return false
	}
	return count >= loginMaxAttempts
}

// RegisterRateLimit 注册专用限流：每 IP 每分钟最多 3 次。
func RegisterRateLimit() gin.HandlerFunc {
	return RateLimit(3, rateLimitWindow)
}
