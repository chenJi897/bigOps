package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	casbinPkg "github.com/bigops/platform/internal/pkg/casbin"
	"github.com/bigops/platform/internal/pkg/response"
)

// 公共路由白名单（所有已认证用户可访问，无需 Casbin 校验）
// 精确匹配 path，仅对 GET 方法放行（写操作仍需 Casbin 授权）
var casbinWhitelist = []string{
	"/api/v1/auth/logout",       // POST 但属于自身操作
	"/api/v1/auth/info",
	"/api/v1/auth/password",     // POST 但属于自身操作
	"/api/v1/menus/user",
	"/api/v1/departments/all",   // 部门下拉（多页面筛选器依赖）
	"/api/v1/service-trees",     // 服务树（资产筛选器依赖）
	"/api/v1/ticket-types/all",  // 工单类型下拉
	"/api/v1/users",             // 用户列表（负责人选择器依赖）
}

// 公共路由前缀白名单（不限方法，所有已认证用户可访问）
var casbinPrefixWhitelistAny = []string{
	"/api/v1/notifications/in-app",
	"/api/v1/ws/",
}

// 公共路由前缀白名单（仅 GET）
var casbinPrefixWhitelistGET = []string{
	"/api/v1/stats/",
	"/api/v1/users/",   // /users/:id/roles 等查询
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
