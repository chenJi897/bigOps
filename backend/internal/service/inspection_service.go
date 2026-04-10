package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type InspectionService struct {
	repo      *repository.InspectionRepository
	taskSvc   *TaskService
	eventRepo *repository.AlertEventRepository
}

func NewInspectionService() *InspectionService {
	return &InspectionService{
		repo:      repository.NewInspectionRepository(),
		taskSvc:   NewTaskService(),
		eventRepo: repository.NewAlertEventRepository(),
	}
}

func (s *InspectionService) ListTemplates(page, size int) ([]*model.InspectionTemplate, int64, error) {
	return s.repo.ListTemplates(page, size)
}

func (s *InspectionService) UpsertTemplate(id int64, req *model.InspectionTemplate) error {
	if req.Name == "" {
		return errors.New("模板名称不能为空")
	}
	if req.TaskID == 0 {
		return errors.New("任务模板不能为空")
	}
	if req.DefaultHosts == "" {
		req.DefaultHosts = "[]"
	}
	if id == 0 {
		return s.repo.CreateTemplate(req)
	}
	old, err := s.repo.GetTemplate(id)
	if err != nil {
		return err
	}
	old.Name = req.Name
	old.Description = req.Description
	old.TaskID = req.TaskID
	old.DefaultHosts = req.DefaultHosts
	old.Enabled = req.Enabled
	old.UpdatedBy = req.UpdatedBy
	return s.repo.UpdateTemplate(old)
}

func (s *InspectionService) ListPlans(page, size int) ([]*model.InspectionPlan, int64, error) {
	return s.repo.ListPlans(page, size)
}

func (s *InspectionService) UpsertPlan(id int64, req *model.InspectionPlan) error {
	if req.Name == "" {
		return errors.New("计划名称不能为空")
	}
	if req.TemplateID == 0 {
		return errors.New("模板不能为空")
	}
	if req.CronExpr == "" {
		return errors.New("cron 表达式不能为空")
	}
	if id == 0 {
		return s.repo.CreatePlan(req)
	}
	old, err := s.repo.GetPlan(id)
	if err != nil {
		return err
	}
	old.Name = req.Name
	old.TemplateID = req.TemplateID
	old.CronExpr = req.CronExpr
	old.Enabled = req.Enabled
	old.UpdatedBy = req.UpdatedBy
	return s.repo.UpdatePlan(old)
}

func (s *InspectionService) ExecutePlan(planID int64) (*model.InspectionRecord, error) {
	plan, err := s.repo.GetPlan(planID)
	if err != nil {
		return nil, err
	}
	tpl, err := s.repo.GetTemplate(plan.TemplateID)
	if err != nil {
		return nil, err
	}
	hosts := make([]string, 0)
	_ = json.Unmarshal([]byte(tpl.DefaultHosts), &hosts)
	if len(hosts) == 0 {
		return nil, errors.New("模板未配置巡检主机")
	}
	exec, err := s.taskSvc.ExecuteTask(tpl.TaskID, hosts, 0)
	if err != nil {
		return nil, err
	}
	now := model.LocalTime(time.Now())
	reportPayload, _ := json.Marshal(map[string]interface{}{
		"plan_id":       plan.ID,
		"template_id":   tpl.ID,
		"template_name": tpl.Name,
		"task_id":       tpl.TaskID,
		"target_hosts":  hosts,
		"started_at":    now,
		"status":        "running",
	})
	record := &model.InspectionRecord{
		PlanID:          plan.ID,
		TemplateID:      tpl.ID,
		TaskExecutionID: exec.ID,
		Status:          "running",
		StartedAt:       &now,
		ReportJSON:      string(reportPayload),
	}
	if err := s.repo.CreateRecord(record); err != nil {
		return nil, err
	}
	plan.LastRunAt = &now
	_ = s.repo.UpdatePlan(plan)
	return record, nil
}

