package repository

import (
	"time"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type AlertSilenceRepository struct{}

func NewAlertSilenceRepository() *AlertSilenceRepository {
	return &AlertSilenceRepository{}
}

func (r *AlertSilenceRepository) Create(item *model.AlertSilence) error {
	return database.GetDB().Create(item).Error
}

func (r *AlertSilenceRepository) Update(item *model.AlertSilence) error {
	return database.GetDB().Save(item).Error
}

func (r *AlertSilenceRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.AlertSilence{}, id).Error
}

func (r *AlertSilenceRepository) GetByID(id int64) (*model.AlertSilence, error) {
	var item model.AlertSilence
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AlertSilenceRepository) List() ([]*model.AlertSilence, error) {
	var items []*model.AlertSilence
	if err := database.GetDB().Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AlertSilenceRepository) ListEnabled() ([]*model.AlertSilence, error) {
	var items []*model.AlertSilence
	if err := database.GetDB().
		Where("enabled = 1").
		Order("id DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AlertSilenceRepository) ListActive(now time.Time) ([]*model.AlertSilence, error) {
	var items []*model.AlertSilence
	if err := database.GetDB().
		Where("enabled = 1 AND starts_at <= ? AND ends_at >= ?", now, now).
		Order("id DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
