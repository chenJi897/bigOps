# 云账号定时同步系统 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将云资产同步从纯手动触发升级为支持 per-account 定时调度的统一同步任务体系，含完整的同步日志记录。

**Architecture:** 新增 `cloud_sync_tasks` 表记录每次同步任务（无论手动/定时）。将 handler 中 200 行同步逻辑抽取为 `SyncRunner`，手动触发和定时调度共用同一执行器。定时调度使用 Go 原生 goroutine + ticker（不引入新依赖），在 `main.go` 启动时拉起 scheduler。CloudAccount 新增 `sync_interval` 字段控制每个账号的同步周期，`sync_enabled` 控制开关。通过 `sync.Mutex` per-account 锁防止重复执行。

**Tech Stack:** Go stdlib (time.Ticker, sync.Mutex), GORM, Gin — 不引入外部调度库

---

## File Structure

### 新建文件

| 文件 | 职责 |
|------|------|
| `backend/internal/model/cloud_sync_task.go` | 同步任务模型（记录每次同步的完整生命周期） |
| `backend/internal/repository/cloud_sync_task_repository.go` | 同步任务 CRUD |
| `backend/internal/service/cloud_sync/runner.go` | **统一同步执行器** — 手动和定时都调这一个 |
| `backend/internal/service/cloud_sync/scheduler.go` | 定时调度器 — 管理 per-account ticker |
| `backend/internal/handler/cloud_sync_task_handler.go` | 同步记录查询 API |
| `frontend/src/views/SyncLogs.vue` | 同步日志前端页面（放在云账号详情或独立页） |

### 修改文件

| 文件 | 变更 |
|------|------|
| `backend/internal/model/cloud_account.go` | 新增 `SyncEnabled`, `SyncInterval` 字段 |
| `backend/internal/handler/cloud_account_handler.go` | Sync 方法改为调用 SyncRunner；新增调度配置接口 |
| `backend/internal/service/cloud_account_service.go` | 新增 `UpdateSyncConfig`、`ListEnabled` 方法 |
| `backend/internal/repository/cloud_account_repository.go` | 新增 `ListEnabled` 查询 |
| `backend/cmd/core/main.go` | 启动 scheduler；AutoMigrate 新增 CloudSyncTask |
| `backend/api/http/router/router.go` | 注册同步日志路由、调度配置路由 |
| `backend/internal/service/cloud_sync/provider.go` | SyncResult 补充 Unchanged/Offline 字段 |
| `frontend/src/api/index.ts` | 新增 syncTaskApi |
| `frontend/src/views/CloudAccounts.vue` | 云账号表单增加同步配置（开关+周期） |

---

## Task 1: CloudSyncTask 同步任务模型

**Files:**
- Create: `backend/internal/model/cloud_sync_task.go`

- [ ] **Step 1: 创建模型**

```go
// backend/internal/model/cloud_sync_task.go
package model

// CloudSyncTask 云同步任务记录，对应 cloud_sync_tasks 表。
// 每次同步（无论手动还是定时）都会创建一条记录。
type CloudSyncTask struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	CloudAccountID int64      `gorm:"index;not null" json:"cloud_account_id"`
	AccountName    string     `gorm:"size:100" json:"account_name"`         // 冗余，方便查询展示
	Provider       string     `gorm:"size:50" json:"provider"`              // aliyun/tencent/aws
	TriggerType    string     `gorm:"size:20;not null" json:"trigger_type"` // manual / schedule
	Status         string     `gorm:"size:20;not null;index" json:"status"` // running / success / failed
	StartedAt      LocalTime  `json:"started_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	FinishedAt     *LocalTime `json:"finished_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DurationMs     int64      `json:"duration_ms"`                                     // 耗时毫秒
	TotalCount     int        `json:"total_count"`                                     // 云端返回的实例总数
	CreatedCount   int        `json:"created_count"`                                   // 新增数
	UpdatedCount   int        `json:"updated_count"`                                   // 更新数
	UnchangedCount int        `json:"unchanged_count"`                                 // 无变化数
	OfflineCount   int        `json:"offline_count"`                                   // 下线数（云端消失）
	ErrorMessage   string     `gorm:"type:text" json:"error_message"`                  // 失败原因
	OperatorID     int64      `gorm:"default:0" json:"operator_id"`                    // 手动触发时的操作人
	OperatorName   string     `gorm:"size:50" json:"operator_name"`                    // 操作人用户名
	Regions        string     `gorm:"size:500" json:"regions"`                         // 本次同步的 region 列表
	CreatedAt      LocalTime  `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
}

func (CloudSyncTask) TableName() string {
	return "cloud_sync_tasks"
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /root/bigOps/backend && go build ./internal/model/
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add backend/internal/model/cloud_sync_task.go
git commit -m "feat: 添加 CloudSyncTask 同步任务记录模型"
```

