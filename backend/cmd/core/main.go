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

	// 初始化 Casbin 权限引擎
	if err := casbinPkg.Init(database.GetDB()); err != nil {
		logger.Fatal("Failed to initialize Casbin", zap.Error(err))
	}
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
