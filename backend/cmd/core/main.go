// Package main 是 BigOps Core 模块的入口。
// 负责按顺序初始化各基础设施组件（配置、日志、数据库、路由），然后启动 HTTP 服务。
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	_ "github.com/bigops/platform/docs"

	"github.com/bigops/platform/api/http/router"
	grpcserver "github.com/bigops/platform/internal/grpc"
	"github.com/bigops/platform/internal/model"
	casbinPkg "github.com/bigops/platform/internal/pkg/casbin"
	"github.com/bigops/platform/internal/pkg/config"
	"github.com/bigops/platform/internal/pkg/database"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/repository"
	"github.com/bigops/platform/internal/service"
	cloudsync "github.com/bigops/platform/internal/service/cloud_sync"
)

// @title BigOps API
// @version 1.0
// @description BigOps 运维平台 API 文档
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer token 认证，格式: Bearer {token}
func main() {
	// 1. 加载配置文件，优先使用环境变量 CONFIG_PATH 指定的路径
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	if err := config.Load(configPath); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	cfg := config.Get()

	// 2. 初始化日志
	loggerCfg := logger.Config{
		Level:      cfg.Log.Level,
		Filename:   cfg.Log.Filename,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	}
	if err := logger.Init(loggerCfg); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting BigOps Core Module")

	// 3. 初始化 MySQL
	mysqlCfg := database.MySQLConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Username: cfg.Database.Username,
		Password: cfg.Database.Password,
		Database: cfg.Database.Database,
		Charset:  cfg.Database.Charset,
	}
	if err := database.InitMySQL(mysqlCfg, logger.Get()); err != nil {
		logger.Fatal("Failed to initialize MySQL", zap.Error(err))
	}
	defer database.Close()

	// 自动迁移数据库表结构（开发阶段使用，生产环境建议使用 SQL 迁移脚本）
	if err := database.GetDB().AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.Menu{},
		&model.UserRole{},
		&model.AuditLog{},
		&model.ServiceTree{},
		&model.CloudAccount{},
		&model.Asset{},
		&model.AssetChange{},
		&model.CloudSyncTask{},
		&model.Department{},
		&model.Ticket{},
		&model.TicketType{},
		&model.TicketActivity{},
		&model.RequestTemplate{},
		&model.ApprovalPolicy{},
		&model.ApprovalPolicyStage{},
		&model.ApprovalInstance{},
		&model.ApprovalRecord{},
		&model.ExecutionOrder{},
		&model.NotificationEvent{},
		&model.NotificationDelivery{},
		&model.InAppNotification{},
		&model.Task{},
		&model.TaskExecution{},
		&model.TaskHostResult{},
		&model.AgentInfo{},
	); err != nil {
		logger.Fatal("Failed to migrate database", zap.Error(err))
	}
	logger.Info("Database migration completed")

	// 初始化任务中心菜单（幂等）
	seedTaskMenus()

	// 初始化 Casbin 权限引擎
	if err := casbinPkg.Init(database.GetDB()); err != nil {
		logger.Fatal("Failed to initialize Casbin", zap.Error(err))
	}
	syncCasbinPolicies()
	logger.Info("Casbin initialized")

	// 4. 初始化 Redis
	redisCfg := database.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	if err := database.InitRedis(redisCfg, logger.Get()); err != nil {
		logger.Fatal("Failed to initialize Redis", zap.Error(err))
	}
	defer database.CloseRedis()

	// 5. 启动 gRPC Server
	grpcPort := cfg.GRPC.Port
	if grpcPort == 0 {
		grpcPort = 9090
	}
	grpcSrv, err := grpcserver.StartGRPCServer(grpcPort)
	if err != nil {
		logger.Fatal("Failed to start gRPC server", zap.Error(err))
	}
	defer grpcSrv.GracefulStop()
	logger.Info(fmt.Sprintf("gRPC server started on :%d", grpcPort))

	// 6. 初始化 HTTP 路由并启动服务
	r := router.Setup(cfg.Server.Mode)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		logger.Info(fmt.Sprintf("Server starting on %s", addr))
		if err := r.Run(addr); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 启动云同步调度器
	syncScheduler := cloudsync.NewScheduler()
	syncScheduler.Start()
	defer syncScheduler.Stop()
	logger.Info("Cloud sync scheduler started")

	notificationScheduler := service.NewNotificationScheduler()
	notificationScheduler.Start()
	defer notificationScheduler.Stop()
	logger.Info("Notification scheduler started")

	// 6. 优雅关闭：等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down...")
}

