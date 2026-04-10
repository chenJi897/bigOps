package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type AlertEventListQuery struct {
	Page     int
	Size     int
	Status   string
	Severity string
	AgentID  string
	Keyword  string
	RuleID   *int64
}

type AlertEventRepository struct{}

func NewAlertEventRepository() *AlertEventRepository {
	return &AlertEventRepository{}
}

func (r *AlertEventRepository) Create(item *model.AlertEvent) error {
	return database.GetDB().Create(item).Error
}

func (r *AlertEventRepository) Update(item *model.AlertEvent) error {
	return database.GetDB().Save(item).Error
}

func (r *AlertEventRepository) FindOpenByRuleAgent(ruleID int64, agentID string) (*model.AlertEvent, error) {
	var item model.AlertEvent
	if err := database.GetDB().
		Where("rule_id = ? AND agent_id = ? AND status IN ?", ruleID, agentID, []string{
			model.AlertEventStatusFiring,
			model.AlertEventStatusAcknowledged,
		}).
		Order("id DESC").
		First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AlertEventRepository) List(q AlertEventListQuery) ([]*model.AlertEvent, int64, error) {
	var items []*model.AlertEvent
	var total int64
	db := database.GetDB().Model(&model.AlertEvent{})
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.Severity != "" {
		db = db.Where("severity = ?", q.Severity)
	}
	if q.AgentID != "" {
		db = db.Where("agent_id = ?", q.AgentID)
	}
	if q.Keyword != "" {
		keyword := "%" + q.Keyword + "%"
		db = db.Where("rule_name LIKE ? OR hostname LIKE ? OR ip LIKE ?", keyword, keyword, keyword)
	}
	if q.RuleID != nil {
		db = db.Where("rule_id = ?", *q.RuleID)
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

func (r *AlertEventRepository) GetByID(id int64) (*model.AlertEvent, error) {
	var item model.AlertEvent
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AlertEventRepository) CountByStatus() (map[string]int64, error) {
	type row struct {
		Status string
		Total  int64
	}
	var rows []row
	if err := database.GetDB().Model(&model.AlertEvent{}).
		Select("status, COUNT(*) as total").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, entry := range rows {
		result[entry.Status] = entry.Total
	}
	return result, nil
}

func (r *AlertEventRepository) CountBySeverity() (map[string]int64, error) {
	type row struct {
		Severity string
		Total    int64
	}
	var rows []row
	if err := database.GetDB().Model(&model.AlertEvent{}).
		Select("severity, COUNT(*) as total").
		Group("severity").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, entry := range rows {
		result[entry.Severity] = entry.Total
	}
	return result, nil
}

func (r *AlertEventRepository) ListLatest(limit int) ([]*model.AlertEvent, error) {
	if limit <= 0 {
		limit = 5
	}
	var items []*model.AlertEvent
	if err := database.GetDB().
		Order("triggered_at DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AlertEventRepository) ListByRuleAgent(ruleID int64, agentID string, limit int) ([]*model.AlertEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	var items []*model.AlertEvent
	if err := database.GetDB().
		Where("rule_id = ? AND agent_id = ?", ruleID, agentID).
		Order("triggered_at DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
