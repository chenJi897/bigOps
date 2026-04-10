package handler

import (
	"encoding/json"
	"net/url"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
	"github.com/gin-gonic/gin"
)

type InspectionHandler struct {
	svc *service.InspectionService
}

func NewInspectionHandler() *InspectionHandler {
	return &InspectionHandler{svc: service.NewInspectionService()}
}

type upsertInspectionTemplateRequest struct {
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description"`
	TaskID       int64    `json:"task_id" binding:"required"`
	DefaultHosts []string `json:"default_hosts"`
	Enabled      int8     `json:"enabled"`
}

type upsertInspectionPlanRequest struct {
	Name       string `json:"name" binding:"required"`
	TemplateID int64  `json:"template_id" binding:"required"`
	CronExpr   string `json:"cron_expr" binding:"required"`
	Enabled    int8   `json:"enabled"`
}

func (h *InspectionHandler) ListTemplates(c *gin.Context) {
	page, size := parsePageSize(c)
	items, total, err := h.svc.ListTemplates(page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

func (h *InspectionHandler) CreateTemplate(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var req upsertInspectionTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	hostsPayload, _ := json.Marshal(req.DefaultHosts)
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	item := &model.InspectionTemplate{
		Name:         req.Name,
		Description:  req.Description,
		TaskID:       req.TaskID,
		DefaultHosts: string(hostsPayload),
		Enabled:      req.Enabled,
		CreatedBy:    currentUserID,
		UpdatedBy:    currentUserID,
	}
	if err := h.svc.UpsertTemplate(0, item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.Success(c, item)
}

func (h *InspectionHandler) UpdateTemplate(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req upsertInspectionTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	hostsPayload, _ := json.Marshal(req.DefaultHosts)
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	item := &model.InspectionTemplate{
		Name:         req.Name,
		Description:  req.Description,
		TaskID:       req.TaskID,
		DefaultHosts: string(hostsPayload),
		Enabled:      req.Enabled,
		UpdatedBy:    currentUserID,
	}
	if err := h.svc.UpsertTemplate(id, item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "更新成功", nil)
}

func (h *InspectionHandler) ListPlans(c *gin.Context) {
	page, size := parsePageSize(c)
	items, total, err := h.svc.ListPlans(page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

func (h *InspectionHandler) CreatePlan(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var req upsertInspectionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	item := &model.InspectionPlan{
		Name:       req.Name,
		TemplateID: req.TemplateID,
		CronExpr:   req.CronExpr,
		Enabled:    req.Enabled,
		CreatedBy:  currentUserID,
		UpdatedBy:  currentUserID,
	}
	if err := h.svc.UpsertPlan(0, item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.Success(c, item)
}

func (h *InspectionHandler) UpdatePlan(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req upsertInspectionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	item := &model.InspectionPlan{
		Name:       req.Name,
		TemplateID: req.TemplateID,
		CronExpr:   req.CronExpr,
		Enabled:    req.Enabled,
		UpdatedBy:  currentUserID,
	}
	if err := h.svc.UpsertPlan(id, item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "更新成功", nil)
}

func (h *InspectionHandler) ExecutePlan(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	record, err := h.svc.ExecutePlan(id)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "执行已发起", record)
}

func (h *InspectionHandler) ListRecords(c *gin.Context) {
	page, size := parsePageSize(c)
	items, total, err := h.svc.ListRecords(page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

func (h *InspectionHandler) GetRecordReport(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	data, err := h.svc.GetRecordReport(id)
	if err != nil {
		response.Error(c, 404, "记录不存在")
		return
	}
	response.Success(c, data)
}

func (h *InspectionHandler) TemplateTrend(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	data, err := h.svc.TemplateTrend(id)
	if err != nil {
		response.Error(c, 404, "模板不存在")
		return
	}
	response.Success(c, data)
}

func (h *InspectionHandler) ExportRecordReport(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	format := c.DefaultQuery("format", "json")
	payload, contentType, filename, err := h.svc.ExportRecordReport(id, format)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", `attachment; filename="`+url.PathEscape(filename)+`"`)
	c.Data(200, contentType, payload)
}
