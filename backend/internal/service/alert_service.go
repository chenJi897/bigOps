package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/safego"
	"github.com/bigops/platform/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertService struct {
	ruleRepo    *repository.AlertRuleRepository
	eventRepo   *repository.AlertEventRepository
	agentRepo   *repository.AgentRepository
	assetRepo   *repository.AssetRepository
	silenceRepo *repository.AlertSilenceRepository
	notifySvc   *NotificationService
	ticketSvc   *TicketService
	taskSvc     *TaskService
	oncallSvc   *OnCallService
}

type AlertEvaluationSummary struct {
	AgentCount     int `json:"agent_count"`
	RuleCount      int `json:"rule_count"`
	TriggeredCount int `json:"triggered_count"`
	ResolvedCount  int `json:"resolved_count"`
	UpdatedCount   int `json:"updated_count"`
	ErrorCount     int `json:"error_count"`
}

type AlertEventGroup struct {
	Fingerprint    string          `json:"fingerprint"`
	RuleName       string          `json:"rule_name"`
	Severity       string          `json:"severity"`
	Status         string          `json:"status"`
	TotalCount     int             `json:"total_count"`
	FirstTriggered model.LocalTime `json:"first_triggered"`
	LastTriggered  model.LocalTime `json:"last_triggered"`
	DurationSec    int64           `json:"duration_sec"`
	LatestEventID  int64           `json:"latest_event_id"`
	LatestHost     string          `json:"latest_host"`
	ServiceTreeID  int64           `json:"service_tree_id"`
	OwnerID        int64           `json:"owner_id"`
}

type AlertTimelineItem struct {
	Type      string           `json:"type"`
	Timestamp *model.LocalTime `json:"timestamp,omitempty"`
	Note      string           `json:"note,omitempty"`
	Operator  int64            `json:"operator,omitempty"`
}

type AlertEventTimeline struct {
	EventID      int64               `json:"event_id"`
	RuleName     string              `json:"rule_name"`
	Severity     string              `json:"severity"`
	Status       string              `json:"status"`
	RelatedCount int                 `json:"related_count"`
	Items        []AlertTimelineItem `json:"items"`
}

type AlertRootCauseResult struct {
	EventID           int64    `json:"event_id"`
	PrimarySuspect    string   `json:"primary_suspect"`
	Confidence        float64  `json:"confidence"`
	Evidence          []string `json:"evidence"`
	RelatedEventCount int      `json:"related_event_count"`
	ServiceTreeID     int64    `json:"service_tree_id"`
	OwnerID           int64    `json:"owner_id"`
}

type AlertEventContext struct {
	EventID         int64    `json:"event_id"`
	RuleName        string   `json:"rule_name"`
	Status          string   `json:"status"`
	Severity        string   `json:"severity"`
	Action          string   `json:"action"`
	TaskExecutionID int64    `json:"task_execution_id"`
	TicketID        int64    `json:"ticket_id"`
	ServiceTreeID   int64    `json:"service_tree_id"`
	OwnerID         int64    `json:"owner_id"`
	Suggestions     []string `json:"suggestions"`
}

func NewAlertService() *AlertService {
	return &AlertService{
		ruleRepo:    repository.NewAlertRuleRepository(),
		eventRepo:   repository.NewAlertEventRepository(),
		agentRepo:   repository.NewAgentRepository(),
		assetRepo:   repository.NewAssetRepository(),
		silenceRepo: repository.NewAlertSilenceRepository(),
		notifySvc:   NewNotificationService(),
		ticketSvc:   NewTicketService(),
		taskSvc:     NewTaskService(),
		oncallSvc:   NewOnCallService(),
	}
}

func (s *AlertService) CreateRule(item *model.AlertRule) error {
	normalizeAlertRule(item)
	if err := validateAlertRule(item); err != nil {
		return err
	}
	if _, err := s.ruleRepo.GetByName(item.Name); err == nil {
		return errors.New("规则名称已存在")
	}
	return s.ruleRepo.Create(item)
}

