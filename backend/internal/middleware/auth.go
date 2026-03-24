// Package middleware 提供 HTTP 中间件，如 JWT 认证、跨域处理等。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	jwtPkg "github.com/bigops/platform/internal/pkg/jwt"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

// AuthMiddleware JWT 认证中间件。
// 从请求头 Authorization 提取 Bearer token，验证有效性后将用户信息注入 Context。
func AuthMiddleware() gin.HandlerFunc {
	authService := service.NewAuthService()

	return func(c *gin.Context) {
		// 从 Header 提取 token（格式：Bearer <token>）
		// WebSocket 无法设置自定义 Header，降级从 query param 读取
		authHeader := c.GetHeader("Authorization")
		var tokenString string
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}
		if tokenString == "" {
			tokenString = c.Query("token")
		}
		if tokenString == "" {
			response.Unauthorized(c, "请求未携带 token")
			c.Abort()
			return
		}

		// 检查 token 是否在黑名单中（已登出）
		if authService.IsTokenBlacklisted(tokenString) {
			response.Unauthorized(c, "token 已失效，请重新登录")
			c.Abort()
			return
		}

		// 解析并验证 token
		claims, err := jwtPkg.ParseToken(tokenString)
		if err != nil {
			response.Unauthorized(c, "token 无效或已过期")
			c.Abort()
			return
		}

		// 将用户信息注入到 Context，后续 handler 可通过 c.Get 获取
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}
