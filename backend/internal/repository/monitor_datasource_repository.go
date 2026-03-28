package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type MonitorDatasourceRepository struct{}

func NewMonitorDatasourceRepository() *MonitorDatasourceRepository {
	return &MonitorDatasourceRepository{}
}

func (r *MonitorDatasourceRepository) Create(item *model.MonitorDatasource) error {
	return database.GetDB().Create(item).Error
}

func (r *MonitorDatasourceRepository) Update(item *model.MonitorDatasource) error {
	return database.GetDB().Save(item).Error
}

func (r *MonitorDatasourceRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.MonitorDatasource{}, id).Error
}

func (r *MonitorDatasourceRepository) GetByID(id int64) (*model.MonitorDatasource, error) {
	var item model.MonitorDatasource
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *MonitorDatasourceRepository) List() ([]*model.MonitorDatasource, error) {
	var items []*model.MonitorDatasource
	if err := database.GetDB().Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
