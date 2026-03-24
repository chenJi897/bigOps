package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type TicketTypeRepository struct{}

func NewTicketTypeRepository() *TicketTypeRepository {
	return &TicketTypeRepository{}
}

func (r *TicketTypeRepository) Create(tt *model.TicketType) error {
	return database.GetDB().Create(tt).Error
}

func (r *TicketTypeRepository) Update(tt *model.TicketType) error {
	return database.GetDB().Save(tt).Error
}

func (r *TicketTypeRepository) GetByID(id int64) (*model.TicketType, error) {
	var tt model.TicketType
	if err := database.GetDB().First(&tt, id).Error; err != nil {
		return nil, err
	}
	return &tt, nil
}

func (r *TicketTypeRepository) GetByCode(code string) (*model.TicketType, error) {
	var tt model.TicketType
	if err := database.GetDB().Where("code = ?", code).First(&tt).Error; err != nil {
		return nil, err
	}
	return &tt, nil
}

func (r *TicketTypeRepository) GetByName(name string) (*model.TicketType, error) {
	var tt model.TicketType
	if err := database.GetDB().Where("name = ?", name).First(&tt).Error; err != nil {
		return nil, err
	}
	return &tt, nil
}

func (r *TicketTypeRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.TicketType{}, id).Error
}

func (r *TicketTypeRepository) List(page, size int) ([]*model.TicketType, int64, error) {
	var items []*model.TicketType
	var total int64
	db := database.GetDB().Model(&model.TicketType{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("sort ASC, id ASC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *TicketTypeRepository) GetAll() ([]*model.TicketType, error) {
	var items []*model.TicketType
	err := database.GetDB().Where("status = 1").Order("sort ASC, id ASC").Find(&items).Error
	return items, err
}
