package repository

import (
	"time"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type AgentMetricSampleListQuery struct {
	Page       int
	Size       int
	AgentID    string
	MetricType string
	IP         string
	StartAt    *time.Time
	EndAt      *time.Time
}

type AgentMetricSampleRepository struct{}

func NewAgentMetricSampleRepository() *AgentMetricSampleRepository {
	return &AgentMetricSampleRepository{}
}

func (r *AgentMetricSampleRepository) CreateBatch(items []model.AgentMetricSample) error {
	if len(items) == 0 {
		return nil
	}
	return database.GetDB().Create(&items).Error
}

func (r *AgentMetricSampleRepository) List(q AgentMetricSampleListQuery) ([]*model.AgentMetricSample, int64, error) {
	var items []*model.AgentMetricSample
	var total int64

	db := database.GetDB().Model(&model.AgentMetricSample{})
	if q.AgentID != "" {
		db = db.Where("agent_id = ?", q.AgentID)
	}
	if q.MetricType != "" {
		db = db.Where("metric_type = ?", q.MetricType)
	}
	if q.IP != "" {
		db = db.Where("ip = ?", q.IP)
	}
	if q.StartAt != nil {
		db = db.Where("collected_at >= ?", *q.StartAt)
	}
	if q.EndAt != nil {
		db = db.Where("collected_at <= ?", *q.EndAt)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (q.Page - 1) * q.Size
	if err := db.Offset(offset).Limit(q.Size).Order("collected_at DESC, id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *AgentMetricSampleRepository) ListTrend(agentID, metricType string, startAt, endAt *time.Time, limit int) ([]*model.AgentMetricSample, error) {
	var items []*model.AgentMetricSample
	db := database.GetDB().Model(&model.AgentMetricSample{}).Where("agent_id = ? AND metric_type = ?", agentID, metricType)
	if startAt != nil {
		db = db.Where("collected_at >= ?", *startAt)
	}
	if endAt != nil {
		db = db.Where("collected_at <= ?", *endAt)
	}
	if limit > 0 {
		db = db.Limit(limit)
	}
	if err := db.Order("collected_at ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AgentMetricSampleRepository) GetLatestCollectedAt() (*model.LocalTime, error) {
	var item model.AgentMetricSample
	if err := database.GetDB().Order("collected_at DESC, id DESC").First(&item).Error; err != nil {
		return nil, err
	}
	return &item.CollectedAt, nil
}
