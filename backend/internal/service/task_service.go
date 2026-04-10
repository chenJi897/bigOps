package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	pb "github.com/bigops/platform/proto/gen/agent"

	agentgrpc "github.com/bigops/platform/internal/grpc"
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/scriptguard"
	"github.com/bigops/platform/internal/repository"
)

// TaskService handles task CRUD and execution dispatch.
type TaskService struct {
	taskRepo     *repository.TaskRepository
	execRepo     *repository.TaskExecutionRepository
	agentRepo    *repository.AgentRepository
	userRepo     *repository.UserRepository
	approvalRepo *repository.TaskApprovalRepository
}

// NewTaskService creates a new TaskService.
func NewTaskService() *TaskService {
	return &TaskService{
		taskRepo:     repository.NewTaskRepository(),
		execRepo:     repository.NewTaskExecutionRepository(),
		agentRepo:    repository.NewAgentRepository(),
		userRepo:     repository.NewUserRepository(),
		approvalRepo: repository.NewTaskApprovalRepository(),
	}
}

var allowedTaskTypes = map[string]bool{"script": true, "file_transfer": true, "api_call": true, "shell": true}
var allowedScriptTypes = map[string]bool{"bash": true, "python": true, "sh": true, "powershell": true}

func normalizeTaskType(task *model.Task) {
	if task.TaskType == "" || task.TaskType == "shell" || task.TaskType == "bash" || task.TaskType == "python" {
		task.TaskType = "script"
	}
	if task.TaskType == "script" && task.ScriptType == "" {
		task.ScriptType = "bash"
	}
}

func validateTaskTypeCombo(task *model.Task) error {
	if !allowedTaskTypes[task.TaskType] {
		return fmt.Errorf("不支持的任务类型: %s（允许: script/file_transfer/api_call）", task.TaskType)
	}
	if task.TaskType == "script" {
		if !allowedScriptTypes[task.ScriptType] {
			return fmt.Errorf("不支持的脚本语言: %s（允许: bash/python/sh/powershell）", task.ScriptType)
		}
		if strings.TrimSpace(task.ScriptContent) == "" {
			return errors.New("脚本类型任务的脚本内容不能为空")
		}
	} else {
		if task.ScriptType != "" {
			return fmt.Errorf("任务类型为 %s 时不应设置 script_type", task.TaskType)
		}
	}
	return nil
}

func inferRiskLevel(task *model.Task) {
	if task.RiskLevel != "" && task.RiskLevel != "auto" {
		return
	}
	task.RiskLevel = "low"
	content := strings.ToLower(task.ScriptContent)
	for _, pattern := range model.DangerousCommandPatterns {
		if strings.Contains(content, pattern) {
			task.RiskLevel = "critical"
			task.RequireApproval = 1
			return
		}
	}
	if task.RunAsUser == "root" {
		if task.RiskLevel == "low" {
			task.RiskLevel = "medium"
		}
	}
	if strings.Contains(content, "systemctl restart") || strings.Contains(content, "service") {
		task.RiskLevel = "high"
		task.RequireApproval = 1
	}
}

// Create validates and creates a new task.
func (s *TaskService) Create(task *model.Task) error {
	if task.Name == "" {
		return errors.New("任务名称不能为空")
	}
	normalizeTaskType(task)
	if err := validateTaskTypeCombo(task); err != nil {
		return err
	}
	inferRiskLevel(task)
	if task.Timeout <= 0 {
		task.Timeout = 60
	}
	if task.Status == 0 {
		task.Status = 1
	}
	return s.taskRepo.CreateTask(task)
}

