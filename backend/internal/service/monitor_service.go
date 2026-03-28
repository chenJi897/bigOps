package service

import (
	"context"
	"errors"
	"encoding/json"
	"strconv"
	"sort"
	"time"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type MonitorSummary struct {
	AgentTotal          int64                `json:"agent_total"`
	AgentOnline         int64                `json:"agent_online"`
	AgentOffline        int64                `json:"agent_offline"`
	RuleEnabledTotal    int64                `json:"rule_enabled_total"`
	AlertFiringTotal    int64                `json:"alert_firing_total"`
	AlertStatusCounts   []AlertStatusCount   `json:"alert_status_counts"`
	AlertSeverityCounts []AlertSeverityCount `json:"alert_severity_counts"`
	RecentAlerts        []AlertEventSummary  `json:"recent_alerts"`
	LastCollectedAt     *model.LocalTime     `json:"last_collected_at"`
	CPUHighAgents       []model.AgentInfo    `json:"cpu_high_agents"`
	MemoryHighAgents    []model.AgentInfo    `json:"memory_high_agents"`
	DiskHighAgents      []model.AgentInfo    `json:"disk_high_agents"`
}

type AlertStatusCount struct {
	Status string `json:"status"`
	Total  int64  `json:"total"`
}

type AlertSeverityCount struct {
	Severity string `json:"severity"`
	Total    int64  `json:"total"`
}

type AlertEventSummary struct {
	ID             int64            `json:"id"`
	RuleName       string           `json:"rule_name"`
	AgentID        string           `json:"agent_id"`
	Hostname       string           `json:"hostname"`
	IP             string           `json:"ip"`
	MetricType     string           `json:"metric_type"`
	MetricValue    float64          `json:"metric_value"`
	Severity       string           `json:"severity"`
	Status         string           `json:"status"`
	TriggeredAt    model.LocalTime  `json:"triggered_at"`
	AcknowledgedAt *model.LocalTime `json:"acknowledged_at,omitempty"`
	ResolvedAt     *model.LocalTime `json:"resolved_at,omitempty"`
	Note           string           `json:"note"`
}

type MonitorService struct {
	agentRepo      *repository.AgentRepository
	ruleRepo       *repository.AlertRuleRepository
	eventRepo      *repository.AlertEventRepository
	sampleRepo     *repository.AgentMetricSampleRepository
	datasourceRepo *repository.MonitorDatasourceRepository
	assetRepo      *repository.AssetRepository
	treeRepo       *repository.ServiceTreeRepository
	userRepo       *repository.UserRepository
}

func NewMonitorService() *MonitorService {
	return &MonitorService{
		agentRepo:      repository.NewAgentRepository(),
		ruleRepo:       repository.NewAlertRuleRepository(),
		eventRepo:      repository.NewAlertEventRepository(),
		sampleRepo:     repository.NewAgentMetricSampleRepository(),
		datasourceRepo: repository.NewMonitorDatasourceRepository(),
		assetRepo:      repository.NewAssetRepository(),
		treeRepo:       repository.NewServiceTreeRepository(),
		userRepo:       repository.NewUserRepository(),
	}
}

type MonitorAggregateItem struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	AgentTotal      int     `json:"agent_total"`
	OnlineTotal     int     `json:"online_total"`
	OfflineTotal    int     `json:"offline_total"`
	AvgCPUUsagePct  float64 `json:"avg_cpu_usage_pct"`
	AvgMemoryPct    float64 `json:"avg_memory_usage_pct"`
	AvgDiskPct      float64 `json:"avg_disk_usage_pct"`
}

