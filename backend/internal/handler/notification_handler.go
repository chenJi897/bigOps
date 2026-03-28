package handler

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/config"
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
	Webhook struct {
		Enabled        bool   `json:"enabled"`
		URL            string `json:"url"`
		Secret         string `json:"secret"`
		TimeoutSeconds int    `json:"timeout_seconds"`
	} `json:"webhook"`
	Email struct {
		Enabled  bool   `json:"enabled"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		From     string `json:"from"`
	} `json:"email"`
	MessagePusher struct {
		Enabled        bool   `json:"enabled"`
		Server         string `json:"server"`
		Username       string `json:"username"`
		Token          string `json:"token"`
		Channel        string `json:"channel"`
		TimeoutSeconds int    `json:"timeout_seconds"`
	} `json:"message_pusher"`
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
		Webhook: config.NotificationWebhookConfig{
			Enabled:        req.Webhook.Enabled,
			URL:            req.Webhook.URL,
			Secret:         req.Webhook.Secret,
			TimeoutSeconds: req.Webhook.TimeoutSeconds,
		},
		Email: config.NotificationEmailConfig{
			Enabled:  req.Email.Enabled,
			Host:     req.Email.Host,
			Port:     req.Email.Port,
			Username: req.Email.Username,
			Password: req.Email.Password,
			From:     req.Email.From,
		},
		MessagePusher: config.MessagePusherConfig{
			Enabled:        req.MessagePusher.Enabled,
			Server:         req.MessagePusher.Server,
			Username:       req.MessagePusher.Username,
			Token:          req.MessagePusher.Token,
			Channel:        req.MessagePusher.Channel,
			TimeoutSeconds: req.MessagePusher.TimeoutSeconds,
		},
	}
	if err := config.UpdateNotificationConfig(cfg); err != nil {
		response.Error(c, 500, err.Error())
		return
	}
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
	response.SuccessWithMessage(c, "已标记为已读", nil)
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	if err := h.svc.MarkAllRead(currentUserID); err != nil {
		response.InternalServerError(c, "批量已读失败")
		return
	}
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
	response.SuccessWithMessage(c, "已清空已读通知", gin.H{"count": count})
}

type NotificationTestRequest struct {
	Title    string   `json:"title" binding:"required"`
	Content  string   `json:"content" binding:"required"`
	Channels []string `json:"channels"`
	UserIDs  []int64  `json:"user_ids"`
}

type NotificationPreferenceRequest struct {
	EnabledChannels   []string `json:"enabled_channels"`
	SubscribedBizTypes []string `json:"subscribed_biz_types"`
	Enabled           int8     `json:"enabled"`
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
	item := &model.NotificationUserSetting{
		UserID:             currentUserID,
		EnabledChannels:    string(channels),
		SubscribedBizTypes: string(bizTypes),
		Enabled:            req.Enabled,
	}
	if err := h.svc.SaveUserPreference(item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
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
		EventType: "notification_test",
		BizType:   "notification",
		BizID:     currentUserID,
		Title:     req.Title,
		Content:   req.Content,
		Level:     "info",
		UserIDs:   userIDs,
		Channels:  req.Channels,
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
	response.SuccessWithMessage(c, "已触发重试", gin.H{"delivery_count": count})
}