// Update modifies an existing task.
func (s *TaskService) Update(id int64, updates *model.Task) error {
	existing, err := s.taskRepo.GetTask(id)
	if err != nil {
		return errors.New("任务不存在")
	}
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.TaskType != "" {
		existing.TaskType = updates.TaskType
	}
	existing.ScriptType = updates.ScriptType
	existing.ScriptContent = updates.ScriptContent
	if updates.Timeout > 0 {
		existing.Timeout = updates.Timeout
	}
	existing.RunAsUser = updates.RunAsUser
	existing.Description = updates.Description
	normalizeTaskType(existing)
	if err := validateTaskTypeCombo(existing); err != nil {
		return err
	}
	if updates.RiskLevel != "" {
		existing.RiskLevel = updates.RiskLevel
	}
	existing.RequireApproval = updates.RequireApproval
	inferRiskLevel(existing)
	return s.taskRepo.UpdateTask(existing)
}

// Delete soft-deletes a task.
func (s *TaskService) Delete(id int64) error {
	_, err := s.taskRepo.GetTask(id)
	if err != nil {
		return errors.New("任务不存在")
	}
	return s.taskRepo.DeleteTask(id)
}

// GetByID returns a task by ID with the creator name filled.
func (s *TaskService) GetByID(id int64) (*model.Task, error) {
	task, err := s.taskRepo.GetTask(id)
	if err != nil {
		return nil, err
	}
	task.CreatorName = s.getUserName(task.CreatorID)
	return task, nil
}

// List returns a paginated list of tasks with creator names.
func (s *TaskService) List(q repository.TaskListQuery) ([]*model.Task, int64, error) {
	items, total, err := s.taskRepo.ListTasks(q)
	if err != nil {
		return nil, 0, err
	}
	// Batch fill creator names
	nameMap := make(map[int64]string)
	for _, item := range items {
		if item.CreatorID > 0 {
			if _, ok := nameMap[item.CreatorID]; !ok {
				nameMap[item.CreatorID] = s.getUserName(item.CreatorID)
			}
			item.CreatorName = nameMap[item.CreatorID]
		}
	}
	return items, total, nil
}

// ExecuteTask creates execution records and dispatches tasks to agents.
func (s *TaskService) ExecuteTask(taskID int64, hostIPs []string, operatorID int64) (*model.TaskExecution, error) {
	return s.executeTask(taskID, hostIPs, operatorID, nil)
}

func (s *TaskService) ExecuteTaskWithEnv(taskID int64, hostIPs []string, operatorID int64, extraEnv map[string]string) (*model.TaskExecution, error) {
	return s.executeTask(taskID, hostIPs, operatorID, extraEnv)
}

