# Phase 1: Backend Foundation Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the backend foundation including Go module initialization, database connections, configuration management, logging, and basic HTTP server setup.

**Architecture:** Layered architecture with clear separation of concerns - configuration layer, database layer, middleware layer, and HTTP routing. All infrastructure code goes in `internal/pkg/` for reusability across modules.

**Tech Stack:** Go 1.21+, Gin, GORM, MySQL, Redis, Viper, Zap, Swag

---

## File Structure Overview

This phase creates the foundational infrastructure:

```
backend/
├── go.mod                              # Go module definition
├── go.sum                              # Dependency lock file
├── cmd/
│   └── core/
│       └── main.go                     # Core module entry point
├── internal/
│   └── pkg/
│       ├── config/
│       │   └── config.go               # Configuration management
│       ├── database/
│       │   ├── mysql.go                # MySQL connection
│       │   └── redis.go                # Redis connection
│       ├── logger/
│       │   └── logger.go               # Logging setup
│       ├── response/
│       │   └── response.go             # HTTP response wrapper
│       └── validator/
│           └── validator.go            # Request validation
├── api/
│   └── http/
│       └── router/
│           └── router.go               # HTTP router setup
└── config/
    ├── config.yaml                     # Default configuration
    └── config.example.yaml             # Configuration template
```

---

## Task 1: Initialize Go Module and Dependencies

**Files:**
- Create: `backend/go.mod`
- Create: `backend/.gitignore`

- [ ] **Step 1: Initialize Go module**

```bash
cd /root/bigOps/backend
go mod init github.com/bigops/platform
```

Expected output: `go: creating new go.mod: module github.com/bigops/platform`

- [ ] **Step 2: Add core dependencies**

```bash
# Web framework
go get -u github.com/gin-gonic/gin@v1.9.1

# Database
go get -u gorm.io/gorm@v1.25.5
go get -u gorm.io/driver/mysql@v1.5.2

# Redis
go get -u github.com/redis/go-redis/v9@v9.3.0

# Configuration
go get -u github.com/spf13/viper@v1.18.0

# Logging
go get -u go.uber.org/zap@v1.26.0

# JWT
go get -u github.com/golang-jwt/jwt/v5@v5.2.0

# Validation
go get -u github.com/go-playground/validator/v10@v10.16.0

# Swagger
go get -u github.com/swaggo/swag/cmd/swag@v1.16.2
go get -u github.com/swaggo/gin-swagger@v1.6.0
go get -u github.com/swaggo/files@v1.0.1
```

- [ ] **Step 3: Create .gitignore**

Create `backend/.gitignore`:
```
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
bin/
dist/

# Test binary
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/

# Go workspace file
go.work

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# Config files with secrets
config/config.local.yaml

# Logs
*.log
logs/

# OS
.DS_Store
Thumbs.db
```

- [ ] **Step 4: Verify dependencies**

```bash
go mod tidy
go mod verify
```

Expected: All dependencies downloaded and verified

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum .gitignore
git commit -m "chore: initialize Go module and add dependencies"
```

---

## Task 2: Configuration Management

**Files:**
- Create: `backend/internal/pkg/config/config.go`
- Create: `backend/config/config.yaml`
- Create: `backend/config/config.example.yaml`

- [ ] **Step 1: Write configuration structure**

Create `backend/internal/pkg/config/config.go`:
```go
package config

import (
	"fmt"
	"github.com/spf13/viper"
)

// Config holds all configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug, release, test
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	Charset  string `mapstructure:"charset"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire"` // seconds
}