---

## Task 2: CloudSyncTask Repository

**Files:**
- Create: `backend/internal/repository/cloud_sync_task_repository.go`

- [ ] **Step 1: 创建 repository**

```go
// backend/internal/repository/cloud_sync_task_repository.go
package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type CloudSyncTaskRepository struct{}

func NewCloudSyncTaskRepository() *CloudSyncTaskRepository {
	return &CloudSyncTaskRepository{}
}

func (r *CloudSyncTaskRepository) Create(task *model.CloudSyncTask) error {
	return database.GetDB().Create(task).Error
}

func (r *CloudSyncTaskRepository) Update(task *model.CloudSyncTask) error {
	return database.GetDB().Save(task).Error
}

func (r *CloudSyncTaskRepository) GetByID(id int64) (*model.CloudSyncTask, error) {
	var task model.CloudSyncTask
	if err := database.GetDB().First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// GetLastByAccountID 获取某云账号最近一次同步任务。
func (r *CloudSyncTaskRepository) GetLastByAccountID(accountID int64) (*model.CloudSyncTask, error) {
	var task model.CloudSyncTask
	if err := database.GetDB().Where("cloud_account_id = ?", accountID).
		Order("id DESC").First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// HasRunning 检查某云账号是否有正在运行的同步任务（防重复执行）。
func (r *CloudSyncTaskRepository) HasRunning(accountID int64) (bool, error) {
	var count int64
	err := database.GetDB().Model(&model.CloudSyncTask{}).
		Where("cloud_account_id = ? AND status = ?", accountID, "running").
		Count(&count).Error
	return count > 0, err
}

// ListByAccountID 按云账号分页查询同步记录。
func (r *CloudSyncTaskRepository) ListByAccountID(accountID int64, page, size int) ([]*model.CloudSyncTask, int64, error) {
	var tasks []*model.CloudSyncTask
	var total int64
	db := database.GetDB().Model(&model.CloudSyncTask{}).
		Where("cloud_account_id = ?", accountID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

// List 全局分页查询同步记录（可按状态/触发类型筛选）。
func (r *CloudSyncTaskRepository) List(page, size int, status, triggerType string, accountID int64) ([]*model.CloudSyncTask, int64, error) {
	var tasks []*model.CloudSyncTask
	var total int64
	db := database.GetDB().Model(&model.CloudSyncTask{})
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if triggerType != "" {
		db = db.Where("trigger_type = ?", triggerType)
	}
	if accountID > 0 {
		db = db.Where("cloud_account_id = ?", accountID)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /root/bigOps/backend && go build ./internal/repository/
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/cloud_sync_task_repository.go
git commit -m "feat: 添加 CloudSyncTask repository"
```

---

## Task 3: CloudAccount 模型新增调度字段

**Files:**
- Modify: `backend/internal/model/cloud_account.go`
- Modify: `backend/internal/service/cloud_account_service.go`
- Modify: `backend/internal/repository/cloud_account_repository.go`

- [ ] **Step 1: CloudAccount 模型新增字段**

在 `cloud_account.go` 的 `Status` 字段后添加：

```go
SyncEnabled  bool   `gorm:"default:false;not null" json:"sync_enabled"`   // 是否启用定时同步
SyncInterval int    `gorm:"default:0;not null" json:"sync_interval"`      // 同步周期（分钟），0=不同步，10/30/60/1440
```

- [ ] **Step 2: CloudAccountRepository 新增 ListEnabled**

在 `cloud_account_repository.go` 末尾添加：

