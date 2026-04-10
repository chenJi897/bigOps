package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/config"
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
	taskRepo       *repository.TaskRepository
	taskExecRepo   *repository.TaskExecutionRepository
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
		taskRepo:       repository.NewTaskRepository(),
		taskExecRepo:   repository.NewTaskExecutionRepository(),
	}
}

type GoldenSignalsSummary struct {
	WindowMinutes         int     `json:"window_minutes"`
	AvailabilityPct       float64 `json:"availability_pct"`
	ErrorRatePct          float64 `json:"error_rate_pct"`
	AvgLatencyMs          float64 `json:"avg_latency_ms"`
	ThroughputPerMinute   float64 `json:"throughput_per_minute"`
	TotalRequests         int64   `json:"total_requests"`
	TotalErrors           int64   `json:"total_errors"`
	SLOTargetAvailability float64 `json:"slo_target_availability"`
	SLOTargetLatencyMs    float64 `json:"slo_target_latency_ms"`
	SLOBreached           bool    `json:"slo_breached"`
	AgentCoverage         int     `json:"agent_coverage"`
	AgentTotal            int     `json:"agent_total"`
	CoverageRatePct       float64 `json:"coverage_rate_pct"`
	ConfidenceLevel       string  `json:"confidence_level"`
}

type GoldenSignalDimensionItem struct {
	DimensionType string  `json:"dimension_type"`
	DimensionKey  string  `json:"dimension_key"`
	DimensionName string  `json:"dimension_name"`
	TotalRequests int64   `json:"total_requests"`
	TotalErrors   int64   `json:"total_errors"`
	ErrorRatePct  float64 `json:"error_rate_pct"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
}

type MonitorAggregateItem struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	AgentTotal     int     `json:"agent_total"`
	OnlineTotal    int     `json:"online_total"`
	OfflineTotal   int     `json:"offline_total"`
	AvgCPUUsagePct float64 `json:"avg_cpu_usage_pct"`
	AvgMemoryPct   float64 `json:"avg_memory_usage_pct"`
	AvgDiskPct     float64 `json:"avg_disk_usage_pct"`
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

func (s *MonitorService) GoldenSignals(windowMinutes int) (*GoldenSignalsSummary, error) {
	if windowMinutes <= 0 {
		windowMinutes = 60
	}
	since := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)

	_ = s.agentRepo.MarkStaleOffline(45 * time.Second)
	agents, _, _ := s.agentRepo.List(1, 10000, "")
	onlineCount := int64(0)
	totalAgents := int64(len(agents))
	for _, a := range agents {
		if a.Status == "online" {
			onlineCount++
		}
	}

	threshold := 80.0
	rows, err := s.sampleRepo.AggregateByWindow(since, threshold)
	if err != nil {
		return nil, err
	}

	var totalSamples int64
	var errorSamples int64
	var latencyMs float64
	hasLatency := false
	for _, row := range rows {
		if row.MetricType == "latency" {
			latencyMs = row.AvgValue
			hasLatency = true
			continue
		}
		totalSamples += row.SampleCnt
		errorSamples += row.OverThresh
	}
	if !hasLatency {
		for _, row := range rows {
			if row.MetricType == "cpu_usage" {
				latencyMs = row.AvgValue
				break
			}
		}
	}

	availability := 100.0
	if totalAgents > 0 {
		availability = float64(onlineCount) / float64(totalAgents) * 100
	}
	errorRate := 0.0
	if totalSamples > 0 {
		errorRate = float64(errorSamples) / float64(totalSamples) * 100
	}
	throughput := 0.0
	if windowMinutes > 0 {
		throughput = float64(totalSamples) / float64(windowMinutes)
	}

	sloAvail := s.getSLOAvailability()
	sloLatency := s.getSLOLatencyMs()

	agentTotal := 0
	agentCoverage := 0
	if _, total, err := s.agentRepo.List(1, 1, ""); err == nil {
		agentTotal = int(total)
	}
	if _, onlineTotal, err := s.agentRepo.List(1, 1, "online"); err == nil {
		agentCoverage = int(onlineTotal)
	}
	coverageRate := 0.0
	if agentTotal > 0 {
		coverageRate = float64(agentCoverage) / float64(agentTotal) * 100
	}
	confidence := "high"
	if totalSamples < 10 {
		confidence = "low"
	} else if coverageRate < 80 || totalSamples < 100 {
		confidence = "medium"
	}

	return &GoldenSignalsSummary{
		WindowMinutes:         windowMinutes,
		AvailabilityPct:       round2(availability),
		ErrorRatePct:          round2(errorRate),
		AvgLatencyMs:          round2(latencyMs),
		ThroughputPerMinute:   round2(throughput),
		TotalRequests:         totalSamples,
		TotalErrors:           errorSamples,
		SLOTargetAvailability: sloAvail,
		SLOTargetLatencyMs:    sloLatency,
		SLOBreached:           availability < sloAvail || latencyMs > sloLatency,
		AgentCoverage:         agentCoverage,
		AgentTotal:            agentTotal,
		CoverageRatePct:       round2(coverageRate),
		ConfidenceLevel:       confidence,
	}, nil
}

func (s *MonitorService) getSLOAvailability() float64 {
	cfg := config.Get()
	if cfg.SLO.TargetAvailability > 0 {
		return cfg.SLO.TargetAvailability
	}
	return 99.9
}

func (s *MonitorService) getSLOLatencyMs() float64 {
	cfg := config.Get()
	if cfg.SLO.TargetLatencyMs > 0 {
		return cfg.SLO.TargetLatencyMs
	}
	return 3000
}

func (s *MonitorService) GoldenSignalsByDimension(windowMinutes int, dimension string) ([]GoldenSignalDimensionItem, error) {
	if windowMinutes <= 0 {
		windowMinutes = 60
	}
	since := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)
	dimType := normalizeGoldenDimension(dimension)

	sampleDim := "ip"
	switch dimType {
	case "service":
		sampleDim = "service"
	case "instance":
		sampleDim = "instance"
	case "interface", "metric_type":
		sampleDim = "metric_type"
	default:
		sampleDim = "ip"
	}

	rows, err := s.sampleRepo.AggregateByDimension(since, sampleDim, 80.0)
	if err != nil {
		return nil, err
	}

	type agg struct {
		total      int64
		errors     int64
		avgValSum  float64
		avgValCnt  int64
	}
	grouped := map[string]*agg{}
	for _, row := range rows {
		key := row.DimensionKey
		item := grouped[key]
		if item == nil {
			item = &agg{}
			grouped[key] = item
		}
		item.total += row.SampleCnt
		item.errors += row.OverThresh
		item.avgValSum += row.AvgValue * float64(row.SampleCnt)
		item.avgValCnt += row.SampleCnt
	}

	agentNameMap := map[string]string{}
	if dimType == "instance" || dimType == "service" {
		agents, _, _ := s.agentRepo.List(1, 10000, "")
		for _, a := range agents {
			agentNameMap[a.AgentID] = firstNonEmptyString(a.Hostname, a.IP, a.AgentID)
			agentNameMap[a.IP] = firstNonEmptyString(a.Hostname, a.IP)
		}
	}

	metricTypeLabel := map[string]string{
		"cpu_usage":    "CPU 使用率",
		"memory_usage": "内存使用率",
		"disk_usage":   "磁盘使用率",
		"latency":      "心跳延迟",
	}

	result := make([]GoldenSignalDimensionItem, 0, len(grouped))
	for key, item := range grouped {
		er := 0.0
		if item.total > 0 {
			er = float64(item.errors) / float64(item.total) * 100
		}
		avg := 0.0
		if item.avgValCnt > 0 {
			avg = item.avgValSum / float64(item.avgValCnt)
		}
		displayName := key
		switch dimType {
		case "instance":
			if name, ok := agentNameMap[key]; ok {
				displayName = name
			}
		case "service":
			if name, ok := agentNameMap[key]; ok && displayName == key {
				displayName = name
			}
		case "interface":
			if label, ok := metricTypeLabel[key]; ok {
				displayName = label
			}
		}
		result = append(result, GoldenSignalDimensionItem{
			DimensionType: dimType,
			DimensionKey:  key,
			DimensionName: displayName,
			TotalRequests: item.total,
			TotalErrors:   item.errors,
			ErrorRatePct:  round2(er),
			AvgLatencyMs:  round2(avg),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalRequests > result[j].TotalRequests
	})
	return result, nil
}

func normalizeGoldenDimension(dimension string) string {
	switch dimension {
	case "service", "interface", "instance", "operator":
		return dimension
	default:
		return "service"
	}
}

func (s *MonitorService) resolveTaskName(taskID int64) string {
	if taskID <= 0 {
		return "未知任务"
	}
	task, err := s.taskRepo.GetTask(taskID)
	if err != nil || task == nil {
		return "任务#" + strconv.FormatInt(taskID, 10)
	}
	return firstNonEmptyString(task.Name, "任务#"+strconv.FormatInt(taskID, 10))
}

func (s *MonitorService) resolveTaskType(taskID int64) string {
	if taskID <= 0 {
		return "unknown"
	}
	task, err := s.taskRepo.GetTask(taskID)
	if err != nil || task == nil || task.TaskType == "" {
		return "unknown"
	}
	return task.TaskType
}

func (s *MonitorService) resolveTaskTypeLabel(taskID int64) string {
	switch s.resolveTaskType(taskID) {
	case "shell":
		return "Shell 脚本"
	case "python":
		return "Python 脚本"
	case "file_transfer":
		return "文件分发"
	default:
		return "未知类型"
	}
}

func (s *MonitorService) resolveOperatorName(userID int64) string {
	if userID <= 0 {
		return "系统"
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return "用户#" + strconv.FormatInt(userID, 10)
	}
	return firstNonEmptyString(user.RealName, user.Username, "用户#"+strconv.FormatInt(userID, 10))
}

// GetSLOConfig returns the current SLO configuration.
func (s *MonitorService) GetSLOConfig() map[string]float64 {
	return map[string]float64{
		"target_availability": s.getSLOAvailability(),
		"target_latency_ms":  s.getSLOLatencyMs(),
	}
}

// UpdateSLOConfig updates SLO configuration in config.yaml (runtime only for now).
func (s *MonitorService) UpdateSLOConfig(availability, latencyMs float64) {
	cfg := config.Get()
	if availability > 0 {
		cfg.SLO.TargetAvailability = availability
	}
	if latencyMs > 0 {
		cfg.SLO.TargetLatencyMs = latencyMs
	}
}

// AnomalyDetection computes baseline stats and detects anomalies.
type AnomalyItem struct {
	AgentID    string  `json:"agent_id"`
	Hostname   string  `json:"hostname"`
	IP         string  `json:"ip"`
	MetricType string  `json:"metric_type"`
	Current    float64 `json:"current_value"`
	Baseline   float64 `json:"baseline_avg"`
	StdDev     float64 `json:"std_dev"`
	ZScore     float64 `json:"z_score"`
	Category   string  `json:"category"` // spike/drop/sustained_high/capacity_risk
}

func (s *MonitorService) DetectAnomalies(stddevMultiplier float64) ([]AnomalyItem, error) {
	if stddevMultiplier <= 0 {
		stddevMultiplier = 2.0
	}
	rows, err := s.sampleRepo.AggregateBaselineByAgent()
	if err != nil {
		return nil, err
	}
	agents, _, _ := s.agentRepo.List(1, 10000, "")
	agentMap := make(map[string]*model.AgentInfo, len(agents))
	for _, a := range agents {
		agentMap[a.AgentID] = a
	}

	var anomalies []AnomalyItem
	for _, row := range rows {
		if row.StdDev == 0 || row.SampleCnt < 10 {
			continue
		}
		agent := agentMap[row.AgentID]
		if agent == nil {
			continue
		}
		current := metricCurrentValue(agent, row.MetricType)
		zScore := (current - row.AvgValue) / row.StdDev
		if zScore > stddevMultiplier || zScore < -stddevMultiplier {
			cat := classifyAnomaly(row.MetricType, current, row.AvgValue, zScore)
			anomalies = append(anomalies, AnomalyItem{
				AgentID:    row.AgentID,
				Hostname:   agent.Hostname,
				IP:         agent.IP,
				MetricType: row.MetricType,
				Current:    round2(current),
				Baseline:   round2(row.AvgValue),
				StdDev:     round2(row.StdDev),
				ZScore:     round2(zScore),
				Category:   cat,
			})
		}
	}
	return anomalies, nil
}

func metricCurrentValue(agent *model.AgentInfo, metricType string) float64 {
	switch metricType {
	case "cpu_usage":
		return agent.CPUUsagePct
	case "memory_usage":
		return agent.MemoryUsagePct
	case "disk_usage":
		return agent.DiskUsagePct
	case "latency":
		return agent.LatencyMs
	default:
		return 0
	}
}

// CapacityPrediction predicts when a metric will reach threshold based on linear regression.
type CapacityPrediction struct {
	AgentID      string  `json:"agent_id"`
	Hostname     string  `json:"hostname"`
	MetricType   string  `json:"metric_type"`
	CurrentValue float64 `json:"current_value"`
	TrendPerDay  float64 `json:"trend_per_day"`
	DaysToFull   float64 `json:"days_to_full"`
	Threshold    float64 `json:"threshold"`
}

func (s *MonitorService) PredictCapacity(metricType string, threshold float64) ([]CapacityPrediction, error) {
	if metricType == "" {
		metricType = "disk_usage"
	}
	if threshold <= 0 {
		threshold = 90.0
	}
	agents, _, _ := s.agentRepo.List(1, 10000, "")
	since := time.Now().Add(-7 * 24 * time.Hour)
	var predictions []CapacityPrediction
	for _, agent := range agents {
		samples, err := s.sampleRepo.ListTrend(agent.AgentID, metricType, &since, nil, 500)
		if err != nil || len(samples) < 5 {
			continue
		}
		sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
		n := float64(len(samples))
		baseTime := time.Time(samples[0].CollectedAt)
		for _, sample := range samples {
			x := time.Time(sample.CollectedAt).Sub(baseTime).Hours() / 24.0
			y := sample.MetricValue
			sumX += x
			sumY += y
			sumXY += x * y
			sumX2 += x * x
		}
		denom := n*sumX2 - sumX*sumX
		if denom == 0 {
			continue
		}
		slope := (n*sumXY - sumX*sumY) / denom
		intercept := (sumY - slope*sumX) / n
		currentValue := samples[len(samples)-1].MetricValue
		if slope <= 0 || currentValue >= threshold {
			continue
		}
		currentX := time.Since(baseTime).Hours() / 24.0
		targetX := (threshold - intercept) / slope
		daysToFull := targetX - currentX
		if daysToFull < 0 || daysToFull > 365 {
			continue
		}
		predictions = append(predictions, CapacityPrediction{
			AgentID:      agent.AgentID,
			Hostname:     agent.Hostname,
			MetricType:   metricType,
			CurrentValue: round2(currentValue),
			TrendPerDay:  round2(slope),
			DaysToFull:   round2(daysToFull),
			Threshold:    threshold,
		})
	}
	return predictions, nil
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func classifyAnomaly(metricType string, current, baseline, zScore float64) string {
	if current > 90 && (metricType == "cpu_usage" || metricType == "memory_usage" || metricType == "disk_usage") {
		return "capacity_risk"
	}
	if zScore > 3 {
		return "spike"
	}
	if zScore < -3 {
		return "drop"
	}
	if current > baseline*1.3 {
		return "sustained_high"
	}
	if zScore > 0 {
		return "spike"
	}
	return "drop"
}
