package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

// CICDPipelineRunListQuery 支持流水线执行记录的多维分页查询。
type CICDPipelineRunListQuery struct {
	Page       int
	Size       int
	PipelineID int64
	ProjectID  int64
	Status     string
}

// CICDPipelineRunRepository 提供流水线运行历史的增删改查。
type CICDPipelineRunRepository struct{}

func NewCICDPipelineRunRepository() *CICDPipelineRunRepository {
	return &CICDPipelineRunRepository{}
}

func (r *CICDPipelineRunRepository) Create(run *model.CICDPipelineRun) error {
	return database.GetDB().Create(run).Error
}

func (r *CICDPipelineRunRepository) Update(run *model.CICDPipelineRun) error {
	return database.GetDB().Save(run).Error
}

func (r *CICDPipelineRunRepository) GetByID(id int64) (*model.CICDPipelineRun, error) {
	var run model.CICDPipelineRun
	if err := database.GetDB().First(&run, id).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *CICDPipelineRunRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.CICDPipelineRun{}, id).Error
}

func (r *CICDPipelineRunRepository) GetLatestByPipeline(pipelineID int64) (*model.CICDPipelineRun, error) {
	var run model.CICDPipelineRun
	if err := database.GetDB().
		Where("pipeline_id = ?", pipelineID).
		Order("run_number DESC, id DESC").
		First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *CICDPipelineRunRepository) List(q CICDPipelineRunListQuery) ([]*model.CICDPipelineRun, int64, error) {
	var runs []*model.CICDPipelineRun
	var total int64
	db := database.GetDB().Model(&model.CICDPipelineRun{})

	if q.PipelineID > 0 {
		db = db.Where("pipeline_id = ?", q.PipelineID)
	}
	if q.ProjectID > 0 {
		db = db.Where("project_id = ?", q.ProjectID)
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
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
	if err := db.Offset(offset).
		Limit(q.Size).
		Order("id DESC").
		Find(&runs).Error; err != nil {
		return nil, 0, err
	}

	return runs, total, nil
}

func (r *CICDPipelineRunRepository) ListWaitingApprovalByTicketID(ticketID int64) ([]*model.CICDPipelineRun, error) {
	var runs []*model.CICDPipelineRun
	if err := database.GetDB().
		Where("approval_ticket_id = ? AND status = ?", ticketID, "waiting_approval").
		Order("id ASC").
		Find(&runs).Error; err != nil {
		return nil, err
	}
	return runs, nil
}
