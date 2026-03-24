package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/bigops/platform/internal/pkg/config"
	"github.com/bigops/platform/internal/pkg/database"
	"github.com/bigops/platform/internal/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type NotificationPublishRequest struct {
	EventType string
	BizType   string
	BizID     int64
	Title     string
	Content   string
	Level     string
	UserIDs   []int64
	Payload   interface{}
	Channels  []string
}

type NotificationDispatchResult struct {
	EventID int64 `json:"event_id"`
}

type deliveryDispatchOutcome struct {
	success   bool
	retryable bool
	response  string
}

type NotificationScheduler struct {
	ctx    context.Context
	cancel context.CancelFunc
	ticker *time.Ticker
}

type NotificationService struct {
	userRepo   *repository.UserRepository
	repo       *repository.NotificationRepository
	httpClient *http.Client
}

func NewNotificationService() *NotificationService {
	timeout := 10 * time.Second
	cfg := config.Get()
	if cfg.Notification.Webhook.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.Notification.Webhook.TimeoutSeconds) * time.Second
	}
	return &NotificationService{
		userRepo:   repository.NewUserRepository(),
		repo:       repository.NewNotificationRepository(),
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (s *NotificationService) PublishTx(tx *gorm.DB, req NotificationPublishRequest) (int64, error) {
	userIDs := dedupeUserIDs(req.UserIDs)
	channels := s.normalizeChannels(req.Channels)
	if len(userIDs) == 0 && len(channels) == 0 {
		return 0, nil
	}

	payloadJSON := "{}"
	if req.Payload != nil {
		if data, err := json.Marshal(req.Payload); err == nil {
			payloadJSON = string(data)
		}
	}

	event := &model.NotificationEvent{
		EventType: req.EventType,
		BizType:   req.BizType,
		BizID:     req.BizID,
		Title:     req.Title,
		Payload:   payloadJSON,
		Status:    "pending",
	}
	if err := tx.Create(event).Error; err != nil {
		return 0, err
	}

	for _, channel := range channels {
		switch channel {
		case "in_app":
			for _, userID := range userIDs {
				delivery := &model.NotificationDelivery{
					EventID:   event.ID,
					Channel:   "in_app",
					Recipient: fmt.Sprintf("%d", userID),
					Status:    "sent",
				}
				if err := tx.Create(delivery).Error; err != nil {
					return 0, err
				}
				notification := &model.InAppNotification{
					UserID:  userID,
					Title:   req.Title,
					Content: req.Content,
					Level:   normalizeLevel(req.Level),
					BizType: req.BizType,
					BizID:   req.BizID,
				}
				if err := tx.Create(notification).Error; err != nil {
					return 0, err
				}
			}
		case "email":
			for _, userID := range userIDs {
				now := model.LocalTime(time.Now())
				delivery := &model.NotificationDelivery{
					EventID:     event.ID,
					Channel:     "email",
					Recipient:   fmt.Sprintf("%d", userID),
					Status:      "pending",
					NextRetryAt: &now,
				}
				if err := tx.Create(delivery).Error; err != nil {
					return 0, err
				}
			}
		case "webhook":
			now := model.LocalTime(time.Now())
			if err := tx.Create(&model.NotificationDelivery{
				EventID:     event.ID,
				Channel:     "webhook",
				Recipient:   "system",
				Status:      "pending",
				NextRetryAt: &now,
			}).Error; err != nil {
				return 0, err
			}
		case "message_pusher":
			now := model.LocalTime(time.Now())
			if err := tx.Create(&model.NotificationDelivery{
				EventID:     event.ID,
				Channel:     "message_pusher",
				Recipient:   "system",
				Status:      "pending",
				NextRetryAt: &now,
			}).Error; err != nil {
				return 0, err
			}
		}
	}

	return event.ID, nil
}

func (s *NotificationService) Publish(req NotificationPublishRequest) (int64, error) {
	var eventID int64
	err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		id, err := s.PublishTx(tx, req)
		if err != nil {
			return err
		}
		eventID = id
		return nil
	})
	if err != nil {
		return 0, err
	}
	if eventID > 0 {
		s.DispatchEventAsync(eventID)
	}
	return eventID, nil
}

func (s *NotificationService) DispatchEventAsync(eventID int64) {
	go func() {
		_ = s.DispatchEvent(eventID)
	}()
}

