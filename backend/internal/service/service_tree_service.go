// backend/internal/service/service_tree_service.go
package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type ServiceTreeService struct {
	repo      *repository.ServiceTreeRepository
	assetRepo *repository.AssetRepository
}

func NewServiceTreeService() *ServiceTreeService {
	return &ServiceTreeService{
		repo:      repository.NewServiceTreeRepository(),
		assetRepo: repository.NewAssetRepository(),
	}
}

func (s *ServiceTreeService) Create(node *model.ServiceTree) error {
	// 校验 code 唯一
	if node.Code != "" {
		_, err := s.repo.GetByCode(node.Code)
		if err == nil {
			return errors.New("节点编码已存在")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("查询失败: %w", err)
		}
	}
	// 如果有父节点，计算 level
	if node.ParentID > 0 {
		parent, err := s.repo.GetByID(node.ParentID)
		if err != nil {
			return errors.New("父节点不存在")
		}
		node.Level = parent.Level + 1
	} else {
		node.Level = 1
	}
	return s.repo.Create(node)
}

func (s *ServiceTreeService) Update(node *model.ServiceTree) error {
	existing, err := s.repo.GetByID(node.ID)
	if err != nil {
		return errors.New("节点不存在")
	}
	existing.Name = node.Name
	existing.Code = node.Code
	existing.Description = node.Description
	existing.Sort = node.Sort
	existing.OwnerID = node.OwnerID
	return s.repo.Update(existing)
}

func (s *ServiceTreeService) Delete(id int64) error {
	hasChildren, err := s.repo.HasChildren(id)
	if err != nil {
		return fmt.Errorf("查询子节点失败: %w", err)
	}
	if hasChildren {
		return errors.New("存在子节点，不允许删除")
	}
	// 校验是否有关联资产
	assetCount, err := s.assetRepo.CountByServiceTreeID(id)
	if err != nil {
		return fmt.Errorf("查询关联资产失败: %w", err)
	}
	if assetCount > 0 {
		return fmt.Errorf("该节点下有 %d 个关联资产，请先移除或迁移后再删除", assetCount)
	}
	return s.repo.Delete(id)
}

func (s *ServiceTreeService) GetByID(id int64) (*model.ServiceTree, error) {
	return s.repo.GetByID(id)
}

func (s *ServiceTreeService) GetTree() ([]*model.ServiceTree, error) {
	nodes, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}
	return buildServiceTree(nodes, 0), nil
}

func (s *ServiceTreeService) Move(id, newParentID int64) error {
	node, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("节点不存在")
	}
	if newParentID == id {
		return errors.New("不能移动到自身")
	}
	newLevel := 1
	if newParentID > 0 {
		parent, err := s.repo.GetByID(newParentID)
		if err != nil {
			return errors.New("目标父节点不存在")
		}
		newLevel = parent.Level + 1

		// 防环校验：目标父节点不能是当前节点的后代
		descendants, err := s.repo.GetAllDescendantIDs(id)
		if err == nil {
			for _, did := range descendants {
				if did == newParentID {
					return errors.New("不能移动到自身的子节点下，会形成环")
				}
			}
		}
	}
	node.ParentID = newParentID
	node.Level = newLevel
	return s.repo.Update(node)
}

func buildServiceTree(nodes []*model.ServiceTree, parentID int64) []*model.ServiceTree {
	var tree []*model.ServiceTree
	for _, node := range nodes {
		if node.ParentID == parentID {
			children := buildServiceTree(nodes, node.ID)
			if len(children) > 0 {
				childSlice := make([]model.ServiceTree, len(children))
				for i, c := range children {
					childSlice[i] = *c
				}
				node.Children = childSlice
			}
			tree = append(tree, node)
		}
	}
	return tree
}
