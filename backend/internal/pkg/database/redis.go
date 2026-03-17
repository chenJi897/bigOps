package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// globalRedis 全局 Redis 客户端实例。
var globalRedis *redis.Client

// RedisConfig Redis 连接配置。
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int // Redis 数据库编号（0-15）
}

// InitRedis 根据配置初始化 Redis 连接，并通过 PING 验证连通性。
func InitRedis(cfg RedisConfig, log *zap.Logger) error {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 发送 PING 验证连接可用
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	globalRedis = client
	log.Info("Redis connected successfully",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.Int("db", cfg.DB),
	)

	return nil
}

// GetRedis 返回全局 Redis 客户端实例。若未初始化则 panic。
func GetRedis() *redis.Client {
	if globalRedis == nil {
		panic("redis not initialized, call InitRedis() first")
	}
	return globalRedis
}

// CloseRedis 关闭 Redis 连接，释放资源。
func CloseRedis() error {
	if globalRedis != nil {
		return globalRedis.Close()
	}
	return nil
}
