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
}

// ServerConfig HTTP 服务器配置。
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // Gin 运行模式：debug, release, test
}

// DatabaseConfig MySQL 数据库连接配置。
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	Charset  string `mapstructure:"charset"`
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

// Load 从指定路径读取 YAML 配置文件，解析到 Config 结构体，
// 并将其存储为全局配置。环境变量可通过 Viper 的 AutomaticEnv 覆盖文件中的值。
func Load(configPath string) error {
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
