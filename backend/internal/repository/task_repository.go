package repository

import (
	"context"
	"strings"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
	"gorm.io/gorm"
)

// TaskListQuery 任务列表查询参数（兼容老接口）
type TaskListQuery struct {
	Page     int
	PageSize int
	TaskType string
	Status   int
	Keyword  string
}

// TaskRepository 任务模板仓储
type TaskRepository struct {
	db *gorm.DB
}

// NewTaskRepository 创建任务仓储
func NewTaskRepository() *TaskRepository {
	return &TaskRepository{db: database.GetDB()}
}

// Create 创建任务（主路径统一到 tasks 表）
func (r *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// CreateTask 写入 tasks 表（任务中心主路径）。
func (r *TaskRepository) CreateTask(task *model.Task) error {
	return r.db.Create(task).Error
}

// GetByID 根据ID查询（主路径统一到 tasks 表）
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*model.Task, error) {
	var task model.Task
	err := r.db.WithContext(ctx).First(&task, id).Error
	return &task, err
}

// GetTask 根据 ID 查询任务（无 context 便捷封装）。
func (r *TaskRepository) GetTask(id int64) (*model.Task, error) {
	var task model.Task
	err := r.db.First(&task, id).Error
	return &task, err
}

// ListTasks 分页列表（任务中心主路径）。
func (r *TaskRepository) ListTasks(q TaskListQuery) ([]*model.Task, int64, error) {
	var tasks []*model.Task
	var total int64
	db := r.db.Model(&model.Task{})

	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		db = db.Where("name LIKE ?", "%"+kw+"%")
	}
	if tt := strings.TrimSpace(q.TaskType); tt != "" {
		db = db.Where("task_type = ?", tt)
	}
	if q.Status == 0 || q.Status == 1 {
		db = db.Where("status = ?", q.Status)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := q.Page
	if page <= 0 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	err := db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks).Error
	return tasks, total, err

}

// Update 更新任务（主路径统一到 tasks 表）
func (r *TaskRepository) Update(ctx context.Context, task *model.Task) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// Delete 删除任务（软删除，主路径统一到 tasks 表）
func (r *TaskRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Task{}, id).Error
}

// UpdateTask 持久化任务变更（无 context）。
func (r *TaskRepository) UpdateTask(task *model.Task) error {
	return r.db.Save(task).Error
}

// DeleteTask 软删除任务（无 context）。
func (r *TaskRepository) DeleteTask(id int64) error {
	return r.db.Delete(&model.Task{}, id).Error
}

// List 列表查询（主路径统一到 tasks 表）
func (r *TaskRepository) List(ctx context.Context, page, pageSize int, category string) ([]model.Task, int64, error) {
	var tasks []model.Task
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Task{})

	if category != "" {
		query = query.Where("task_type = ?", category)
	}

	query.Count(&total)
	err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks).Error

	return tasks, total, err
}

// TaskInstanceRepository 任务实例仓储
type TaskInstanceRepository struct {
	db *gorm.DB
}

func NewTaskInstanceRepository() *TaskInstanceRepository {
	return &TaskInstanceRepository{db: database.GetDB()}
}

func (r *TaskInstanceRepository) Create(ctx context.Context, instance *model.TaskInstance) error {
	return r.db.WithContext(ctx).Create(instance).Error
}

func (r *TaskInstanceRepository) GetByExecutionID(ctx context.Context, executionID string) (*model.TaskInstance, error) {
	var instance model.TaskInstance
	err := r.db.WithContext(ctx).
		Preload("TaskDefinition").
		Preload("HostResults").
		Where("execution_id = ?", executionID).
		First(&instance).Error
	return &instance, err
}

func (r *TaskInstanceRepository) List(ctx context.Context, page, pageSize int, status string) ([]model.TaskInstance, int64, error) {
	var instances []model.TaskInstance
	var total int64

	query := r.db.WithContext(ctx).Model(&model.TaskInstance{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	err := query.Offset((page - 1) * pageSize).
		Limit(pageSize).
		Preload("TaskDefinition").
		Order("created_at desc").
		Find(&instances).Error

	return instances, total, err
}

// ListByTaskID 按任务ID查询执行记录（兼容老接口）
func (r *TaskInstanceRepository) ListByTaskID(ctx context.Context, taskID int64, page, pageSize int) ([]model.TaskInstance, int64, error) {
	var instances []model.TaskInstance
	var total int64

	query := r.db.WithContext(ctx).Model(&model.TaskInstance{}).Where("task_definition_id = ?", taskID)

	query.Count(&total)
	err := query.Offset((page - 1) * pageSize).
		Limit(pageSize).
		Preload("TaskDefinition").
		Order("created_at desc").
		Find(&instances).Error

	return instances, total, err
}

// TaskStepRepository 任务步骤仓储
type TaskStepRepository struct {
	db *gorm.DB
}

func NewTaskStepRepository() *TaskStepRepository {
	return &TaskStepRepository{db: database.GetDB()}
}

func (r *TaskStepRepository) Create(ctx context.Context, step *model.TaskStep) error {
	return r.db.WithContext(ctx).Create(step).Error
}
