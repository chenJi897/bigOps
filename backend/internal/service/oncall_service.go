package service

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type OnCallService struct {
	repo *repository.OnCallScheduleRepository
}

func NewOnCallService() *OnCallService {
	return &OnCallService{repo: repository.NewOnCallScheduleRepository()}
}

func (s *OnCallService) List() ([]*model.OnCallSchedule, error) {
	return s.repo.List()
}

func (s *OnCallService) Create(item *model.OnCallSchedule) error {
	if item.Name == "" {
		return errors.New("值班名称不能为空")
	}
	return s.repo.Create(item)
}

func (s *OnCallService) Update(item *model.OnCallSchedule) error {
	if item.Name == "" {
		return errors.New("值班名称不能为空")
	}
	return s.repo.Update(item)
}

func (s *OnCallService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *OnCallService) CurrentUserIDs(scheduleID int64, now time.Time) ([]int64, error) {
	schedule, err := s.repo.GetByID(scheduleID)
	if err != nil {
		return nil, err
	}
	var userIDs []int64
	if err := json.Unmarshal([]byte(schedule.UsersJSON), &userIDs); err != nil {
		return nil, err
	}
	if len(userIDs) == 0 {
		return nil, nil
	}
	rotationDays := schedule.RotationDays
	if rotationDays <= 0 {
		rotationDays = 1
	}
	slot := int(now.Unix() / int64(24*3600*rotationDays))
	return []int64{userIDs[slot%len(userIDs)]}, nil
}
