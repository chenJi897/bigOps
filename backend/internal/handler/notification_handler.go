package handler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/config"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

var _ model.InAppNotification // swag

type NotificationHandler struct {
	svc *service.NotificationService
}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{svc: service.NewNotificationService()}
}

type NotificationConfigRequest struct {
	DefaultChannels          []string `json:"default_channels"`
	MaxRetries               int      `json:"max_retries"`
	RetryIntervalSeconds     int      `json:"retry_interval_seconds"`
	RetryScanIntervalSeconds int      `json:"retry_scan_interval_seconds"`
}

// GetConfig 获取通知配置。
// @Summary 获取通知配置
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Router /notifications/config [get]
func (h *NotificationHandler) GetConfig(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	response.Success(c, config.Get().Notification)
}

// UpdateConfig 更新通知配置。
// @Summary 更新通知配置
// @Tags 通知管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body NotificationConfigRequest true "配置请求"
// @Success 200 {object} response.Response
// @Router /notifications/config [post]
func (h *NotificationHandler) UpdateConfig(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var req NotificationConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	cfg := config.NotificationConfig{
		DefaultChannels:          req.DefaultChannels,
		MaxRetries:               req.MaxRetries,
		RetryIntervalSeconds:     req.RetryIntervalSeconds,
		RetryScanIntervalSeconds: req.RetryScanIntervalSeconds,
	}
	if err := config.UpdateNotificationConfig(cfg); err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	logger.Info("更新通知配置", zap.String("operator", c.GetString("username")), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "通知配置已更新", cfg)
}

// ListInApp 站内通知列表。
// @Summary 站内通知列表
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Param unread_only query string false "仅未读" default(0)
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /notifications/in-app [get]
func (h *NotificationHandler) ListInApp(c *gin.Context) {
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	unreadOnly := c.DefaultQuery("unread_only", "0") == "1"
	page, size := parsePageSize(c)
	items, total, err := h.svc.ListInAppByUserID(currentUserID, unreadOnly, page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

// CountUnread 未读通知数量。
// @Summary 未读通知数量
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Router /notifications/in-app/unread-count [get]
func (h *NotificationHandler) CountUnread(c *gin.Context) {
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	count, err := h.svc.CountUnreadByUserID(currentUserID)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, gin.H{"count": count})
}

// MarkRead 标记通知已读。
// @Summary 标记通知已读
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "通知ID"
// @Success 200 {object} response.Response
// @Router /notifications/in-app/{id}/read [post]
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.MarkRead(currentUserID, id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("标记通知已读", zap.String("operator", c.GetString("username")), zap.Int64("notification_id", id), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "已标记为已读", nil)
}

// MarkAllRead 标记全部通知已读。
// @Summary 标记全部通知已读
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Router /notifications/in-app/read-all [post]
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	if err := h.svc.MarkAllRead(currentUserID); err != nil {
		response.InternalServerError(c, "批量已读失败")
		return
	}
	logger.Info("标记全部通知已读", zap.String("operator", c.GetString("username")), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "已全部标记为已读", nil)
}

// ClearRead 清空已读通知。
// @Summary 清空已读通知
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Router /notifications/in-app/clear-read [post]
func (h *NotificationHandler) ClearRead(c *gin.Context) {
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	count, err := h.svc.ClearRead(currentUserID)
	if err != nil {
		response.InternalServerError(c, "清空已读失败")
		return
	}
	logger.Info("清除已读通知", zap.String("operator", c.GetString("username")), zap.Int64("count", count), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "已清空已读通知", gin.H{"count": count})
}

type NotificationTestRequest struct {
	Title    string   `json:"title" binding:"required"`
	Content  string   `json:"content" binding:"required"`
	Channels []string `json:"channels"`
	UserIDs  []int64  `json:"user_ids"`
}

type NotificationPreferenceRequest struct {
	EnabledChannels    []string `json:"enabled_channels"`
	SubscribedBizTypes []string `json:"subscribed_biz_types"`
	// ChannelTargets 表示“个人通道按业务 -> Message Pusher 通道名”的映射
	// 例如：{"alert_event":{"lark":"lark-ops-group-001"},"cicd_pipeline":{"lark":"lark-dev-group-002"}}
	ChannelTargets map[string]map[string]string `json:"channel_targets"`
	Enabled            int8     `json:"enabled"`
}

// GetPreference 获取个人通知偏好。
// @Summary 获取个人通知偏好
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Router /notifications/preferences [get]
func (h *NotificationHandler) GetPreference(c *gin.Context) {
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	item, err := h.svc.GetUserPreference(currentUserID)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.Success(c, item)
}