type LogConfig struct {
	Level      string `mapstructure:"level"` // debug, info, warn, error
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`    // megabytes
	MaxBackups int    `mapstructure:"max_backups"` // number of backups
	MaxAge     int    `mapstructure:"max_age"`     // days
	Compress   bool   `mapstructure:"compress"`
}

var globalConfig *Config

// Load loads configuration from file
func Load(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Allow environment variables to override config
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

// Get returns the global configuration
func Get() *Config {
	if globalConfig == nil {
		panic("config not loaded, call Load() first")
	}
	return globalConfig
}
```

- [ ] **Step 2: Create default configuration file**

Create `backend/config/config.yaml`:
```yaml
server:
  port: 8080
  mode: debug

database:
  host: localhost
  port: 3306
  username: bigops
  password: bigops123
  database: bigops
  charset: utf8mb4

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

jwt:
  secret: your-secret-key-change-in-production
  expire: 7200  # 2 hours

log:
  level: debug
  filename: logs/bigops.log
  max_size: 100
  max_backups: 10
  max_age: 30
  compress: true
```

- [ ] **Step 3: Create example configuration**

```bash
cp backend/config/config.yaml backend/config/config.example.yaml
```

- [ ] **Step 4: Test configuration loading**

Create temporary test file `backend/internal/pkg/config/config_test.go`:
```go
package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	content := `
server:
  port: 8080
  mode: debug
database:
  host: localhost
  port: 3306
  username: test
  password: test
  database: test
  charset: utf8mb4
redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
jwt:
  secret: test-secret
  expire: 7200
log:
  level: debug
  filename: test.log
  max_size: 100
  max_backups: 10
  max_age: 30
  compress: true
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Test loading
	if err := Load(tmpfile.Name()); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	cfg := Get()
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Database.Username != "test" {
		t.Errorf("Database.Username = %s, want test", cfg.Database.Username)
	}
}
```

Run test:
```bash
cd /root/bigOps/backend
go test ./internal/pkg/config -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/pkg/config/ config/
git commit -m "feat: add configuration management with Viper"
```

---

## Task 3: Logger Setup

**Files:**
- Create: `backend/internal/pkg/logger/logger.go`

- [ ] **Step 1: Write logger implementation**

Create `backend/internal/pkg/logger/logger.go`:
```go
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var globalLogger *zap.Logger

// Config holds logger configuration
type Config struct {
	Level      string
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

// Init initializes the global logger
func Init(cfg Config) error {
	// Parse log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	// Create encoder config
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

	// Create file writer with rotation
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	})

	// Create console writer
	consoleWriter := zapcore.AddSync(os.Stdout)

	// Create core that writes to both file and console
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

	// Create logger
	globalLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return nil
}

// Get returns the global logger
func Get() *zap.Logger {
	if globalLogger == nil {
		// Return a default logger if not initialized
		globalLogger, _ = zap.NewProduction()
	}
	return globalLogger
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// Sync flushes any buffered log entries
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}
```

- [ ] **Step 2: Add lumberjack dependency**

```bash
cd /root/bigOps/backend
go get -u gopkg.in/natefinch/lumberjack.v2
go mod tidy
```

- [ ] **Step 3: Write logger test**

Create `backend/internal/pkg/logger/logger_test.go`:
```go
package logger

import (
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestInit(t *testing.T) {
	// Create temp log file
	tmpfile, err := os.CreateTemp("", "test-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	cfg := Config{
		Level:      "debug",
		Filename:   tmpfile.Name(),
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
	}

	if err := Init(cfg); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test logging
	Info("test info message", zap.String("key", "value"))
	Debug("test debug message")
	Warn("test warn message")

	// Sync to ensure logs are written
	if err := Sync(); err != nil {
		t.Errorf("Sync() error = %v", err)
	}

	// Check if log file has content
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(content) == 0 {
		t.Error("Log file is empty")
	}
}
```

Run test:
```bash
cd /root/bigOps/backend
go test ./internal/pkg/logger -v
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/pkg/logger/
git commit -m "feat: add structured logging with Zap and log rotation"
```

---

## Task 4: Database Connection - MySQL

**Files:**
- Create: `backend/internal/pkg/database/mysql.go`

- [ ] **Step 1: Write MySQL connection code**

Create `backend/internal/pkg/database/mysql.go`:
```go
package database

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var globalDB *gorm.DB

// MySQLConfig holds MySQL configuration
type MySQLConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Charset  string
}

// InitMySQL initializes MySQL connection
func InitMySQL(cfg MySQLConfig, log *zap.Logger) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
	)

	// Configure GORM logger
	gormLogger := logger.Default.LogMode(logger.Info)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// Get underlying sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test connection
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

// GetDB returns the global database instance
func GetDB() *gorm.DB {
	if globalDB == nil {
		panic("database not initialized, call InitMySQL() first")
	}
	return globalDB
}

