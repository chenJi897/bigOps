// backend/internal/repository/asset_repository.go
package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type AssetRepository struct{}

func NewAssetRepository() *AssetRepository {
	return &AssetRepository{}
}

func (r *AssetRepository) Create(asset *model.Asset) error {
	return database.GetDB().Create(asset).Error
}

func (r *AssetRepository) GetByID(id int64) (*model.Asset, error) {
	var asset model.Asset
	if err := database.GetDB().First(&asset, id).Error; err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *AssetRepository) GetByHostname(hostname string) (*model.Asset, error) {
	var asset model.Asset
	if err := database.GetDB().Where("hostname = ?", hostname).First(&asset).Error; err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *AssetRepository) GetByCloudInstanceID(cloudInstanceID string) (*model.Asset, error) {
	var asset model.Asset
	if err := database.GetDB().Where("cloud_instance_id = ?", cloudInstanceID).First(&asset).Error; err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *AssetRepository) Update(asset *model.Asset) error {
	return database.GetDB().Save(asset).Error
}

func (r *AssetRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.Asset{}, id).Error
}

type AssetListQuery struct {
	Page          int
	Size          int
	Status        string
	ServiceTreeID int64
	Source        string
	Keyword       string // 搜索 hostname 或 ip
}

func (r *AssetRepository) List(q AssetListQuery) ([]*model.Asset, int64, error) {
	var assets []*model.Asset
	var total int64
	db := database.GetDB().Model(&model.Asset{})

	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.ServiceTreeID > 0 {
		db = db.Where("service_tree_id = ?", q.ServiceTreeID)
	}
	if q.Source != "" {
		db = db.Where("source = ?", q.Source)
	}
	if q.Keyword != "" {
		db = db.Where("hostname LIKE ? OR ip LIKE ?", "%"+q.Keyword+"%", "%"+q.Keyword+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (q.Page - 1) * q.Size
	if err := db.Offset(offset).Limit(q.Size).Order("id DESC").Find(&assets).Error; err != nil {
		return nil, 0, err
	}
	return assets, total, nil
}

func (r *AssetRepository) CountByServiceTreeID(serviceTreeID int64) (int64, error) {
	var count int64
	err := database.GetDB().Model(&model.Asset{}).Where("service_tree_id = ?", serviceTreeID).Count(&count).Error
	return count, err
}
