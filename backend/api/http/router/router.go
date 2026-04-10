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

		// --- 认证模块（公开路由，含限流） ---
		authHandler := handler.NewAuthHandler()
		cicdHandler := handler.NewCICDHandler()
		v1.POST("/auth/register", middleware.RegisterRateLimit(), middleware.AuditLog(), authHandler.Register)
		v1.POST("/auth/login", middleware.LoginRateLimit(), middleware.AuditLog(), authHandler.Login)
		v1.POST("/cicd/webhook/:code", cicdHandler.TriggerByWebhook)

		// --- 需要认证的路由 ---
		authGroup := v1.Group("")
		authGroup.Use(middleware.AuthMiddleware())
		authGroup.Use(middleware.AuditLog())
		authGroup.Use(middleware.CasbinMiddleware())
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
			authGroup.POST("/users/:id", userHandler.Update)
			authGroup.POST("/users/:id/status", userHandler.UpdateStatus)
			authGroup.POST("/users/:id/delete", userHandler.Delete)
			authGroup.POST("/users/:id/department", userHandler.SetDepartment)

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

			// --- 通知中心 ---
			notificationHandler := handler.NewNotificationHandler()
			authGroup.GET("/notifications/in-app", notificationHandler.ListInApp)
			authGroup.GET("/notifications/in-app/unread-count", notificationHandler.CountUnread)
			authGroup.POST("/notifications/in-app/:id/read", notificationHandler.MarkRead)
			authGroup.POST("/notifications/in-app/read-all", notificationHandler.MarkAllRead)
			authGroup.POST("/notifications/in-app/clear-read", notificationHandler.ClearRead)
			authGroup.GET("/notifications/preferences", notificationHandler.GetPreference)
			authGroup.POST("/notifications/preferences", notificationHandler.UpdatePreference)
			authGroup.GET("/notifications/config", notificationHandler.GetConfig)
			authGroup.POST("/notifications/config", notificationHandler.UpdateConfig)
			authGroup.POST("/notifications/test", notificationHandler.TestSend)
			authGroup.GET("/notifications/events", notificationHandler.ListEvents)
			authGroup.POST("/notifications/events/:id/retry", notificationHandler.RetryEvent)
			authGroup.GET("/notifications/templates", notificationHandler.ListTemplates)
			authGroup.POST("/notifications/templates/:id", notificationHandler.UpdateTemplate)
			authGroup.POST("/notifications/templates/preview", notificationHandler.PreviewTemplate)
			authGroup.POST("/notifications/test-webhook", notificationHandler.TestWebhook)
			authGroup.GET("/notifications/enabled-channel-types", notificationHandler.GetEnabledChannelTypes)

			// --- 发送组管理 ---
			authGroup.GET("/notify-groups", notificationHandler.ListGroups)
			authGroup.GET("/notify-groups/all", notificationHandler.ListAllGroups)
			authGroup.GET("/notify-groups/:id", notificationHandler.GetGroup)
			authGroup.POST("/notify-groups", notificationHandler.CreateGroup)
			authGroup.POST("/notify-groups/:id", notificationHandler.UpdateGroup)
			authGroup.POST("/notify-groups/:id/delete", notificationHandler.DeleteGroup)
			authGroup.POST("/notify-groups/:id/test", notificationHandler.TestGroup)

			// --- 部门管理 ---
			departmentHandler := handler.NewDepartmentHandler()
			authGroup.GET("/departments", departmentHandler.List)
			authGroup.GET("/departments/all", departmentHandler.GetAll)
			authGroup.GET("/departments/:id", departmentHandler.GetByID)
			authGroup.POST("/departments", departmentHandler.Create)
			authGroup.POST("/departments/:id", departmentHandler.Update)
			authGroup.POST("/departments/:id/delete", departmentHandler.Delete)

			// --- 统计 ---
			statsHandler := handler.NewStatsHandler()
			authGroup.GET("/stats/summary", statsHandler.Summary)
			authGroup.GET("/stats/asset-distribution", statsHandler.AssetDistribution)
			authGroup.GET("/dashboard/personal", statsHandler.Personal)

			// --- 监控 ---
			monitorHandler := handler.NewMonitorHandler()
			monitorDatasourceHandler := handler.NewMonitorDatasourceHandler()
			alertRuleHandler := handler.NewAlertRuleHandler()
			alertSilenceHandler := handler.NewAlertSilenceHandler()
			onCallHandler := handler.NewOnCallHandler()
			authGroup.GET("/monitor/summary", monitorHandler.Summary)
			authGroup.GET("/monitor/agents", monitorHandler.Agents)
			authGroup.GET("/monitor/agents/:agent_id/trends", monitorHandler.AgentTrend)
			authGroup.GET("/monitor/aggregates/service-trees", monitorHandler.AggregateServiceTrees)
			authGroup.GET("/monitor/aggregates/owners", monitorHandler.AggregateOwners)
			authGroup.GET("/monitor/datasources", monitorDatasourceHandler.List)
			authGroup.POST("/monitor/datasources", monitorDatasourceHandler.Create)
			authGroup.POST("/monitor/datasources/:id", monitorDatasourceHandler.Update)
			authGroup.POST("/monitor/datasources/:id/delete", monitorDatasourceHandler.Delete)
			authGroup.GET("/monitor/datasources/:id/health", monitorDatasourceHandler.Health)
			authGroup.POST("/monitor/query", monitorHandler.Query)
			authGroup.POST("/monitor/query-range", monitorHandler.QueryRange)
			authGroup.GET("/monitor/golden-signals", monitorHandler.GoldenSignals)
			authGroup.GET("/monitor/golden-signals/dimensions", monitorHandler.GoldenSignalsDimensions)
			authGroup.GET("/alert-silences", alertSilenceHandler.List)
			authGroup.POST("/alert-silences", alertSilenceHandler.Create)
			authGroup.POST("/alert-silences/:id", alertSilenceHandler.Update)
			authGroup.POST("/alert-silences/:id/delete", alertSilenceHandler.Delete)
			authGroup.GET("/oncall-schedules", onCallHandler.List)
			authGroup.POST("/oncall-schedules", onCallHandler.Create)
			authGroup.POST("/oncall-schedules/:id", onCallHandler.Update)
			authGroup.POST("/oncall-schedules/:id/delete", onCallHandler.Delete)
			authGroup.GET("/alert-rules", alertRuleHandler.List)
			authGroup.POST("/alert-rules", alertRuleHandler.Create)
			authGroup.POST("/alert-rules/:id", alertRuleHandler.Update)
			authGroup.POST("/alert-rules/:id/delete", alertRuleHandler.Delete)
			authGroup.POST("/alert-rules/evaluate", alertRuleHandler.Evaluate)
			authGroup.GET("/alert-events", alertRuleHandler.Events)
			authGroup.GET("/alert-events/groups", alertRuleHandler.EventGroups)
			authGroup.GET("/alert-events/:id", alertRuleHandler.GetEvent)
			authGroup.GET("/alert-events/:id/timeline", alertRuleHandler.EventTimeline)
			authGroup.GET("/alert-events/:id/root-cause", alertRuleHandler.EventRootCause)
			authGroup.GET("/alert-events/:id/context", alertRuleHandler.EventContext)
			authGroup.POST("/alert-events/:id/ack", alertRuleHandler.AcknowledgeEvent)
			authGroup.POST("/alert-events/:id/resolve", alertRuleHandler.ResolveEvent)

			// --- CI/CD ---
			authGroup.GET("/cicd/projects", cicdHandler.ListProjects)
			authGroup.POST("/cicd/projects", cicdHandler.CreateProject)
			authGroup.POST("/cicd/projects/:id", cicdHandler.UpdateProject)
			authGroup.POST("/cicd/projects/:id/status", cicdHandler.UpdateProjectStatus)
			authGroup.POST("/cicd/projects/:id/delete", cicdHandler.DeleteProject)
			authGroup.GET("/cicd/pipelines", cicdHandler.ListPipelines)
			authGroup.POST("/cicd/pipelines", cicdHandler.CreatePipeline)
			authGroup.POST("/cicd/pipelines/:id", cicdHandler.UpdatePipeline)
			authGroup.POST("/cicd/pipelines/:id/trigger", cicdHandler.TriggerPipeline)
			authGroup.POST("/cicd/pipelines/:id/delete", cicdHandler.DeletePipeline)
			authGroup.GET("/cicd/runs", cicdHandler.ListRuns)
			authGroup.GET("/cicd/runs/:id", cicdHandler.GetRunDetail)
			authGroup.POST("/cicd/runs/:id/retry", cicdHandler.RetryRun)
			authGroup.POST("/cicd/runs/:id/rollback", cicdHandler.RollbackRun)

			// --- 工单类型 ---
			ticketTypeHandler := handler.NewTicketTypeHandler()
			authGroup.GET("/ticket-types", ticketTypeHandler.List)
			authGroup.GET("/ticket-types/all", ticketTypeHandler.GetAll)
			authGroup.POST("/ticket-types", ticketTypeHandler.Create)
			authGroup.POST("/ticket-types/:id", ticketTypeHandler.Update)
			authGroup.POST("/ticket-types/:id/delete", ticketTypeHandler.Delete)

			// --- 请求模板 ---
			requestTemplateHandler := handler.NewRequestTemplateHandler()
			authGroup.GET("/request-templates", requestTemplateHandler.List)
			authGroup.GET("/request-templates/:id", requestTemplateHandler.GetByID)
			authGroup.POST("/request-templates", requestTemplateHandler.Create)
			authGroup.POST("/request-templates/:id", requestTemplateHandler.Update)
			authGroup.POST("/request-templates/:id/delete", requestTemplateHandler.Delete)

			// --- 审批策略 ---
			approvalPolicyHandler := handler.NewApprovalPolicyHandler()
			authGroup.GET("/approval-policies", approvalPolicyHandler.List)
			authGroup.GET("/approval-policies/:id", approvalPolicyHandler.GetByID)
			authGroup.POST("/approval-policies", approvalPolicyHandler.Create)
			authGroup.POST("/approval-policies/:id", approvalPolicyHandler.Update)
			authGroup.POST("/approval-policies/:id/delete", approvalPolicyHandler.Delete)

			// --- 审批待办 ---
			approvalHandler := handler.NewApprovalHandler()
			authGroup.GET("/approval-instances/pending", approvalHandler.Pending)
			authGroup.POST("/approval-instances/:id/approve", approvalHandler.Approve)
			authGroup.POST("/approval-instances/:id/reject", approvalHandler.Reject)

			// --- 工单管理 ---
			ticketHandler := handler.NewTicketHandler()
			authGroup.GET("/tickets", ticketHandler.List)
			authGroup.GET("/tickets/:id", ticketHandler.GetByID)
			authGroup.GET("/tickets/:id/approval-instance", ticketHandler.ApprovalInstance)
			authGroup.POST("/tickets", ticketHandler.Create)
			authGroup.POST("/tickets/:id/assign", ticketHandler.Assign)
			authGroup.POST("/tickets/:id/process", ticketHandler.Process)
			authGroup.POST("/tickets/:id/close", ticketHandler.Close)
			authGroup.POST("/tickets/:id/reopen", ticketHandler.Reopen)
			authGroup.POST("/tickets/:id/comment", ticketHandler.Comment)
			authGroup.POST("/tickets/:id/transfer", ticketHandler.Transfer)
			authGroup.GET("/tickets/:id/activities", ticketHandler.Activities)

			// --- 任务管理 ---
			taskHandler := handler.NewTaskHandler()
			authGroup.GET("/tasks", taskHandler.List)
			authGroup.GET("/tasks/:id", taskHandler.GetByID)
			authGroup.POST("/tasks", taskHandler.Create)
			authGroup.POST("/tasks/:id", taskHandler.Update)
			authGroup.POST("/tasks/:id/delete", taskHandler.Delete)
			authGroup.POST("/tasks/:id/execute", taskHandler.Execute)
			authGroup.GET("/task-executions/:id", taskHandler.GetExecution)
			authGroup.GET("/task-executions", taskHandler.ListExecutions)
			authGroup.POST("/task-executions/:id/cancel", taskHandler.CancelExecution)
			authGroup.POST("/task-executions/:id/retry", taskHandler.RetryExecution)
			inspectionHandler := handler.NewInspectionHandler()
			authGroup.GET("/inspection/templates", inspectionHandler.ListTemplates)
			authGroup.POST("/inspection/templates", inspectionHandler.CreateTemplate)
			authGroup.POST("/inspection/templates/:id", inspectionHandler.UpdateTemplate)
			authGroup.GET("/inspection/plans", inspectionHandler.ListPlans)
			authGroup.POST("/inspection/plans", inspectionHandler.CreatePlan)
			authGroup.POST("/inspection/plans/:id", inspectionHandler.UpdatePlan)
			authGroup.POST("/inspection/plans/:id/run", inspectionHandler.ExecutePlan)
			authGroup.GET("/inspection/records", inspectionHandler.ListRecords)
			authGroup.GET("/inspection/records/:id/report", inspectionHandler.GetRecordReport)
			authGroup.GET("/inspection/records/:id/report/export", inspectionHandler.ExportRecordReport)
			authGroup.GET("/inspection/templates/:id/trend", inspectionHandler.TemplateTrend)
			authGroup.GET("/agents", taskHandler.ListAgents)

		}

		// WebSocket 路由（认证但不审计）
		wsGroup := v1.Group("")
		wsGroup.Use(middleware.AuthMiddleware())
		{
			taskHandler := handler.NewTaskHandler()
			wsGroup.GET("/ws/task-executions/:id/logs", taskHandler.WSLogs)
		}
	}

	return r
}
