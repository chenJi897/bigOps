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
	if dimension == "service" {
		return r.aggregateByServiceTree(since, threshold)
	}
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

func (r *AgentMetricSampleRepository) aggregateByServiceTree(since time.Time, threshold float64) ([]MetricDimensionRow, error) {
	var rows []MetricDimensionRow
	q := database.GetDB().Table("agent_metric_samples s").
		Joins("LEFT JOIN agent_infos a ON s.agent_id = a.agent_id").
		Joins("LEFT JOIN assets ast ON (a.private_ip = ast.ip OR a.public_ip = ast.ip OR a.ip = ast.ip) AND ast.deleted_at IS NULL").
		Joins("LEFT JOIN service_trees st ON ast.service_tree_id = st.id AND st.deleted_at IS NULL").
		Select(`COALESCE(st.name, CONCAT('IP:', s.ip)) as dimension_key, s.metric_type,
			AVG(s.metric_value) as avg_value,
			MAX(s.metric_value) as max_value,
			COUNT(*) as sample_cnt,
			SUM(CASE WHEN s.metric_value >= ? THEN 1 ELSE 0 END) as over_thresh`, threshold).
		Where("s.collected_at >= ? AND s.deleted_at IS NULL", since).
		Group("dimension_key, s.metric_type")
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

type AgentBaselineRow struct {
	AgentID    string  `json:"agent_id"`
	MetricType string  `json:"metric_type"`
	AvgValue   float64 `json:"avg_value"`
	StdDev     float64 `json:"std_dev"`
	SampleCnt  int64   `json:"sample_count"`
}

func (r *AgentMetricSampleRepository) AggregateBaselineByAgent() ([]AgentBaselineRow, error) {
	since := time.Now().Add(-24 * time.Hour)
	var rows []AgentBaselineRow
	if err := database.GetDB().Table("agent_metric_samples").
		Select(`agent_id, metric_type,
			AVG(metric_value) as avg_value,
			STDDEV_POP(metric_value) as std_dev,
			COUNT(*) as sample_cnt`).
		Where("collected_at >= ? AND deleted_at IS NULL", since).
		Group("agent_id, metric_type").
		Having("COUNT(*) >= 10").
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
