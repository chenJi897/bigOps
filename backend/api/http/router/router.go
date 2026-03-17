// Package router 负责 HTTP 路由的注册与初始化。
// 包含全局中间件、健康检查、Swagger 文档以及 API v1 路由组。
package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/bigops/platform/internal/handler"
	"github.com/bigops/platform/internal/middleware"
)

// Setup 创建并配置 Gin 路由引擎。
// mode 对应 Gin 的运行模式：debug / release / test。
func Setup(mode string) *gin.Engine {
	gin.SetMode(mode)

	r := gin.New()

	// 全局中间件：请求日志 + panic 恢复
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// 健康检查，供负载均衡器 / K8s 探针使用
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// Swagger API 文档（仅开发环境建议开启）
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})

		// --- 认证模块路由 ---
		authHandler := handler.NewAuthHandler()

		// 公开路由（无需认证）
		v1.POST("/auth/register", authHandler.Register)
		v1.POST("/auth/login", authHandler.Login)

		// 需要认证的路由
		authGroup := v1.Group("")
		authGroup.Use(middleware.AuthMiddleware())
		{
			authGroup.POST("/auth/logout", authHandler.Logout)
			authGroup.GET("/auth/info", authHandler.GetUserInfo)
			authGroup.PUT("/auth/password", authHandler.ChangePassword)
		}
	}

	return r
}
