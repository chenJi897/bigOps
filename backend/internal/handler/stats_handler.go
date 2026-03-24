package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/pkg/database"
	"github.com/bigops/platform/internal/pkg/response"
)

type StatsHandler struct{}

func NewStatsHandler() *StatsHandler {
	return &StatsHandler{}
}

type SummaryResponse struct {
	AssetTotal         int64 `json:"asset_total"`
	AssetOnline        int64 `json:"asset_online"`
	AssetOffline       int64 `json:"asset_offline"`
	CloudAccountTotal  int64 `json:"cloud_account_total"`
	CloudAccountFailed int64 `json:"cloud_account_failed"` // last_sync_status=failed
	ServiceTreeTotal   int64 `json:"service_tree_total"`
	UserTotal          int64 `json:"user_total"`
	DepartmentTotal    int64 `json:"department_total"`
	TicketOpen         int64 `json:"ticket_open"`
	TicketTotal        int64 `json:"ticket_total"`
}

// Summary 平台摘要统计。
// @Summary 平台摘要
// @Description 一次返回首页所需的所有统计数字
// @Tags 统计
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=SummaryResponse}
// @Router /stats/summary [get]
func (h *StatsHandler) Summary(c *gin.Context) {
	db := database.GetDB()
	var s SummaryResponse

	db.Table("assets").Where("deleted_at IS NULL").Count(&s.AssetTotal)
	db.Table("assets").Where("deleted_at IS NULL AND status = 'online'").Count(&s.AssetOnline)
	db.Table("assets").Where("deleted_at IS NULL AND status = 'offline'").Count(&s.AssetOffline)
	db.Table("cloud_accounts").Where("deleted_at IS NULL").Count(&s.CloudAccountTotal)
	db.Table("cloud_accounts").Where("deleted_at IS NULL AND last_sync_status = 'failed'").Count(&s.CloudAccountFailed)
	db.Table("service_trees").Where("deleted_at IS NULL").Count(&s.ServiceTreeTotal)
	db.Table("users").Where("deleted_at IS NULL").Count(&s.UserTotal)
	db.Table("departments").Where("deleted_at IS NULL").Count(&s.DepartmentTotal)
	db.Table("tickets").Where("deleted_at IS NULL").Count(&s.TicketTotal)
	db.Table("tickets").Where("deleted_at IS NULL AND status IN ('open','processing')").Count(&s.TicketOpen)

	response.Success(c, s)
}

type DistItem struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type TopItem struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type AssetDistributionResponse struct {
	StatusDist  []DistItem `json:"status_dist"`
	SourceDist  []DistItem `json:"source_dist"`
	TopServices []TopItem  `json:"top_services"`
}

// AssetDistribution 资产分布统计。
// @Summary 资产分布
// @Description 资产状态分布、来源分布、服务树资产 Top 10
// @Tags 统计
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=AssetDistributionResponse}
// @Router /stats/asset-distribution [get]
func (h *StatsHandler) AssetDistribution(c *gin.Context) {
	db := database.GetDB()
	var resp AssetDistributionResponse

	// 状态分布
	var statusRows []struct {
		Status string `gorm:"column:status"`
		Cnt    int64  `gorm:"column:cnt"`
	}
	db.Table("assets").Select("status, COUNT(*) as cnt").
		Where("deleted_at IS NULL").Group("status").Find(&statusRows)
	for _, r := range statusRows {
		resp.StatusDist = append(resp.StatusDist, DistItem{Label: r.Status, Count: r.Cnt})
	}

	// 来源分布
	var sourceRows []struct {
		Source string `gorm:"column:source"`
		Cnt    int64  `gorm:"column:cnt"`
	}
	db.Table("assets").Select("source, COUNT(*) as cnt").
		Where("deleted_at IS NULL").Group("source").Find(&sourceRows)
	for _, r := range sourceRows {
		resp.SourceDist = append(resp.SourceDist, DistItem{Label: r.Source, Count: r.Cnt})
	}

	// 服务树资产 Top 10
	var topRows []struct {
		ServiceTreeID int64  `gorm:"column:service_tree_id"`
		Cnt           int64  `gorm:"column:cnt"`
	}
	db.Table("assets").Select("service_tree_id, COUNT(*) as cnt").
		Where("deleted_at IS NULL AND service_tree_id > 0").
		Group("service_tree_id").Order("cnt DESC").Limit(10).Find(&topRows)
	for _, r := range topRows {
		item := TopItem{ID: r.ServiceTreeID, Count: r.Cnt}
		// 查节点名称
		var name string
		db.Table("service_trees").Select("name").Where("id = ?", r.ServiceTreeID).Scan(&name)
		item.Name = name
		resp.TopServices = append(resp.TopServices, item)
	}

	response.Success(c, resp)
}
