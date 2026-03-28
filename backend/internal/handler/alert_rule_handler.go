package handler

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/repository"
	"github.com/bigops/platform/internal/service"
)

type AlertRuleHandler struct {
	alertSvc *service.AlertService
}

func NewAlertRuleHandler() *AlertRuleHandler {
	return &AlertRuleHandler{alertSvc: service.NewAlertService()}
}

type UpsertAlertRuleRequest struct {
	Name          string  `json:"name" binding:"required"`
	MetricType    string  `json:"metric_type" binding:"required"`
	Operator      string  `json:"operator"`
	Threshold     float64 `json:"threshold"`
	Severity      string  `json:"severity"`
	Enabled       int8    `json:"enabled"`
	Description   string  `json:"description"`
	NotifyUserIDs []int64 `json:"notify_user_ids"`
	NotifyChannels []string `json:"notify_channels"`
	Action        string  `json:"action"`
	RepairTaskID  int64   `json:"repair_task_id"`
	TicketTypeID  int64   `json:"ticket_type_id"`
	OnCallScheduleID int64 `json:"oncall_schedule_id"`
	ServiceTreeID int64   `json:"service_tree_id"`
	OwnerID       int64   `json:"owner_id"`
}

type AlertEventStatusRequest struct {
	Note string `json:"note"`
}

func (h *AlertRuleHandler) List(c *gin.Context) {
	page, size := parsePageSize(c)
	var enabled *int8
	if c.Query("enabled") != "" {
		value, _ := strconv.ParseInt(c.Query("enabled"), 10, 8)
		parsed := int8(value)
		enabled = &parsed
	}
	items, total, err := h.alertSvc.ListRules(repository.AlertRuleListQuery{
		Page:       page,
		Size:       size,
		Keyword:    c.Query("keyword"),
		MetricType: c.Query("metric_type"),
		Severity:   c.Query("severity"),
		Enabled:    enabled,
	})
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

func (h *AlertRuleHandler) Create(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var req UpsertAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	payload, _ := json.Marshal(req.NotifyUserIDs)
	channelsPayload, _ := json.Marshal(req.NotifyChannels)
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	item := &model.AlertRule{
		Name:          req.Name,
		MetricType:    req.MetricType,
		Operator:      req.Operator,
		Threshold:     req.Threshold,
		Severity:      req.Severity,
		Enabled:       req.Enabled,
		Description:   req.Description,
		NotifyUserIDs: string(payload),
		NotifyChannels: string(channelsPayload),
		Action:        req.Action,
		RepairTaskID:  req.RepairTaskID,
		TicketTypeID:  req.TicketTypeID,
		OnCallScheduleID: req.OnCallScheduleID,
		ServiceTreeID: req.ServiceTreeID,
		OwnerID:       req.OwnerID,
		CreatedBy:     currentUserID,
		UpdatedBy:     currentUserID,
	}
	if item.Operator == "" {
		item.Operator = "gt"
	}
	if item.Severity == "" {
		item.Severity = "warning"
	}
	if err := h.alertSvc.CreateRule(item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "创建成功", item)
}

func (h *AlertRuleHandler) Update(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpsertAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	payload, _ := json.Marshal(req.NotifyUserIDs)
	channelsPayload, _ := json.Marshal(req.NotifyChannels)
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	item := &model.AlertRule{
		Name:          req.Name,
		MetricType:    req.MetricType,
		Operator:      req.Operator,
		Threshold:     req.Threshold,
		Severity:      req.Severity,
		Enabled:       req.Enabled,
		Description:   req.Description,
		NotifyUserIDs: string(payload),
		NotifyChannels: string(channelsPayload),
		Action:        req.Action,
		RepairTaskID:  req.RepairTaskID,
		TicketTypeID:  req.TicketTypeID,
		OnCallScheduleID: req.OnCallScheduleID,
		ServiceTreeID: req.ServiceTreeID,
		OwnerID:       req.OwnerID,
		UpdatedBy:     currentUserID,
	}
	if err := h.alertSvc.UpdateRule(id, item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "更新成功", nil)
}

func (h *AlertRuleHandler) Delete(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.alertSvc.DeleteRule(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "删除成功", nil)
}

func (h *AlertRuleHandler) Events(c *gin.Context) {
	page, size := parsePageSize(c)
	var ruleID *int64
	if c.Query("rule_id") != "" {
		if parsed, err := strconv.ParseInt(c.Query("rule_id"), 10, 64); err == nil {
			ruleID = &parsed
		}
	}
	items, total, err := h.alertSvc.ListEvents(repository.AlertEventListQuery{
		Page:     page,
		Size:     size,
		Status:   c.Query("status"),
		Severity: c.Query("severity"),
		AgentID:  c.Query("agent_id"),
		Keyword:  c.Query("keyword"),
		RuleID:   ruleID,
	})
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

func (h *AlertRuleHandler) GetEvent(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	event, err := h.alertSvc.GetEvent(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "事件不存在")
			return
		}
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, event)
}

func (h *AlertRuleHandler) AcknowledgeEvent(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req AlertEventStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	if err := h.alertSvc.AcknowledgeEvent(id, currentUserID, req.Note); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "事件不存在")
			return
		}
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "事件已确认", nil)
}

func (h *AlertRuleHandler) ResolveEvent(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req AlertEventStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	if err := h.alertSvc.ResolveEvent(id, currentUserID, req.Note); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "事件不存在")
			return
		}
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "事件已关闭", nil)
}

func (h *AlertRuleHandler) Evaluate(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	summary, err := h.alertSvc.EvaluateAll()
	if err != nil {
		response.InternalServerError(c, "执行巡检失败")
		return
	}
	response.SuccessWithMessage(c, "巡检完成", summary)
}