func (s *MonitorService) Summary() (*MonitorSummary, error) {
	_ = s.agentRepo.MarkStaleOffline(45 * time.Second)
	agents, _, err := s.agentRepo.List(1, 500, "")
	if err != nil {
		return nil, err
	}
	summary := &MonitorSummary{}
	summary.AgentTotal = int64(len(agents))
	for _, agent := range agents {
		if agent.Status == "online" {
			summary.AgentOnline++
		} else {
			summary.AgentOffline++
		}
		if agent.CPUUsagePct >= 80 {
			summary.CPUHighAgents = append(summary.CPUHighAgents, *agent)
		}
		if agent.MemoryUsagePct >= 80 {
			summary.MemoryHighAgents = append(summary.MemoryHighAgents, *agent)
		}
		if agent.DiskUsagePct >= 80 {
			summary.DiskHighAgents = append(summary.DiskHighAgents, *agent)
		}
	}
	sortAgentMetrics(summary.CPUHighAgents, func(item model.AgentInfo) float64 { return item.CPUUsagePct })
	sortAgentMetrics(summary.MemoryHighAgents, func(item model.AgentInfo) float64 { return item.MemoryUsagePct })
	sortAgentMetrics(summary.DiskHighAgents, func(item model.AgentInfo) float64 { return item.DiskUsagePct })
	summary.CPUHighAgents = limitAgents(summary.CPUHighAgents, 5)
	summary.MemoryHighAgents = limitAgents(summary.MemoryHighAgents, 5)
	summary.DiskHighAgents = limitAgents(summary.DiskHighAgents, 5)

	if total, err := s.ruleRepo.CountByEnabled(1); err == nil {
		summary.RuleEnabledTotal = total
	}
	statusCounts, err := s.eventRepo.CountByStatus()
	if err == nil {
		summary.AlertStatusCounts = buildStatusCounts(statusCounts)
		summary.AlertFiringTotal = statusCounts[model.AlertEventStatusFiring]
	} else if _, total, listErr := s.eventRepo.List(repository.AlertEventListQuery{
		Page:   1,
		Size:   20,
		Status: model.AlertEventStatusFiring,
	}); listErr == nil {
		summary.AlertFiringTotal = total
	}

	if severityCounts, err := s.eventRepo.CountBySeverity(); err == nil {
		summary.AlertSeverityCounts = buildSeverityCounts(severityCounts)
	}

	if events, err := s.eventRepo.ListLatest(5); err == nil {
		summary.RecentAlerts = buildAlertEventSummaries(events)
	}
	if latest, err := s.sampleRepo.GetLatestCollectedAt(); err == nil {
		summary.LastCollectedAt = latest
	}
	return summary, nil
}

func sortAgentMetrics(items []model.AgentInfo, score func(model.AgentInfo) float64) {
	sort.Slice(items, func(i, j int) bool {
		return score(items[i]) > score(items[j])
	})
}

