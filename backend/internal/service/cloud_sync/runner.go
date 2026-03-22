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
	result := upsertAssets(cloudAssets, accountID, account.ServiceTreeID)

	return finishTask(taskRepo, accountSvc, task, accountID, len(cloudAssets), result, nil)
}

// upsertAssets 执行资产 upsert + diff 变更记录。
func upsertAssets(cloudAssets []*model.Asset, accountID int64, serviceTreeID int64) SyncResult {
	assetSvc := service.NewAssetService()
	assetRepo := repository.NewAssetRepository()
	changeRepo := repository.NewAssetChangeRepository()

	var result SyncResult
	for _, ca := range cloudAssets {
		ca.CloudAccountID = accountID
		if serviceTreeID > 0 {
			ca.ServiceTreeID = serviceTreeID
		}
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
