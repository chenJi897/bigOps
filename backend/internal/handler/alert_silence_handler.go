package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

type AlertSilenceHandler struct {
	svc *service.AlertSilenceService
}

func NewAlertSilenceHandler() *AlertSilenceHandler {
	return &AlertSilenceHandler{svc: service.NewAlertSilenceService()}
}

// upsertAlertSilenceRequest 用于创建/更新告警静默记录。
type upsertAlertSilenceRequest struct {
	// Name 静默规则名称。
	Name string `json:"name" binding:"required"`
	// RuleID 作用到的告警规则 ID。
	RuleID int64 `json:"rule_id"`
	// AgentID 作用到的 agent ID。
	AgentID string `json:"agent_id"`
	// ServiceTreeID 作用到的服务树节点 ID。
	ServiceTreeID int64 `json:"service_tree_id"`
	// OwnerID 作用到的负责人 ID。
	OwnerID int64 `json:"owner_id"`
	// Reason 静默原因。
	Reason string `json:"reason"`
	// Enabled 是否启用（1/0）。
	Enabled int8 `json:"enabled"`
	// StartsAt 开始时间（2006-01-02 15:04:05）。
	StartsAt string `json:"starts_at"`
	// EndsAt 结束时间（2006-01-02 15:04:05）。
	EndsAt string `json:"ends_at"`
}

// List 返回所有告警静默。
// @Summary 告警静默列表
// @Description 查询全部静默规则
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]model.AlertSilence}
// @Router /alert-silences [get]
func (h *AlertSilenceHandler) List(c *gin.Context) {
	items, err := h.svc.List()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

// Create 新建告警静默。
// @Summary 创建静默规则
// @Description 需要管理员权限
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Param body body upsertAlertSilenceRequest true "静默规则"
// @Success 200 {object} response.Response{data=model.AlertSilence}
// @Router /alert-silences [post]
func (h *AlertSilenceHandler) Create(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var req upsertAlertSilenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	item, err := buildAlertSilenceModel(req)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	userID, _ := c.Get("userID")
	item.CreatedBy, _ = userID.(int64)
	item.UpdatedBy = item.CreatedBy
	if err := h.svc.Create(item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建告警静默", zap.String("operator", c.GetString("username")), zap.Int64("silence_id", item.ID), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "创建成功", item)
}

// Update 修改告警静默。
// @Summary 更新静默规则
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Param id path int true "静默规则 ID"
// @Param body body upsertAlertSilenceRequest true "静默规则"
// @Success 200 {object} response.Response{data=model.AlertSilence}
// @Router /alert-silences/{id} [post]
func (h *AlertSilenceHandler) Update(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req upsertAlertSilenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	item, err := buildAlertSilenceModel(req)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	item.ID = id
	userID, _ := c.Get("userID")
	item.UpdatedBy, _ = userID.(int64)
	if err := h.svc.Update(item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新告警静默", zap.String("operator", c.GetString("username")), zap.Int64("silence_id", id), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "更新成功", item)
}

// Delete 删除静默规则。
// @Summary 删除静默规则
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Param id path int true "静默规则 ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response "删除失败"
// @Router /alert-silences/{id}/delete [post]
func (h *AlertSilenceHandler) Delete(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除告警静默", zap.String("operator", c.GetString("username")), zap.Int64("silence_id", id), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "删除成功", nil)
}

func buildAlertSilenceModel(req upsertAlertSilenceRequest) (*model.AlertSilence, error) {
	start, err := time.ParseInLocation("2006-01-02 15:04:05", req.StartsAt, time.Local)
	if err != nil {
		return nil, err
	}
	end, err := time.ParseInLocation("2006-01-02 15:04:05", req.EndsAt, time.Local)
	if err != nil {
		return nil, err
	}
	return &model.AlertSilence{
		Name:          req.Name,
		RuleID:        req.RuleID,
		AgentID:       req.AgentID,
		ServiceTreeID: req.ServiceTreeID,
		OwnerID:       req.OwnerID,
		Reason:        req.Reason,
		Enabled:       req.Enabled,
		StartsAt:      model.LocalTime(start),
		EndsAt:        model.LocalTime(end),
	}, nil
}