```go
// ListEnabled 获取所有启用定时同步的云账号。
func (r *CloudAccountRepository) ListEnabled() ([]*model.CloudAccount, error) {
	var accounts []*model.CloudAccount
	err := database.GetDB().Where("sync_enabled = ? AND sync_interval > 0 AND status = 1", true).
		Find(&accounts).Error
	return accounts, err
}
```

- [ ] **Step 3: CloudAccountService 新增 UpdateSyncConfig**

在 `cloud_account_service.go` 末尾添加：

```go
// UpdateSyncConfig 更新云账号的同步调度配置。
func (s *CloudAccountService) UpdateSyncConfig(id int64, syncEnabled bool, syncInterval int) error {
	account, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("云账号不存在")
	}
	account.SyncEnabled = syncEnabled
	account.SyncInterval = syncInterval
	return s.repo.Update(account)
}

// ListEnabled 获取所有启用定时同步的云账号。
func (s *CloudAccountService) ListEnabled() ([]*model.CloudAccount, error) {
	return s.repo.ListEnabled()
}
```

- [ ] **Step 4: 验证编译**

```bash
cd /root/bigOps/backend && go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/model/cloud_account.go backend/internal/service/cloud_account_service.go backend/internal/repository/cloud_account_repository.go
git commit -m "feat: CloudAccount 新增 sync_enabled/sync_interval 字段"
```

---

## Task 4: SyncRunner 统一同步执行器

**Files:**
- Create: `backend/internal/service/cloud_sync/runner.go`
- Modify: `backend/internal/service/cloud_sync/provider.go`（SyncResult 补字段）

这是核心：将 handler 中的 200 行 upsert 逻辑抽取为独立执行器，手动和定时共用。

- [ ] **Step 1: 更新 SyncResult**

修改 `provider.go` 中的 SyncResult：

```go
// SyncResult 同步结果。
type SyncResult struct {
	Created   int
	Updated   int
	Unchanged int
	Offline   int
	Total     int
}
```

- [ ] **Step 2: 创建 SyncRunner**

