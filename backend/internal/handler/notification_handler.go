package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/model"
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

type NotificationTestRequest struct {
	Title    string   `json:"title" binding:"required"`
	Content  string   `json:"content" binding:"required"`
	Channels []string `json:"channels"`
	UserIDs  []int64  `json:"user_ids"`
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
