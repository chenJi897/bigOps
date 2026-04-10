package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type TaskApprovalRepository struct{}

func NewTaskApprovalRepository() *TaskApprovalRepository {
	return &TaskApprovalRepository{}
}

func (r *TaskApprovalRepository) Create(item *model.TaskApproval) error {
	return database.GetDB().Create(item).Error
}

func (r *TaskApprovalRepository) GetByID(id int64) (*model.TaskApproval, error) {
	var item model.TaskApproval
	err := database.GetDB().First(&item, id).Error
	return &item, err
}

func (r *TaskApprovalRepository) FindPendingByTask(taskID int64) (*model.TaskApproval, error) {
	var item model.TaskApproval
	err := database.GetDB().Where("task_id = ? AND status = ?", taskID, model.TaskApprovalStatusPending).First(&item).Error
	return &item, err
}

func (r *TaskApprovalRepository) FindLatestApprovedByTask(taskID int64) (*model.TaskApproval, error) {
	var item model.TaskApproval
	err := database.GetDB().Where("task_id = ? AND status = ?", taskID, model.TaskApprovalStatusApproved).
		Order("updated_at DESC").First(&item).Error
	return &item, err
}

func (r *TaskApprovalRepository) ListPending(page, size int) ([]*model.TaskApproval, int64, error) {
	var items []*model.TaskApproval
	var total int64
	db := database.GetDB().Model(&model.TaskApproval{}).Where("status = ?", model.TaskApprovalStatusPending)
	db.Count(&total)
	err := db.Offset((page-1)*size).Limit(size).Order("created_at DESC").Find(&items).Error
	return items, total, err
}

func (r *TaskApprovalRepository) ListByTask(taskID int64) ([]*model.TaskApproval, error) {
	var items []*model.TaskApproval
	err := database.GetDB().Where("task_id = ?", taskID).Order("created_at DESC").Find(&items).Error
	return items, err
}

func (r *TaskApprovalRepository) Update(item *model.TaskApproval) error {
	return database.GetDB().Save(item).Error
}