```go
// backend/internal/service/cloud_sync/runner.go
package cloudsync

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/config"
	"github.com/bigops/platform/internal/pkg/crypto"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/repository"
	"github.com/bigops/platform/internal/service"
)

// accountLocks per-account 锁，防止同一账号并发同步。
var (
	accountLocks   = make(map[int64]*sync.Mutex)
	accountLocksMu sync.Mutex
)

func getAccountLock(accountID int64) *sync.Mutex {
	accountLocksMu.Lock()
	defer accountLocksMu.Unlock()
	if _, ok := accountLocks[accountID]; !ok {
		accountLocks[accountID] = &sync.Mutex{}
	}
	return accountLocks[accountID]
}

// RunSync 执行一次同步任务。这是唯一的同步入口，手动和定时都调它。
// triggerType: "manual" 或 "schedule"
// operatorID/operatorName: 手动触发时传操作人信息，定时传 0/""
func RunSync(accountID int64, triggerType string, operatorID int64, operatorName string) (*model.CloudSyncTask, error) {
	// 获取 per-account 锁（TryLock，拿不到直接跳过）
	lock := getAccountLock(accountID)
	if !lock.TryLock() {
		return nil, fmt.Errorf("云账号 %d 同步正在执行中，跳过本次", accountID)
	}
	defer lock.Unlock()

	// 查账号信息
	accountSvc := service.NewCloudAccountService()
	account, err := accountSvc.GetByID(accountID)
	if err != nil {
		return nil, fmt.Errorf("云账号不存在: %w", err)
	}

	// 解密 AK/SK
	encryptKey := config.Get().Encrypt.Key
	accessKey, err := crypto.AESDecrypt(account.AccessKey, encryptKey)
	if err != nil {
		return nil, fmt.Errorf("解密 AccessKey 失败: %w", err)
	}
	secretKey, err := crypto.AESDecrypt(account.SecretKey, encryptKey)
	if err != nil {
		return nil, fmt.Errorf("解密 SecretKey 失败: %w", err)
	}

	// 创建同步任务记录
	taskRepo := repository.NewCloudSyncTaskRepository()
	now := model.LocalTime(time.Now())
	task := &model.CloudSyncTask{
		CloudAccountID: accountID,
		AccountName:    account.Name,
		Provider:       account.Provider,
		TriggerType:    triggerType,
		Status:         "running",
		StartedAt:      now,
		OperatorID:     operatorID,
		OperatorName:   operatorName,
		Regions:        account.Region,
	}
	if err := taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("创建同步任务记录失败: %w", err)
	}

	// 更新云账号状态为 syncing
	accountSvc.UpdateSyncStatus(accountID, "syncing", "", nil)

	logger.Info("同步任务开始",
		zap.Int64("task_id", task.ID),
		zap.Int64("account_id", accountID),
		zap.String("trigger", triggerType),
		zap.String("provider", account.Provider),
	)

	// 选择 provider
	var provider CloudProvider
	switch account.Provider {
	case "aliyun":
		provider = NewAliyunProvider()
	default:
		return finishTask(taskRepo, accountSvc, task, accountID, 0, SyncResult{},
			fmt.Errorf("暂不支持该云厂商: %s", account.Provider))
	}

	// 执行同步
	regions := strings.Split(account.Region, ",")
	cloudAssets, err := provider.SyncInstances(accessKey, secretKey, regions)
	if err != nil {
		return finishTask(taskRepo, accountSvc, task, accountID, 0, SyncResult{}, err)
	}

	// Upsert 逻辑
	result := upsertAssets(cloudAssets, accountID)

	return finishTask(taskRepo, accountSvc, task, accountID, len(cloudAssets), result, nil)
}

// upsertAssets 执行资产 upsert + diff 变更记录。
func upsertAssets(cloudAssets []*model.Asset, accountID int64) SyncResult {
	assetSvc := service.NewAssetService()
	assetRepo := repository.NewAssetRepository()
	changeRepo := repository.NewAssetChangeRepository()

	var result SyncResult
	for _, ca := range cloudAssets {
		ca.CloudAccountID = accountID
		existing, err := assetRepo.GetByCloudInstanceID(ca.CloudInstanceID)
		if err != nil {
			// 新资产
			if createErr := assetSvc.Create(ca); createErr == nil {
				result.Created++
			}
		} else {
			// 已存在：对比 diff
			changes := diffAsset(existing, ca)
			if len(changes) == 0 {
				result.Unchanged++
				continue
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
			existing.CloudAccountID = accountID
			if updateErr := assetRepo.Update(existing); updateErr == nil {
				result.Updated++
				for i := range changes {
					changes[i].AssetID = existing.ID
					changes[i].ChangeType = "sync"
					changeRepo.Create(&changes[i])
				}
			}
		}
	}
	return result
}

// diffAsset 对比两个 Asset 的关键字段，返回变更列表。
func diffAsset(old, new *model.Asset) []model.AssetChange {
	var changes []model.AssetChange
	check := func(field, oldVal, newVal string) {
		if oldVal != newVal {
			changes = append(changes, model.AssetChange{Field: field, OldValue: oldVal, NewValue: newVal})
		}
	}
	check("ip", old.IP, new.IP)
	check("inner_ip", old.InnerIP, new.InnerIP)
	check("os", old.OS, new.OS)
	check("status", old.Status, new.Status)
	check("hostname", old.Hostname, new.Hostname)
	if old.CPUCores != new.CPUCores {
		changes = append(changes, model.AssetChange{Field: "cpu_cores", OldValue: strconv.Itoa(old.CPUCores), NewValue: strconv.Itoa(new.CPUCores)})
	}
	if old.MemoryMB != new.MemoryMB {
		changes = append(changes, model.AssetChange{Field: "memory_mb", OldValue: strconv.Itoa(old.MemoryMB), NewValue: strconv.Itoa(new.MemoryMB)})
	}
	return changes
}

// finishTask 完成同步任务：更新任务记录 + 更新云账号同步状态。
func finishTask(
	taskRepo *repository.CloudSyncTaskRepository,
	accountSvc *service.CloudAccountService,
	task *model.CloudSyncTask,
	accountID int64,
	totalFromCloud int,
	result SyncResult,
	syncErr error,
) (*model.CloudSyncTask, error) {
	finishedAt := model.LocalTime(time.Now())
	task.FinishedAt = &finishedAt
	task.DurationMs = time.Time(finishedAt).Sub(time.Time(task.StartedAt)).Milliseconds()
	task.TotalCount = totalFromCloud
	task.CreatedCount = result.Created
	task.UpdatedCount = result.Updated
	task.UnchangedCount = result.Unchanged
	task.OfflineCount = result.Offline

	if syncErr != nil {
		task.Status = "failed"
		task.ErrorMessage = syncErr.Error()
		taskRepo.Update(task)
		accountSvc.UpdateSyncStatus(accountID, "failed", syncErr.Error(), nil)
		logger.Error("同步任务失败",
			zap.Int64("task_id", task.ID),
			zap.Int64("account_id", accountID),
			zap.Error(syncErr),
		)
		return task, syncErr
	}

	task.Status = "success"
	taskRepo.Update(task)
	now := model.LocalTime(time.Now())
	msg := fmt.Sprintf("同步完成: 新增 %d, 更新 %d, 无变化 %d, 总计 %d",
		result.Created, result.Updated, result.Unchanged, totalFromCloud)
	accountSvc.UpdateSyncStatus(accountID, "success", msg, &now)
	logger.Info("同步任务完成",
		zap.Int64("task_id", task.ID),
		zap.Int64("account_id", accountID),
		zap.Int("created", result.Created),
		zap.Int("updated", result.Updated),
		zap.Int("unchanged", result.Unchanged),
		zap.Int("total", totalFromCloud),
		zap.Int64("duration_ms", task.DurationMs),
	)
	return task, nil
}
```

