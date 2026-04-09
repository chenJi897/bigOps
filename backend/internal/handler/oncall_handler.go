package handler

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

type OnCallHandler struct {
	svc *service.OnCallService
}

func NewOnCallHandler() *OnCallHandler {
	return &OnCallHandler{svc: service.NewOnCallService()}
}

// upsertOnCallRequest 用于创建/更新值班表。
type upsertOnCallRequest struct {
	Name              string   `json:"name" binding:"required" example:"平台运维值班"`
	Description       string   `json:"description" example:"负责平台监控和告警处理"`
	Timezone          string   `json:"timezone" example:"Asia/Shanghai"`
	UserIDs           []int64  `json:"user_ids" example:"1,2"`
	RotationDays      int      `json:"rotation_days" example:"1"`
	NotifyChannels    []string `json:"notify_channels" example:"in_app,email"`
	EscalationMinutes int      `json:"escalation_minutes" example:"15"`
	Enabled           int8     `json:"enabled" example:"1"`
}

// List 返回全部 OnCall 值班表。
// @Summary OnCall 值班表列表
// @Description 查询全部值班轮转配置
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]model.OnCallSchedule}
// @Failure 500 {object} response.Response "查询失败"
// @Router /oncall-schedules [get]
func (h *OnCallHandler) List(c *gin.Context) {
	items, err := h.svc.List()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

// Create 新建 OnCall 值班表。
// @Summary 创建 OnCall 值班表
// @Description 仅管理员可创建值班轮转配置
// @Tags 监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body upsertOnCallRequest true "值班表请求"
// @Success 200 {object} response.Response{data=model.OnCallSchedule}
// @Failure 400 {object} response.Response "参数错误"
// @Router /oncall-schedules [post]
func (h *OnCallHandler) Create(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var req upsertOnCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	item := buildOnCallModel(req)
	userID, _ := c.Get("userID")
	item.CreatedBy, _ = userID.(int64)
	item.UpdatedBy = item.CreatedBy
	if err := h.svc.Create(item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建值班计划", zap.String("operator", c.GetString("username")), zap.Int64("schedule_id", item.ID), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "创建成功", item)
}

// Update 更新 OnCall 值班表。
// @Summary 更新 OnCall 值班表
// @Tags 监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "值班表 ID"
// @Param body body upsertOnCallRequest true "值班表请求"
// @Success 200 {object} response.Response{data=model.OnCallSchedule}
// @Failure 400 {object} response.Response "参数错误"
// @Router /oncall-schedules/{id} [post]
func (h *OnCallHandler) Update(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req upsertOnCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	item := buildOnCallModel(req)
	item.ID = id
	userID, _ := c.Get("userID")
	item.UpdatedBy, _ = userID.(int64)
	if err := h.svc.Update(item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新值班计划", zap.String("operator", c.GetString("username")), zap.Int64("schedule_id", id), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "更新成功", item)
}

// Delete 删除 OnCall 值班表。
// @Summary 删除 OnCall 值班表
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Param id path int true "值班表 ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response "删除失败"
// @Router /oncall-schedules/{id}/delete [post]
func (h *OnCallHandler) Delete(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除值班计划", zap.String("operator", c.GetString("username")), zap.Int64("schedule_id", id), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "删除成功", nil)
}

func buildOnCallModel(req upsertOnCallRequest) *model.OnCallSchedule {
	usersJSON, _ := json.Marshal(req.UserIDs)
	channelsJSON, _ := json.Marshal(req.NotifyChannels)
	return &model.OnCallSchedule{
		Name:               req.Name,
		Description:        req.Description,
		Timezone:           req.Timezone,
		UsersJSON:          string(usersJSON),
		RotationDays:       req.RotationDays,
		NotifyChannelsJSON: string(channelsJSON),
		EscalationMinutes:  req.EscalationMinutes,
		Enabled:            req.Enabled,
	}
}
