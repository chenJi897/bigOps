package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type CICDProjectListQuery struct {
	Page    int
	Size    int
	Keyword string
	Status  *int8
}

type CICDProjectRepository struct{}

func NewCICDProjectRepository() *CICDProjectRepository {
	return &CICDProjectRepository{}
}

func (r *CICDProjectRepository) Create(item *model.CICDProject) error {
	return database.GetDB().Create(item).Error
}

func (r *CICDProjectRepository) Update(item *model.CICDProject) error {
	return database.GetDB().Save(item).Error
}

func (r *CICDProjectRepository) GetByID(id int64) (*model.CICDProject, error) {
	var item model.CICDProject
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CICDProjectRepository) GetByName(name string) (*model.CICDProject, error) {
	var item model.CICDProject
	if err := database.GetDB().Where("name = ?", name).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CICDProjectRepository) GetByCode(code string) (*model.CICDProject, error) {
	var item model.CICDProject
	if err := database.GetDB().Where("code = ?", code).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CICDProjectRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.CICDProject{}, id).Error
}

func (r *CICDProjectRepository) List(q CICDProjectListQuery) ([]*model.CICDProject, int64, error) {
	var items []*model.CICDProject
	var total int64
	db := database.GetDB().Model(&model.CICDProject{})
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		db = db.Where("name LIKE ? OR code LIKE ? OR repo_url LIKE ?", like, like, like)
	}
	if q.Status != nil {
		db = db.Where("status = ?", *q.Status)
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