func (s *AlertService) UpdateRule(id int64, item *model.AlertRule) error {
	normalizeAlertRule(item)
	if err := validateAlertRule(item); err != nil {
		return err
	}
	existing, err := s.ruleRepo.GetByID(id)
	if err != nil {
		return errors.New("规则不存在")
	}
	if item.Name != "" {
		if _, err := s.ruleRepo.GetByNameExcludingID(item.Name, id); err == nil {
			return errors.New("规则名称已存在")
		}
	}
	existing.Name = item.Name
	existing.MetricType = item.MetricType
	existing.Operator = item.Operator
	existing.Threshold = item.Threshold
	existing.Severity = item.Severity
	existing.Enabled = item.Enabled
	existing.Description = item.Description
	existing.NotifyUserIDs = item.NotifyUserIDs
	existing.Action = item.Action
	existing.RepairTaskID = item.RepairTaskID
	existing.TicketTypeID = item.TicketTypeID
	existing.OnCallScheduleID = item.OnCallScheduleID
	existing.ServiceTreeID = item.ServiceTreeID
	existing.OwnerID = item.OwnerID
	existing.UpdatedBy = item.UpdatedBy
	return s.ruleRepo.Update(existing)
}

func (s *AlertService) DeleteRule(id int64) error {
	if _, err := s.ruleRepo.GetByID(id); err != nil {
		return errors.New("规则不存在")
	}
	return s.ruleRepo.Delete(id)
}

func (s *AlertService) ListRules(q repository.AlertRuleListQuery) ([]*model.AlertRule, int64, error) {
	return s.ruleRepo.List(q)
}

func (s *AlertService) ListEvents(q repository.AlertEventListQuery) ([]*model.AlertEvent, int64, error) {
	return s.eventRepo.List(q)
}

func (s *AlertService) GetEvent(id int64) (*model.AlertEvent, error) {
	return s.eventRepo.GetByID(id)
}

func (s *AlertService) ListEventGroups(q repository.AlertEventListQuery, windowMinutes int) ([]*AlertEventGroup, int64, error) {
	if windowMinutes <= 0 {
		windowMinutes = 5
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Size <= 0 {
		q.Size = 20
	}
	baseQuery := q
	baseQuery.Page = 1
	baseQuery.Size = 1000
	items, _, err := s.eventRepo.List(baseQuery)
	if err != nil {
		return nil, 0, err
	}
	type acc struct {
		group *AlertEventGroup
	}
	buckets := make(map[string]*acc)
	for _, ev := range items {
		if ev == nil {
			continue
		}
		ts := time.Time(ev.TriggeredAt)
		slot := ts.Unix() / int64(windowMinutes*60)
		fp := buildAlertFingerprint(ev)
		key := fmt.Sprintf("%s:%d", fp, slot)
		existed, ok := buckets[key]
		if !ok {
			buckets[key] = &acc{
				group: &AlertEventGroup{
					Fingerprint:    fp,
					RuleName:       ev.RuleName,
					Severity:       ev.Severity,
					Status:         ev.Status,
					TotalCount:     1,
					FirstTriggered: ev.TriggeredAt,
					LastTriggered:  ev.TriggeredAt,
					LatestEventID:  ev.ID,
					LatestHost:     firstNonEmptyAlertHost(ev),
					ServiceTreeID:  ev.ServiceTreeID,
					OwnerID:        ev.OwnerID,
				},
			}
			continue
		}
		g := existed.group
		g.TotalCount++
		if time.Time(ev.TriggeredAt).Before(time.Time(g.FirstTriggered)) {
			g.FirstTriggered = ev.TriggeredAt
		}
		if time.Time(ev.TriggeredAt).After(time.Time(g.LastTriggered)) {
			g.LastTriggered = ev.TriggeredAt
			g.LatestEventID = ev.ID
			g.LatestHost = firstNonEmptyAlertHost(ev)
			g.Status = ev.Status
		}
	}
	groups := make([]*AlertEventGroup, 0, len(buckets))
	for _, item := range buckets {
		g := item.group
		g.DurationSec = int64(time.Time(g.LastTriggered).Sub(time.Time(g.FirstTriggered)).Seconds())
		if g.DurationSec < 0 {
			g.DurationSec = 0
		}
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return time.Time(groups[i].LastTriggered).After(time.Time(groups[j].LastTriggered))
	})
	total := int64(len(groups))
	start := (q.Page - 1) * q.Size
	if start >= len(groups) {
		return []*AlertEventGroup{}, total, nil
	}
	end := start + q.Size
	if end > len(groups) {
		end = len(groups)
	}
	return groups[start:end], total, nil
}

