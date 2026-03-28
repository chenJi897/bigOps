package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

type MonitorHandler struct {
	monitorSvc *service.MonitorService
}

func NewMonitorHandler() *MonitorHandler {
	return &MonitorHandler{monitorSvc: service.NewMonitorService()}
}

// monitorQueryRequest 表示一次 Prometheus 即时查询的参数。
type monitorQueryRequest struct {
	// DatasourceID 目标数据源 ID。
	DatasourceID int64 `json:"datasource_id" binding:"required"`
	// Query PromQL 表达式。
	Query string `json:"query" binding:"required"`
	// Time 查询时间戳，RFC3339 格式，默认为当前时间。
	Time string `json:"time"`
}

// monitorRangeRequest 表示一次 Prometheus 范围查询的参数。
type monitorRangeRequest struct {
	// DatasourceID 目标数据源 ID。
	DatasourceID int64 `json:"datasource_id" binding:"required"`
	// Query PromQL 表达式。
	Query string `json:"query" binding:"required"`
	// Start 起始时间，RFC3339 格式。
	Start string `json:"start"`
	// End 结束时间，RFC3339 格式。
	End string `json:"end"`
	// Step 查询步长，支持时间间隔（例如 30s、1m）。
	Step string `json:"step"`
}

// Summary 汇总监控维度。
// @Summary 监控概览
// @Description 返回 agent、告警、规则等概览数据
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=service.MonitorSummary}
// @Router /monitor/summary [get]
func (h *MonitorHandler) Summary(c *gin.Context) {
	data, err := h.monitorSvc.Summary()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, data)
}

// Agents 查询 agent 列表。
// @Summary agent 列表
// @Description 支持状态筛选与关键字搜索
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param status query string false "状态" Enums(online,offline)
// @Param keyword query string false "关键字（主机名/IP）"
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.AgentInfo}}
// @Router /monitor/agents [get]
func (h *MonitorHandler) Agents(c *gin.Context) {
	page, size := parsePageSize(c)
	items, total, err := h.monitorSvc.ListAgents(page, size, c.Query("status"), c.Query("keyword"))
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

// AgentTrend 查询 agent 指标趋势数据。
// @Summary agent 指标趋势
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Param agent_id path string true "agent ID"
// @Param metric_type query string true "指标类型"
// @Param minutes query int false "回溯分钟数" default(60)
// @Param limit query int false "最大采样点" default(120)
// @Success 200 {object} response.Response{data=[]model.AgentMetricSample}
// @Router /monitor/agents/{agent_id}/trends [get]
func (h *MonitorHandler) AgentTrend(c *gin.Context) {
	minutes, _ := strconv.Atoi(c.DefaultQuery("minutes", "60"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "120"))
	items, err := h.monitorSvc.AgentTrend(c.Param("agent_id"), c.Query("metric_type"), minutes, limit)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.Success(c, items)
}

// AggregateServiceTrees 统计每个服务树下的 agent 分布与平均指标。
// @Summary 服务树聚合
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]service.MonitorAggregateItem}
// @Router /monitor/aggregates/service-trees [get]
func (h *MonitorHandler) AggregateServiceTrees(c *gin.Context) {
	items, err := h.monitorSvc.AggregateByServiceTree()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

// AggregateOwners 统计每个负责人下的 agent 聚合指标。
// @Summary 负责人聚合
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]service.MonitorAggregateItem}
// @Router /monitor/aggregates/owners [get]
func (h *MonitorHandler) AggregateOwners(c *gin.Context) {
	items, err := h.monitorSvc.AggregateByOwner()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

// Query 触发 Prometheus 即时查询。
// @Summary Prometheus 即时查询
// @Description 通过指定数据源和 PromQL 返回向量/矩阵数据
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Param body body monitorQueryRequest true "Prometheus 查询参数"
// @Success 200 {object} response.Response{data=map[string]any}
// @Router /monitor/query [post]
func (h *MonitorHandler) Query(c *gin.Context) {
	var req monitorQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	ts := time.Now()
	if req.Time != "" {
		if parsed, err := time.Parse(time.RFC3339, req.Time); err == nil {
			ts = parsed
		}
	}
	result, err := h.monitorSvc.QueryPrometheus(c.Request.Context(), req.DatasourceID, req.Query, ts)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.Success(c, result)
}

// QueryRange 触发 Prometheus 范围查询。
// @Summary Prometheus 范围查询
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Param body body monitorRangeRequest true "Prometheus 查询范围参数"
// @Success 200 {object} response.Response{data=map[string]any}
// @Router /monitor/query-range [post]
func (h *MonitorHandler) QueryRange(c *gin.Context) {
	var req monitorRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	if req.Start != "" {
		if parsed, err := time.Parse(time.RFC3339, req.Start); err == nil {
			start = parsed
		}
	}
	if req.End != "" {
		if parsed, err := time.Parse(time.RFC3339, req.End); err == nil {
			end = parsed
		}
	}
	step := time.Minute
	if req.Step != "" {
		if parsed, err := time.ParseDuration(req.Step); err == nil {
			step = parsed
		}
	}
	result, err := h.monitorSvc.QueryPrometheusRange(c.Request.Context(), req.DatasourceID, req.Query, start, end, step)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.Success(c, result)
}
