package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type RequestTemplateRepository struct{}

func NewRequestTemplateRepository() *RequestTemplateRepository {
	return &RequestTemplateRepository{}
}

func (r *RequestTemplateRepository) Create(item *model.RequestTemplate) error {
	return database.GetDB().Create(item).Error
}

func (r *RequestTemplateRepository) Update(item *model.RequestTemplate) error {
	return database.GetDB().Save(item).Error
}

func (r *RequestTemplateRepository) GetByID(id int64) (*model.RequestTemplate, error) {
	var item model.RequestTemplate
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *RequestTemplateRepository) GetByCode(code string) (*model.RequestTemplate, error) {
	var item model.RequestTemplate
	if err := database.GetDB().Where("code = ?", code).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *RequestTemplateRepository) GetByName(name string) (*model.RequestTemplate, error) {
	var item model.RequestTemplate
	if err := database.GetDB().Where("name = ?", name).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// GetByIDs 批量按 ID 查询，返回 id->RequestTemplate 映射。
func (r *RequestTemplateRepository) GetByIDs(ids []int64) (map[int64]*model.RequestTemplate, error) {
	result := make(map[int64]*model.RequestTemplate)
	if len(ids) == 0 {
		return result, nil
	}
	var items []*model.RequestTemplate
	if err := database.GetDB().Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.ID] = item
	}
	return result, nil
}

func (r *RequestTemplateRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.RequestTemplate{}, id).Error
}

func (r *RequestTemplateRepository) List(enabledOnly bool) ([]*model.RequestTemplate, error) {
	var items []*model.RequestTemplate
	db := database.GetDB().Model(&model.RequestTemplate{})
	if enabledOnly {
		db = db.Where("status = 1")
	}
	if err := db.Order("sort ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