func (s *AlertService) GetEventTimeline(id int64) (*AlertEventTimeline, error) {
	event, err := s.eventRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	related, err := s.eventRepo.ListByRuleAgent(event.RuleID, event.AgentID, 100)
	if err != nil {
		return nil, err
	}
	items := make([]AlertTimelineItem, 0, 4)
	items = append(items, AlertTimelineItem{
		Type:      "triggered",
		Timestamp: &event.TriggeredAt,
		Note:      event.Description,
	})
	if event.AcknowledgedAt != nil {
		items = append(items, AlertTimelineItem{
			Type:      "acknowledged",
			Timestamp: event.AcknowledgedAt,
			Note:      event.AcknowledgementNote,
			Operator:  event.AcknowledgedBy,
		})
	}
	if event.ResolvedAt != nil {
		items = append(items, AlertTimelineItem{
			Type:      "resolved",
			Timestamp: event.ResolvedAt,
			Note:      event.ResolutionNote,
			Operator:  event.ResolvedBy,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Timestamp == nil {
			return false
		}
		if items[j].Timestamp == nil {
			return true
		}
		return time.Time(*items[i].Timestamp).Before(time.Time(*items[j].Timestamp))
	})
	return &AlertEventTimeline{
		EventID:      event.ID,
		RuleName:     event.RuleName,
		Severity:     event.Severity,
		Status:       event.Status,
		RelatedCount: len(related),
		Items:        items,
	}, nil
}

func (s *AlertService) AnalyzeRootCause(id int64) (*AlertRootCauseResult, error) {
	event, err := s.eventRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	related, err := s.eventRepo.ListByRuleAgent(event.RuleID, event.AgentID, 20)
	if err != nil {
		return nil, err
	}
	evidence := []string{
		fmt.Sprintf("规则=%s", event.RuleName),
		fmt.Sprintf("主机=%s", firstNonEmptyAlertHost(event)),
		fmt.Sprintf("指标=%s 当前值=%.2f 阈值=%.2f", event.MetricType, event.MetricValue, event.Threshold),
	}
	confidence := 0.65
	if len(related) >= 3 {
		confidence = 0.8
		evidence = append(evidence, fmt.Sprintf("最近存在 %d 条同规则同主机事件，具备持续性特征", len(related)))
	}
	if event.ServiceTreeID > 0 {
		confidence += 0.05
		evidence = append(evidence, fmt.Sprintf("已关联服务树 ID=%d", event.ServiceTreeID))
	}
	if confidence > 0.95 {
		confidence = 0.95
	}
	return &AlertRootCauseResult{
		EventID:           event.ID,
		PrimarySuspect:    "rule_agent_hotspot",
		Confidence:        confidence,
		Evidence:          evidence,
		RelatedEventCount: len(related),
		ServiceTreeID:     event.ServiceTreeID,
		OwnerID:           event.OwnerID,
	}, nil
}

func (s *AlertService) GetEventContext(id int64) (*AlertEventContext, error) {
	event, err := s.eventRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	suggestions := make([]string, 0, 4)
	if event.Status == model.AlertEventStatusFiring {
		suggestions = append(suggestions, "优先确认是否为持续抖动告警，必要时使用静默策略。")
	}
	if event.TaskExecutionID > 0 {
		suggestions = append(suggestions, fmt.Sprintf("检查自动修复任务执行记录：task_execution_id=%d", event.TaskExecutionID))
	}
	if event.TicketID > 0 {
		suggestions = append(suggestions, fmt.Sprintf("跟进关联工单处理进度：ticket_id=%d", event.TicketID))
	}
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "建议查看近 5 分钟同规则同主机事件，确认是否需升级告警级别。")
	}
	return &AlertEventContext{
		EventID:         event.ID,
		RuleName:        event.RuleName,
		Status:          event.Status,
		Severity:        event.Severity,
		Action:          event.Action,
		TaskExecutionID: event.TaskExecutionID,
		TicketID:        event.TicketID,
		ServiceTreeID:   event.ServiceTreeID,
		OwnerID:         event.OwnerID,
		Suggestions:     suggestions,
	}, nil
}

