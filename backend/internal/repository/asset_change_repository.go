// backend/internal/repository/asset_change_repository.go
package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type AssetChangeRepository struct{}

func NewAssetChangeRepository() *AssetChangeRepository {
	return &AssetChangeRepository{}
}

func (r *AssetChangeRepository) Create(change *model.AssetChange) error {
	return database.GetDB().Create(change).Error
}

func (r *AssetChangeRepository) ListByAssetID(assetID int64, page, size int) ([]*model.AssetChange, int64, error) {
	var changes []*model.AssetChange
	var total int64
	db := database.GetDB().Model(&model.AssetChange{}).Where("asset_id = ?", assetID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&changes).Error; err != nil {
		return nil, 0, err
	}
	return changes, total, nil
}
