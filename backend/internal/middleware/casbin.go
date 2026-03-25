package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	casbinPkg "github.com/bigops/platform/internal/pkg/casbin"
	"github.com/bigops/platform/internal/pkg/response"
)

// 公共路由白名单（所有已认证用户可访问，无需 Casbin 校验）
var casbinWhitelist = []string{
	"/api/v1/auth/logout",
	"/api/v1/auth/info",
	"/api/v1/auth/password",
	"/api/v1/menus/user",
}

// 公共路由前缀白名单
var casbinPrefixWhitelist = []string{
	"/api/v1/stats/",
	"/api/v1/notifications/in-app",
	"/api/v1/ws/",
}

// CasbinMiddleware Casbin 权限校验中间件。
// 依赖 AuthMiddleware 在 Context 中设置的 username。
// admin 角色在 Casbin 模型中已配置为跳过校验（拥有全部权限）。
func CasbinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Context 获取用户名（由 AuthMiddleware 设置）
		username, exists := c.Get("username")
		if !exists {
			response.Unauthorized(c, "用户未认证")
			c.Abort()
			return
		}

		// 获取请求的 API 路径和方法
		obj := c.Request.URL.Path
		act := c.Request.Method

		// 白名单放行
		for _, path := range casbinWhitelist {
			if obj == path {
				c.Next()
				return
			}
		}
		for _, prefix := range casbinPrefixWhitelist {
			if strings.HasPrefix(obj, prefix) {
				c.Next()
				return
			}
		}

		// Casbin 权限校验
		enforcer := casbinPkg.GetEnforcer()
		ok, err := enforcer.Enforce(username.(string), obj, act)
		if err != nil {
			response.InternalServerError(c, "权限校验失败")
			c.Abort()
			return
		}

		if !ok {
			response.Forbidden(c, "无操作权限")
			c.Abort()
			return
		}

		c.Next()
	}
}
