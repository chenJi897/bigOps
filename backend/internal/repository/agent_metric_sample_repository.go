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

type MetricAggRow struct {
	MetricType string  `json:"metric_type"`
	AvgValue   float64 `json:"avg_value"`
	MaxValue   float64 `json:"max_value"`
	SampleCnt  int64   `json:"sample_count"`
	OverThresh int64   `json:"over_threshold"`
}

func (r *AgentMetricSampleRepository) AggregateByWindow(since time.Time, threshold float64) ([]MetricAggRow, error) {
	var rows []MetricAggRow
	if err := database.GetDB().Table("agent_metric_samples").
		Select(`metric_type,
			AVG(metric_value) as avg_value,
			MAX(metric_value) as max_value,
			COUNT(*) as sample_cnt,
			SUM(CASE WHEN metric_value >= ? THEN 1 ELSE 0 END) as over_thresh`, threshold).
		Where("collected_at >= ? AND deleted_at IS NULL", since).
		Group("metric_type").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

type MetricDimensionRow struct {
	DimensionKey string  `json:"dimension_key"`
	MetricType   string  `json:"metric_type"`
	AvgValue     float64 `json:"avg_value"`
	MaxValue     float64 `json:"max_value"`
	SampleCnt    int64   `json:"sample_count"`
	OverThresh   int64   `json:"over_threshold"`
}

func (r *AgentMetricSampleRepository) AggregateByDimension(since time.Time, dimension string, threshold float64) ([]MetricDimensionRow, error) {
	dimCol := "agent_id"
	switch dimension {
	case "instance":
		dimCol = "agent_id"
	case "metric_type":
		dimCol = "metric_type"
	default:
		dimCol = "ip"
	}
	var rows []MetricDimensionRow
	if err := database.GetDB().Table("agent_metric_samples").
		Select(dimCol+` as dimension_key, metric_type,
			AVG(metric_value) as avg_value,
			MAX(metric_value) as max_value,
			COUNT(*) as sample_cnt,
			SUM(CASE WHEN metric_value >= ? THEN 1 ELSE 0 END) as over_thresh`, threshold).
		Where("collected_at >= ? AND deleted_at IS NULL", since).
		Group(dimCol+", metric_type").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *AgentMetricSampleRepository) GetLatestCollectedAt() (*model.LocalTime, error) {
	var item model.AgentMetricSample
	if err := database.GetDB().Order("collected_at DESC, id DESC").First(&item).Error; err != nil {
		return nil, err
	}
	return &item.CollectedAt, nil
}