- [ ] **Step 3: 验证编译**

```bash
cd /root/bigOps/backend && go build ./internal/service/cloud_sync/
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/cloud_sync/runner.go backend/internal/service/cloud_sync/provider.go
git commit -m "feat: SyncRunner 统一同步执行器 + per-account 防重复锁"
```

---

## Task 5: 重构 Handler 调用 SyncRunner

**Files:**
- Modify: `backend/internal/handler/cloud_account_handler.go`

将 Sync 方法从 200 行内联逻辑改为调用 `cloudsync.RunSync()`，同时新增同步配置接口。

- [ ] **Step 1: 重写 Sync 方法**

将 `cloud_account_handler.go` 的 Sync 方法替换为：

```go
func (h *CloudAccountHandler) Sync(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	// 获取操作人信息
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	operatorName := getOperator(c)

	task, err := cloudsync.RunSync(id, "manual", operatorID, operatorName)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	msg := fmt.Sprintf("同步完成: 新增 %d, 更新 %d, 无变化 %d, 总计 %d",
		task.CreatedCount, task.UpdatedCount, task.UnchangedCount, task.TotalCount)
	c.Set("audit_action", "update")
	c.Set("audit_resource", "cloud_account")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", msg)
	response.SuccessWithMessage(c, msg, task)
}
```

同时删除 handler 中的 `diffAsset` 函数（已移至 runner.go）。

- [ ] **Step 2: 添加同步配置接口**

在 handler 中添加：

```go
type UpdateSyncConfigRequest struct {
	SyncEnabled  bool `json:"sync_enabled"`
	SyncInterval int  `json:"sync_interval" example:"30" enums:"0,10,30,60,1440"` // 分钟
}

// UpdateSyncConfig 更新云账号同步调度配置。
// @Summary 更新同步调度配置
// @Description 设置云账号的定时同步开关和周期
// @Tags 云账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "云账号ID"
// @Param body body UpdateSyncConfigRequest true "同步配置"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /cloud-accounts/{id}/sync-config [post]
func (h *CloudAccountHandler) UpdateSyncConfig(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpdateSyncConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	// 校验 interval 合法值
	validIntervals := map[int]bool{0: true, 10: true, 30: true, 60: true, 1440: true}
	if !validIntervals[req.SyncInterval] {
		response.BadRequest(c, "同步周期只支持 0/10/30/60/1440 分钟")
		return
	}
	if err := h.svc.UpdateSyncConfig(id, req.SyncEnabled, req.SyncInterval); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新同步配置", zap.String("operator", getOperator(c)), zap.Int64("account_id", id),
		zap.Bool("enabled", req.SyncEnabled), zap.Int("interval", req.SyncInterval))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "cloud_account")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", fmt.Sprintf("更新同步配置: enabled=%v, interval=%d min", req.SyncEnabled, req.SyncInterval))
	response.SuccessWithMessage(c, "更新成功", nil)
}
```

