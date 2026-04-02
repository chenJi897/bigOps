package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"strings"
	"text/template"
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
	EventType      string
	BizType        string
	BizID          int64
	Title          string
	Content        string
	Level          string
	UserIDs        []int64
	Payload        interface{}
	Channels       []string
	SkipPreference bool
	NotifyConfig   map[string]WebhookTarget // 业务内嵌的 Webhook 配置（渠道类型→地址）
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
	repo       *repository.NotificationRepository
	prefRepo   *repository.NotificationUserSettingRepository
	tmplRepo   *repository.NotificationTemplateRepository
	httpClient *http.Client
}

func NewNotificationService() *NotificationService {
	timeout := 10 * time.Second
	cfg := config.Get()
	if cfg.Notification.MessagePusher.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.Notification.MessagePusher.TimeoutSeconds) * time.Second
	}
	return &NotificationService{
		repo:       repository.NewNotificationRepository(),
		prefRepo:   repository.NewNotificationUserSettingRepository(),
		tmplRepo:   repository.NewNotificationTemplateRepository(),
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (s *NotificationService) PublishTx(tx *gorm.DB, req NotificationPublishRequest) (int64, error) {
	userIDs := dedupeUserIDs(req.UserIDs)
	if len(userIDs) == 0 && len(req.NotifyConfig) == 0 {
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

	hasExternal := false

	// 1. 站内通知：强制发送，不可取消
	for _, userID := range userIDs {
		if err := tx.Create(&model.NotificationDelivery{
			EventID:   event.ID,
			Channel:   "in_app",
			Recipient: fmt.Sprintf("%d", userID),
			Status:    "sent",
		}).Error; err != nil {
			return 0, err
		}
		if err := tx.Create(&model.InAppNotification{
			UserID:  userID,
			Title:   req.Title,
			Content: req.Content,
			Level:   normalizeLevel(req.Level),
			BizType: req.BizType,
			BizID:   req.BizID,
		}).Error; err != nil {
			return 0, err
		}
	}

	// 2. 外部渠道：按 NotifyConfig 中的 Webhook 地址创建投递记录
	now := model.LocalTime(time.Now())
	for channelType, target := range req.NotifyConfig {
		if strings.TrimSpace(target.WebhookURL) == "" {
			continue
		}
		if err := tx.Create(&model.NotificationDelivery{
			EventID:       event.ID,
			Channel:       channelType,
			Recipient:     channelType,
			WebhookURL:    target.WebhookURL,
			WebhookSecret: target.Secret,
			Status:        "pending",
			NextRetryAt:   &now,
		}).Error; err != nil {
			return 0, err
		}
		hasExternal = true
	}

	// 兼容旧调用：如果没有 NotifyConfig 但有 Channels，走旧 Message Pusher 逻辑
	if len(req.NotifyConfig) == 0 && len(req.Channels) > 0 {
		for _, channel := range req.Channels {
			if channel == "in_app" {
				continue
			}
			mpCh := config.Get().Notification.ChannelMapping[channel]
			if err := tx.Create(&model.NotificationDelivery{
				EventID:     event.ID,
				Channel:     channel,
				Recipient:   mpCh,
				Status:      "pending",
				NextRetryAt: &now,
			}).Error; err != nil {
				return 0, err
			}
			hasExternal = true
		}
	}

	if !hasExternal {
		event.Status = "sent"
		if err := tx.Save(event).Error; err != nil {
			return 0, err
		}
	}

	return event.ID, nil
}

func (s *NotificationService) GetUserPreference(userID int64) (*model.NotificationUserSetting, error) {
	item, err := s.prefRepo.GetByUserID(userID)
	if err == nil {
		return item, nil
	}
	return &model.NotificationUserSetting{
		UserID:             userID,
		EnabledChannels:    "[\"in_app\",\"wecom\",\"dingtalk\",\"lark\"]",
		SubscribedBizTypes: "[\"alert_event\",\"ticket\",\"cicd_pipeline\",\"task_execution\",\"notification\"]",
		ChannelTargets:     "{}",
		Enabled:            1,
	}, nil
}

func (s *NotificationService) SaveUserPreference(item *model.NotificationUserSetting) error {
	item.EnabledChannels = normalizeNotifyJSONList(item.EnabledChannels)
	item.SubscribedBizTypes = normalizeNotifyJSONList(item.SubscribedBizTypes)
	item.ChannelTargets = normalizeNotifyJSONBizMap(item.ChannelTargets)
	if item.Enabled != 0 {
		item.Enabled = 1
	}
	return s.prefRepo.Upsert(item)
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

func (s *NotificationService) MarkAllRead(userID int64) error {
	now := model.LocalTime(time.Now())
	return s.repo.MarkAllReadByUserID(userID, now)
}

func (s *NotificationService) ClearRead(userID int64) (int64, error) {
	return s.repo.ClearReadByUserID(userID)
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
	case "in_app":
		delivery.Status = "sent"
		delivery.SentAt = &now
		delivery.NextRetryAt = nil
		_ = s.repo.UpdateDelivery(delivery)
		return
	default:
		// 新模式：delivery 自带 Webhook URL
		if strings.TrimSpace(delivery.WebhookURL) != "" {
			outcome = s.dispatchViaWebhook(event, delivery)
		} else {
			// 旧模式兼容：走 Message Pusher
			outcome = s.pushViaMessagePusher(event, delivery)
		}
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

func normalizeNotifyJSONList(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "[]"
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return "[]"
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func normalizeNotifyJSONBizMap(raw string) string {
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "null" {
		return "{}"
	}
	// 外部通道的目标映射：bizType -> channelType -> message-pusher-channel-name
	var m map[string]map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return "{}"
	}
	// 清理空键/空值，避免覆盖逻辑里误用空字符串。
	clean := make(map[string]map[string]string)
	for bizType, byChannel := range m {
		bizType = strings.TrimSpace(bizType)
		if bizType == "" || byChannel == nil {
			continue
		}
		for ch, target := range byChannel {
			ch = strings.TrimSpace(ch)
			target = strings.TrimSpace(target)
			if ch == "" || target == "" {
				continue
			}
			if _, ok := clean[bizType]; !ok {
				clean[bizType] = make(map[string]string)
			}
			clean[bizType][ch] = target
		}
	}
	data, err := json.Marshal(clean)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (s *NotificationService) resolveRecipientsByPreference(userIDs []int64, bizType string, channels []string, skipPreference bool) (map[string][]int64, []string) {
	recipients := map[string][]int64{
		"in_app": {},
	}
	if len(channels) == 0 {
		return recipients, channels
	}
	if skipPreference {
		for _, channel := range channels {
			recipients[channel] = append(recipients[channel], userIDs...)
		}
		return recipients, channels
	}
	settings, err := s.prefRepo.ListByUserIDs(userIDs)
	if err != nil {
		for _, channel := range channels {
			recipients[channel] = append(recipients[channel], userIDs...)
		}
		return recipients, channels
	}
	channelEnabled := make(map[string]bool)
	for _, userID := range userIDs {
		setting := settings[userID]
		if setting != nil && setting.Enabled == 0 {
			continue
		}
		allowedChannels := map[string]bool{
			"in_app":  true,
			"wecom":   true,
			"dingtalk": true,
			"lark":    true,
			"webhook": true,
		}
		allowedBizTypes := map[string]bool{}
		if setting != nil {
			var prefChannels []string
			var prefBizTypes []string
			_ = json.Unmarshal([]byte(setting.EnabledChannels), &prefChannels)
			_ = json.Unmarshal([]byte(setting.SubscribedBizTypes), &prefBizTypes)
			if len(prefChannels) > 0 {
				allowedChannels = map[string]bool{}
				for _, channel := range prefChannels {
					allowedChannels[channel] = true
				}
			}
			for _, item := range prefBizTypes {
				allowedBizTypes[item] = true
			}
		}
		if len(allowedBizTypes) > 0 && !allowedBizTypes[bizType] {
			continue
		}
		for _, channel := range channels {
			if allowedChannels[channel] {
				recipients[channel] = append(recipients[channel], userID)
				channelEnabled[channel] = true
			}
		}
	}
	finalChannels := make([]string, 0, len(channels))
	for _, channel := range channels {
		if channelEnabled[channel] {
			finalChannels = append(finalChannels, channel)
		}
	}
	return recipients, finalChannels
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

type RenderedTemplate struct {
	Title   string
	Content string
}

func (s *NotificationService) ListTemplates() ([]*model.NotificationTemplate, error) {
	return s.tmplRepo.List()
}

func (s *NotificationService) UpdateTemplate(id int64, title, content string) error {
	item, err := s.tmplRepo.GetByID(id)
	if err != nil {
		return errors.New("模板不存在")
	}
	item.Title = title
	item.Content = content
	return s.tmplRepo.Update(item)
}

func (s *NotificationService) RenderTemplate(titleTpl, contentTpl string, vars map[string]interface{}) (*RenderedTemplate, error) {
	if vars == nil {
		vars = map[string]interface{}{}
	}
	renderedTitle, err := executeTemplate(titleTpl, vars)
	if err != nil {
		return nil, fmt.Errorf("title template error: %w", err)
	}
	renderedContent, err := executeTemplate(contentTpl, vars)
	if err != nil {
		return nil, fmt.Errorf("content template error: %w", err)
	}
	return &RenderedTemplate{Title: renderedTitle, Content: renderedContent}, nil
}

func executeTemplate(tplStr string, vars map[string]interface{}) (string, error) {
	t, err := template.New("").Option("missingkey=zero").Parse(tplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s *NotificationService) renderEventContent(event *model.NotificationEvent) (string, string) {
	tmpl, err := s.tmplRepo.GetByEventType(event.EventType)
	if err != nil || tmpl == nil {
		return event.Title, event.Payload
	}
	var vars map[string]interface{}
	_ = json.Unmarshal([]byte(event.Payload), &vars)
	if vars == nil {
		vars = map[string]interface{}{}
	}
	rendered, err := s.RenderTemplate(tmpl.Title, tmpl.Content, vars)
	if err != nil {
		return event.Title, event.Payload
	}
	return rendered.Title, rendered.Content
}

func (s *NotificationService) dispatchViaWebhook(event *model.NotificationEvent, delivery *model.NotificationDelivery) deliveryDispatchOutcome {
	renderedTitle, renderedContent := s.renderEventContent(event)
	err := DispatchWebhook(delivery.Channel, delivery.WebhookURL, delivery.WebhookSecret, renderedTitle, renderedContent)
	if err != nil {
		return dispatchErrorOutcome(err)
	}
	return deliveryDispatchOutcome{success: true, response: fmt.Sprintf("%s webhook ok", delivery.Channel)}
}

func (s *NotificationService) pushViaMessagePusher(event *model.NotificationEvent, delivery *model.NotificationDelivery) deliveryDispatchOutcome {
	cfg := config.Get().Notification.MessagePusher
	if !cfg.Enabled || cfg.Server == "" || cfg.Username == "" || cfg.Token == "" {
		return deliveryDispatchOutcome{response: "message_pusher channel not configured"}
	}
	renderedTitle, renderedContent := s.renderEventContent(event)
	server := strings.TrimRight(cfg.Server, "/")
	body := map[string]interface{}{
		"title":       renderedTitle,
		"description": renderedTitle,
		"content":     renderedContent,
		"token":       cfg.Token,
	}
	// 外部投递：优先使用 delivery.Recipient 里的"最终 Message Pusher 通道名"，
	// 否则回退到平台全局的 channel_mapping（兼容旧数据 Recipient="system"）。
	mpCh := strings.TrimSpace(delivery.Recipient)
	if mpCh == "" || mpCh == "system" {
		mpCh = config.Get().Notification.ChannelMapping[delivery.Channel]
	}
	if strings.TrimSpace(mpCh) != "" {
		body["channel"] = mpCh
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