func (s *TaskService) executeTask(taskID int64, hostIPs []string, operatorID int64, extraEnv map[string]string) (*model.TaskExecution, error) {
	task, err := s.taskRepo.GetTask(taskID)
	if err != nil {
		return nil, errors.New("任务不存在")
	}
	if task.Status != 1 {
		return nil, errors.New("任务已禁用")
	}
	if task.RequireApproval == 1 {
		approved, err := s.approvalRepo.FindLatestApprovedByTask(taskID)
		if err != nil || approved.ID == 0 {
			return nil, errors.New("该任务需要审批，当前不允许直接执行")
		}
	}
	if len(hostIPs) == 0 {
		return nil, errors.New("目标主机列表不能为空")
	}

	// Server-side script safety pre-check
	if err := scriptguard.Validate(task.ScriptContent, task.ScriptType); err != nil {
		return nil, fmt.Errorf("脚本安全检测未通过: %w", err)
	}

	// Marshal target hosts to JSON
	hostsJSON, err2 := json.Marshal(hostIPs)
	if err2 != nil {
		logger.Warn("序列化目标主机列表失败", zap.Error(err2))
		hostsJSON = []byte("[]")
	}

	exec := &model.TaskExecution{
		TaskID:      taskID,
		Status:      model.TaskExecStatusPending,
		TargetHosts: string(hostsJSON),
		TotalCount:  len(hostIPs),
		OperatorID:  operatorID,
		StartedAt:   nil,
	}
	if err := s.execRepo.Create(exec); err != nil {
		return nil, fmt.Errorf("创建执行记录失败: %w", err)
	}

	mgr := agentgrpc.GetAgentManager()
	mgr.PublishLog(exec.ID, &agentgrpc.LogLine{
		HostResultID: 0,
		HostIP:       "",
		Line:         fmt.Sprintf("任务开始执行: task_id=%d, targets=%d", taskID, len(hostIPs)),
		IsStderr:     false,
		Phase:        "running",
		Timestamp:    time.Now().Unix(),
	})
	dispatchedCount := 0

	for _, ip := range hostIPs {
		// Find agent by IP
		agentInfo, err := s.agentRepo.GetByAnyIP(ip)
		if err != nil {
			// No agent registered for this IP, create host result as failed
			fin := model.LocalTime(time.Now())
			if err := s.execRepo.CreateHostResult(&model.TaskHostResult{
				ExecutionID:  exec.ID,
				HostIP:       ip,
				Status:       "failed",
				Stderr:       "Agent 未注册",
				FinishedAt:   &fin,
				ErrorSummary: "Agent 未注册",
			}); err != nil {
				logger.Warn("创建主机结果记录失败", zap.String("host_ip", ip), zap.Error(err))
			}
			mgr.PublishLog(exec.ID, &agentgrpc.LogLine{
				HostResultID: 0,
				HostIP:       ip,
				Line:         "Agent 未注册，跳过下发",
				IsStderr:     true,
				Phase:        "error",
				Timestamp:    time.Now().Unix(),
			})
			exec.FailCount++
			continue
		}

		hr := &model.TaskHostResult{
			ExecutionID: exec.ID,
			AgentID:     agentInfo.AgentID,
			HostIP:      ip,
			Hostname:    agentInfo.Hostname,
			Status:      "pending",
		}
		if err := s.execRepo.CreateHostResult(hr); err != nil {
			continue
		}

		// Check if agent is online
		agentStream := mgr.GetAgent(agentInfo.AgentID)
		if agentStream == nil {
			hr.Status = "failed"
			hr.Stderr = "Agent 离线"
			fin := model.LocalTime(time.Now())
			hr.FinishedAt = &fin
			hr.ErrorSummary = "Agent 离线"
			if err := s.execRepo.UpdateHostResult(hr); err != nil {
				logger.Warn("创建主机结果记录失败", zap.String("host_ip", ip), zap.Error(err))
			}
			mgr.PublishLog(exec.ID, &agentgrpc.LogLine{
				HostResultID: hr.ID,
				HostIP:       ip,
				Line:         "Agent 离线，任务未下发",
				IsStderr:     true,
				Phase:        "error",
				Timestamp:    time.Now().Unix(),
			})
			exec.FailCount++
			continue
		}

		// Send task to agent
		execReq := &pb.ExecuteRequest{
			TaskId:         strconv.FormatInt(taskID, 10),
			ExecutionId:    exec.ID,
			HostResultId:   hr.ID,
			ScriptType:     task.ScriptType,
			ScriptContent:  task.ScriptContent,
			TimeoutSeconds: int32(task.Timeout),
			RunAsUser:      task.RunAsUser,
			Env:            extraEnv,
		}

		if err := mgr.SendTaskToAgent(agentInfo.AgentID, execReq); err != nil {
			logger.Warn("Failed to dispatch task",
				zap.String("agent_id", agentInfo.AgentID),
				zap.String("ip", ip),
				zap.Error(err),
			)
			hr.Status = "failed"
			hr.Stderr = "下发失败: " + err.Error()
			fin := model.LocalTime(time.Now())
			hr.FinishedAt = &fin
			hr.ErrorSummary = truncateRunes("下发失败: "+err.Error(), 500)
			if err := s.execRepo.UpdateHostResult(hr); err != nil {
				logger.Warn("创建主机结果记录失败", zap.String("host_ip", ip), zap.Error(err))
			}
			mgr.PublishLog(exec.ID, &agentgrpc.LogLine{
				HostResultID: hr.ID,
				HostIP:       ip,
				Line:         "任务下发失败: " + err.Error(),
				IsStderr:     true,
				Phase:        "error",
				Timestamp:    time.Now().Unix(),
			})
			exec.FailCount++
			continue
		}

		mgr.PublishLog(exec.ID, &agentgrpc.LogLine{
			HostResultID: hr.ID,
			HostIP:       ip,
			Line:         fmt.Sprintf("任务已下发到 Agent(%s)", agentInfo.AgentID),
			IsStderr:     false,
			Phase:        "running",
			Timestamp:    time.Now().Unix(),
		})
		dispatchedCount++
	}

	// Update execution status：pending -> running（并记录真正开始时间）
	if dispatchedCount > 0 {
		if !model.CanTransitionTaskExecution(exec.Status, model.TaskExecStatusRunning) {
			logger.Warn("任务执行状态流转非常规路径",
				zap.String("from", exec.Status), zap.String("to", model.TaskExecStatusRunning))
		}
		exec.Status = model.TaskExecStatusRunning
		st := model.LocalTime(time.Now())
		exec.StartedAt = &st
		mgr.PublishLog(exec.ID, &agentgrpc.LogLine{
			HostResultID: 0,
			HostIP:       "",
			Line:         fmt.Sprintf("下发完成: success=%d fail=%d", dispatchedCount, exec.FailCount),
			IsStderr:     false,
			Phase:        "running",
			Timestamp:    time.Now().Unix(),
		})
	} else {
		if !model.CanTransitionTaskExecution(exec.Status, model.TaskExecStatusFailed) {
			logger.Warn("任务执行状态流转非常规路径",
				zap.String("from", exec.Status), zap.String("to", model.TaskExecStatusFailed))
		}
		exec.Status = model.TaskExecStatusFailed
		finishedAt := model.LocalTime(time.Now())
		exec.StartedAt = &finishedAt
		exec.FinishedAt = &finishedAt
		mgr.PublishLog(exec.ID, &agentgrpc.LogLine{
			HostResultID: 0,
			HostIP:       "",
			Line:         "没有可用 Agent，任务执行失败",
			IsStderr:     true,
			Phase:        "error",
			Timestamp:    time.Now().Unix(),
		})
	}
	if err := s.execRepo.Update(exec); err != nil {
		logger.Warn("更新执行状态失败", zap.Int64("execution_id", exec.ID), zap.Error(err))
	}

	logger.Info("Task execution dispatched",
		zap.Int64("task_id", taskID),
		zap.Int64("execution_id", exec.ID),
		zap.Int("dispatched", dispatchedCount),
		zap.Int("total", len(hostIPs)),
	)

	// 即时发送执行汇总（不再使用延迟补发）
	mgr.PublishLog(exec.ID, &agentgrpc.LogLine{
		HostResultID: 0,
		HostIP:       "",
		Line:         fmt.Sprintf("执行汇总: status=%s success=%d fail=%d", exec.Status, dispatchedCount, exec.FailCount),
		IsStderr:     exec.Status == model.TaskExecStatusFailed,
		Phase:        "finished",
		Timestamp:    time.Now().Unix(),
	})

	return exec, nil
}