// AcknowledgeEvent 允许人工确认当前仍在 firing/acknowledged 状态的告警。
func (s *AlertService) AcknowledgeEvent(id int64, operator int64, note string) error {
	event, err := s.eventRepo.GetByID(id)
	if err != nil {
		return err
	}
	if event.Status == model.AlertEventStatusResolved {
		return errors.New("事件已经结束，无法确认")
	}
	now := model.LocalTime(time.Now())
	event.Status = model.AlertEventStatusAcknowledged
	event.AcknowledgedBy = operator
	event.AcknowledgedAt = &now
	event.AcknowledgementNote = note
	return s.eventRepo.Update(event)
}

// ResolveEvent 支持人工关闭告警事件。
func (s *AlertService) ResolveEvent(id int64, operator int64, note string) error {
	event, err := s.eventRepo.GetByID(id)
	if err != nil {
		return err
	}
	if event.Status == model.AlertEventStatusResolved {
		return nil
	}
	now := model.LocalTime(time.Now())
	event.Status = model.AlertEventStatusResolved
	event.ResolvedAt = &now
	event.ResolvedBy = operator
	event.ResolutionNote = note
	return s.eventRepo.Update(event)
}

func (s *AlertService) EvaluateAll() (*AlertEvaluationSummary, error) {
	_ = s.agentRepo.MarkStaleOffline(45 * time.Second)

	rules, err := s.ruleRepo.ListEnabled()
	if err != nil {
		return nil, err
	}
	summary := &AlertEvaluationSummary{
		RuleCount: len(rules),
	}

	const batchSize = 200
	for page := 1; ; page++ {
		agents, _, err := s.agentRepo.List(page, batchSize, "")
		if err != nil {
			return nil, err
		}
		if len(agents) == 0 {
			break
		}
		summary.AgentCount += len(agents)
		for _, agent := range agents {
			for _, rule := range rules {
				outcome, err := s.evaluateRule(agent, rule)
				if err != nil {
					summary.ErrorCount++
					logger.Warn("evaluate alert rule failed", zap.String("agent_id", agent.AgentID), zap.Int64("rule_id", rule.ID), zap.Error(err))
					continue
				}
				switch outcome {
				case "triggered":
					summary.TriggeredCount++
				case "resolved":
					summary.ResolvedCount++
				case "updated":
					summary.UpdatedCount++
				}
			}
		}
		if len(agents) < batchSize {
			break
		}
	}
	return summary, nil
}

