// backend/internal/service/asset_service.go
package service

import (
	"errors"

	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type AssetService struct {
	assetRepo       *repository.AssetRepository
	serviceTreeRepo *repository.ServiceTreeRepository
}

func NewAssetService() *AssetService {
	return &AssetService{
		assetRepo:       repository.NewAssetRepository(),
		serviceTreeRepo: repository.NewServiceTreeRepository(),
	}
}

func (s *AssetService) Create(asset *model.Asset) error {
	if asset.Hostname == "" {
		return errors.New("主机名不能为空")
	}
	_, err := s.assetRepo.GetByHostname(asset.Hostname)
	if err == nil {
		return errors.New("主机名已存在")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if asset.Source == "" {
		asset.Source = "manual"
	}
	if asset.Status == "" {
		asset.Status = "online"
	}
	return s.assetRepo.Create(asset)
}

func (s *AssetService) Update(asset *model.Asset) error {
	existing, err := s.assetRepo.GetByID(asset.ID)
	if err != nil {
		return errors.New("资产不存在")
	}
	existing.Hostname = asset.Hostname
	existing.IP = asset.IP
	existing.InnerIP = asset.InnerIP
	existing.OS = asset.OS
	existing.OSVersion = asset.OSVersion
	existing.CPUCores = asset.CPUCores
	existing.MemoryMB = asset.MemoryMB
	existing.DiskGB = asset.DiskGB
	existing.Status = asset.Status
	existing.AssetType = asset.AssetType
	existing.ServiceTreeID = asset.ServiceTreeID
	existing.IDC = asset.IDC
	existing.SN = asset.SN
	existing.Tags = asset.Tags
	existing.Remark = asset.Remark
	return s.assetRepo.Update(existing)
}

func (s *AssetService) Delete(id int64) error {
	return s.assetRepo.Delete(id)
}

func (s *AssetService) GetByID(id int64) (*model.Asset, error) {
	asset, err := s.assetRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	s.fillServiceTreeName(asset)
	return asset, nil
}

func (s *AssetService) List(q repository.AssetListQuery) ([]*model.Asset, int64, error) {
	assets, total, err := s.assetRepo.List(q)
	if err != nil {
		return nil, 0, err
	}
	for _, a := range assets {
		s.fillServiceTreeName(a)
	}
	return assets, total, nil
}

func (s *AssetService) fillServiceTreeName(asset *model.Asset) {
	if asset.ServiceTreeID > 0 {
		if node, err := s.serviceTreeRepo.GetByID(asset.ServiceTreeID); err == nil {
			asset.ServiceTreeName = node.Name
		}
	}
}
