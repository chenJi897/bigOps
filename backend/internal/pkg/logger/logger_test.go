package logger

import (
	"os"
	"testing"

	"go.uber.org/zap"
)

// TestInit 验证日志初始化及写入流程：
// 初始化后写入多条不同级别的日志，确认日志文件非空。
func TestInit(t *testing.T) {
	// 创建临时日志文件
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

	// 写入不同级别的日志
	Info("测试 info 消息", zap.String("key", "value"))
	Debug("测试 debug 消息")
	Warn("测试 warn 消息")

	// 刷新缓冲区，确保日志写入磁盘
	// 注意：Sync 对 stdout 会返回 "invalid argument"，这是 Linux 上的已知行为，可忽略
	_ = Sync()

	// 检查日志文件是否有内容
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(content) == 0 {
		t.Error("日志文件为空")
	}
}
