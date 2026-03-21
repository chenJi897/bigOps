package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type CloudAccountRepository struct{}

func NewCloudAccountRepository() *CloudAccountRepository {
	return &CloudAccountRepository{}
}

func (r *CloudAccountRepository) Create(account *model.CloudAccount) error {
	return database.GetDB().Create(account).Error
}

func (r *CloudAccountRepository) GetByID(id int64) (*model.CloudAccount, error) {
	var account model.CloudAccount
	if err := database.GetDB().First(&account, id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *CloudAccountRepository) Update(account *model.CloudAccount) error {
	return database.GetDB().Save(account).Error
}

func (r *CloudAccountRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.CloudAccount{}, id).Error
}

func (r *CloudAccountRepository) List(page, size int) ([]*model.CloudAccount, int64, error) {
	var accounts []*model.CloudAccount
	var total int64
	db := database.GetDB().Model(&model.CloudAccount{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&accounts).Error; err != nil {
		return nil, 0, err
	}
	return accounts, total, nil
}