// GetExecution returns an execution with host results.
func (s *TaskService) GetExecution(id int64) (*model.TaskExecution, error) {
	exec, err := s.execRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	// Fill task name
	if task, err := s.taskRepo.GetTask(exec.TaskID); err == nil {
		exec.TaskName = task.Name
	}
	// Fill operator name
	exec.OperatorName = s.getUserName(exec.OperatorID)
	// Fill host results (convert []*TaskHostResult to []TaskHostResult)
	results, err := s.execRepo.GetHostResultsByExecutionID(id)
	if err == nil {
		hostResults := make([]model.TaskHostResult, len(results))
		for i, r := range results {
			hostResults[i] = *r
		}
		exec.HostResults = hostResults
	}
	return exec, nil
}

// ListExecutions returns paginated execution records for a task.
func (s *TaskService) ListExecutions(taskID int64, page, size int) ([]*model.TaskExecution, int64, error) {
	items, total, err := s.execRepo.ListByTaskID(taskID, page, size)
	if err != nil {
		return nil, 0, err
	}
	operatorNameMap := make(map[int64]string)
	taskNameMap := make(map[int64]string)
	for _, item := range items {
		if item.OperatorID > 0 {
			if _, ok := operatorNameMap[item.OperatorID]; !ok {
				operatorNameMap[item.OperatorID] = s.getUserName(item.OperatorID)
			}
			item.OperatorName = operatorNameMap[item.OperatorID]
		}
		if item.TaskID > 0 {
			if _, ok := taskNameMap[item.TaskID]; !ok {
				if t, err := s.taskRepo.GetTask(item.TaskID); err == nil {
					taskNameMap[item.TaskID] = t.Name
				}
			}
			item.TaskName = taskNameMap[item.TaskID]
		}
	}
	return items, total, nil
}

