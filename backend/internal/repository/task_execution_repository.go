package repository

import (
	"strings"
	"time"

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
	db := database.GetDB().Model(&model.TaskExecution{})
	if taskID > 0 {
		db = db.Where("task_id = ?", taskID)
	}
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

func (r *TaskExecutionRepository) CancelUnfinishedHostResults(executionID int64, reason string) error {
	now := time.Now()
	lt := model.LocalTime(now)
	suffix := reason
	if suffix != "" && !strings.HasSuffix(suffix, "\n") {
		suffix += "\n"
	}
	// MySQL：批量取消时补齐结束时间与耗时（微秒差/1000 ->毫秒）
	return database.GetDB().Exec(`
UPDATE task_host_results SET
	status = 'canceled',
	stderr = CONCAT(COALESCE(stderr,''), ?),
	finished_at = ?,
	duration_ms = CASE
		WHEN started_at IS NOT NULL THEN TIMESTAMPDIFF(MICROSECOND, started_at, NOW())/1000
		ELSE 0
	END
WHERE execution_id = ? AND status IN ('pending','running')`,
		suffix, lt, executionID,
	).Error
}

func (r *TaskExecutionRepository) ListRecent(since *time.Time, limit int) ([]*model.TaskExecution, error) {
	if limit <= 0 {
		limit = 200
	}
	var items []*model.TaskExecution
	db := database.GetDB().Model(&model.TaskExecution{})
	if since != nil {
		db = db.Where("started_at >= ?", *since)
	}
	if err := db.Order("id DESC").Limit(limit).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
