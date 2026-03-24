package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type TaskExecutionRepository struct{}

func NewTaskExecutionRepository() *TaskExecutionRepository {
	return &TaskExecutionRepository{}
}

func (r *TaskExecutionRepository) Create(e *model.TaskExecution) error {
	return database.GetDB().Create(e).Error
}

func (r *TaskExecutionRepository) Update(e *model.TaskExecution) error {
	return database.GetDB().Save(e).Error
}

func (r *TaskExecutionRepository) GetByID(id int64) (*model.TaskExecution, error) {
	var e model.TaskExecution
	if err := database.GetDB().First(&e, id).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *TaskExecutionRepository) ListByTaskID(taskID int64, page, size int) ([]*model.TaskExecution, int64, error) {
	var items []*model.TaskExecution
	var total int64
	db := database.GetDB().Model(&model.TaskExecution{}).Where("task_id = ?", taskID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *TaskExecutionRepository) CreateHostResult(hr *model.TaskHostResult) error {
	return database.GetDB().Create(hr).Error
}

func (r *TaskExecutionRepository) UpdateHostResult(hr *model.TaskHostResult) error {
	return database.GetDB().Save(hr).Error
}

func (r *TaskExecutionRepository) GetHostResultsByExecutionID(executionID int64) ([]*model.TaskHostResult, error) {
	var items []*model.TaskHostResult
	if err := database.GetDB().Where("execution_id = ?", executionID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *TaskExecutionRepository) GetHostResult(id int64) (*model.TaskHostResult, error) {
	var hr model.TaskHostResult
	if err := database.GetDB().First(&hr, id).Error; err != nil {
		return nil, err
	}
	return &hr, nil
}
