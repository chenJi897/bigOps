// backend/internal/repository/asset_repository.go
package repository

import (
	"fmt"

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

func (r *AssetRepository) GetByIDs(ids []int64) ([]*model.Asset, error) {
	var assets []*model.Asset
	if len(ids) == 0 {
		return assets, nil
	}
	if err := database.GetDB().Where("id IN ?", ids).Find(&assets).Error; err != nil {
		return nil, err
	}
	return assets, nil
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

// GetByCloudInstanceIDUnscoped 查找含软删除的资产（用于同步恢复场景）。
func (r *AssetRepository) GetByCloudInstanceIDUnscoped(cloudInstanceID string) (*model.Asset, error) {
	var asset model.Asset
	if err := database.GetDB().Unscoped().Where("cloud_instance_id = ?", cloudInstanceID).First(&asset).Error; err != nil {
		return nil, err
	}
	return &asset, nil
}

// RestoreSoftDeleted 恢复软删除的资产。
func (r *AssetRepository) RestoreSoftDeleted(id int64) error {
	return database.GetDB().Unscoped().Model(&model.Asset{}).Where("id = ?", id).Update("deleted_at", nil).Error
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
	Recursive     bool   // 是否递归查询子节点下的资产
	Source        string
	Keyword       string // 搜索 hostname 或 ip
	OwnerID       int64  // 按负责人筛选
}

func (r *AssetRepository) List(q AssetListQuery) ([]*model.Asset, int64, error) {
	var assets []*model.Asset
	var total int64
	db := database.GetDB().Model(&model.Asset{})

	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.ServiceTreeID > 0 {
		if q.Recursive {
			// 递归查找所有子节点 ID
			treeRepo := NewServiceTreeRepository()
			ids, err := treeRepo.GetAllDescendantIDs(q.ServiceTreeID)
			if err == nil && len(ids) > 0 {
				db = db.Where("service_tree_id IN ?", ids)
			} else {
				db = db.Where("service_tree_id = ?", q.ServiceTreeID)
			}
		} else {
			db = db.Where("service_tree_id = ?", q.ServiceTreeID)
		}
	}
	if q.Source != "" {
		db = db.Where("source = ?", q.Source)
	}
	if q.Keyword != "" {
		db = db.Where("hostname LIKE ? OR ip LIKE ?", "%"+q.Keyword+"%", "%"+q.Keyword+"%")
	}
	if q.OwnerID > 0 {
		// JSON 数组搜索: owner_ids 包含该 ID
		db = db.Where("JSON_CONTAINS(owner_ids, ?)", fmt.Sprintf("%d", q.OwnerID))
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

// ListByCloudAccountID 查询指定云账号下所有云同步来源的资产（不含手工资产）。
func (r *AssetRepository) ListByCloudAccountID(accountID int64) ([]*model.Asset, error) {
	var assets []*model.Asset
	err := database.GetDB().Where("cloud_account_id = ? AND source != 'manual'", accountID).Find(&assets).Error
	return assets, err
}

// CountByServiceTreeIDs 批量统计多个 service_tree_id 的资产数量。
func (r *AssetRepository) CountByServiceTreeIDs(ids []int64) (map[int64]int64, error) {
	type Result struct {
		ServiceTreeID int64 `gorm:"column:service_tree_id"`
		Count         int64 `gorm:"column:cnt"`
	}
	var results []Result
	err := database.GetDB().Model(&model.Asset{}).
		Select("service_tree_id, COUNT(*) as cnt").
		Where("service_tree_id IN ?", ids).
		Group("service_tree_id").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	m := make(map[int64]int64)
	for _, r := range results {
		m[r.ServiceTreeID] = r.Count
	}
	return m, nil
}