func (s *AlertService) evaluateRule(agent *model.AgentInfo, rule *model.AlertRule) (string, error) {
	value := metricValueByType(agent, rule.MetricType)
	triggered := compareMetric(value, rule.Operator, rule.Threshold)
	event, err := s.eventRepo.FindOpenByRuleAgent(rule.ID, agent.AgentID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	now := model.LocalTime(time.Now())
	var userIDs []int64
	var channels []string
	var notifyConfig map[string]WebhookTarget
	var sendResolved int8 = 1

	// 发送组模式 vs 简单模式
	if rule.NotifyGroupID > 0 {
		groupRepo := repository.NewNotifyGroupRepository()
		group, groupErr := groupRepo.GetByID(rule.NotifyGroupID)
		if groupErr == nil {
			_ = json.Unmarshal([]byte(group.NotifyUserIDs), &userIDs)
			notifyConfig = ResolveGroupWebhookTargets(group.WebhooksJSON)
			sendResolved = group.SendResolved
		}
	} else {
		_ = json.Unmarshal([]byte(rule.NotifyUserIDs), &userIDs)
		_ = json.Unmarshal([]byte(rule.NotifyChannels), &channels)
		notifyConfig = ParseNotifyConfig(rule.NotifyConfig)
	}
	if rule.OnCallScheduleID > 0 {
		if oncallIDs, oncallErr := s.oncallSvc.CurrentUserIDs(rule.OnCallScheduleID, time.Now()); oncallErr == nil {
			userIDs = dedupeInt64s(append(userIDs, oncallIDs...))
		}
	}
	asset := s.matchAgentAsset(agent)
	if s.isSilenced(rule, agent, asset, time.Now()) {
		return "suppressed", nil
	}

	switch {
	case triggered && errors.Is(err, gorm.ErrRecordNotFound):
		var notificationEventID int64
		if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
			alertEvent := &model.AlertEvent{
				RuleID:        rule.ID,
				RuleName:      rule.Name,
				AgentID:       agent.AgentID,
				Hostname:      agent.Hostname,
				IP:            agent.IP,
				MetricType:    rule.MetricType,
				MetricValue:   value,
				Threshold:     rule.Threshold,
				Operator:      rule.Operator,
				Severity:      rule.Severity,
				Action:        rule.Action,
				ServiceTreeID: rule.ServiceTreeID,
				OwnerID:       rule.OwnerID,
				Status:        model.AlertEventStatusFiring,
				Description:   rule.Description,
				TriggeredAt:   now,
				LastNotifyAt:  &now,
			}
			if err := tx.Create(alertEvent).Error; err != nil {
				return err
			}
			if err := s.applyRuleAction(tx, rule, agent, alertEvent); err != nil {
				return err
			}
			eventID, notifyErr := s.notifySvc.PublishTx(tx, NotificationPublishRequest{
				EventType:    "alert_firing",
				BizType:      "alert_event",
				BizID:        alertEvent.ID,
				Title:        fmt.Sprintf("告警触发：%s", rule.Name),
				Content:      fmt.Sprintf("%s(%s) %s = %.2f", agent.Hostname, agent.IP, rule.MetricType, value),
				Level:        severityToNotifyLevel(rule.Severity),
				UserIDs:      userIDs,
				Channels:     channels,
				NotifyConfig: notifyConfig,
				Payload: map[string]interface{}{
					"alert_event_id": alertEvent.ID,
					"rule_id":        rule.ID,
					"rule_name":      rule.Name,
					"agent_id":       agent.AgentID,
					"hostname":       agent.Hostname,
					"ip":             agent.IP,
					"metric_type":    rule.MetricType,
					"metric_value":   value,
					"threshold":      rule.Threshold,
				},
			})
			if notifyErr != nil {
				return notifyErr
			}
			notificationEventID = eventID
			alertEvent.NotificationEventID = eventID
			return tx.Save(alertEvent).Error
		}); err != nil {
			return "", err
		}
		if notificationEventID > 0 {
			s.notifySvc.DispatchEventAsync(notificationEventID)
		}
		return "triggered", nil
	case triggered && event != nil:
		event.MetricValue = value
		if err := s.eventRepo.Update(event); err != nil {
			return "", err
		}
		return "updated", nil
	case !triggered && event != nil:
		var notificationEventID int64
		if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
			event.Status = model.AlertEventStatusResolved
			event.MetricValue = value
			event.ResolvedAt = &now
			event.ResolvedBy = 0
			if event.ResolutionNote == "" {
				event.ResolutionNote = "指标恢复"
			}
			if err := tx.Save(event).Error; err != nil {
				return err
			}
			// 发送组可控制是否发恢复通知
			if sendResolved == 0 {
				return nil
			}
			eventID, notifyErr := s.notifySvc.PublishTx(tx, NotificationPublishRequest{
				EventType:    "alert_resolved",
				BizType:      "alert_event",
				BizID:        event.ID,
				Title:        fmt.Sprintf("告警恢复：%s", rule.Name),
				Content:      fmt.Sprintf("%s(%s) %s 已恢复到 %.2f", agent.Hostname, agent.IP, rule.MetricType, value),
				Level:        "info",
				UserIDs:      userIDs,
				Channels:     channels,
				NotifyConfig: notifyConfig,
				Payload: map[string]interface{}{
					"alert_event_id": event.ID,
					"rule_id":        rule.ID,
					"rule_name":      rule.Name,
					"agent_id":       agent.AgentID,
					"hostname":       agent.Hostname,
					"ip":             agent.IP,
					"metric_type":    rule.MetricType,
					"metric_value":   value,
					"threshold":      rule.Threshold,
				},
			})
			if notifyErr != nil {
				return notifyErr
			}
			notificationEventID = eventID
			return nil
		}); err != nil {
			return "", err
		}
		if notificationEventID > 0 {
			s.notifySvc.DispatchEventAsync(notificationEventID)
		}
		return "resolved", nil
	default:
		return "noop", nil
	}
}

