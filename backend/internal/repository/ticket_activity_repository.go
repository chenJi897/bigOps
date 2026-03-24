package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type TicketActivityRepository struct{}

func NewTicketActivityRepository() *TicketActivityRepository {
	return &TicketActivityRepository{}
}

func (r *TicketActivityRepository) Create(a *model.TicketActivity) error {
	return database.GetDB().Create(a).Error
}

func (r *TicketActivityRepository) ListByTicketID(ticketID int64, page, size int) ([]*model.TicketActivity, int64, error) {
	var items []*model.TicketActivity
	var total int64
	db := database.GetDB().Model(&model.TicketActivity{}).Where("ticket_id = ?", ticketID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("id ASC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