- [ ] **Step 3: 验证编译**

```bash
cd /root/bigOps/backend && go build ./cmd/core/
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/cloud_account_handler.go
git commit -m "refactor: Sync handler 改为调用 SyncRunner + 新增同步配置接口"
```

---

## Task 6: 定时调度器 Scheduler

**Files:**
- Create: `backend/internal/service/cloud_sync/scheduler.go`
- Modify: `backend/cmd/core/main.go`

- [ ] **Step 1: 创建 Scheduler**

```go
// backend/internal/service/cloud_sync/scheduler.go
package cloudsync

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/service"
)

// Scheduler 定时同步调度器。
// 每分钟检查一次哪些云账号需要同步，按各自的 sync_interval 判断是否到期。
type Scheduler struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	// lastRun 记录每个账号上次同步时间，用于判断是否到期
	lastRun   map[int64]time.Time
	lastRunMu sync.Mutex
}

// NewScheduler 创建调度器实例。
func NewScheduler() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		ctx:     ctx,
		cancel:  cancel,
		lastRun: make(map[int64]time.Time),
	}
}

// Start 启动调度器，每 60 秒巡检一次。
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		logger.Info("云同步调度器已启动，巡检间隔 60s")
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				logger.Info("云同步调度器已停止")
				return
			case <-ticker.C:
				s.check()
			}
		}
	}()
}

// Stop 优雅停止调度器。
func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
}

// check 巡检所有启用定时同步的云账号。
func (s *Scheduler) check() {
	accountSvc := service.NewCloudAccountService()
	accounts, err := accountSvc.ListEnabled()
	if err != nil {
		logger.Error("调度器: 查询云账号失败", zap.Error(err))
		return
	}

	for _, account := range accounts {
		if account.SyncInterval <= 0 {
			continue
		}

		s.lastRunMu.Lock()
		last, ok := s.lastRun[account.ID]
		s.lastRunMu.Unlock()

		interval := time.Duration(account.SyncInterval) * time.Minute
		if ok && time.Since(last) < interval {
			continue // 还没到期
		}

		// 到期了，异步执行同步
		accountID := account.ID
		accountName := account.Name
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			logger.Info("调度器: 触发定时同步",
				zap.Int64("account_id", accountID),
				zap.String("account_name", accountName),
			)
			_, err := RunSync(accountID, "schedule", 0, "system")
			if err != nil {
				logger.Error("调度器: 同步失败",
					zap.Int64("account_id", accountID),
					zap.Error(err),
				)
			}
			// 记录本次执行时间（无论成功失败都记录，避免失败时疯狂重试）
			s.lastRunMu.Lock()
			s.lastRun[accountID] = time.Now()
			s.lastRunMu.Unlock()
		}()
	}
}
```

- [ ] **Step 2: 修改 main.go 启动调度器**

在 `main.go` 的 AutoMigrate 中添加 `&model.CloudSyncTask{}`：

```go
if err := database.GetDB().AutoMigrate(&model.User{}, &model.Role{}, &model.Menu{}, &model.UserRole{}, &model.AuditLog{}, &model.ServiceTree{}, &model.CloudAccount{}, &model.Asset{}, &model.AssetChange{}, &model.CloudSyncTask{}); err != nil {
```

在启动 HTTP server 的 `go func()` 之后、等待信号之前，添加：

```go
// 启动云同步调度器
syncScheduler := cloudsync.NewScheduler()
syncScheduler.Start()
defer syncScheduler.Stop()
logger.Info("Cloud sync scheduler started")
```

注意：需要在 import 中添加 `cloudsync "github.com/bigops/platform/internal/service/cloud_sync"`

- [ ] **Step 3: 验证编译**

```bash
cd /root/bigOps/backend && go build ./cmd/core/
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/cloud_sync/scheduler.go backend/cmd/core/main.go
git commit -m "feat: 定时同步调度器 + main.go 启动集成"
```

---

## Task 7: 同步日志查询 API + 路由注册

**Files:**
- Create: `backend/internal/handler/cloud_sync_task_handler.go`
- Modify: `backend/api/http/router/router.go`

- [ ] **Step 1: 创建同步日志 Handler**

