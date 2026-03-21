package main

// 云同步集成测试
// 使用 MockProvider 模拟阿里云 ECS 返回，验证：
// 1. 首次同步 — 创建资产
// 2. 增量同步 — 检测变更并记录
// 3. 云端删除 — 本地未处理（当前设计不自动删除）
//
// 运行：cd /root/bigOps/backend && go run cmd/sync_test/main.go
// 前提：MySQL 和 Redis 已启动，config.yaml 配置正确

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/config"
	"github.com/bigops/platform/internal/pkg/database"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/repository"
	"github.com/bigops/platform/internal/service"
	cloudsync "github.com/bigops/platform/internal/service/cloud_sync"
)

// MockProvider 模拟阿里云 ECS 返回
type MockProvider struct {
	instances []*model.Asset
}

func (m *MockProvider) SyncInstances(accessKey, secretKey string, regions []string) ([]*model.Asset, error) {
	return m.instances, nil
}

// 模拟同步逻辑（从 cloud_account_handler.go Sync 方法中提取的核心逻辑）
func doSync(provider cloudsync.CloudProvider, cloudAccountID int64) (created, updated, total int, err error) {
	cloudAssets, err := provider.SyncInstances("fake-ak", "fake-sk", []string{"cn-hangzhou"})
	if err != nil {
		return 0, 0, 0, err
	}

	assetSvc := service.NewAssetService()
	assetRepo := repository.NewAssetRepository()
	changeRepo := repository.NewAssetChangeRepository()

	for _, ca := range cloudAssets {
		ca.CloudAccountID = cloudAccountID
		existing, getErr := assetRepo.GetByCloudInstanceID(ca.CloudInstanceID)
		if getErr != nil {
			// 新资产
			if createErr := assetSvc.Create(ca); createErr == nil {
				created++
			} else {
				fmt.Printf("  创建失败: %s - %v\n", ca.Hostname, createErr)
			}
		} else {
			// 已存在：对比 diff 并更新
			changes := diffAsset(existing, ca)
			fmt.Printf("  [debug] %s: %d changes detected\n", ca.Hostname, len(changes))
			for _, ch := range changes {
				fmt.Printf("    %s: %s → %s\n", ch.Field, ch.OldValue, ch.NewValue)
			}
			existing.Hostname = ca.Hostname
			existing.IP = ca.IP
			existing.InnerIP = ca.InnerIP
			existing.OS = ca.OS
			existing.OSVersion = ca.OSVersion
			existing.CPUCores = ca.CPUCores
			existing.MemoryMB = ca.MemoryMB
			existing.DiskGB = ca.DiskGB
			existing.Status = ca.Status
			existing.IDC = ca.IDC
			existing.SN = ca.SN
			existing.CloudAccountID = cloudAccountID
			if len(changes) == 0 {
				continue
			}
			if updateErr := assetRepo.Update(existing); updateErr == nil {
				updated++
				for i := range changes {
					changes[i].AssetID = existing.ID
					changes[i].ChangeType = "sync"
					changeRepo.Create(&changes[i])
				}
			}
		}
	}

	return created, updated, len(cloudAssets), nil
}

// 从 handler 中复制的 diffAsset（测试需要用到同样的逻辑）
func diffAsset(old, new *model.Asset) []model.AssetChange {
	var changes []model.AssetChange
	check := func(field, oldVal, newVal string) {
		if oldVal != newVal {
			changes = append(changes, model.AssetChange{
				Field:    field,
				OldValue: oldVal,
				NewValue: newVal,
			})
		}
	}
	check("ip", old.IP, new.IP)
	check("inner_ip", old.InnerIP, new.InnerIP)
	check("os", old.OS, new.OS)
	check("status", old.Status, new.Status)
	check("hostname", old.Hostname, new.Hostname)
	if old.CPUCores != new.CPUCores {
		changes = append(changes, model.AssetChange{
			Field:    "cpu_cores",
			OldValue: fmt.Sprintf("%d", old.CPUCores),
			NewValue: fmt.Sprintf("%d", new.CPUCores),
		})
	}
	if old.MemoryMB != new.MemoryMB {
		changes = append(changes, model.AssetChange{
			Field:    "memory_mb",
			OldValue: fmt.Sprintf("%d", old.MemoryMB),
			NewValue: fmt.Sprintf("%d", new.MemoryMB),
		})
	}
	return changes
}