func limitAgents(items []model.AgentInfo, limit int) []model.AgentInfo {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func buildStatusCounts(counts map[string]int64) []AlertStatusCount {
	order := []string{
		model.AlertEventStatusFiring,
		model.AlertEventStatusAcknowledged,
		model.AlertEventStatusResolved,
	}
	result := make([]AlertStatusCount, 0, len(order)+len(counts))
	for _, status := range order {
		result = append(result, AlertStatusCount{
			Status: status,
			Total:  counts[status],
		})
	}
	for status, total := range counts {
		if status == model.AlertEventStatusFiring ||
			status == model.AlertEventStatusAcknowledged ||
			status == model.AlertEventStatusResolved {
			continue
		}
		result = append(result, AlertStatusCount{
			Status: status,
			Total:  total,
		})
	}
	return result
}

func buildSeverityCounts(counts map[string]int64) []AlertSeverityCount {
	order := []string{"critical", "warning", "info"}
	result := make([]AlertSeverityCount, 0, len(order)+len(counts))
	seen := map[string]struct{}{}
	for _, severity := range order {
		result = append(result, AlertSeverityCount{
			Severity: severity,
			Total:    counts[severity],
		})
		seen[severity] = struct{}{}
	}
	for severity, total := range counts {
		if _, ok := seen[severity]; ok {
			continue
		}
		result = append(result, AlertSeverityCount{
			Severity: severity,
			Total:    total,
		})
	}
	return result
}

func buildAlertEventSummaries(items []*model.AlertEvent) []AlertEventSummary {
	result := make([]AlertEventSummary, 0, len(items))
	for _, item := range items {
		result = append(result, AlertEventSummary{
			ID:             item.ID,
			RuleName:       item.RuleName,
			AgentID:        item.AgentID,
			Hostname:       item.Hostname,
			IP:             item.IP,
			MetricType:     item.MetricType,
			MetricValue:    item.MetricValue,
			Severity:       item.Severity,
			Status:         item.Status,
			TriggeredAt:    item.TriggeredAt,
			AcknowledgedAt: item.AcknowledgedAt,
			ResolvedAt:     item.ResolvedAt,
			Note:           selectEventNote(item),
		})
	}
	return result
}

func selectEventNote(item *model.AlertEvent) string {
	switch item.Status {
	case model.AlertEventStatusAcknowledged:
		if item.AcknowledgementNote != "" {
			return item.AcknowledgementNote
		}
	case model.AlertEventStatusResolved:
		if item.ResolutionNote != "" {
			return item.ResolutionNote
		}
	}
	if item.Description != "" {
		return item.Description
	}
	return ""
}

func (s *MonitorService) QueryPrometheus(ctx context.Context, datasourceID int64, query string, ts time.Time) (map[string]any, error) {
	ds, err := s.datasourceRepo.GetByID(datasourceID)
	if err != nil {
		return nil, err
	}
	client := NewPrometheusClient(ds.BaseURL, ds.Username, ds.Password, parseHeaders(ds.HeadersJSON))
	return client.Query(ctx, query, ts)
}

func (s *MonitorService) QueryPrometheusRange(ctx context.Context, datasourceID int64, query string, start, end time.Time, step time.Duration) (map[string]any, error) {
	ds, err := s.datasourceRepo.GetByID(datasourceID)
	if err != nil {
		return nil, err
	}
	client := NewPrometheusClient(ds.BaseURL, ds.Username, ds.Password, parseHeaders(ds.HeadersJSON))
	return client.QueryRange(ctx, query, start, end, step)
}

func (s *MonitorService) ListAgents(page, size int, status, keyword string) ([]*model.AgentInfo, int64, error) {
	_ = s.agentRepo.MarkStaleOffline(45 * time.Second)
	return s.agentRepo.ListForMonitor(repository.AgentMonitorListQuery{
		Page:    page,
		Size:    size,
		Status:  status,
		Keyword: keyword,
	})
}

func (s *MonitorService) AgentTrend(agentID, metricType string, minutes int, limit int) ([]*model.AgentMetricSample, error) {
	if agentID == "" {
		return nil, errors.New("agent_id 不能为空")
	}
	if metricType == "" {
		return nil, errors.New("metric_type 不能为空")
	}
	var startAt *time.Time
	if minutes > 0 {
		t := time.Now().Add(-time.Duration(minutes) * time.Minute)
		startAt = &t
	}
	return s.sampleRepo.ListTrend(agentID, metricType, startAt, nil, limit)
}

func (s *MonitorService) AggregateByServiceTree() ([]MonitorAggregateItem, error) {
	_ = s.agentRepo.MarkStaleOffline(45 * time.Second)
	agents, _, err := s.agentRepo.ListForMonitor(repository.AgentMonitorListQuery{Page: 1, Size: 2000})
	if err != nil {
		return nil, err
	}
	assets, _, err := s.assetRepo.List(repository.AssetListQuery{Page: 1, Size: 5000})
	if err != nil {
		return nil, err
	}
	assetByIP := make(map[string]*model.Asset)
	for _, asset := range assets {
		if asset.IP != "" {
			assetByIP[asset.IP] = asset
		}
		if asset.InnerIP != "" {
			assetByIP[asset.InnerIP] = asset
		}
	}
	grouped := make(map[int64]*MonitorAggregateItem)
	for _, agent := range agents {
		asset := assetByIP[firstNonEmptyString(agent.PrivateIP, agent.PublicIP, agent.IP)]
		if asset == nil && agent.PublicIP != "" {
			asset = assetByIP[agent.PublicIP]
		}
		if asset == nil && agent.IP != "" {
			asset = assetByIP[agent.IP]
		}
		if asset == nil || asset.ServiceTreeID == 0 {
			continue
		}
		item := grouped[asset.ServiceTreeID]
		if item == nil {
			name := strconv.FormatInt(asset.ServiceTreeID, 10)
			if node, err := s.treeRepo.GetByID(asset.ServiceTreeID); err == nil {
				name = node.Name
			}
			item = &MonitorAggregateItem{ID: asset.ServiceTreeID, Name: name}
			grouped[asset.ServiceTreeID] = item
		}
		accumulateAggregate(item, agent)
	}
	return finalizeAggregates(grouped), nil
}

func (s *MonitorService) AggregateByOwner() ([]MonitorAggregateItem, error) {
	_ = s.agentRepo.MarkStaleOffline(45 * time.Second)
	agents, _, err := s.agentRepo.ListForMonitor(repository.AgentMonitorListQuery{Page: 1, Size: 2000})
	if err != nil {
		return nil, err
	}
	assets, _, err := s.assetRepo.List(repository.AssetListQuery{Page: 1, Size: 5000})
	if err != nil {
		return nil, err
	}
	assetByIP := make(map[string]*model.Asset)
	for _, asset := range assets {
		if asset.IP != "" {
			assetByIP[asset.IP] = asset
		}
		if asset.InnerIP != "" {
			assetByIP[asset.InnerIP] = asset
		}
	}
	grouped := make(map[int64]*MonitorAggregateItem)
	for _, agent := range agents {
		asset := assetByIP[firstNonEmptyString(agent.PrivateIP, agent.PublicIP, agent.IP)]
		if asset == nil && agent.PublicIP != "" {
			asset = assetByIP[agent.PublicIP]
		}
		if asset == nil && agent.IP != "" {
			asset = assetByIP[agent.IP]
		}
		if asset == nil || asset.OwnerIDs == "" {
			continue
		}
		var ownerIDs []int64
		_ = json.Unmarshal([]byte(asset.OwnerIDs), &ownerIDs)
		for _, ownerID := range ownerIDs {
			item := grouped[ownerID]
			if item == nil {
				name := strconv.FormatInt(ownerID, 10)
				if user, err := s.userRepo.GetByID(ownerID); err == nil {
					name = user.Username
					if user.RealName != "" {
						name = user.RealName
					}
				}
				item = &MonitorAggregateItem{ID: ownerID, Name: name}
				grouped[ownerID] = item
			}
			accumulateAggregate(item, agent)
		}
	}
	return finalizeAggregates(grouped), nil
}

func accumulateAggregate(item *MonitorAggregateItem, agent *model.AgentInfo) {
	item.AgentTotal++
	if agent.Status == "online" {
		item.OnlineTotal++
	} else {
		item.OfflineTotal++
	}
	item.AvgCPUUsagePct += agent.CPUUsagePct
	item.AvgMemoryPct += agent.MemoryUsagePct
	item.AvgDiskPct += agent.DiskUsagePct
}

func firstNonEmptyString(items ...string) string {
	for _, item := range items {
		if item != "" {
			return item
		}
	}
	return ""
}

func finalizeAggregates(grouped map[int64]*MonitorAggregateItem) []MonitorAggregateItem {
	result := make([]MonitorAggregateItem, 0, len(grouped))
	for _, item := range grouped {
		if item.AgentTotal > 0 {
			item.AvgCPUUsagePct = item.AvgCPUUsagePct / float64(item.AgentTotal)
			item.AvgMemoryPct = item.AvgMemoryPct / float64(item.AgentTotal)
			item.AvgDiskPct = item.AvgDiskPct / float64(item.AgentTotal)
		}
		result = append(result, *item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].AgentTotal > result[j].AgentTotal })
	return result
}
