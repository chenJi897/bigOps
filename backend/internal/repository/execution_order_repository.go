package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type ExecutionOrderRepository struct{}

func NewExecutionOrderRepository() *ExecutionOrderRepository {
	return &ExecutionOrderRepository{}
}

func (r *ExecutionOrderRepository) Create(item *model.ExecutionOrder) error {
	return database.GetDB().Create(item).Error
}

func (r *ExecutionOrderRepository) Update(item *model.ExecutionOrder) error {
	return database.GetDB().Save(item).Error
}

func (r *ExecutionOrderRepository) GetByID(id int64) (*model.ExecutionOrder, error) {
	var item model.ExecutionOrder
	if err := database.GetDB().First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ExecutionOrderRepository) ListByTicketID(ticketID int64) ([]*model.ExecutionOrder, error) {
	var items []*model.ExecutionOrder
	if err := database.GetDB().Where("ticket_id = ?", ticketID).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