// Close closes the database connection
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
```

- [ ] **Step 2: Verify code compiles**

```bash
cd /root/bigOps/backend
go build ./internal/pkg/database
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/pkg/database/mysql.go
git commit -m "feat: add MySQL connection with GORM"
```

---

## Task 5: Database Connection - Redis

**Files:**
- Create: `backend/internal/pkg/database/redis.go`

- [ ] **Step 1: Write Redis connection code**

Create `backend/internal/pkg/database/redis.go`:
```go
package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var globalRedis *redis.Client

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// InitRedis initializes Redis connection
func InitRedis(cfg RedisConfig, log *zap.Logger) error {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
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

// GetRedis returns the global Redis client
func GetRedis() *redis.Client {
	if globalRedis == nil {
		panic("redis not initialized, call InitRedis() first")
	}
	return globalRedis
}

// CloseRedis closes the Redis connection
func CloseRedis() error {
	if globalRedis != nil {
		return globalRedis.Close()
	}
	return nil
}
```

- [ ] **Step 2: Verify code compiles**

```bash
cd /root/bigOps/backend
go build ./internal/pkg/database
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/pkg/database/redis.go
git commit -m "feat: add Redis connection"
```

---

## Task 6: HTTP Response Wrapper

**Files:**
- Create: `backend/internal/pkg/response/response.go`

- [ ] **Step 1: Write response wrapper**

Create `backend/internal/pkg/response/response.go`:
```go
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents a standard API response
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageData represents paginated data
type PageData struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// Success sends a success response
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage sends a success response with custom message
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Error sends an error response
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithData sends an error response with data
func ErrorWithData(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// BadRequest sends a 400 bad request response
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

// Unauthorized sends a 401 unauthorized response
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

// Forbidden sends a 403 forbidden response
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message)
}

// NotFound sends a 404 not found response
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

// InternalServerError sends a 500 internal server error response
func InternalServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}

// Page sends a paginated response
func Page(c *gin.Context, list interface{}, total int64, page, size int) {
	Success(c, PageData{
		List:  list,
		Total: total,
		Page:  page,
		Size:  size,
	})
}
```

- [ ] **Step 2: Verify code compiles**

```bash
cd /root/bigOps/backend
go build ./internal/pkg/response
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/pkg/response/
git commit -m "feat: add HTTP response wrapper"
```

---

## Task 7: Request Validator

**Files:**
- Create: `backend/internal/pkg/validator/validator.go`

- [ ] **Step 1: Write validator wrapper**

Create `backend/internal/pkg/validator/validator.go`:
```go
package validator

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		validate = v
	}
}

// Validate validates a struct
func Validate(obj interface{}) error {
	if validate == nil {
		return fmt.Errorf("validator not initialized")
	}
	return validate.Struct(obj)
}

// TranslateError translates validation errors to readable messages
func TranslateError(err error) string {
	if err == nil {
		return ""
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error()
	}

	var messages []string
	for _, e := range validationErrors {
		messages = append(messages, translateFieldError(e))
	}

	return strings.Join(messages, "; ")
}

func translateFieldError(e validator.FieldError) string {
	field := e.Field()
	tag := e.Tag()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, e.Param())
	case "len":
		return fmt.Sprintf("%s must be %s characters", field, e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of [%s]", field, e.Param())
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}
```

- [ ] **Step 2: Verify code compiles**

```bash
cd /root/bigOps/backend
go build ./internal/pkg/validator
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/pkg/validator/
git commit -m "feat: add request validator wrapper"
```

---

## Task 8: HTTP Router Setup

**Files:**
- Create: `backend/api/http/router/router.go`

- [ ] **Step 1: Write router setup**

Create `backend/api/http/router/router.go`:
```go
package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Setup sets up the HTTP router
func Setup(mode string) *gin.Engine {
	// Set Gin mode
	gin.SetMode(mode)

	// Create router
	r := gin.New()

	// Global middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Auth routes will be added here
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})
	}

	return r
}
```

- [ ] **Step 2: Verify code compiles**

```bash
cd /root/bigOps/backend
go build ./api/http/router
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add api/http/router/
git commit -m "feat: add HTTP router setup with Gin"
```

---

## Task 9: Main Entry Point

**Files:**
- Create: `backend/cmd/core/main.go`

- [ ] **Step 1: Write main entry point**

Create `backend/cmd/core/main.go`:
```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bigops/platform/api/http/router"
	"github.com/bigops/platform/internal/pkg/config"
	"github.com/bigops/platform/internal/pkg/database"
	"github.com/bigops/platform/internal/pkg/logger"
)

