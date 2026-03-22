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