func (s *NotificationService) DispatchEvent(eventID int64) error {
	event, err := s.repo.GetEventByID(eventID)
	if err != nil {
		return err
	}
	deliveries, err := s.repo.ListDeliveriesByEventID(eventID, true)
	if err != nil {
		return err
	}
	for _, delivery := range deliveries {
		s.dispatchDelivery(event, delivery)
	}
	_ = s.refreshEventStatus(event.ID)
	return nil
}

func (s *NotificationService) ListInAppByUserID(userID int64, unreadOnly bool) ([]*model.InAppNotification, error) {
	return s.repo.ListInAppByUserID(userID, unreadOnly)
}

func (s *NotificationService) CountUnreadByUserID(userID int64) (int64, error) {
	return s.repo.CountUnreadByUserID(userID)
}

func (s *NotificationService) MarkRead(userID, notificationID int64) error {
	item, err := s.repo.GetInAppByID(notificationID)
	if err != nil {
		return errors.New("通知不存在")
	}
	if item.UserID != userID {
		return errors.New("无权操作该通知")
	}
	if item.ReadAt != nil {
		return nil
	}
	now := model.LocalTime(time.Now())
	return s.repo.MarkRead(notificationID, now)
}

func (s *NotificationService) ListEvents(limit int) ([]*model.NotificationEvent, error) {
	events, err := s.repo.ListEvents(limit)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		deliveries, err := s.repo.ListDeliveriesByEventID(event.ID, false)
		if err != nil {
			return nil, err
		}
		event.Deliveries = make([]model.NotificationDelivery, 0, len(deliveries))
		event.CanRetry = false
		for _, delivery := range deliveries {
			delivery.StatusSummary = deliveryStatusSummary(delivery)
			delivery.CanRetry = delivery.Status != "sent"
			if delivery.CanRetry {
				event.CanRetry = true
			}
			event.Deliveries = append(event.Deliveries, *delivery)
		}
		event.StatusSummary = eventStatusSummary(event.Status)
	}
	return events, nil
}

func (s *NotificationService) RetryEvent(eventID int64) (int, error) {
	event, err := s.repo.GetEventByID(eventID)
	if err != nil {
		return 0, err
	}
	deliveries, err := s.repo.ListDeliveriesByEventIDAndStatuses(eventID, []string{"pending", "failed", "dead"})
	if err != nil {
		return 0, err
	}
	if len(deliveries) == 0 {
		return 0, errors.New("该事件没有可重试的投递记录")
	}
	for _, delivery := range deliveries {
		s.dispatchDelivery(event, delivery)
	}
	_ = s.refreshEventStatus(eventID)
	return len(deliveries), nil
}

func (s *NotificationService) DispatchDueDeliveries(limit int) error {
	now := model.LocalTime(time.Now())
	deliveries, err := s.repo.ListRetryableDeliveries(now, limit)
	if err != nil {
		return err
	}
	for _, delivery := range deliveries {
		event, err := s.repo.GetEventByID(delivery.EventID)
		if err != nil {
			continue
		}
		s.dispatchDelivery(event, delivery)
		_ = s.refreshEventStatus(event.ID)
	}
	return nil
}

func (s *NotificationService) dispatchDelivery(event *model.NotificationEvent, delivery *model.NotificationDelivery) {
	cfg := config.Get().Notification
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	retryInterval := cfg.RetryIntervalSeconds
	if retryInterval <= 0 {
		retryInterval = 60
	}

	now := model.LocalTime(time.Now())
	delivery.LastAttemptAt = &now

	var outcome deliveryDispatchOutcome
	switch delivery.Channel {
	case "email":
		outcome = s.sendEmail(event, delivery)
	case "webhook":
		outcome = s.sendWebhook(event, delivery)
	case "message_pusher":
		outcome = s.sendMessagePusher(event, delivery)
	default:
		return
	}

	delivery.RetryCount += 1
	delivery.Response = outcome.response
	if outcome.success {
		delivery.Status = "sent"
		delivery.SentAt = &now
		delivery.NextRetryAt = nil
	} else {
		if !outcome.retryable || delivery.RetryCount >= maxRetries {
			delivery.Status = "dead"
			delivery.NextRetryAt = nil
		} else {
			delivery.Status = "failed"
			next := model.LocalTime(time.Now().Add(time.Duration(retryInterval) * time.Second))
			delivery.NextRetryAt = &next
		}
	}
	_ = s.repo.UpdateDelivery(delivery)
}

