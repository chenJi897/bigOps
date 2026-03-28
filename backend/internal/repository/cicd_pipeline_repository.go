package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

// CICDPipelineListQuery encapsulates pagination and filters for pipelines.
type CICDPipelineListQuery struct {
	Page      int
	Size      int
	ProjectID int64
	Keyword   string
	Status    *int8
}

// CICDPipelineRepository 实现流水线定义的基础 CRUD。
type CICDPipelineRepository struct{}

func NewCICDPipelineRepository() *CICDPipelineRepository {
	return &CICDPipelineRepository{}
}

func (r *CICDPipelineRepository) Create(item *model.CICDPipeline) error {
	return database.GetDB().Create(item).Error
}

func (r *CICDPipelineRepository) Update(item *model.CICDPipeline) error {
	return database.GetDB().Save(item).Error
}

func (r *CICDPipelineRepository) GetByID(id int64) (*model.CICDPipeline, error) {
	var item model.CICDPipeline
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CICDPipelineRepository) GetByName(name string) (*model.CICDPipeline, error) {
	var item model.CICDPipeline
	if err := database.GetDB().Where("name = ?", name).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CICDPipelineRepository) GetByCode(code string) (*model.CICDPipeline, error) {
	var item model.CICDPipeline
	if err := database.GetDB().Where("code = ?", code).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CICDPipelineRepository) GetByProjectAndCode(projectID int64, code string) (*model.CICDPipeline, error) {
	var item model.CICDPipeline
	if err := database.GetDB().Where("project_id = ? AND code = ?", projectID, code).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CICDPipelineRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.CICDPipeline{}, id).Error
}

func (r *CICDPipelineRepository) List(q CICDPipelineListQuery) ([]*model.CICDPipeline, int64, error) {
	var items []*model.CICDPipeline
	var total int64
	db := database.GetDB().Model(&model.CICDPipeline{})

	if q.ProjectID > 0 {
		db = db.Where("project_id = ?", q.ProjectID)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		db = db.Where("name LIKE ? OR code LIKE ?", like, like)
	}
	if q.Status != nil {
		db = db.Where("status = ?", *q.Status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Size <= 0 {
		q.Size = 20
	}
	offset := (q.Page - 1) * q.Size
	if err := db.Offset(offset).Limit(q.Size).Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
