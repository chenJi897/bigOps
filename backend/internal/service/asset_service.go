// backend/internal/service/asset_service.go
package service

import (
	"encoding/json"
	"errors"
	"strconv"

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

func (s *AssetService) Update(asset *model.Asset, operatorID int64, operatorName string) error {
	existing, err := s.assetRepo.GetByID(asset.ID)
	if err != nil {
		return errors.New("资产不存在")
	}

	// diff 对比变更
	changes := s.diffAsset(existing, asset)

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
	existing.OwnerIDs = asset.OwnerIDs

	if err := s.assetRepo.Update(existing); err != nil {
		return err
	}

	// 记录变更历史
	if len(changes) > 0 {
		changeRepo := repository.NewAssetChangeRepository()
		for i := range changes {
			changes[i].AssetID = existing.ID
			changes[i].ChangeType = "manual"
			changes[i].OperatorID = operatorID
			changes[i].OperatorName = operatorName
			changeRepo.Create(&changes[i])
		}
	}
	return nil
}

// diffAsset 对比资产关键字段，返回变更列表。
func (s *AssetService) diffAsset(old, new *model.Asset) []model.AssetChange {
	var changes []model.AssetChange
	check := func(field, oldVal, newVal string) {
		if oldVal != newVal {
			changes = append(changes, model.AssetChange{Field: field, OldValue: oldVal, NewValue: newVal})
		}
	}
	check("hostname", old.Hostname, new.Hostname)
	check("ip", old.IP, new.IP)
	check("inner_ip", old.InnerIP, new.InnerIP)
	check("os", old.OS, new.OS)
	check("os_version", old.OSVersion, new.OSVersion)
	check("status", old.Status, new.Status)
	check("asset_type", old.AssetType, new.AssetType)
	check("idc", old.IDC, new.IDC)
	check("sn", old.SN, new.SN)
	check("remark", old.Remark, new.Remark)
	if old.CPUCores != new.CPUCores {
		changes = append(changes, model.AssetChange{Field: "cpu_cores", OldValue: strconv.Itoa(old.CPUCores), NewValue: strconv.Itoa(new.CPUCores)})
	}
	if old.MemoryMB != new.MemoryMB {
		changes = append(changes, model.AssetChange{Field: "memory_mb", OldValue: strconv.Itoa(old.MemoryMB), NewValue: strconv.Itoa(new.MemoryMB)})
	}
	if old.DiskGB != new.DiskGB {
		changes = append(changes, model.AssetChange{Field: "disk_gb", OldValue: strconv.Itoa(old.DiskGB), NewValue: strconv.Itoa(new.DiskGB)})
	}
	if old.ServiceTreeID != new.ServiceTreeID {
		changes = append(changes, model.AssetChange{Field: "service_tree_id", OldValue: strconv.FormatInt(old.ServiceTreeID, 10), NewValue: strconv.FormatInt(new.ServiceTreeID, 10)})
	}
	check("owner_ids", old.OwnerIDs, new.OwnerIDs)
	return changes
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
	s.fillOwnerNames(asset)
	return asset, nil
}

func (s *AssetService) List(q repository.AssetListQuery) ([]*model.Asset, int64, error) {
	assets, total, err := s.assetRepo.List(q)
	if err != nil {
		return nil, 0, err
	}
	for _, a := range assets {
		s.fillServiceTreeName(a)
		s.fillOwnerNames(a)
	}
	return assets, total, nil
}

func (s *AssetService) fillServiceTreeName(asset *model.Asset) {
	if asset.ServiceTreeID <= 0 {
		return
	}
	node, err := s.serviceTreeRepo.GetByID(asset.ServiceTreeID)
	if err != nil {
		return
	}
	asset.ServiceTreeName = node.Name

	// 向上追溯构建完整路径: "根 / 父 / 子"
	var parts []string
	current := node
	for current != nil {
		parts = append(parts, current.Name)
		if current.ParentID <= 0 {
			break
		}
		parent, err := s.serviceTreeRepo.GetByID(current.ParentID)
		if err != nil {
			break
		}
		current = parent
	}
	// 反转: parts 现在是 [子, 父, 根]，需要变成 [根, 父, 子]
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	asset.ServiceTreePath = joinPath(parts)
}

func joinPath(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " / "
		}
		result += p
	}
	return result
}

func (s *AssetService) fillOwnerNames(asset *model.Asset) {
	if asset.OwnerIDs == "" || asset.OwnerIDs == "[]" {
		return
	}
	var ids []int64
	json.Unmarshal([]byte(asset.OwnerIDs), &ids)
	if len(ids) == 0 {
		return
	}
	userRepo := repository.NewUserRepository()
	nameMap := userRepo.GetNamesByIDs(ids)
	for _, id := range ids {
		if name, ok := nameMap[id]; ok {
			asset.OwnerNames = append(asset.OwnerNames, name)
		}
	}
}