// SyncRecordStatus 在任务执行完成后回写巡检记录状态、真实报告和告警联动。
// 由 gRPC checkExecutionCompletion 调用。
func (s *InspectionService) SyncRecordStatus(executionID int64) {
	record, err := s.repo.GetRecordByExecutionID(executionID)
	if err != nil {
		return
	}
	if record.Status != "running" {
		return
	}
	exec, err := s.taskSvc.GetExecution(executionID)
	if err != nil {
		return
	}

	now := model.LocalTime(time.Now())
	record.Status = exec.Status
	record.FinishedAt = &now

	hostSummaries := make([]map[string]interface{}, 0, len(exec.HostResults))
	for _, hr := range exec.HostResults {
		summary := map[string]interface{}{
			"host_ip":       hr.HostIP,
			"hostname":      hr.Hostname,
			"status":        hr.Status,
			"exit_code":     hr.ExitCode,
			"duration_ms":   hr.DurationMs,
			"error_summary": hr.ErrorSummary,
		}
		if len(hr.Stdout) > 1024 {
			summary["stdout_tail"] = string([]rune(hr.Stdout)[max(0, len([]rune(hr.Stdout))-512):])
		} else {
			summary["stdout"] = hr.Stdout
		}
		if hr.Stderr != "" {
			if len(hr.Stderr) > 512 {
				summary["stderr_tail"] = string([]rune(hr.Stderr)[max(0, len([]rune(hr.Stderr))-256):])
			} else {
				summary["stderr"] = hr.Stderr
			}
		}
		hostSummaries = append(hostSummaries, summary)
	}

	report := map[string]interface{}{
		"execution_id":  exec.ID,
		"task_id":       exec.TaskID,
		"task_name":     exec.TaskName,
		"status":        exec.Status,
		"total_count":   exec.TotalCount,
		"success_count": exec.SuccessCount,
		"fail_count":    exec.FailCount,
		"started_at":    exec.StartedAt,
		"finished_at":   exec.FinishedAt,
		"host_results":  hostSummaries,
	}
	if reportJSON, err := json.Marshal(report); err == nil {
		record.ReportJSON = string(reportJSON)
	}
	_ = s.repo.UpdateRecord(record)

	if exec.Status == "failed" || exec.Status == "partial_fail" || exec.Status == "canceled" {
		tplName := ""
		if tpl, err := s.repo.GetTemplate(record.TemplateID); err == nil {
			tplName = tpl.Name
		}
		severity := "warning"
		if exec.Status == "failed" {
			severity = "critical"
		}
		_ = s.eventRepo.Create(&model.AlertEvent{
			RuleID:          0,
			RuleName:        "巡检任务异常",
			AgentID:         "inspection",
			Hostname:        tplName,
			MetricType:      "inspection_failed",
			MetricValue:     float64(exec.FailCount),
			Threshold:       0,
			Operator:        "gt",
			Severity:        severity,
			Action:          model.AlertRuleActionNotifyOnly,
			Status:          model.AlertEventStatusFiring,
			Description:     fmt.Sprintf("巡检执行%s: 成功%d 失败%d 总计%d", exec.Status, exec.SuccessCount, exec.FailCount, exec.TotalCount),
			TriggeredAt:     now,
			TaskExecutionID: exec.ID,
		})
	}
}

func (s *InspectionService) ListRecords(page, size int) ([]*model.InspectionRecord, int64, error) {
	return s.repo.ListRecords(page, size)
}

func (s *InspectionService) GetRecordReport(recordID int64) (map[string]interface{}, error) {
	record, err := s.repo.GetRecord(recordID)
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{
		"record_id":         record.ID,
		"plan_id":           record.PlanID,
		"template_id":       record.TemplateID,
		"task_execution_id": record.TaskExecutionID,
		"status":            record.Status,
		"started_at":        record.StartedAt,
		"finished_at":       record.FinishedAt,
	}
	if record.ReportJSON != "" {
		var detail map[string]interface{}
		if err := json.Unmarshal([]byte(record.ReportJSON), &detail); err == nil {
			result["detail"] = detail
		}
	}
	return result, nil
}

func (s *InspectionService) TemplateTrend(templateID int64) (map[string]interface{}, error) {
	items, err := s.repo.ListRecordsByTemplate(templateID, 30)
	if err != nil {
		return nil, err
	}
	success := 0
	failed := 0
	running := 0
	series := make([]map[string]interface{}, 0, len(items))
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		switch item.Status {
		case "success":
			success++
		case "failed", "canceled":
			failed++
		default:
			running++
		}
		series = append(series, map[string]interface{}{
			"id":         item.ID,
			"status":     item.Status,
			"created_at": item.CreatedAt,
		})
	}
	return map[string]interface{}{
		"template_id": templateID,
		"success":     success,
		"failed":      failed,
		"running":     running,
		"series":      series,
	}, nil
}

func (s *InspectionService) ExportRecordReport(recordID int64, format string) ([]byte, string, string, error) {
	report, err := s.GetRecordReport(recordID)
	if err != nil {
		return nil, "", "", err
	}
	filenamePrefix := fmt.Sprintf("inspection-record-%d", recordID)
	switch format {
	case "json":
		payload, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr != nil {
			return nil, "", "", marshalErr
		}
		return payload, "application/json; charset=utf-8", filenamePrefix + ".json", nil
	case "csv":
		payload, csvErr := buildInspectionReportCSV(report)
		if csvErr != nil {
			return nil, "", "", csvErr
		}
		return payload, "text/csv; charset=utf-8", filenamePrefix + ".csv", nil
	default:
		return nil, "", "", errors.New("仅支持导出 json 或 csv")
	}
}

func buildInspectionReportCSV(report map[string]interface{}) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	writer := csv.NewWriter(buffer)
	if err := writer.Write([]string{"field", "value"}); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(report))
	for key := range report {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value, _ := json.Marshal(report[key])
		if err := writer.Write([]string{key, string(value)}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
