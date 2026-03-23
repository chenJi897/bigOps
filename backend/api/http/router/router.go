// Package router 负责 HTTP 路由的注册与初始化。
package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/bigops/platform/internal/handler"
	"github.com/bigops/platform/internal/middleware"
	"github.com/bigops/platform/internal/pkg/response"
)

// Setup 创建并配置 Gin 路由引擎。
func Setup(mode string) *gin.Engine {
	gin.SetMode(mode)

	r := gin.New()

	// 全局中间件：请求日志（Zap） + panic 恢复
	r.Use(middleware.GinLogger())
	r.Use(gin.Recovery())

	// 404 处理
	// 404 路由不存在
	r.NoRoute(func(c *gin.Context) {
		response.NotFound(c, "接口不存在")
	})

	// 405 方法不允许
	r.NoMethod(func(c *gin.Context) {
		response.Error(c, 405, "请求方法不允许")
	})
	r.HandleMethodNotAllowed = true

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger API 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong"})
		})

		// --- 认证模块（公开路由） ---
		authHandler := handler.NewAuthHandler()
		v1.POST("/auth/register", middleware.AuditLog(), authHandler.Register)
		v1.POST("/auth/login", middleware.AuditLog(), authHandler.Login)

		// --- 需要认证的路由 ---
		authGroup := v1.Group("")
		authGroup.Use(middleware.AuthMiddleware())
		authGroup.Use(middleware.AuditLog())
		{
			// 认证相关
			authGroup.POST("/auth/logout", authHandler.Logout)
			authGroup.GET("/auth/info", authHandler.GetUserInfo)
			authGroup.POST("/auth/password", authHandler.ChangePassword)

			// --- 角色管理 ---
			roleHandler := handler.NewRoleHandler()
			authGroup.GET("/roles", roleHandler.List)
			authGroup.GET("/roles/:id", roleHandler.GetByID)
			authGroup.POST("/roles", roleHandler.Create)
			authGroup.POST("/roles/:id", roleHandler.Update)
			authGroup.POST("/roles/:id/delete", roleHandler.Delete)
			authGroup.POST("/roles/:id/status", roleHandler.UpdateStatus)
			authGroup.POST("/roles/:id/menus", roleHandler.SetMenus)

			// --- 用户管理 ---
			userHandler := handler.NewUserHandler()
			authGroup.GET("/users", userHandler.List)
			authGroup.POST("/users/:id/status", userHandler.UpdateStatus)
			authGroup.POST("/users/:id/delete", userHandler.Delete)

			// --- 用户角色管理 ---
			authGroup.GET("/users/:id/roles", roleHandler.GetUserRoles)
			authGroup.POST("/users/:id/roles", roleHandler.SetUserRoles)

			// --- 菜单管理 ---
			menuHandler := handler.NewMenuHandler()
			authGroup.GET("/menus", menuHandler.GetTree)
			authGroup.GET("/menus/user", menuHandler.GetUserMenus)
			authGroup.POST("/menus", menuHandler.Create)
			authGroup.POST("/menus/:id", menuHandler.Update)
			authGroup.POST("/menus/:id/delete", menuHandler.Delete)

			// --- 服务树管理 ---
			serviceTreeHandler := handler.NewServiceTreeHandler()
			authGroup.GET("/service-trees", serviceTreeHandler.GetTree)
			authGroup.GET("/service-trees/asset-counts", serviceTreeHandler.AssetCounts)
			authGroup.GET("/service-trees/:id", serviceTreeHandler.GetByID)
			authGroup.POST("/service-trees", serviceTreeHandler.Create)
			authGroup.POST("/service-trees/:id", serviceTreeHandler.Update)
			authGroup.POST("/service-trees/:id/delete", serviceTreeHandler.Delete)
			authGroup.POST("/service-trees/:id/move", serviceTreeHandler.Move)

			// --- 云账号管理 ---
			cloudAccountHandler := handler.NewCloudAccountHandler()
			syncTaskHandler := handler.NewCloudSyncTaskHandler()
			authGroup.GET("/cloud-accounts", cloudAccountHandler.List)
			authGroup.GET("/cloud-accounts/:id", cloudAccountHandler.GetByID)
			authGroup.POST("/cloud-accounts", cloudAccountHandler.Create)
			authGroup.POST("/cloud-accounts/:id", cloudAccountHandler.Update)
			authGroup.POST("/cloud-accounts/:id/keys", cloudAccountHandler.UpdateKeys)
			authGroup.POST("/cloud-accounts/:id/delete", cloudAccountHandler.Delete)
			authGroup.POST("/cloud-accounts/:id/sync", cloudAccountHandler.Sync)
			authGroup.POST("/cloud-accounts/:id/sync-config", cloudAccountHandler.UpdateSyncConfig)
			authGroup.GET("/cloud-accounts/:id/sync-tasks", syncTaskHandler.GetByAccountID)

			// --- 同步日志 ---
			authGroup.GET("/sync-tasks", syncTaskHandler.List)

			// --- 资产管理 ---
			assetHandler := handler.NewAssetHandler()
			authGroup.GET("/assets", assetHandler.List)
			authGroup.GET("/assets/:id", assetHandler.GetByID)
			authGroup.POST("/assets", assetHandler.Create)
			authGroup.POST("/assets/:id", assetHandler.Update)
			authGroup.POST("/assets/:id/delete", assetHandler.Delete)
			authGroup.GET("/assets/:id/changes", assetHandler.GetChanges)

			// --- 审计日志 ---
			auditLogHandler := handler.NewAuditLogHandler()
			authGroup.GET("/audit-logs", auditLogHandler.List)

			// --- 统计 ---
			statsHandler := handler.NewStatsHandler()
			authGroup.GET("/stats/summary", statsHandler.Summary)
			authGroup.GET("/stats/asset-distribution", statsHandler.AssetDistribution)
		}
	}

	return r
}
