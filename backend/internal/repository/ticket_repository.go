package repository

import (
	"fmt"
	"time"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type TicketRepository struct{}

func NewTicketRepository() *TicketRepository {
	return &TicketRepository{}
}

func (r *TicketRepository) Create(t *model.Ticket) error {
	return database.GetDB().Create(t).Error
}

func (r *TicketRepository) Update(t *model.Ticket) error {
	return database.GetDB().Save(t).Error
}

func (r *TicketRepository) GetByID(id int64) (*model.Ticket, error) {
	var t model.Ticket
	if err := database.GetDB().First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TicketRepository) GetByTicketNo(no string) (*model.Ticket, error) {
	var t model.Ticket
	if err := database.GetDB().Where("ticket_no = ?", no).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TicketRepository) GetByDedupeKey(key string) (*model.Ticket, error) {
	var t model.Ticket
	if err := database.GetDB().Where("dedupe_key = ? AND status != 'closed'", key).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// GetNextSeq 获取当天工单序号。
func (r *TicketRepository) GetNextSeq(date string) int {
	var maxNo string
	prefix := fmt.Sprintf("TK-%s-", date)
	database.GetDB().Model(&model.Ticket{}).
		Where("ticket_no LIKE ?", prefix+"%").
		Select("MAX(ticket_no)").Scan(&maxNo)
	if maxNo == "" {
		return 1
	}
	// 解析末尾4位
	var seq int
	fmt.Sscanf(maxNo, "TK-"+date+"-%04d", &seq)
	return seq + 1
}

// GenerateTicketNo 生成工单编号。
func (r *TicketRepository) GenerateTicketNo() string {
	date := time.Now().Format("20060102")
	seq := r.GetNextSeq(date)
	return fmt.Sprintf("TK-%s-%04d", date, seq)
}

type TicketListQuery struct {
	Page          int
	Size          int
	Status        string
	Priority      string
	TypeID        int64
	Source        string
	ResourceType  string
	CreatorID     int64
	AssigneeID    int64
	HandleDeptID  int64
	ServiceTreeID int64
	Keyword       string
	// scope: my_created/my_assigned/my_dept/all
	Scope         string
	CurrentUserID int64
	CurrentDeptID int64
}

func (r *TicketRepository) List(q TicketListQuery) ([]*model.Ticket, int64, error) {
	var items []*model.Ticket
	var total int64
	db := database.GetDB().Model(&model.Ticket{})

	// scope 处理
	switch q.Scope {
	case "my_created":
		db = db.Where("creator_id = ?", q.CurrentUserID)
	case "my_assigned":
		db = db.Where("assignee_id = ?", q.CurrentUserID)
	case "my_dept":
		if q.CurrentDeptID > 0 {
			db = db.Where("handle_dept_id = ? OR submit_dept_id = ?", q.CurrentDeptID, q.CurrentDeptID)
		}
	}

	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.Priority != "" {
		db = db.Where("priority = ?", q.Priority)
	}
	if q.TypeID > 0 {
		db = db.Where("type_id = ?", q.TypeID)
	}
	if q.Source != "" {
		db = db.Where("source = ?", q.Source)
	}
	if q.ResourceType != "" {
		db = db.Where("resource_type = ?", q.ResourceType)
	}
	if q.CreatorID > 0 {
		db = db.Where("creator_id = ?", q.CreatorID)
	}
	if q.AssigneeID > 0 {
		db = db.Where("assignee_id = ?", q.AssigneeID)
	}
	if q.HandleDeptID > 0 {
		db = db.Where("handle_dept_id = ?", q.HandleDeptID)
	}
	if q.ServiceTreeID > 0 {
		db = db.Where("service_tree_id = ?", q.ServiceTreeID)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		db = db.Where("title LIKE ? OR ticket_no LIKE ? OR description LIKE ?", like, like, like)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (q.Page - 1) * q.Size
	if err := db.Offset(offset).Limit(q.Size).Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