```go
// backend/internal/handler/cloud_sync_task_handler.go
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/repository"
)

var _ model.CloudSyncTask // swag import

type CloudSyncTaskHandler struct {
	repo *repository.CloudSyncTaskRepository
}

func NewCloudSyncTaskHandler() *CloudSyncTaskHandler {
	return &CloudSyncTaskHandler{repo: repository.NewCloudSyncTaskRepository()}
}

// List 同步任务日志列表。
// @Summary 同步任务日志列表
// @Description 分页查询云同步任务记录，支持按状态/触发类型/云账号筛选
// @Tags 同步日志
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param status query string false "状态" Enums(running,success,failed)
// @Param trigger_type query string false "触发类型" Enums(manual,schedule)
// @Param cloud_account_id query int false "云账号ID"
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.CloudSyncTask}} "同步日志列表"
// @Failure 500 {object} response.Response "查询失败"
// @Router /sync-tasks [get]
func (h *CloudSyncTaskHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := c.Query("status")
	triggerType := c.Query("trigger_type")
	accountID, _ := strconv.ParseInt(c.Query("cloud_account_id"), 10, 64)

	tasks, total, err := h.repo.List(page, size, status, triggerType, accountID)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, tasks, total, page, size)
}

// GetByAccountID 查询某云账号的同步历史。
// @Summary 云账号同步历史
// @Description 查询指定云账号的同步任务记录
// @Tags 同步日志
// @Produce json
// @Security BearerAuth
// @Param id path int true "云账号ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(10)
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.CloudSyncTask}} "同步历史"
// @Router /cloud-accounts/{id}/sync-tasks [get]
func (h *CloudSyncTaskHandler) GetByAccountID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	tasks, total, err := h.repo.ListByAccountID(id, page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, tasks, total, page, size)
}
```

- [ ] **Step 2: 注册路由**

在 `router.go` 的云账号路由区域添加：

```go
authGroup.POST("/cloud-accounts/:id/sync-config", cloudAccountHandler.UpdateSyncConfig)
authGroup.GET("/cloud-accounts/:id/sync-tasks", syncTaskHandler.GetByAccountID)

// --- 同步日志 ---
syncTaskHandler := handler.NewCloudSyncTaskHandler()
authGroup.GET("/sync-tasks", syncTaskHandler.List)
```

注意：`syncTaskHandler` 声明要在使用之前，`GetByAccountID` 路由可以放在云账号区域也可以放在同步日志区域。推荐这样组织：

```go
// --- 云账号管理 ---
cloudAccountHandler := handler.NewCloudAccountHandler()
syncTaskHandler := handler.NewCloudSyncTaskHandler()
authGroup.GET("/cloud-accounts", cloudAccountHandler.List)
authGroup.GET("/cloud-accounts/:id", cloudAccountHandler.GetByID)
authGroup.POST("/cloud-accounts", cloudAccountHandler.Create)
authGroup.POST("/cloud-accounts/:id", cloudAccountHandler.Update)
authGroup.POST("/cloud-accounts/:id/keys", cloudAccountHandler.UpdateKeys)
authGroup.POST("/cloud-accounts/:id/delete", cloudAccountHandler.Delete)
authGroup.POST("/cloud-accounts/:id/sync", cloudAccountHandler.Sync)
authGroup.POST("/cloud-accounts/:id/sync-config", cloudAccountHandler.UpdateSyncConfig)
authGroup.GET("/cloud-accounts/:id/sync-tasks", syncTaskHandler.GetByAccountID)

// --- 同步日志 ---
authGroup.GET("/sync-tasks", syncTaskHandler.List)
```

- [ ] **Step 3: 验证编译**

```bash
cd /root/bigOps/backend && go build ./cmd/core/
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/cloud_sync_task_handler.go backend/api/http/router/router.go
git commit -m "feat: 同步日志查询 API + 路由注册"
```

---

## Task 8: 前端 — 云账号同步配置 + 同步日志

**Files:**
- Modify: `frontend/src/api/index.ts`
- Modify: `frontend/src/views/CloudAccounts.vue`

- [ ] **Step 1: 前端 API 补充**

在 `api/index.ts` 的 cloudAccountApi 中添加：

