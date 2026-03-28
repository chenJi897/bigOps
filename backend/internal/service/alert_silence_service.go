package service

import (
	"errors"
	"time"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type AlertSilenceService struct {
	repo *repository.AlertSilenceRepository
}

func NewAlertSilenceService() *AlertSilenceService {
	return &AlertSilenceService{repo: repository.NewAlertSilenceRepository()}
}

func (s *AlertSilenceService) List() ([]*model.AlertSilence, error) {
	return s.repo.List()
}

func (s *AlertSilenceService) Create(item *model.AlertSilence) error {
	if item.Name == "" {
		return errors.New("静默名称不能为空")
	}
	if time.Time(item.EndsAt).Before(time.Time(item.StartsAt)) {
		return errors.New("结束时间不能早于开始时间")
	}
	return s.repo.Create(item)
}

func (s *AlertSilenceService) Update(item *model.AlertSilence) error {
	if item.Name == "" {
		return errors.New("静默名称不能为空")
	}
	if time.Time(item.EndsAt).Before(time.Time(item.StartsAt)) {
		return errors.New("结束时间不能早于开始时间")
	}
	return s.repo.Update(item)
}

func (s *AlertSilenceService) Delete(id int64) error {
	return s.repo.Delete(id)
}