// ListAgents returns paginated agent list.
func (s *TaskService) ListAgents(page, size int, status string) ([]*model.AgentInfo, int64, error) {
	return s.agentRepo.List(page, size, status)
}

// CancelExecution cancels a running or pending execution.
func (s *TaskService) CancelExecution(id int64, operatorID int64) (*model.TaskExecution, error) {
	exec, err := s.execRepo.GetByID(id)
	if err != nil {
		return nil, errors.New("执行记录不存在")
	}
	if exec.Status != model.TaskExecStatusPending && exec.Status != model.TaskExecStatusRunning {
		return nil, fmt.Errorf("当前状态不允许取消: %s", exec.Status)
	}
	if !model.CanTransitionTaskExecution(exec.Status, model.TaskExecStatusCanceled) {
		return nil, fmt.Errorf("当前状态不允许取消: %s", exec.Status)
	}

	now := model.LocalTime(time.Now())
	exec.Status = model.TaskExecStatusCanceled
	exec.FinishedAt = &now

	if err := s.execRepo.CancelUnfinishedHostResults(id, fmt.Sprintf("任务已取消(操作者:%d)", operatorID)); err != nil {
		logger.Warn("批量取消主机执行记录失败", zap.Int64("execution_id", id), zap.Error(err))
	}
	if err := s.execRepo.Update(exec); err != nil {
		return nil, fmt.Errorf("取消执行失败: %w", err)
	}

	agentgrpc.GetAgentManager().PublishLog(id, &agentgrpc.LogLine{
		HostResultID: 0,
		HostIP:       "",
		Line:         fmt.Sprintf("执行已取消: operator=%d", operatorID),
		IsStderr:     false,
		Phase:        "finished",
		Timestamp:    time.Now().Unix(),
	})
	return exec, nil
}

