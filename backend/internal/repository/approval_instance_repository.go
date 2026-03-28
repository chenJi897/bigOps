package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type ApprovalInstanceRepository struct{}

func NewApprovalInstanceRepository() *ApprovalInstanceRepository {
	return &ApprovalInstanceRepository{}
}

func (r *ApprovalInstanceRepository) Create(item *model.ApprovalInstance) error {
	return database.GetDB().Create(item).Error
}

func (r *ApprovalInstanceRepository) Update(item *model.ApprovalInstance) error {
	return database.GetDB().Save(item).Error
}

func (r *ApprovalInstanceRepository) GetByID(id int64) (*model.ApprovalInstance, error) {
	var item model.ApprovalInstance
	if err := database.GetDB().Preload("Records").First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ApprovalInstanceRepository) GetByTicketID(ticketID int64) (*model.ApprovalInstance, error) {
	var item model.ApprovalInstance
	if err := database.GetDB().Where("ticket_id = ?", ticketID).Preload("Records").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ApprovalInstanceRepository) CreateRecord(record *model.ApprovalRecord) error {
	return database.GetDB().Create(record).Error
}

func (r *ApprovalInstanceRepository) GetRecordByInstanceStageApprover(instanceID int64, stageNo int, approverID int64) (*model.ApprovalRecord, error) {
	var item model.ApprovalRecord
	if err := database.GetDB().
		Where("instance_id = ? AND stage_no = ? AND approver_id = ?", instanceID, stageNo, approverID).
		First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ApprovalInstanceRepository) ListRecordsByInstanceStage(instanceID int64, stageNo int) ([]*model.ApprovalRecord, error) {
	var items []*model.ApprovalRecord
	if err := database.GetDB().
		Where("instance_id = ? AND stage_no = ?", instanceID, stageNo).
		Order("id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ApprovalInstanceRepository) UpdateRecord(record *model.ApprovalRecord) error {
	return database.GetDB().Save(record).Error
}

func (r *ApprovalInstanceRepository) ListPendingRecordsByApproverID(approverID int64) ([]*model.ApprovalRecord, error) {
	var items []*model.ApprovalRecord
	if err := database.GetDB().
		Where("approver_id = ? AND status = ?", approverID, "pending").
		Order("id DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