func (s *NotificationService) refreshEventStatus(eventID int64) error {
	event, err := s.repo.GetEventByID(eventID)
	if err != nil {
		return err
	}
	deliveries, err := s.repo.ListDeliveriesByEventID(eventID, false)
	if err != nil {
		return err
	}
	if len(deliveries) == 0 {
		return nil
	}
	total := len(deliveries)
	sentCount := 0
	pendingCount := 0
	failedCount := 0
	deadCount := 0
	for _, delivery := range deliveries {
		switch delivery.Status {
		case "sent":
			sentCount++
		case "failed":
			failedCount++
		case "pending":
			pendingCount++
		case "dead":
			deadCount++
		default:
			pendingCount++
		}
	}
	switch {
	case sentCount == total:
		event.Status = "sent"
	case pendingCount > 0:
		event.Status = "pending"
	case failedCount > 0:
		event.Status = "retrying"
	case deadCount == total:
		event.Status = "failed"
	case deadCount > 0 && sentCount > 0:
		event.Status = "partial_failed"
	default:
		event.Status = "pending"
	}
	return s.repo.UpdateEvent(event)
}

func NewNotificationScheduler() *NotificationScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &NotificationScheduler{ctx: ctx, cancel: cancel}
}

func (s *NotificationScheduler) Start() {
	interval := config.Get().Notification.RetryScanIntervalSeconds
	if interval <= 0 {
		interval = 60
	}
	s.ticker = time.NewTicker(time.Duration(interval) * time.Second)
	go func() {
		notifySvc := NewNotificationService()
		logger.Info("通知重试调度器已启动", zap.Int("interval_seconds", interval))
		for {
			select {
			case <-s.ctx.Done():
				logger.Info("通知重试调度器已停止")
				return
			case <-s.ticker.C:
				if err := notifySvc.DispatchDueDeliveries(100); err != nil {
					logger.Error("通知重试调度失败", zap.Error(err))
				}
			}
		}
	}()
}

func (s *NotificationScheduler) Stop() {
	s.cancel()
	if s.ticker != nil {
		s.ticker.Stop()
	}
}

func (s *NotificationService) normalizeChannels(channels []string) []string {
	if len(channels) == 0 {
		cfgChannels := config.Get().Notification.DefaultChannels
		if len(cfgChannels) > 0 {
			channels = cfgChannels
		} else {
			channels = []string{"in_app"}
		}
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(channels))
	for _, channel := range channels {
		if channel == "" || seen[channel] {
			continue
		}
		seen[channel] = true
		result = append(result, channel)
	}
	return result
}

func dedupeUserIDs(userIDs []int64) []int64 {
	result := make([]int64, 0, len(userIDs))
	seen := make(map[int64]bool, len(userIDs))
	for _, userID := range userIDs {
		if userID <= 0 || seen[userID] {
			continue
		}
		seen[userID] = true
		result = append(result, userID)
	}
	return result
}

func normalizeLevel(level string) string {
	switch level {
	case "success", "warning", "error":
		return level
	default:
		return "info"
	}
}

func (s *NotificationService) sendEmail(event *model.NotificationEvent, delivery *model.NotificationDelivery) deliveryDispatchOutcome {
	cfg := config.Get().Notification.Email
	if !cfg.Enabled || cfg.Host == "" || cfg.Port == 0 || cfg.From == "" {
		return deliveryDispatchOutcome{response: "email channel not configured"}
	}
	userID, err := strconv.ParseInt(delivery.Recipient, 10, 64)
	if err != nil {
		return dispatchErrorOutcome(err)
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return dispatchErrorOutcome(err)
	}
	if user.Email == nil || *user.Email == "" {
		return deliveryDispatchOutcome{response: "recipient email is empty"}
	}

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	msg := []byte("To: " + *user.Email + "\r\n" +
		"Subject: " + event.Title + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		event.Payload)

	if err := smtp.SendMail(
		fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		auth,
		cfg.From,
		[]string{*user.Email},
		msg,
	); err != nil {
		return dispatchErrorOutcome(err)
	}
	return deliveryDispatchOutcome{
		success:  true,
		response: fmt.Sprintf("smtp delivered to %s", *user.Email),
	}
}

