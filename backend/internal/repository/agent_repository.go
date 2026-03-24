package repository

import (
	"time"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type AgentRepository struct{}

func NewAgentRepository() *AgentRepository {
	return &AgentRepository{}
}

func (r *AgentRepository) Upsert(a *model.AgentInfo) error {
	var existing model.AgentInfo
	err := database.GetDB().Where("agent_id = ?", a.AgentID).First(&existing).Error
	if err != nil {
		// Not found, create
		return database.GetDB().Create(a).Error
	}
	// Update existing
	existing.Hostname = a.Hostname
	existing.IP = a.IP
	existing.Version = a.Version
	existing.OS = a.OS
	existing.Status = a.Status
	existing.Labels = a.Labels
	existing.CPUCount = a.CPUCount
	existing.MemoryTotal = a.MemoryTotal
	existing.LastHeartbeat = a.LastHeartbeat
	return database.GetDB().Save(&existing).Error
}

func (r *AgentRepository) GetByAgentID(agentID string) (*model.AgentInfo, error) {
	var a model.AgentInfo
	if err := database.GetDB().Where("agent_id = ?", agentID).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AgentRepository) GetByIP(ip string) (*model.AgentInfo, error) {
	var a model.AgentInfo
	if err := database.GetDB().Where("ip = ?", ip).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AgentRepository) List(page, size int, status string) ([]*model.AgentInfo, int64, error) {
	var items []*model.AgentInfo
	var total int64
	db := database.GetDB().Model(&model.AgentInfo{})
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("status DESC, last_heartbeat DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *AgentRepository) UpdateStatus(agentID string, status string) error {
	return database.GetDB().Model(&model.AgentInfo{}).Where("agent_id = ?", agentID).Update("status", status).Error
}

func (r *AgentRepository) ListOnline() ([]*model.AgentInfo, error) {
	var items []*model.AgentInfo
	if err := database.GetDB().Where("status = ?", "online").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// MarkStaleOffline marks agents offline if they haven't sent heartbeat in the given duration.
func (r *AgentRepository) MarkStaleOffline(staleDuration time.Duration) error {
	threshold := time.Now().Add(-staleDuration)
	return database.GetDB().Model(&model.AgentInfo{}).
		Where("status = ? AND last_heartbeat < ?", "online", threshold).
		Update("status", "offline").Error
}