func initInfra() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}
	if err := config.Load(configPath); err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}
	cfg := config.Get()

	loggerCfg := logger.Config{Level: "info", Filename: "", MaxSize: 10, MaxBackups: 3, MaxAge: 7}
	logger.Init(loggerCfg)

	mysqlCfg := database.MySQLConfig{
		Host: cfg.Database.Host, Port: cfg.Database.Port,
		Username: cfg.Database.Username, Password: cfg.Database.Password,
		Database: cfg.Database.Database, Charset: cfg.Database.Charset,
	}
	if err := database.InitMySQL(mysqlCfg, logger.Get()); err != nil {
		panic(fmt.Sprintf("MySQL 初始化失败: %v", err))
	}

	// AutoMigrate
	database.GetDB().AutoMigrate(&model.Asset{}, &model.AssetChange{})
}

func main() {
	initInfra()
	defer database.Close()

	assetRepo := repository.NewAssetRepository()
	changeRepo := repository.NewAssetChangeRepository()

	fmt.Println("========================================")
	fmt.Println("云同步集成测试 (MockProvider)")
	fmt.Println("========================================")

	// 清理测试数据
	database.GetDB().Exec("DELETE FROM assets WHERE cloud_instance_id LIKE 'i-test-%'")
	database.GetDB().Exec("DELETE FROM asset_changes WHERE asset_id IN (SELECT id FROM assets WHERE cloud_instance_id LIKE 'i-test-%')")

	// ==============================
	// 测试 1: 首次同步 — 全部新建
	// ==============================
	fmt.Println("\n--- 测试 1: 首次同步 (3台新机器) ---")
	mock := &MockProvider{
		instances: []*model.Asset{
			{
				CloudInstanceID: "i-test-001",
				Hostname:        "web-server-01",
				IP:              "47.98.1.1",
				InnerIP:         "10.0.0.1",
				OS:              "CentOS 7.9 64bit",
				OSVersion:       "linux",
				CPUCores:        4,
				MemoryMB:        8192,
				Status:          "online",
				AssetType:       "server",
				Source:          "aliyun",
				IDC:             "cn-hangzhou",
				SN:              "SN-001",
			},
			{
				CloudInstanceID: "i-test-002",
				Hostname:        "db-server-01",
				IP:              "47.98.1.2",
				InnerIP:         "10.0.0.2",
				OS:              "CentOS 7.9 64bit",
				OSVersion:       "linux",
				CPUCores:        8,
				MemoryMB:        32768,
				Status:          "online",
				AssetType:       "server",
				Source:          "aliyun",
				IDC:             "cn-hangzhou",
				SN:              "SN-002",
			},
			{
				CloudInstanceID: "i-test-003",
				Hostname:        "cache-server-01",
				IP:              "47.98.1.3",
				InnerIP:         "10.0.0.3",
				OS:              "Ubuntu 22.04 64bit",
				OSVersion:       "linux",
				CPUCores:        2,
				MemoryMB:        4096,
				Status:          "online",
				AssetType:       "server",
				Source:          "aliyun",
				IDC:             "cn-hangzhou",
				SN:              "SN-003",
			},
		},
	}

	created, updated, total, err := doSync(mock, 999)
	if err != nil {
		fmt.Printf("同步失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("结果: 新增=%d, 更新=%d, 总计=%d\n", created, updated, total)
	assert("首次同步-新增数", created, 3)
	assert("首次同步-更新数", updated, 0)

	// 验证数据库中确实有 3 条记录
	a1, _ := assetRepo.GetByCloudInstanceID("i-test-001")
	assert("web-server-01 存在", a1 != nil, true)
	assert("web-server-01 IP", a1.IP, "47.98.1.1")
	assert("web-server-01 CPU", a1.CPUCores, 4)
	assert("web-server-01 Source", a1.Source, "aliyun")

	// ==============================
	// 测试 2: 增量同步 — 检测变更
	// ==============================
	fmt.Println("\n--- 测试 2: 增量同步 (1台升配 + 1台关机 + 1台不变) ---")
	time.Sleep(time.Second) // 确保时间戳不同

	mock.instances = []*model.Asset{
		{
			// web-server-01: CPU 4→8, Memory 8192→16384 (升配)
			CloudInstanceID: "i-test-001",
			Hostname:        "web-server-01",
			IP:              "47.98.1.1",
			InnerIP:         "10.0.0.1",
			OS:              "CentOS 7.9 64bit",
			OSVersion:       "linux",
			CPUCores:        8,     // 变了
			MemoryMB:        16384, // 变了
			Status:          "online",
			AssetType:       "server",
			Source:          "aliyun",
			IDC:             "cn-hangzhou",
			SN:              "SN-001",
		},
		{
			// db-server-01: 状态从 online → offline (关机)
			CloudInstanceID: "i-test-002",
			Hostname:        "db-server-01",
			IP:              "47.98.1.2",
			InnerIP:         "10.0.0.2",
			OS:              "CentOS 7.9 64bit",
			OSVersion:       "linux",
			CPUCores:        8,
			MemoryMB:        32768,
			Status:          "offline", // 变了
			AssetType:       "server",
			Source:          "aliyun",
			IDC:             "cn-hangzhou",
			SN:              "SN-002",
		},
		{
			// cache-server-01: 无变化
			CloudInstanceID: "i-test-003",
			Hostname:        "cache-server-01",
			IP:              "47.98.1.3",
			InnerIP:         "10.0.0.3",
			OS:              "Ubuntu 22.04 64bit",
			OSVersion:       "linux",
			CPUCores:        2,
			MemoryMB:        4096,
			Status:          "online",
			AssetType:       "server",
			Source:          "aliyun",
			IDC:             "cn-hangzhou",
			SN:              "SN-003",
		},
	}

	created, updated, total, err = doSync(mock, 999)
	if err != nil {
		fmt.Printf("同步失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("结果: 新增=%d, 更新=%d, 总计=%d\n", created, updated, total)
	assert("增量同步-新增数", created, 0)
	assert("增量同步-更新数", updated, 2) // web-server-01 升配 + db-server-01 关机

	// 验证 web-server-01 升配后的数据
	a1, _ = assetRepo.GetByCloudInstanceID("i-test-001")
	assert("web-server-01 CPU升配后", a1.CPUCores, 8)
	assert("web-server-01 内存升配后", a1.MemoryMB, 16384)

	// 验证 db-server-01 关机后的状态
	a2, _ := assetRepo.GetByCloudInstanceID("i-test-002")
	assert("db-server-01 关机后状态", a2.Status, "offline")

	// 验证变更历史记录
	changes1, total1, _ := changeRepo.ListByAssetID(a1.ID, 1, 20)
	fmt.Printf("web-server-01 变更记录数: %d\n", total1)
	assert("web-server-01 变更记录>=2", total1 >= 2, true)
	for _, ch := range changes1 {
		fmt.Printf("  字段=%s, 旧值=%s, 新值=%s, 类型=%s\n", ch.Field, ch.OldValue, ch.NewValue, ch.ChangeType)
	}

	changes2, total2, _ := changeRepo.ListByAssetID(a2.ID, 1, 20)
	fmt.Printf("db-server-01 变更记录数: %d\n", total2)
	assert("db-server-01 变更记录>=1", total2 >= 1, true)
	for _, ch := range changes2 {
		fmt.Printf("  字段=%s, 旧值=%s, 新值=%s, 类型=%s\n", ch.Field, ch.OldValue, ch.NewValue, ch.ChangeType)
	}

	// cache-server-01 无变化，不应有变更记录
	a3, _ := assetRepo.GetByCloudInstanceID("i-test-003")
	_, total3, _ := changeRepo.ListByAssetID(a3.ID, 1, 20)
	assert("cache-server-01 无变更记录", total3, int64(0))

	// ==============================
	// 测试 3: 云端删除机器 — 验证本地行为
	// ==============================
	fmt.Println("\n--- 测试 3: 云端删除 (i-test-003 消失) ---")
	mock.instances = []*model.Asset{
		{
			CloudInstanceID: "i-test-001",
			Hostname:        "web-server-01",
			IP:              "47.98.1.1",
			InnerIP:         "10.0.0.1",
			OS:              "CentOS 7.9 64bit",
			OSVersion:       "linux",
			CPUCores:        8,
			MemoryMB:        16384,
			Status:          "online",
			AssetType:       "server",
			Source:          "aliyun",
			IDC:             "cn-hangzhou",
			SN:              "SN-001",
		},
		{
			CloudInstanceID: "i-test-002",
			Hostname:        "db-server-01",
			IP:              "47.98.1.2",
			InnerIP:         "10.0.0.2",
			OS:              "CentOS 7.9 64bit",
			OSVersion:       "linux",
			CPUCores:        8,
			MemoryMB:        32768,
			Status:          "offline",
			AssetType:       "server",
			Source:          "aliyun",
			IDC:             "cn-hangzhou",
			SN:              "SN-002",
		},
		// i-test-003 消失了！
	}

	created, updated, total, err = doSync(mock, 999)
	if err != nil {
		fmt.Printf("同步失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("结果: 新增=%d, 更新=%d, 总计=%d\n", created, updated, total)

	// i-test-003 在本地仍然存在（当前设计不自动删除）
	a3still, _ := assetRepo.GetByCloudInstanceID("i-test-003")
	fmt.Printf("i-test-003 本地仍存在: %v (status=%s)\n", a3still != nil, a3still.Status)
	assert("云端删除-本地仍存在", a3still != nil, true)
	fmt.Println("⚠️  注意: 当前设计不自动删除/标记云端已删除的资产，需要后续优化")

	// ==============================
	// 测试 4: 再次同步无变化 — 幂等性
	// ==============================
	fmt.Println("\n--- 测试 4: 无变化再次同步 (幂等性) ---")
	created, updated, total, err = doSync(mock, 999)
	if err != nil {
		fmt.Printf("同步失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("结果: 新增=%d, 更新=%d, 总计=%d\n", created, updated, total)
	assert("幂等同步-新增数", created, 0)
	assert("幂等同步-更新数", updated, 0)

	// ==============================
	// 清理测试数据
	// ==============================
	fmt.Println("\n--- 清理测试数据 ---")
	database.GetDB().Exec("DELETE FROM asset_changes WHERE asset_id IN (SELECT id FROM assets WHERE cloud_instance_id LIKE 'i-test-%')")
	database.GetDB().Exec("DELETE FROM assets WHERE cloud_instance_id LIKE 'i-test-%'")
	fmt.Println("清理完毕")

	fmt.Println("\n========================================")
	fmt.Println("所有测试通过 ✓")
	fmt.Println("========================================")
}

func assert(name string, got, want interface{}) {
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
		fmt.Printf("❌ FAIL: %s: got=%v, want=%v\n", name, got, want)
		logger.Error("测试失败", zap.String("test", name), zap.Any("got", got), zap.Any("want", want))
		os.Exit(1)
	}
	fmt.Printf("✅ PASS: %s = %v\n", name, got)
}