func (s *NotificationService) sendWebhook(event *model.NotificationEvent, _ *model.NotificationDelivery) deliveryDispatchOutcome {
	cfg := config.Get().Notification.Webhook
	if !cfg.Enabled || cfg.URL == "" {
		return deliveryDispatchOutcome{response: "webhook channel not configured"}
	}
	body := map[string]interface{}{
		"event_type": event.EventType,
		"biz_type":   event.BizType,
		"biz_id":     event.BizID,
		"title":      event.Title,
		"payload":    json.RawMessage(event.Payload),
		"timestamp":  time.Now().Unix(),
	}
	data, err := json.Marshal(body)
	if err != nil {
		return dispatchErrorOutcome(err)
	}
	req, err := http.NewRequest(http.MethodPost, cfg.URL, bytes.NewBuffer(data))
	if err != nil {
		return dispatchErrorOutcome(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Secret != "" {
		mac := hmac.New(sha256.New, []byte(cfg.Secret))
		mac.Write(data)
		req.Header.Set("X-BigOps-Signature", fmt.Sprintf("%x", mac.Sum(nil)))
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return dispatchErrorOutcome(err)
	}
	defer resp.Body.Close()
	return buildHTTPOutcome("webhook", resp)
}

func (s *NotificationService) sendMessagePusher(event *model.NotificationEvent, _ *model.NotificationDelivery) deliveryDispatchOutcome {
	cfg := config.Get().Notification.MessagePusher
	if !cfg.Enabled || cfg.Server == "" || cfg.Username == "" || cfg.Token == "" {
		return deliveryDispatchOutcome{response: "message_pusher channel not configured"}
	}
	server := strings.TrimRight(cfg.Server, "/")
	body := map[string]interface{}{
		"title":       event.Title,
		"description": event.Title,
		"content":     event.Payload,
		"token":       cfg.Token,
	}
	if cfg.Channel != "" {
		body["channel"] = cfg.Channel
	}
	data, err := json.Marshal(body)
	if err != nil {
		return dispatchErrorOutcome(err)
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	client := s.httpClient
	if timeout > 0 {
		client = &http.Client{Timeout: timeout}
	}
	resp, err := client.Post(server+"/push/"+cfg.Username, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return dispatchErrorOutcome(err)
	}
	defer resp.Body.Close()
	return buildHTTPOutcome("message_pusher", resp)
}

func dispatchErrorOutcome(err error) deliveryDispatchOutcome {
	outcome := deliveryDispatchOutcome{
		success:   false,
		retryable: isRetryableError(err),
		response:  truncateText(err.Error(), 500),
	}
	if outcome.response == "" {
		outcome.response = "unknown error"
	}
	return outcome
}

func buildHTTPOutcome(channel string, resp *http.Response) deliveryDispatchOutcome {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	bodyText := strings.TrimSpace(string(body))
	summary := fmt.Sprintf("%s status=%d", channel, resp.StatusCode)
	if bodyText != "" {
		summary = fmt.Sprintf("%s body=%s", summary, truncateText(bodyText, 400))
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return deliveryDispatchOutcome{
			success:  true,
			response: summary,
		}
	}
	return deliveryDispatchOutcome{
		success:   false,
		retryable: resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests,
		response:  summary,
	}
}

func truncateText(text string, max int) string {
	if max <= 0 || len(text) <= max {
		return text
	}
	return text[:max] + "..."
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
		type temporary interface{ Temporary() bool }
		if tempErr, ok := any(netErr).(temporary); ok && tempErr.Temporary() {
			return true
		}
	}
	var protoErr *textproto.Error
	if errors.As(err, &protoErr) {
		return protoErr.Code >= 400 && protoErr.Code < 500
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "timeout"):
		return true
	case strings.Contains(message, "connection reset"):
		return true
	case strings.Contains(message, "connection refused"):
		return true
	case strings.Contains(message, "tempor"):
		return true
	default:
		return false
	}
}

func deliveryStatusSummary(delivery *model.NotificationDelivery) string {
	if delivery == nil {
		return ""
	}
	switch delivery.Status {
	case "sent":
		return "发送成功"
	case "pending":
		return "等待发送"
	case "failed":
		return "发送失败，等待重试"
	case "dead":
		return "发送失败，不再自动重试"
	default:
		return delivery.Status
	}
}

func eventStatusSummary(status string) string {
	switch status {
	case "sent":
		return "全部发送成功"
	case "pending":
		return "仍有投递等待中"
	case "retrying":
		return "存在失败投递，系统会继续重试"
	case "failed":
		return "全部投递失败"
	case "partial_failed":
		return "部分投递成功，部分失败"
	default:
		return status
	}
}