// @title BigOps API
// @version 1.0
// @description BigOps platform API documentation
// @host localhost:8080
// @BasePath /api/v1
func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	if err := config.Load(configPath); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	cfg := config.Get()

	// Initialize logger
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

	// Initialize MySQL
	mysqlCfg := database.MySQLConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Username: cfg.Database.Username,
		Password: cfg.Database.Password,
		Database: cfg.Database.Database,
		Charset:  cfg.Database.Charset,
	}
	if err := database.InitMySQL(mysqlCfg, logger.Get()); err != nil {
		logger.Fatal("Failed to initialize MySQL", logger.Error(err.Error()))
	}
	defer database.Close()

	// Initialize Redis
	redisCfg := database.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	if err := database.InitRedis(redisCfg, logger.Get()); err != nil {
		logger.Fatal("Failed to initialize Redis", logger.Error(err.Error()))
	}
	defer database.CloseRedis()

	// Setup HTTP router
	r := router.Setup(cfg.Server.Mode)

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		logger.Info(fmt.Sprintf("Server starting on %s", addr))
		if err := r.Run(addr); err != nil {
			logger.Fatal("Failed to start server", logger.Error(err.Error()))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down...")
}
```

- [ ] **Step 2: Create logs directory**

```bash
mkdir -p /root/bigOps/backend/logs
echo "logs/" >> /root/bigOps/backend/.gitignore
```

- [ ] **Step 3: Build the application**

```bash
cd /root/bigOps/backend
go build -o bin/bigops-core ./cmd/core
```

Expected: Binary created at `bin/bigops-core`

- [ ] **Step 4: Add bin to .gitignore**

```bash
echo "bin/" >> /root/bigOps/backend/.gitignore
```

- [ ] **Step 5: Commit**

```bash
git add cmd/core/ .gitignore
git commit -m "feat: add main entry point for core module"
```

---

## Task 10: Create Makefile

**Files:**
- Create: `backend/Makefile`

- [ ] **Step 1: Write Makefile**

Create `backend/Makefile`:
```makefile
.PHONY: help build run test clean swagger

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building..."
	@go build -o bin/bigops-core ./cmd/core
	@echo "Build complete: bin/bigops-core"

run: ## Run the application
	@echo "Running..."
	@go run ./cmd/core/main.go

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

swagger: ## Generate swagger documentation
	@echo "Generating swagger docs..."
	@swag init -g cmd/core/main.go -o docs/swagger
	@echo "Swagger docs generated"

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete"

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run
	@echo "Lint complete"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies downloaded"

.DEFAULT_GOAL := help
```

- [ ] **Step 2: Test Makefile**

```bash
cd /root/bigOps/backend
make help
```

Expected: Display help menu

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "chore: add Makefile for common tasks"
```

---

## Task 11: Integration Test

**Files:**
- None (testing existing code)

- [ ] **Step 1: Run all tests**

```bash
cd /root/bigOps/backend
make test
```

Expected: All tests pass

- [ ] **Step 2: Build the application**

```bash
make build
```

Expected: Binary created successfully

- [ ] **Step 3: Verify configuration loads**

```bash
./bin/bigops-core
```

Expected: Application starts (will fail to connect to MySQL/Redis, but should load config and logger)
Press Ctrl+C to stop

- [ ] **Step 4: Generate swagger docs**

```bash
make swagger
```

Expected: Swagger docs generated in `docs/swagger/`

- [ ] **Step 5: Add swagger docs to .gitignore**

```bash
echo "docs/swagger/" >> .gitignore
```

- [ ] **Step 6: Final commit**

```bash
git add .gitignore
git commit -m "chore: add swagger docs to gitignore"
```

---

## Completion Checklist

After completing all tasks, verify:

- [ ] All Go files compile without errors
- [ ] All tests pass
- [ ] Application builds successfully
- [ ] Configuration loads correctly
- [ ] Logger initializes and writes logs
- [ ] HTTP server starts on configured port
- [ ] Health check endpoint responds
- [ ] Swagger documentation is accessible
- [ ] All code is committed to git

## Next Steps

After Phase 1 is complete, proceed to:
- Phase 2: User Authentication Module (JWT, password hashing, login/logout)
- Phase 3: RBAC Permission System (Casbin integration)
- Phase 4: Database Models and Migrations

