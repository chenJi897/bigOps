package service

import (
	"context"
	"encoding/json"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type MonitorDatasourceService struct {
	repo *repository.MonitorDatasourceRepository
}

type MonitorDatasourceHealth struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func NewMonitorDatasourceService() *MonitorDatasourceService {
	return &MonitorDatasourceService{
		repo: repository.NewMonitorDatasourceRepository(),
	}
}

func (s *MonitorDatasourceService) List() ([]*model.MonitorDatasource, error) {
	return s.repo.List()
}

func (s *MonitorDatasourceService) Create(item *model.MonitorDatasource) error {
	if item.Type == "" {
		item.Type = "prometheus"
	}
	if item.AccessType == "" {
		item.AccessType = "proxy"
	}
	return s.repo.Create(item)
}

func (s *MonitorDatasourceService) Update(item *model.MonitorDatasource) error {
	return s.repo.Update(item)
}

func (s *MonitorDatasourceService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *MonitorDatasourceService) GetByID(id int64) (*model.MonitorDatasource, error) {
	return s.repo.GetByID(id)
}

func (s *MonitorDatasourceService) HealthCheck(ctx context.Context, id int64) (*MonitorDatasourceHealth, error) {
	ds, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	client := NewPrometheusClient(ds.BaseURL, ds.Username, ds.Password, parseHeaders(ds.HeadersJSON))
	if err := client.HealthCheck(ctx); err != nil {
		return &MonitorDatasourceHealth{OK: false, Message: err.Error()}, nil
	}
	return &MonitorDatasourceHealth{OK: true, Message: "ok"}, nil
}

func parseHeaders(raw string) map[string]string {
	result := make(map[string]string)
	if raw == "" {
		return result
	}
	_ = json.Unmarshal([]byte(raw), &result)
	return result
}
