// Package database 提供 MySQL 和 Redis 的连接管理。
// 采用全局单例模式，应用启动时初始化，运行期间通过 GetDB / GetRedis 获取实例。
package database

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// globalDB 全局 MySQL 数据库实例。
var globalDB *gorm.DB

// MySQLConfig MySQL 连接配置。
type MySQLConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Charset  string
}

// InitMySQL 根据配置初始化 MySQL 连接，包括：
//   - 构建 DSN 并建立连接
//   - 配置连接池参数（最大连接数、空闲连接数、连接最大存活时间）
//   - 执行 Ping 验证连接可用性
func InitMySQL(cfg MySQLConfig, log *zap.Logger) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
	)

	// 设置 GORM 日志级别为 Info，便于开发阶段排查 SQL 问题
	gormLogger := logger.Default.LogMode(logger.Info)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// 获取底层 sql.DB 以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// 连接池参数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大存活时间

	// 验证连接可用
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping MySQL: %w", err)
	}

	globalDB = db
	log.Info("MySQL connected successfully",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Database),
	)

	return nil
}

// GetDB 返回全局 MySQL 数据库实例。若未初始化则 panic。
func GetDB() *gorm.DB {
	if globalDB == nil {
		panic("database not initialized, call InitMySQL() first")
	}
	return globalDB
}

// Close 关闭 MySQL 数据库连接，释放连接池资源。
func Close() error {
	if globalDB != nil {
		sqlDB, err := globalDB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