// seedTaskMenus 幂等插入任务中心菜单（4 条）。
func seedTaskMenus() {
	db := database.GetDB()
	var count int64
	db.Model(&model.Menu{}).Where("name = ?", "task_dir").Count(&count)
	if count > 0 {
		return // 已存在，跳过
	}

	dir := model.Menu{
		ParentID: 0, Name: "task_dir", Title: "任务中心",
		Icon: "Operation", Type: 1, Sort: 60, Visible: 1, Status: 1,
	}
	if err := db.Create(&dir).Error; err != nil {
		logger.Warn("seed task_dir menu failed", zap.Error(err))
		return
	}

	children := []model.Menu{
		{ParentID: dir.ID, Name: "task_list", Title: "任务管理", Icon: "List", Path: "/task/list", Component: "TaskList", APIPath: "/api/v1/tasks", APIMethod: "GET", Type: 2, Sort: 1, Visible: 1, Status: 1},
		{ParentID: dir.ID, Name: "task_create", Title: "创建任务", Path: "/task/create", Component: "TaskCreate", APIPath: "/api/v1/tasks", APIMethod: "POST", Type: 2, Sort: 2, Visible: 0, Status: 1},
		{ParentID: dir.ID, Name: "task_execution", Title: "执行详情", Path: "/task/execution", Component: "TaskExecution", APIPath: "/api/v1/task-executions/:id", APIMethod: "GET", Type: 2, Sort: 3, Visible: 0, Status: 1},
		{ParentID: dir.ID, Name: "agent_list", Title: "Agent 管理", Icon: "Monitor", Path: "/task/agents", Component: "AgentList", APIPath: "/api/v1/agents", APIMethod: "GET", Type: 2, Sort: 4, Visible: 1, Status: 1},
	}
	for _, m := range children {
		if err := db.Create(&m).Error; err != nil {
			logger.Warn("seed task menu failed", zap.String("name", m.Name), zap.Error(err))
		}
	}
	logger.Info("Task center menus seeded")
}

// syncCasbinPolicies 启动时从 DB 同步所有 Casbin 规则。
func syncCasbinPolicies() {
	enforcer := casbinPkg.GetEnforcer()

	// 清空现有策略，重新从 DB 同步
	enforcer.ClearPolicy()

	db := database.GetDB()

	// 1. 同步 policy: 遍历所有角色 → 获取其菜单 → 写入 p(role, api_path, api_method)
	var roles []model.Role
	db.Where("status = 1").Find(&roles)

	roleRepo := repository.NewRoleRepository()
	menuRepo := repository.NewMenuRepository()

	for _, role := range roles {
		if role.Name == "admin" {
			continue // admin 在 matcher 中 bypass
		}
		menuIDs, err := roleRepo.GetMenusByRoleID(role.ID)
		if err != nil || len(menuIDs) == 0 {
			continue
		}
		menus, err := menuRepo.GetByIDs(menuIDs)
		if err != nil {
			continue
		}
		for _, menu := range menus {
			if menu.APIPath != "" && menu.APIMethod != "" {
				enforcer.AddPolicy(role.Name, menu.APIPath, menu.APIMethod)
			}
		}
	}

	// 2. 同步 grouping: 遍历所有用户-角色关系 → 写入 g(username, role_name)
	var userRoles []model.UserRole
	db.Find(&userRoles)

	userRepo := repository.NewUserRepository()
	for _, ur := range userRoles {
		user, err := userRepo.GetByID(ur.UserID)
		if err != nil {
			continue
		}
		role, err := roleRepo.GetByID(ur.RoleID)
		if err != nil {
			continue
		}
		enforcer.AddRoleForUser(user.Username, role.Name)
	}

	enforcer.SavePolicy()
	logger.Info("Casbin policies synced from database")
}
