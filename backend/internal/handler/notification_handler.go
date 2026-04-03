package handler

import (
	"encoding/json"
	"strconv"
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

func (h *NotificationHandler) GetConfig(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	response.Success(c, config.Get().Notification)
}

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

func (h *NotificationHandler) ListInApp(c *gin.Context) {
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	unreadOnly := c.DefaultQuery("unread_only", "0") == "1"
	items, err := h.svc.ListInAppByUserID(currentUserID, unreadOnly)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

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

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.MarkRead(currentUserID, id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("标记通知已读", zap.String("operator", c.GetString("username")), zap.Int64("notification_id", id), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "已标记为已读", nil)
}

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

func (h *NotificationHandler) ListEvents(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	items, err := h.svc.ListEvents(50)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

func (h *NotificationHandler) RetryEvent(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	count, err := h.svc.RetryEvent(id)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("重试通知事件", zap.String("operator", c.GetString("username")), zap.Int64("event_id", id), zap.Int("retry_count", count), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "已触发重试", gin.H{"delivery_count": count})
}

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

func (h *NotificationHandler) UpdateTemplate(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
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
func (h *NotificationHandler) GetEnabledChannelTypes(c *gin.Context) {
	cfg := config.Get().Notification
	types := cfg.EnabledChannelTypes
	if len(types) == 0 {
		types = []string{"lark", "dingtalk", "wecom", "webhook"}
	}
	response.Success(c, types)
}

// ========== 发送组 CRUD ==========

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

func (h *NotificationHandler) ListAllGroups(c *gin.Context) {
	groupSvc := service.NewNotifyGroupService()
	items, err := groupSvc.ListAll()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

func (h *NotificationHandler) GetGroup(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	groupSvc := service.NewNotifyGroupService()
	item, err := groupSvc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "发送组不存在")
		return
	}
	response.Success(c, item)
}

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

func (h *NotificationHandler) UpdateGroup(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
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

func (h *NotificationHandler) DeleteGroup(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	groupSvc := service.NewNotifyGroupService()
	if err := groupSvc.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除发送组", zap.String("operator", c.GetString("username")), zap.Int64("group_id", id))
	response.SuccessWithMessage(c, "删除成功", nil)
}

func (h *NotificationHandler) TestGroup(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
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