// RetryExecution retries failed/timeout/canceled hosts from an existing execution.
// onlyHosts 非空时，仅重试列表中与 scope 规则交集内的主机（用于单主机重试与审计）。
func (s *TaskService) RetryExecution(id int64, operatorID int64, scope string, onlyHosts []string) (*model.TaskExecution, error) {
	exec, err := s.execRepo.GetByID(id)
	if err != nil {
		return nil, errors.New("执行记录不存在")
	}
	if exec.Status == model.TaskExecStatusPending || exec.Status == model.TaskExecStatusRunning {
		return nil, errors.New("执行进行中，不能重试")
	}

	results, err := s.execRepo.GetHostResultsByExecutionID(id)
	if err != nil {
		return nil, fmt.Errorf("查询主机执行结果失败: %w", err)
	}
	retryIPs := make([]string, 0, len(results))
	for _, item := range results {
		if item == nil {
			continue
		}
		if scope == "all" || item.Status == "failed" || item.Status == "timeout" || item.Status == "canceled" {
			retryIPs = append(retryIPs, item.HostIP)
		}
	}
	if len(onlyHosts) > 0 {
		want := make(map[string]struct{})
		for _, h := range onlyHosts {
			h = strings.TrimSpace(h)
			if h != "" {
				want[h] = struct{}{}
			}
		}
		filtered := make([]string, 0)
		for _, ip := range retryIPs {
			if _, ok := want[ip]; ok {
				filtered = append(filtered, ip)
			}
		}
		if len(filtered) == 0 {
			return nil, errors.New("指定主机不在可重试范围内或状态不允许重试")
		}
		retryIPs = filtered
	}
	if len(retryIPs) == 0 {
		if scope == "all" {
			return nil, errors.New("没有可重试的目标主机")
		}
		return nil, errors.New("没有可重试的失败主机")
	}

	extraEnv := map[string]string{
		"BIGOPS_RETRY_FROM_EXECUTION_ID": strconv.FormatInt(id, 10),
	}
	if len(onlyHosts) > 0 {
		extraEnv["BIGOPS_RETRY_HOST_SCOPE"] = "explicit_hosts"
		extraEnv["BIGOPS_RETRY_HOST_IPS"] = strings.Join(retryIPs, ",")
	}
	newExec, err := s.executeTask(exec.TaskID, retryIPs, operatorID, extraEnv)
	if err != nil {
		return nil, err
	}
	agentgrpc.GetAgentManager().PublishLog(newExec.ID, &agentgrpc.LogLine{
		HostResultID: 0,
		HostIP:       "",
		Line: fmt.Sprintf("重试任务创建成功: from_execution=%d scope=%s retry_hosts=%d hosts=%s",
			id, scope, len(retryIPs), strings.Join(retryIPs, ",")),
		IsStderr:  false,
		Phase:     "running",
		Timestamp: time.Now().Unix(),
	})
	return newExec, nil
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if max <= 0 || len(r) <= max {
		return s
	}
	return string(r[:max])
}

func (s *TaskService) getUserName(id int64) string {
	if id == 0 {
		return ""
	}
	if user, err := s.userRepo.GetByID(id); err == nil {
		if user.RealName != "" {
			return user.RealName
		}
		return user.Username
	}
	return ""
}

// ---------- Approval ----------

// RequestApproval creates a pending approval request for a high-risk task.
func (s *TaskService) RequestApproval(taskID int64, requestorID int64, hostIPs []string) (*model.TaskApproval, error) {
	task, err := s.taskRepo.GetTask(taskID)
	if err != nil {
		return nil, errors.New("任务不存在")
	}
	if task.RequireApproval != 1 {
		return nil, errors.New("该任务不需要审批")
	}
	if existing, err := s.approvalRepo.FindPendingByTask(taskID); err == nil && existing.ID > 0 {
		return nil, errors.New("该任务已有待审批的申请")
	}
	hostsJSON, _ := json.Marshal(hostIPs)
	approval := &model.TaskApproval{
		TaskID:      taskID,
		RequestorID: requestorID,
		Status:      model.TaskApprovalStatusPending,
		HostIPs:     string(hostsJSON),
	}
	if err := s.approvalRepo.Create(approval); err != nil {
		return nil, err
	}
	approval.TaskName = task.Name
	return approval, nil
}

// ApproveTask approves a pending task approval request.
func (s *TaskService) ApproveTask(approvalID int64, approverID int64, comment string) error {
	approval, err := s.approvalRepo.GetByID(approvalID)
	if err != nil {
		return errors.New("审批记录不存在")
	}
	if approval.Status != model.TaskApprovalStatusPending {
		return errors.New("该审批已处理")
	}
	approval.Status = model.TaskApprovalStatusApproved
	approval.ApproverID = approverID
	approval.Comment = comment
	logger.Info("task approval approved",
		zap.Int64("approval_id", approvalID),
		zap.Int64("task_id", approval.TaskID),
		zap.Int64("approver_id", approverID),
	)
	return s.approvalRepo.Update(approval)
}

// RejectTask rejects a pending task approval request.
func (s *TaskService) RejectTask(approvalID int64, approverID int64, comment string) error {
	approval, err := s.approvalRepo.GetByID(approvalID)
	if err != nil {
		return errors.New("审批记录不存在")
	}
	if approval.Status != model.TaskApprovalStatusPending {
		return errors.New("该审批已处理")
	}
	approval.Status = model.TaskApprovalStatusRejected
	approval.ApproverID = approverID
	approval.Comment = comment
	logger.Info("task approval rejected",
		zap.Int64("approval_id", approvalID),
		zap.Int64("task_id", approval.TaskID),
		zap.Int64("approver_id", approverID),
	)
	return s.approvalRepo.Update(approval)
}

