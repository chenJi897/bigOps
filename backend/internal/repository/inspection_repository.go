package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type InspectionRepository struct{}

func NewInspectionRepository() *InspectionRepository {
	return &InspectionRepository{}
}

func (r *InspectionRepository) ListTemplates(page, size int) ([]*model.InspectionTemplate, int64, error) {
	var items []*model.InspectionTemplate
	var total int64
	db := database.GetDB().Model(&model.InspectionTemplate{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *InspectionRepository) CreateTemplate(item *model.InspectionTemplate) error {
	return database.GetDB().Create(item).Error
}

func (r *InspectionRepository) UpdateTemplate(item *model.InspectionTemplate) error {
	return database.GetDB().Save(item).Error
}

func (r *InspectionRepository) GetTemplate(id int64) (*model.InspectionTemplate, error) {
	var item model.InspectionTemplate
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *InspectionRepository) ListPlans(page, size int) ([]*model.InspectionPlan, int64, error) {
	var items []*model.InspectionPlan
	var total int64
	db := database.GetDB().Model(&model.InspectionPlan{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *InspectionRepository) ListEnabledPlans() ([]*model.InspectionPlan, error) {
	var items []*model.InspectionPlan
	if err := database.GetDB().
		Where("enabled = ?", 1).
		Order("id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *InspectionRepository) CreatePlan(item *model.InspectionPlan) error {
	return database.GetDB().Create(item).Error
}

func (r *InspectionRepository) UpdatePlan(item *model.InspectionPlan) error {
	return database.GetDB().Save(item).Error
}

func (r *InspectionRepository) GetPlan(id int64) (*model.InspectionPlan, error) {
	var item model.InspectionPlan
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *InspectionRepository) CreateRecord(item *model.InspectionRecord) error {
	return database.GetDB().Create(item).Error
}

func (r *InspectionRepository) UpdateRecord(item *model.InspectionRecord) error {
	return database.GetDB().Save(item).Error
}

func (r *InspectionRepository) ListRecords(page, size int) ([]*model.InspectionRecord, int64, error) {
	var items []*model.InspectionRecord
	var total int64
	db := database.GetDB().Model(&model.InspectionRecord{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *InspectionRepository) GetRecord(id int64) (*model.InspectionRecord, error) {
	var item model.InspectionRecord
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *InspectionRepository) ListRecordsByTemplate(templateID int64, limit int) ([]*model.InspectionRecord, error) {
	if limit <= 0 {
		limit = 30
	}
	var items []*model.InspectionRecord
	if err := database.GetDB().
		Where("template_id = ?", templateID).
		Order("id DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