func metricValueByType(agent *model.AgentInfo, metricType string) float64 {
	switch metricType {
	case "cpu_usage":
		return agent.CPUUsagePct
	case "memory_usage":
		return agent.MemoryUsagePct
	case "disk_usage":
		return agent.DiskUsagePct
	case "agent_offline":
		if agent.Status != "online" {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func compareMetric(value float64, operator string, threshold float64) bool {
	switch operator {
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	case "neq":
		return value != threshold
	default:
		return value > threshold
	}
}

func severityToNotifyLevel(severity string) string {
	switch severity {
	case "critical":
		return "error"
	case "warning":
		return "warning"
	case "info":
		return "info"
	default:
		return "warning"
	}
}

func normalizeAlertRule(item *model.AlertRule) {
	item.Name = strings.TrimSpace(item.Name)
	item.MetricType = strings.TrimSpace(item.MetricType)
	item.Operator = strings.TrimSpace(item.Operator)
	item.Severity = strings.TrimSpace(item.Severity)
	item.Description = strings.TrimSpace(item.Description)
	item.Action = strings.TrimSpace(item.Action)
	if item.Operator == "" {
		item.Operator = "gt"
	}
	if item.Severity == "" {
		item.Severity = "warning"
	}
	if item.Enabled != 0 {
		item.Enabled = 1
	}
	if item.NotifyUserIDs == "" {
		item.NotifyUserIDs = "[]"
	}
	if item.Action == "" {
		item.Action = model.AlertRuleActionNotifyOnly
	}
	item.NotifyChannels = normalizeAlertChannels(item.NotifyChannels)
}

func validateAlertRule(item *model.AlertRule) error {
	if item.Name == "" {
		return errors.New("规则名称不能为空")
	}
	if item.MetricType == "" {
		return errors.New("监控项不能为空")
	}
	switch item.MetricType {
	case "cpu_usage", "memory_usage", "disk_usage", "agent_offline":
	default:
		return errors.New("不支持的监控项")
	}
	switch item.Operator {
	case "gt", "gte", "lt", "lte", "eq", "neq":
	default:
		return errors.New("不支持的比较运算符")
	}
	switch item.Severity {
	case "info", "warning", "critical":
	default:
		return errors.New("不支持的告警级别")
	}
	switch item.Action {
	case "", model.AlertRuleActionNotifyOnly:
		item.Action = model.AlertRuleActionNotifyOnly
	case model.AlertRuleActionCreateTicket:
		// 工单类型允许为空，未配置时走通用 incident 工单
	case model.AlertRuleActionExecuteTask:
		if item.RepairTaskID == 0 {
			return errors.New("修复任务未配置")
		}
	default:
		return errors.New("不支持的规则动作")
	}
	if item.Enabled != 0 && item.Enabled != 1 {
		return errors.New("启用状态无效")
	}
	if item.MetricType != "agent_offline" && (item.Threshold < 0 || item.Threshold > 100) {
		return errors.New("阈值范围必须在 0-100 之间")
	}
	if item.NotifyChannels != "" && item.NotifyChannels != "[]" {
		var channels []string
		if err := json.Unmarshal([]byte(item.NotifyChannels), &channels); err != nil {
			return errors.New("通知渠道配置不合法")
		}
		for _, channel := range channels {
			switch channel {
			case "in_app", "email", "webhook":
			default:
				return errors.New("存在不支持的通知渠道")
			}
		}
	}
	return nil
}

func normalizeAlertChannels(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "[]"
	}
	var channels []string
	if err := json.Unmarshal([]byte(raw), &channels); err != nil {
		return "[]"
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(channels))
	for _, channel := range channels {
		channel = strings.TrimSpace(channel)
		if channel == "" {
			continue
		}
		if _, ok := seen[channel]; ok {
			continue
		}
		seen[channel] = struct{}{}
		result = append(result, channel)
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func (s *AlertService) matchAgentAsset(agent *model.AgentInfo) *model.Asset {
	if agent == nil {
		return nil
	}
	for _, ip := range []string{agent.PublicIP, agent.PrivateIP, agent.IP} {
		if strings.TrimSpace(ip) == "" {
			continue
		}
		assets, _, err := s.assetRepo.List(repository.AssetListQuery{
			Page:    1,
			Size:    20,
			Keyword: ip,
		})
		if err != nil {
			continue
		}
		for _, asset := range assets {
			if asset.IP == ip || asset.InnerIP == ip {
				return asset
			}
		}
	}
	return nil
}

func (s *AlertService) isSilenced(rule *model.AlertRule, agent *model.AgentInfo, asset *model.Asset, now time.Time) bool {
	items, err := s.silenceRepo.ListActive(now)
	if err != nil {
		return false
	}
	for _, item := range items {
		if item.RuleID > 0 && item.RuleID != rule.ID {
			continue
		}
		if item.AgentID != "" && item.AgentID != agent.AgentID {
			continue
		}
		if item.ServiceTreeID > 0 {
			if asset == nil || asset.ServiceTreeID != item.ServiceTreeID {
				continue
			}
		}
		if item.OwnerID > 0 {
			if asset == nil || !strings.Contains(asset.OwnerIDs, strconv.FormatInt(item.OwnerID, 10)) {
				continue
			}
		}
		return true
	}
	return false
}

func dedupeInt64s(items []int64) []int64 {
	if len(items) == 0 {
		return items
	}
	seen := make(map[int64]struct{}, len(items))
	result := make([]int64, 0, len(items))
	for _, item := range items {
		if item <= 0 {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func buildAlertFingerprint(ev *model.AlertEvent) string {
	return fmt.Sprintf("%d|%s|%d|%d|%s", ev.RuleID, ev.Severity, ev.ServiceTreeID, ev.OwnerID, ev.MetricType)
}

func firstNonEmptyAlertHost(ev *model.AlertEvent) string {
	if strings.TrimSpace(ev.Hostname) != "" {
		return ev.Hostname
	}
	if strings.TrimSpace(ev.IP) != "" {
		return ev.IP
	}
	return ev.AgentID
}

func (s *AlertService) applyRuleAction(tx *gorm.DB, rule *model.AlertRule, agent *model.AgentInfo, event *model.AlertEvent) error {
	switch rule.Action {
	case model.AlertRuleActionCreateTicket:
		if err := s.createAlertTicket(rule, agent, event); err != nil {
			return err
		}
	case model.AlertRuleActionExecuteTask:
		if err := s.executeRepairTask(rule, agent, event); err != nil {
			return err
		}
	default:
		return nil
	}
	return tx.Save(event).Error
}

func (s *AlertService) createAlertTicket(rule *model.AlertRule, agent *model.AgentInfo, event *model.AlertEvent) error {
	ticket := &model.Ticket{
		Title:           fmt.Sprintf("告警自动建单：%s", rule.Name),
		TypeID:          rule.TicketTypeID,
		Description:     fmt.Sprintf("主机 %s(%s) 告警 %s=%.2f", agent.Hostname, agent.AgentID, rule.MetricType, event.MetricValue),
		Source:          "monitor",
		SourceEventType: "alert",
		SourceEventID:   fmt.Sprintf("%d", event.ID),
		ServiceTreeID:   rule.ServiceTreeID,
		SubmitDeptID:    0,
		HandleDeptID:    0,
		CreatorID:       0,
	}
	if err := s.ticketSvc.Create(ticket, 0, "alert_scheduler"); err != nil {
		return err
	}
	event.TicketID = ticket.ID
	if rule.ServiceTreeID > 0 {
		event.ServiceTreeID = rule.ServiceTreeID
	}
	if rule.OwnerID > 0 {
		event.OwnerID = rule.OwnerID
	}
	return nil
}

func (s *AlertService) executeRepairTask(rule *model.AlertRule, agent *model.AgentInfo, event *model.AlertEvent) error {
	if rule.RepairTaskID == 0 {
		return errors.New("修复任务未配置")
	}
	hosts := s.selectRepairHosts(agent)
	if len(hosts) == 0 {
		return errors.New("无法确定目标主机")
	}
	exec, err := s.taskSvc.ExecuteTask(rule.RepairTaskID, hosts, 0)
	if err != nil {
		return err
	}
	event.TaskExecutionID = exec.ID
	if rule.ServiceTreeID > 0 {
		event.ServiceTreeID = rule.ServiceTreeID
	}
	if rule.OwnerID > 0 {
		event.OwnerID = rule.OwnerID
	}
	return nil
}

func (s *AlertService) selectRepairHosts(agent *model.AgentInfo) []string {
	hosts := make([]string, 0, 1)
	for _, ip := range []string{agent.PrivateIP, agent.PublicIP, agent.IP} {
		if ip == "" {
			continue
		}
		duplicated := false
		for _, host := range hosts {
			if host == ip {
				duplicated = true
				break
			}
		}
		if !duplicated {
			hosts = append(hosts, ip)
		}
	}
	if len(hosts) == 0 && agent.Hostname != "" {
		hosts = append(hosts, agent.Hostname)
	}
	if len(hosts) == 0 && agent.AgentID != "" {
		hosts = append(hosts, agent.AgentID)
	}
	return hosts
}

type AlertScheduler struct {
	stop chan struct{}
}

func NewAlertScheduler() *AlertScheduler {
	return &AlertScheduler{stop: make(chan struct{})}
}

func (s *AlertScheduler) Start() {
	ticker := time.NewTicker(30 * time.Second)
	safego.Go(func() {
		defer ticker.Stop()
		alertSvc := NewAlertService()
		for {
			select {
			case <-s.stop:
				return
			case <-ticker.C:
				if _, err := alertSvc.EvaluateAll(); err != nil {
					logger.Warn("alert scheduler evaluate failed", zap.Error(err))
				}
			}
		}
	})
}

func (s *AlertScheduler) Stop() {
	close(s.stop)
}
