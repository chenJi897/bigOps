package handler

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
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
	Name             string                 `json:"name" binding:"required"`
	MetricType       string                 `json:"metric_type" binding:"required"`
	Operator         string                 `json:"operator"`
	Threshold        float64                `json:"threshold"`
	Severity         string                 `json:"severity"`
	Enabled          int8                   `json:"enabled"`
	Description      string                 `json:"description"`
	NotifyUserIDs    []int64                `json:"notify_user_ids"`
	NotifyChannels   []string               `json:"notify_channels"`
	NotifyConfig     map[string]interface{} `json:"notify_config"`
	NotifyGroupID    int64                  `json:"notify_group_id"`
	Action           string                 `json:"action"`
	RepairTaskID     int64                  `json:"repair_task_id"`
	TicketTypeID     int64                  `json:"ticket_type_id"`
	OnCallScheduleID int64                  `json:"oncall_schedule_id"`
	ServiceTreeID    int64                  `json:"service_tree_id"`
	OwnerID          int64                  `json:"owner_id"`
}

type AlertEventStatusRequest struct {
	Note string `json:"note"`
}

// List godoc
// @Summary 获取告警规则列表
// @Tags 告警规则
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Success 200 {object} response.Response{data=[]model.AlertRule}
// @Router /alert-rules [get]
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

// Create godoc
// @Summary 创建告警规则
// @Tags 告警规则
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body UpsertAlertRuleRequest true "告警规则信息"
// @Success 200 {object} response.Response
// @Router /alert-rules [post]
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
	notifyConfigPayload, _ := json.Marshal(req.NotifyConfig)
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	item := &model.AlertRule{
		Name:             req.Name,
		MetricType:       req.MetricType,
		Operator:         req.Operator,
		Threshold:        req.Threshold,
		Severity:         req.Severity,
		Enabled:          req.Enabled,
		Description:      req.Description,
		NotifyUserIDs:    string(payload),
		NotifyChannels:   string(channelsPayload),
		NotifyConfig:     string(notifyConfigPayload),
		NotifyGroupID:    req.NotifyGroupID,
		Action:           req.Action,
		RepairTaskID:     req.RepairTaskID,
		TicketTypeID:     req.TicketTypeID,
		OnCallScheduleID: req.OnCallScheduleID,
		ServiceTreeID:    req.ServiceTreeID,
		OwnerID:          req.OwnerID,
		CreatedBy:        currentUserID,
		UpdatedBy:        currentUserID,
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
	logger.Info("创建告警规则", zap.String("operator", c.GetString("username")), zap.Int64("rule_id", item.ID), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "创建成功", item)
}

