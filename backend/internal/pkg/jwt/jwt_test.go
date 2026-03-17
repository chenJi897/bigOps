package jwt

import (
	"os"
	"testing"

	"github.com/bigops/platform/internal/pkg/config"
)

// setupConfig 创建临时配置文件并加载，供测试使用。
func setupConfig(t *testing.T) {
	t.Helper()
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
  secret: test-secret-key-for-unit-test
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
	t.Cleanup(func() { os.Remove(tmpfile.Name()) })
	tmpfile.Write([]byte(content))
	tmpfile.Close()

	if err := config.Load(tmpfile.Name()); err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
}

// TestGenerateAndParse 验证 token 生成和解析的完整流程。
func TestGenerateAndParse(t *testing.T) {
	setupConfig(t)

	// 生成 token
	token, err := GenerateToken(1, "admin")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken() 返回空 token")
	}

	// 解析 token
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != 1 {
		t.Errorf("UserID = %d, want 1", claims.UserID)
	}
	if claims.Username != "admin" {
		t.Errorf("Username = %s, want admin", claims.Username)
	}

	// 无效 token 应报错
	_, err = ParseToken("invalid-token")
	if err == nil {
		t.Error("ParseToken() 无效 token 应返回错误")
	}
}