```ts
syncConfig: (id: number, sync_enabled: boolean, sync_interval: number) =>
  api.post(`/cloud-accounts/${id}/sync-config`, { sync_enabled, sync_interval }),
syncTasks: (id: number, page = 1, size = 10) =>
  api.get(`/cloud-accounts/${id}/sync-tasks`, { params: { page, size } }),
```

新增 syncTaskApi：

```ts
// 同步日志
export const syncTaskApi = {
  list: (params: { page?: number; size?: number; status?: string; trigger_type?: string; cloud_account_id?: number }) =>
    api.get('/sync-tasks', { params }),
}
```

- [ ] **Step 2: 云账号编辑表单增加同步配置**

在 `CloudAccounts.vue` 的编辑表单中添加同步开关和周期选择：

```html
<el-form-item label="定时同步">
  <el-switch v-model="form.sync_enabled" />
</el-form-item>
<el-form-item label="同步周期" v-if="form.sync_enabled">
  <el-select v-model="form.sync_interval" style="width: 100%;">
    <el-option label="每 10 分钟" :value="10" />
    <el-option label="每 30 分钟" :value="30" />
    <el-option label="每小时" :value="60" />
    <el-option label="每天" :value="1440" />
  </el-select>
</el-form-item>
```

在列表的操作列添加"同步记录"按钮，点击打开抽屉展示同步历史。

在表格中增加"定时同步"列：

```html
<el-table-column label="定时同步" width="100">
  <template #default="{ row }">
    <el-tag v-if="row.sync_enabled" type="success" size="small">{{ row.sync_interval }}分钟</el-tag>
    <el-tag v-else type="info" size="small">关闭</el-tag>
  </template>
</el-table-column>
```

- [ ] **Step 3: 添加同步记录抽屉**

在 CloudAccounts.vue 中添加同步记录抽屉组件，展示同步历史列表（含状态/耗时/结果）。

- [ ] **Step 4: 验证编译**

```bash
cd /root/bigOps/frontend && npm run build
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api/index.ts frontend/src/views/CloudAccounts.vue
git commit -m "feat: 前端云账号同步配置 + 同步日志抽屉"
```

---

## Task 9: 数据库菜单 + 最终验证

**Files:**
- SQL: 在 menus 表中添加同步日志菜单项（如果需要独立页面）
- 全局验证

- [ ] **Step 1: 添加 SyncLogs 到 viewModules（如果做独立页面）**

如果同步日志作为独立菜单页面，需要：
1. 创建 `frontend/src/views/SyncLogs.vue`
2. 在 `router/index.ts` 的 viewModules 中添加 `'SyncLogs': () => import('../views/SyncLogs.vue')`
3. 在数据库 menus 表中添加菜单记录

**建议先不做独立页面**，同步日志放在云账号详情的抽屉里即可，减少页面数量。

- [ ] **Step 2: 全栈验证**

```bash
# 后端编译
cd /root/bigOps/backend && go build ./cmd/core/

# 前端编译
cd /root/bigOps/frontend && npm run build

# 启动后端验证
go run ./cmd/core/main.go
# 检查日志中是否有 "Cloud sync scheduler started"

# 手动触发同步 API 测试
curl -X POST http://localhost:8080/api/v1/cloud-accounts/1/sync \
  -H "Authorization: Bearer <token>"

# 查看同步日志
curl http://localhost:8080/api/v1/sync-tasks \
  -H "Authorization: Bearer <token>"
```

- [ ] **Step 3: 最终 Commit**

```bash
git add -A
git commit -m "feat: 云账号定时同步系统完成 — 统一执行器 + 调度器 + 同步日志"
```

---

## 架构决策备忘

| 决策 | 选择 | 理由 |
|------|------|------|
| 调度库 | Go stdlib ticker | 项目简单，不需要 cron 表达式，避免引入 robfig/cron 等依赖 |
| 防重复 | sync.Mutex TryLock | 单进程部署，不需要分布式锁 |
| 同步入口 | SyncRunner 统一函数 | 手动/定时共用一套逻辑，维护成本最低 |
| 巡检方式 | 每 60s 全量扫描 enabled 账号 | 账号数量少（<100），不需要精确 cron |
| 同步日志存放 | 独立表 cloud_sync_tasks | 不与 audit_logs 混用，字段差异大 |
| 失败重试 | 不立即重试，等下一个周期 | 避免对云 API 造成压力，失败有日志可查 |