// UpdatePreference 更新个人通知偏好。
// @Summary 更新个人通知偏好
// @Tags 通知管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body NotificationPreferenceRequest true "偏好设置"
// @Success 200 {object} response.Response
// @Router /notifications/preferences [post]
func (h *NotificationHandler) UpdatePreference(c *gin.Context) {
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	var req NotificationPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	channels, _ := json.Marshal(req.EnabledChannels)
	bizTypes, _ := json.Marshal(req.SubscribedBizTypes)
	channelTargets, _ := json.Marshal(req.ChannelTargets)
	item := &model.NotificationUserSetting{
		UserID:             currentUserID,
		EnabledChannels:    string(channels),
		SubscribedBizTypes: string(bizTypes),
		ChannelTargets:     string(channelTargets),
		Enabled:            req.Enabled,
	}
	if err := h.svc.SaveUserPreference(item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新通知偏好", zap.String("operator", c.GetString("username")), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "个人通知设置已保存", item)
}

// TestSend 发送测试通知。
// @Summary 发送测试通知
// @Tags 通知管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body NotificationTestRequest true "测试请求"
// @Success 200 {object} response.Response
// @Router /notifications/test [post]
func (h *NotificationHandler) TestSend(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var req NotificationTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	userIDs := req.UserIDs
	if len(userIDs) == 0 {
		userIDs = []int64{currentUserID}
	}
	eventID, err := h.svc.Publish(service.NotificationPublishRequest{
		EventType:      "notification_test",
		BizType:        "notification",
		BizID:          currentUserID,
		Title:          req.Title,
		Content:        req.Content,
		Level:          "info",
		UserIDs:        userIDs,
		Channels:       req.Channels,
		SkipPreference: true,
		Payload: map[string]interface{}{
			"title":    req.Title,
			"content":  req.Content,
			"channels": req.Channels,
		},
	})
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("发送测试通知", zap.String("operator", c.GetString("username")), zap.Int64("event_id", eventID), zap.Strings("channels", req.Channels), zap.Int("user_count", len(userIDs)), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "测试消息已发送", gin.H{"event_id": eventID})
}

// ListEvents 通知事件列表。
// @Summary 通知事件列表
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /notifications/events [get]
func (h *NotificationHandler) ListEvents(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	page, size := parsePageSize(c)
	items, total, err := h.svc.ListEvents(page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

// RetryEvent 重试通知事件。
// @Summary 重试通知事件
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "事件ID"
// @Success 200 {object} response.Response
// @Router /notifications/events/{id}/retry [post]
func (h *NotificationHandler) RetryEvent(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	count, err := h.svc.RetryEvent(id)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("重试通知事件", zap.String("operator", c.GetString("username")), zap.Int64("event_id", id), zap.Int("retry_count", count), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "已触发重试", gin.H{"delivery_count": count})
}

// ListTemplates 通知模板列表。
// @Summary 通知模板列表
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Router /notifications/templates [get]
func (h *NotificationHandler) ListTemplates(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	items, err := h.svc.ListTemplates()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

// UpdateTemplate 更新通知模板。
// @Summary 更新通知模板
// @Tags 通知管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "模板ID"
// @Param body body object true "模板内容 {title, content}"
// @Success 200 {object} response.Response
// @Router /notifications/templates/{id} [post]
func (h *NotificationHandler) UpdateTemplate(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.UpdateTemplate(id, req.Title, req.Content); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新通知模板", zap.String("operator", c.GetString("username")), zap.Int64("template_id", id), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "模板已更新", nil)
}

// PreviewTemplate 预览通知模板渲染结果。
// @Summary 预览通知模板
// @Tags 通知管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object true "预览请求 {title, content, variables}"
// @Success 200 {object} response.Response
// @Router /notifications/templates/preview [post]
func (h *NotificationHandler) PreviewTemplate(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var req struct {
		Title     string                 `json:"title" binding:"required"`
		Content   string                 `json:"content" binding:"required"`
		Variables map[string]interface{} `json:"variables"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	rendered, err := h.svc.RenderTemplate(req.Title, req.Content, req.Variables)
	if err != nil {
		response.Error(c, 400, "模板渲染失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"title": rendered.Title, "content": rendered.Content})
}

// TestWebhook 测试 Webhook 连通性。
// @Summary 测试 Webhook 连通性
// @Tags 通知管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object true "Webhook 测试 {channel_type, webhook_url, secret}"
// @Success 200 {object} response.Response
// @Router /notifications/test-webhook [post]
func (h *NotificationHandler) TestWebhook(c *gin.Context) {
	var req struct {
		ChannelType string `json:"channel_type" binding:"required"`
		WebhookURL  string `json:"webhook_url" binding:"required"`
		Secret      string `json:"secret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	title := "BigOps 通知测试"
	markdown := "## 通知测试\n\n这是一条来自 BigOps 的测试消息，收到说明 Webhook 配置正确。\n\n- 渠道类型: " + req.ChannelType + "\n- 时间: " + time.Now().Format("2006-01-02 15:04:05")
	if err := service.DispatchWebhook(req.ChannelType, req.WebhookURL, req.Secret, title, markdown); err != nil {
		response.Error(c, 400, "发送失败: "+err.Error())
		return
	}
	response.SuccessWithMessage(c, "测试消息发送成功", nil)
}

// GetEnabledChannelTypes 获取管理员允许的通知渠道类型列表。
// @Summary 获取启用的通知渠道类型
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Router /notifications/enabled-channel-types [get]
func (h *NotificationHandler) GetEnabledChannelTypes(c *gin.Context) {
	cfg := config.Get().Notification
	types := cfg.EnabledChannelTypes
	if len(types) == 0 {
		types = []string{"lark", "dingtalk", "wecom", "webhook"}
	}
	response.Success(c, types)
}

// ========== 发送组 CRUD ==========

// ListGroups 发送组分页列表。
// @Summary 发送组分页列表
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param keyword query string false "关键字"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /notify-groups [get]
func (h *NotificationHandler) ListGroups(c *gin.Context) {
	page, size := parsePageSize(c)
	groupSvc := service.NewNotifyGroupService()
	items, total, err := groupSvc.List(page, size, c.Query("keyword"))
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

// ListAllGroups 发送组全量列表。
// @Summary 发送组全量列表
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Router /notify-groups/all [get]
func (h *NotificationHandler) ListAllGroups(c *gin.Context) {
	groupSvc := service.NewNotifyGroupService()
	items, err := groupSvc.ListAll()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

// GetGroup 发送组详情。
// @Summary 发送组详情
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "发送组ID"
// @Success 200 {object} response.Response
// @Router /notify-groups/{id} [get]
func (h *NotificationHandler) GetGroup(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	groupSvc := service.NewNotifyGroupService()
	item, err := groupSvc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "发送组不存在")
		return
	}
	response.Success(c, item)
}

// CreateGroup 创建发送组。
// @Summary 创建发送组
// @Tags 通知管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body model.NotifyGroup true "发送组"
// @Success 200 {object} response.Response
// @Router /notify-groups [post]
func (h *NotificationHandler) CreateGroup(c *gin.Context) {
	var item model.NotifyGroup
	if err := c.ShouldBindJSON(&item); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID, _ := c.Get("userID")
	item.CreatedBy, _ = userID.(int64)
	groupSvc := service.NewNotifyGroupService()
	if err := groupSvc.Create(&item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建发送组", zap.String("operator", c.GetString("username")), zap.Int64("group_id", item.ID), zap.String("name", item.Name))
	response.SuccessWithMessage(c, "创建成功", item)
}

// UpdateGroup 更新发送组。
// @Summary 更新发送组
// @Tags 通知管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "发送组ID"
// @Param body body model.NotifyGroup true "发送组"
// @Success 200 {object} response.Response
// @Router /notify-groups/{id} [post]
func (h *NotificationHandler) UpdateGroup(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var item model.NotifyGroup
	if err := c.ShouldBindJSON(&item); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	groupSvc := service.NewNotifyGroupService()
	if err := groupSvc.Update(id, &item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新发送组", zap.String("operator", c.GetString("username")), zap.Int64("group_id", id))
	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteGroup 删除发送组。
// @Summary 删除发送组
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "发送组ID"
// @Success 200 {object} response.Response
// @Router /notify-groups/{id}/delete [post]
func (h *NotificationHandler) DeleteGroup(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	groupSvc := service.NewNotifyGroupService()
	if err := groupSvc.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除发送组", zap.String("operator", c.GetString("username")), zap.Int64("group_id", id))
	response.SuccessWithMessage(c, "删除成功", nil)
}

// TestGroup 测试发送组。
// @Summary 测试发送组
// @Tags 通知管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "发送组ID"
// @Success 200 {object} response.Response
// @Router /notify-groups/{id}/test [post]
func (h *NotificationHandler) TestGroup(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	groupSvc := service.NewNotifyGroupService()
	group, err := groupSvc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "发送组不存在")
		return
	}
	targets := service.ResolveGroupWebhookTargets(group.WebhooksJSON)
	if len(targets) == 0 {
		response.Error(c, 400, "发送组无 Webhook 配置")
		return
	}
	title := "BigOps 发送组测试"
	markdown := "## 发送组测试\n\n发送组 **" + group.Name + "** 的测试消息。\n\n收到说明 Webhook 配置正确。\n\n- 时间: " + time.Now().Format("2006-01-02 15:04:05")
	var failed []string
	for key, target := range targets {
		if err := service.DispatchWebhook(extractChannelType(key), target.WebhookURL, target.Secret, title, markdown); err != nil {
			failed = append(failed, key+": "+err.Error())
		}
	}
	if len(failed) > 0 {
		response.Error(c, 400, "部分发送失败: "+strings.Join(failed, "; "))
		return
	}
	response.SuccessWithMessage(c, "测试消息已发送到所有渠道", nil)
}

func extractChannelType(key string) string {
	// key may be "lark", "lark_SRE群" etc
	if idx := strings.Index(key, "_"); idx > 0 {
		return key[:idx]
	}
	return key
}
