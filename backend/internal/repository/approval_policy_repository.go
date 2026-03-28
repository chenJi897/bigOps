package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type ApprovalPolicyRepository struct{}

func NewApprovalPolicyRepository() *ApprovalPolicyRepository {
	return &ApprovalPolicyRepository{}
}

func (r *ApprovalPolicyRepository) Create(item *model.ApprovalPolicy) error {
	return database.GetDB().Create(item).Error
}

func (r *ApprovalPolicyRepository) Update(item *model.ApprovalPolicy) error {
	return database.GetDB().Save(item).Error
}

func (r *ApprovalPolicyRepository) GetByID(id int64) (*model.ApprovalPolicy, error) {
	var item model.ApprovalPolicy
	if err := database.GetDB().Preload("Stages").First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ApprovalPolicyRepository) GetByCode(code string) (*model.ApprovalPolicy, error) {
	var item model.ApprovalPolicy
	if err := database.GetDB().Where("code = ?", code).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ApprovalPolicyRepository) GetByName(name string) (*model.ApprovalPolicy, error) {
	var item model.ApprovalPolicy
	if err := database.GetDB().Where("name = ?", name).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ApprovalPolicyRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.ApprovalPolicy{}, id).Error
}

func (r *ApprovalPolicyRepository) List() ([]*model.ApprovalPolicy, error) {
	var items []*model.ApprovalPolicy
	if err := database.GetDB().Preload("Stages").Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ApprovalPolicyRepository) ReplaceStages(policyID int64, stages []model.ApprovalPolicyStage) error {
	if err := database.GetDB().Where("policy_id = ?", policyID).Delete(&model.ApprovalPolicyStage{}).Error; err != nil {
		return err
	}
	for i := range stages {
		stages[i].PolicyID = policyID
		if err := database.GetDB().Create(&stages[i]).Error; err != nil {
			return err
		}
	}
	return nil
}
