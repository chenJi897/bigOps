package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type AlertRuleListQuery struct {
	Page       int
	Size       int
	Keyword    string
	MetricType string
	Severity   string
	Enabled    *int8
}

type AlertRuleRepository struct{}

func NewAlertRuleRepository() *AlertRuleRepository {
	return &AlertRuleRepository{}
}

func (r *AlertRuleRepository) Create(item *model.AlertRule) error {
	return database.GetDB().Create(item).Error
}

func (r *AlertRuleRepository) Update(item *model.AlertRule) error {
	return database.GetDB().Save(item).Error
}

func (r *AlertRuleRepository) GetByID(id int64) (*model.AlertRule, error) {
	var item model.AlertRule
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AlertRuleRepository) GetByName(name string) (*model.AlertRule, error) {
	var item model.AlertRule
	if err := database.GetDB().Where("name = ?", name).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AlertRuleRepository) GetByNameExcludingID(name string, excludeID int64) (*model.AlertRule, error) {
	var item model.AlertRule
	if err := database.GetDB().Where("name = ? AND id <> ?", name, excludeID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AlertRuleRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.AlertRule{}, id).Error
}

func (r *AlertRuleRepository) List(q AlertRuleListQuery) ([]*model.AlertRule, int64, error) {
	var items []*model.AlertRule
	var total int64
	db := database.GetDB().Model(&model.AlertRule{})
	if q.Keyword != "" {
		db = db.Where("name LIKE ?", "%"+q.Keyword+"%")
	}
	if q.MetricType != "" {
		db = db.Where("metric_type = ?", q.MetricType)
	}
	if q.Severity != "" {
		db = db.Where("severity = ?", q.Severity)
	}
	if q.Enabled != nil {
		db = db.Where("enabled = ?", *q.Enabled)
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

func (r *AlertRuleRepository) ListEnabled() ([]*model.AlertRule, error) {
	var items []*model.AlertRule
	if err := database.GetDB().
		Where("enabled = ?", 1).
		Order("id DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AlertRuleRepository) CountByEnabled(enabled int8) (int64, error) {
	var total int64
	err := database.GetDB().Model(&model.AlertRule{}).Where("enabled = ?", enabled).Count(&total).Error
	return total, err
}
