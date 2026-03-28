package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type TaskRepository struct{}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{}
}

func (r *TaskRepository) Create(t *model.Task) error {
	return database.GetDB().Create(t).Error
}

func (r *TaskRepository) Update(t *model.Task) error {
	return database.GetDB().Save(t).Error
}

func (r *TaskRepository) GetByID(id int64) (*model.Task, error) {
	var t model.Task
	if err := database.GetDB().First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TaskRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.Task{}, id).Error
}

type TaskListQuery struct {
	Page     int
	Size     int
	Keyword  string
	TaskType string
}

func (r *TaskRepository) List(q TaskListQuery) ([]*model.Task, int64, error) {
	var items []*model.Task
	var total int64
	db := database.GetDB().Model(&model.Task{})
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		db = db.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if q.TaskType != "" {
		db = db.Where("task_type = ?", q.TaskType)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (q.Page - 1) * q.Size
	if err := db.Offset(offset).Limit(q.Size).Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
