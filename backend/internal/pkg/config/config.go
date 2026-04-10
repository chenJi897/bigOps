// Package config 提供 BigOps 平台的全局配置管理。
// 基于 Viper 加载 YAML 配置文件，并支持通过环境变量覆盖配置项。
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config 根配置结构体，聚合所有子系统的配置。
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
	Encrypt  EncryptConfig  `mapstructure:"encrypt"`
	Aliyun   AliyunConfig   `mapstructure:"aliyun"`
	Notification NotificationConfig `mapstructure:"notification"`
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	Agent    AgentConfig    `mapstructure:"agent"`
}

// GRPCConfig gRPC 服务器配置。
type GRPCConfig struct {
	Port int `mapstructure:"port"`
}

// EncryptConfig 加密配置。
type EncryptConfig struct {
	Key string `mapstructure:"key"` // 32 字节 AES-256 密钥（hex 编码）
}

// AliyunConfig 阿里云相关配置。
type AliyunConfig struct {
	ECSEndpoint string `mapstructure:"ecs_endpoint"` // ECS API 地址模板，含 %s 占位符表示 region，例如 ecs.%s.aliyuncs.com
}

// AgentConfig Agent进程配置（对应 agent.yaml）。
type AgentConfig struct {
	ID                    string            `mapstructure:"id" json:"id"`
	StateFile             string            `mapstructure:"state_file" json:"state_file"`
	Hostname              string            `mapstructure:"hostname" json:"hostname"`
	PublicIP              string            `mapstructure:"public_ip" json:"public_ip"`
	PublicIPProvider      string            `mapstructure:"public_ip_provider" json:"public_ip_provider"`
	PublicIPCacheFile     string            `mapstructure:"public_ip_cache_file" json:"public_ip_cache_file"`
	PublicIPTimeoutSeconds int              `mapstructure:"public_ip_timeout_seconds" json:"public_ip_timeout_seconds"`
	PublicIPRefreshHours  int               `mapstructure:"public_ip_refresh_hours" json:"public_ip_refresh_hours"`
	Labels                map[string]string `mapstructure:"labels" json:"labels"`
	MaxCPURate            float64           `mapstructure:"max_cpu_rate" json:"max_cpu_rate"`
	MaxMemMB              int               `mapstructure:"max_mem_mb" json:"max_mem_mb"`
}

// NotificationConfig 通知中心配置。
type NotificationConfig struct {
	DefaultChannels          []string `mapstructure:"default_channels" json:"default_channels"`
	EnabledChannelTypes      []string `mapstructure:"enabled_channel_types" json:"enabled_channel_types"` // 管理员允许的外部渠道类型
	MaxRetries               int      `mapstructure:"max_retries" json:"max_retries"`
	RetryIntervalSeconds     int      `mapstructure:"retry_interval_seconds" json:"retry_interval_seconds"`
	RetryScanIntervalSeconds int      `mapstructure:"retry_scan_interval_seconds" json:"retry_scan_interval_seconds"`
}

// ServerConfig HTTP 服务器配置。
type ServerConfig struct {
	Address string `mapstructure:"address" json:"address"` // gRPC server address for agent
	Port    int    `mapstructure:"port"`
	Mode    string `mapstructure:"mode"` // Gin 运行模式：debug, release, test
}

// DatabaseConfig MySQL 数据库连接配置。
type DatabaseConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	Database    string `mapstructure:"database"`
	Charset     string `mapstructure:"charset"`
	AutoMigrate bool   `mapstructure:"auto_migrate"` // 是否自动迁移表结构，生产环境建议关闭
}

// RedisConfig Redis 连接配置。
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig JWT 认证配置。
type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire"` // 令牌有效期，单位：秒
}

// LogConfig 日志及日志轮转配置（基于 lumberjack）。
type LogConfig struct {
	Level      string `mapstructure:"level"`       // 日志级别：debug, info, warn, error
	Filename   string `mapstructure:"filename"`    // 日志文件路径
	MaxSize    int    `mapstructure:"max_size"`    // 单个日志文件最大体积（MB），超过后触发轮转
	MaxBackups int    `mapstructure:"max_backups"` // 保留的旧日志文件数量上限
	MaxAge     int    `mapstructure:"max_age"`     // 旧日志文件最大保留天数
	Compress   bool   `mapstructure:"compress"`    // 是否对轮转后的日志文件进行 gzip 压缩
}

// globalConfig 全局配置单例，由 Load 函数设置。
var globalConfig *Config
var currentConfigPath string

// Load 从指定路径读取 YAML 配置文件，解析到 Config 结构体，
// 并将其存储为全局配置。环境变量可通过 Viper 的 AutomaticEnv 覆盖文件中的值。
func Load(configPath string) error {
	currentConfigPath = configPath
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 允许环境变量（如 SERVER_PORT）覆盖配置文件中的值
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	globalConfig = &cfg
	return nil
}

// Get 返回全局配置实例。若 Load 尚未调用，会直接 panic，
// 以便在应用启动阶段尽早暴露配置问题。
func Get() *Config {
	if globalConfig == nil {
		panic("config not loaded, call Load() first")
	}
	return globalConfig
}

func ConfigPath() string {
	return currentConfigPath
}

// GetAgentConfig 返回Agent配置
func GetAgentConfig() AgentConfig {
	if globalConfig == nil {
		panic("config not loaded, call Load() first")
	}
	return globalConfig.Agent
}

func UpdateNotificationConfig(notificationCfg NotificationConfig) error {
	if currentConfigPath == "" {
		return fmt.Errorf("config path is empty")
	}
	v := viper.New()
	v.SetConfigFile(currentConfigPath)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	v.Set("notification.default_channels", notificationCfg.DefaultChannels)
	v.Set("notification.max_retries", notificationCfg.MaxRetries)
	v.Set("notification.retry_interval_seconds", notificationCfg.RetryIntervalSeconds)
	v.Set("notification.retry_scan_interval_seconds", notificationCfg.RetryScanIntervalSeconds)

	if err := v.WriteConfigAs(currentConfigPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	if globalConfig != nil {
		globalConfig.Notification = notificationCfg
	}
	return nil
}
