package config

import (
	"os"
	"testing"
)

// TestLoad 验证从临时 YAML 文件加载配置的完整流程，
// 包括文件读取、反序列化以及字段值的正确性。
func TestLoad(t *testing.T) {
	// 构造一份完整的测试配置
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
	// 写入临时文件，测试结束后自动清理
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

	// 加载配置并验证
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
