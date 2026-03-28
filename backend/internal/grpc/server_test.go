package grpc

import (
	"fmt"
	"os"
	"testing"
	"time"

	pb "github.com/bigops/platform/proto/gen/agent"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
	"github.com/bigops/platform/internal/repository"
	"go.uber.org/zap"
)

func initTestDB(t *testing.T) {
	t.Helper()
	if func() bool {
		defer func() {
			_ = recover()
		}()
		_ = database.GetDB()
		return true
	}() {
		return
	}
	cfg := database.MySQLConfig{
		Host:     getenvOr("BIGOPS_TEST_DB_HOST", "127.0.0.1"),
		Port:     getenvIntOr("BIGOPS_TEST_DB_PORT", 3306),
		Username: getenvOr("BIGOPS_TEST_DB_USER", "root"),
		Password: getenvOr("BIGOPS_TEST_DB_PASSWORD", "DBB4CuIwxc"),
		Database: getenvOr("BIGOPS_TEST_DB_NAME", "bigops2"),
		Charset:  "utf8mb4",
	}
	if err := database.InitMySQL(cfg, zap.NewNop()); err != nil {
		t.Fatalf("init test mysql failed: %v", err)
	}
}

func getenvOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvIntOr(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		var parsed int
		_, _ = fmt.Sscanf(value, "%d", &parsed)
		if parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func TestUpsertHeartbeatAutoCreatesAsset(t *testing.T) {
	initTestDB(t)
	db := database.GetDB()

	hostname := fmt.Sprintf("agent-auto-asset-%d", time.Now().UnixNano())
	agentID := fmt.Sprintf("agent-auto-%d", time.Now().UnixNano())
	publicIP := fmt.Sprintf("47.96.%d.%d", time.Now().Second()%200+1, time.Now().Nanosecond()%200+1)
	privateIP := fmt.Sprintf("10.66.%d.%d", time.Now().Second()%200+1, (time.Now().Nanosecond()/10)%200+1)

	defer db.Where("agent_id = ?", agentID).Delete(&model.AgentInfo{})
	defer db.Where("hostname = ? OR ip IN ? OR inner_ip IN ?", hostname, []string{publicIP, privateIP}, []string{publicIP, privateIP}).Delete(&model.Asset{})

	server := &Server{
		agentRepo:  repository.NewAgentRepository(),
		execRepo:   repository.NewTaskExecutionRepository(),
		sampleRepo: repository.NewAgentMetricSampleRepository(),
	}

	server.upsertHeartbeat(agentID, &pb.HeartbeatRequest{
		AgentId:              agentID,
		Hostname:             hostname,
		Ip:                   privateIP,
		PrivateIp:            privateIP,
		PublicIp:             publicIP,
		Os:                   "linux/amd64",
		Version:              "test",
		CpuCount:             4,
		CpuUsagePercent:      10.5,
		MemoryTotal:          8192,
		MemoryUsed:           4096,
		MemoryUsagePercent:   50,
		DiskTotal:            102400,
		DiskUsed:             51200,
		DiskUsagePercent:     50,
	}, model.LocalTime(time.Now()), "")

	var asset model.Asset
	if err := db.Where("ip = ?", publicIP).First(&asset).Error; err != nil {
		t.Fatalf("expected asset auto created by heartbeat, got error: %v", err)
	}
	if asset.Hostname != hostname {
		t.Fatalf("expected asset hostname %q, got %q", hostname, asset.Hostname)
	}
	if asset.InnerIP != privateIP {
		t.Fatalf("expected asset inner_ip %q, got %q", privateIP, asset.InnerIP)
	}
	if asset.Source != "agent" {
		t.Fatalf("expected asset source 'agent', got %q", asset.Source)
	}
}
