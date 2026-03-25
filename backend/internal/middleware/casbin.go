package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	casbinPkg "github.com/bigops/platform/internal/pkg/casbin"
	"github.com/bigops/platform/internal/pkg/response"
)

// 公共路由白名单（所有已认证用户可访问，无需 Casbin 校验）
// 精确匹配 path，仅对 GET 方法放行（auth 相关不限方法）
var casbinWhitelist = []string{
	"/api/v1/auth/logout",
	"/api/v1/auth/info",
	"/api/v1/auth/password",
	"/api/v1/menus/user",
	"/api/v1/departments/all",
	"/api/v1/ticket-types/all",
	"/api/v1/request-templates",
	"/api/v1/approval-policies",
}

// 公共路由前缀白名单（不限方法，所有已认证用户可访问）
var casbinPrefixWhitelistAny = []string{
	"/api/v1/notifications/in-app",
	"/api/v1/ws/",
	"/api/v1/approval-instances/",
}

// 公共路由前缀白名单（仅 GET）
var casbinPrefixWhitelistGET = []string{
	"/api/v1/stats/",
	"/api/v1/users/",
	"/api/v1/users",
	"/api/v1/service-trees",
	"/api/v1/cloud-accounts",
	"/api/v1/sync-tasks",
}

// CasbinMiddleware Casbin 权限校验中间件。
// 依赖 AuthMiddleware 在 Context 中设置的 username。
// admin 角色在 Casbin 模型中已配置为跳过校验（拥有全部权限）。
func CasbinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, exists := c.Get("username")
		if !exists {
			response.Unauthorized(c, "用户未认证")
			c.Abort()
			return
		}

		obj := c.Request.URL.Path
		act := c.Request.Method

		// 白名单放行（auth 相关不限方法，其余仅 GET）
		for _, path := range casbinWhitelist {
			if obj == path {
				if act == "GET" || strings.HasPrefix(path, "/api/v1/auth/") {
					c.Next()
					return
				}
			}
		}
		for _, prefix := range casbinPrefixWhitelistAny {
			if strings.HasPrefix(obj, prefix) {
				c.Next()
				return
			}
		}
		for _, prefix := range casbinPrefixWhitelistGET {
			if strings.HasPrefix(obj, prefix) && act == "GET" {
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