// ListPendingApprovals returns paginated pending approval records.
func (s *TaskService) ListPendingApprovals(page, size int) ([]*model.TaskApproval, int64, error) {
	items, total, err := s.approvalRepo.ListPending(page, size)
	if err != nil {
		return nil, 0, err
	}
	for _, item := range items {
		if task, e := s.taskRepo.GetTask(item.TaskID); e == nil {
			item.TaskName = task.Name
		}
	}
	return items, total, nil
}

// ListTaskApprovals returns all approval records for a specific task.
func (s *TaskService) ListTaskApprovals(taskID int64) ([]*model.TaskApproval, error) {
	return s.approvalRepo.ListByTask(taskID)
}

// GenerateMarkdownReport produces a human-readable Markdown report from an execution record.
func (s *TaskService) GenerateMarkdownReport(exec *model.TaskExecution) string {
	var b strings.Builder
	b.WriteString("# 任务执行报告\n\n")
	b.WriteString(fmt.Sprintf("- **执行 ID**: %d\n", exec.ID))
	b.WriteString(fmt.Sprintf("- **任务名称**: %s\n", exec.TaskName))
	b.WriteString(fmt.Sprintf("- **操作人**: %s\n", exec.OperatorName))
	b.WriteString(fmt.Sprintf("- **状态**: %s\n", exec.Status))
	if exec.StartedAt != nil {
		b.WriteString(fmt.Sprintf("- **开始时间**: %s\n", time.Time(*exec.StartedAt).Format("2006-01-02 15:04:05")))
	}
	if exec.FinishedAt != nil {
		b.WriteString(fmt.Sprintf("- **结束时间**: %s\n", time.Time(*exec.FinishedAt).Format("2006-01-02 15:04:05")))
	}
	b.WriteString(fmt.Sprintf("- **主机总数**: %d\n", exec.TotalCount))
	b.WriteString(fmt.Sprintf("- **成功**: %d\n", exec.SuccessCount))
	b.WriteString(fmt.Sprintf("- **失败**: %d\n", exec.FailCount))

	successRate := 0.0
	if exec.TotalCount > 0 {
		successRate = float64(exec.SuccessCount) / float64(exec.TotalCount) * 100
	}
	b.WriteString(fmt.Sprintf("- **成功率**: %.1f%%\n", successRate))
	b.WriteString("\n---\n\n")

	if len(exec.HostResults) > 0 {
		b.WriteString("## 主机执行明细\n\n")
		b.WriteString("| 主机 IP | 状态 | 退出码 | 耗时(ms) | 错误摘要 |\n")
		b.WriteString("|---------|------|--------|----------|----------|\n")

		var failedHosts []model.TaskHostResult
		for _, hr := range exec.HostResults {
			dur := hr.DurationMs
			errSummary := hr.ErrorSummary
			if errSummary == "" {
				errSummary = "-"
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %s |\n",
				hr.HostIP, hr.Status, hr.ExitCode, dur, errSummary))
			if hr.Status == "failed" || hr.Status == "timeout" {
				failedHosts = append(failedHosts, hr)
			}
		}

		if len(failedHosts) > 0 {
			b.WriteString("\n## 失败主机 Top\n\n")
			for _, fh := range failedHosts {
				b.WriteString(fmt.Sprintf("### %s (退出码: %d)\n\n", fh.HostIP, fh.ExitCode))
				if fh.ErrorSummary != "" {
					b.WriteString(fmt.Sprintf("```\n%s\n```\n\n", fh.ErrorSummary))
				}
			}
		}
	}

	b.WriteString("\n---\n\n_报告由 BigOps 自动生成_\n")
	return b.String()
}
