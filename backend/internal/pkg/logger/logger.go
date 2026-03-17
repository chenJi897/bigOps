// Package logger 提供基于 Zap 的结构化日志功能。
// 支持同时输出到文件和控制台，文件日志通过 lumberjack 实现自动轮转。
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// globalLogger 全局日志实例，由 Init 函数初始化。
var globalLogger *zap.Logger

// Config 日志配置，与 config.LogConfig 中的字段一一对应。
type Config struct {
	Level      string // 日志级别：debug, info, warn, error
	Filename   string // 日志文件路径
	MaxSize    int    // 单个日志文件最大体积（MB）
	MaxBackups int    // 保留的旧日志文件数量上限
	MaxAge     int    // 旧日志文件最大保留天数
	Compress   bool   // 是否对轮转后的日志进行 gzip 压缩
}

// Init 根据给定配置初始化全局日志实例。
// 日志会同时写入文件（JSON 格式）和控制台（可读格式），方便开发调试和生产排查。
func Init(cfg Config) error {
	// 解析日志级别，无法识别时默认使用 Info
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	// 编码器配置：统一时间格式、字段命名
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 文件输出：通过 lumberjack 实现自动轮转
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	})

	// 控制台输出
	consoleWriter := zapcore.AddSync(os.Stdout)

	// 多路输出：文件使用 JSON 格式，控制台使用可读格式
	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			fileWriter,
			level,
		),
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			consoleWriter,
			level,
		),
	)

	// 创建日志实例，附加调用者信息（跳过一层封装函数）
	globalLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return nil
}

// Get 返回全局日志实例。若未初始化，返回一个默认的生产级日志实例。
func Get() *zap.Logger {
	if globalLogger == nil {
		globalLogger, _ = zap.NewProduction()
	}
	return globalLogger
}

// Debug 输出 Debug 级别日志。
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info 输出 Info 级别日志。
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn 输出 Warn 级别日志。
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error 输出 Error 级别日志。
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal 输出 Fatal 级别日志并终止程序。
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// Sync 刷新缓冲区中的日志条目，应在程序退出前调用。
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}