// Update godoc
// @Summary 更新告警规则
// @Tags 告警规则
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "规则ID"
// @Param body body UpsertAlertRuleRequest true "告警规则信息"
// @Success 200 {object} response.Response
// @Router /alert-rules/{id} [post]
func (h *AlertRuleHandler) Update(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req UpsertAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	payload, _ := json.Marshal(req.NotifyUserIDs)
	channelsPayload, _ := json.Marshal(req.NotifyChannels)
	notifyConfigPayload, _ := json.Marshal(req.NotifyConfig)
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	item := &model.AlertRule{
		Name:             req.Name,
		MetricType:       req.MetricType,
		Operator:         req.Operator,
		Threshold:        req.Threshold,
		Severity:         req.Severity,
		Enabled:          req.Enabled,
		Description:      req.Description,
		NotifyUserIDs:    string(payload),
		NotifyChannels:   string(channelsPayload),
		NotifyConfig:     string(notifyConfigPayload),
		NotifyGroupID:    req.NotifyGroupID,
		Action:           req.Action,
		RepairTaskID:     req.RepairTaskID,
		TicketTypeID:     req.TicketTypeID,
		OnCallScheduleID: req.OnCallScheduleID,
		ServiceTreeID:    req.ServiceTreeID,
		OwnerID:          req.OwnerID,
		UpdatedBy:        currentUserID,
	}
	if err := h.alertSvc.UpdateRule(id, item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新告警规则", zap.String("operator", c.GetString("username")), zap.Int64("rule_id", id), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete godoc
// @Summary 删除告警规则
// @Tags 告警规则
// @Security BearerAuth
// @Produce json
// @Param id path int true "规则ID"
// @Success 200 {object} response.Response
// @Router /alert-rules/{id}/delete [post]
func (h *AlertRuleHandler) Delete(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.alertSvc.DeleteRule(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除告警规则", zap.String("operator", c.GetString("username")), zap.Int64("rule_id", id), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "删除成功", nil)
}

// Events 告警事件列表。
// @Summary 告警事件列表
// @Tags 告警管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param rule_id query int false "规则ID"
// @Param status query string false "状态"
// @Param severity query string false "严重级别"
// @Param agent_id query string false "Agent ID"
// @Param keyword query string false "关键字"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /alert-events [get]
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

// EventGroups 告警事件收敛分组列表。
// @Summary 告警事件收敛分组
// @Tags 告警管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param status query string false "状态"
// @Param severity query string false "严重级别"
// @Param agent_id query string false "Agent ID"
// @Param keyword query string false "关键字"
// @Param window_minutes query int false "收敛窗口(分钟)" default(5)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /alert-events/groups [get]
func (h *AlertRuleHandler) EventGroups(c *gin.Context) {
	page, size := parsePageSize(c)
	windowMinutes, _ := strconv.Atoi(c.DefaultQuery("window_minutes", "5"))
	items, total, err := h.alertSvc.ListEventGroups(repository.AlertEventListQuery{
		Page:     page,
		Size:     size,
		Status:   c.Query("status"),
		Severity: c.Query("severity"),
		AgentID:  c.Query("agent_id"),
		Keyword:  c.Query("keyword"),
	}, windowMinutes)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

// GetEvent 告警事件详情。
// @Summary 告警事件详情
// @Tags 告警管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "事件ID"
// @Success 200 {object} response.Response
// @Router /alert-events/{id} [get]
func (h *AlertRuleHandler) GetEvent(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
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

// EventTimeline 告警事件时间轴。
// @Summary 告警事件时间轴
// @Tags 告警管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "事件ID"
// @Success 200 {object} response.Response
// @Router /alert-events/{id}/timeline [get]
func (h *AlertRuleHandler) EventTimeline(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	data, err := h.alertSvc.GetEventTimeline(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "事件不存在")
			return
		}
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, data)
}

// EventRootCause 告警根因分析。
// @Summary 告警根因分析
// @Tags 告警管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "事件ID"
// @Success 200 {object} response.Response
// @Router /alert-events/{id}/root-cause [get]
func (h *AlertRuleHandler) EventRootCause(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	data, err := h.alertSvc.AnalyzeRootCause(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "事件不存在")
			return
		}
		response.InternalServerError(c, "分析失败")
		return
	}
	response.Success(c, data)
}

// EventContext 告警详情上下文。
// @Summary 告警详情上下文
// @Tags 告警管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "事件ID"
// @Success 200 {object} response.Response
// @Router /alert-events/{id}/context [get]
func (h *AlertRuleHandler) EventContext(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	data, err := h.alertSvc.GetEventContext(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "事件不存在")
			return
		}
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, data)
}

// AcknowledgeEvent 确认告警事件。
// @Summary 确认告警事件
// @Tags 告警管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "事件ID"
// @Param body body AlertEventStatusRequest true "确认请求"
// @Success 200 {object} response.Response
// @Router /alert-events/{id}/ack [post]
func (h *AlertRuleHandler) AcknowledgeEvent(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
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
	logger.Info("确认告警事件", zap.String("operator", c.GetString("username")), zap.Int64("event_id", id), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "事件已确认", nil)
}

// ResolveEvent 解决告警事件。
// @Summary 解决告警事件
// @Tags 告警管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "事件ID"
// @Param body body AlertEventStatusRequest true "解决请求"
// @Success 200 {object} response.Response
// @Router /alert-events/{id}/resolve [post]
func (h *AlertRuleHandler) ResolveEvent(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
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
	logger.Info("解决告警事件", zap.String("operator", c.GetString("username")), zap.Int64("event_id", id), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "事件已关闭", nil)
}

// CommentEvent adds a comment to an alert event timeline.
func (h *AlertRuleHandler) CommentEvent(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req AlertEventStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	if err := h.alertSvc.CommentEvent(id, currentUserID, req.Note); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "评论已添加", nil)
}

// AssignEvent assigns an alert event to a specific user.
func (h *AlertRuleHandler) AssignEvent(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req struct {
		AssigneeID int64 `json:"assignee_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	if err := h.alertSvc.AssignEvent(id, currentUserID, req.AssigneeID); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "已指派", nil)
}

// TopologyView returns health status of hosts in the same service tree as the alert.
func (h *AlertRuleHandler) TopologyView(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	data, err := h.alertSvc.TopologyView(id)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.Success(c, data)
}

// ChangeRiskAssessment evaluates risk before executing a change.
func (h *AlertRuleHandler) ChangeRiskAssessment(c *gin.Context) {
	var req struct {
		TaskID int64    `json:"task_id" binding:"required"`
		Hosts  []string `json:"hosts" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	data, err := h.alertSvc.AssessChangeRisk(req.TaskID, req.Hosts)
	if err != nil {
		response.InternalServerError(c, "风险评估失败")
		return
	}
	response.Success(c, data)
}

// Evaluate 手动触发告警巡检。
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
