package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
	taskRepo  *repository.TaskRepository
	execRepo  *repository.TaskExecutionRepository
	agentRepo *repository.AgentRepository
	userRepo  *repository.UserRepository
}

// NewTaskService creates a new TaskService.
func NewTaskService() *TaskService {
	return &TaskService{
		taskRepo:  repository.NewTaskRepository(),
		execRepo:  repository.NewTaskExecutionRepository(),
		agentRepo: repository.NewAgentRepository(),
		userRepo:  repository.NewUserRepository(),
	}
}

// Create validates and creates a new task.
func (s *TaskService) Create(task *model.Task) error {
	if task.Name == "" {
		return errors.New("任务名称不能为空")
	}
	if task.ScriptContent == "" {
		return errors.New("脚本内容不能为空")
	}
	if task.TaskType == "" {
		task.TaskType = "shell"
	}
	if task.ScriptType == "" {
		task.ScriptType = "bash"
	}
	if task.Timeout <= 0 {
		task.Timeout = 60
	}
	if task.Status == 0 {
		task.Status = 1
	}
	return s.taskRepo.Create(task)
}

// Update modifies an existing task.
func (s *TaskService) Update(id int64, updates *model.Task) error {
	existing, err := s.taskRepo.GetByID(id)
	if err != nil {
		return errors.New("任务不存在")
	}
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.TaskType != "" {
		existing.TaskType = updates.TaskType
	}
	if updates.ScriptType != "" {
		existing.ScriptType = updates.ScriptType
	}
	if updates.ScriptContent != "" {
		existing.ScriptContent = updates.ScriptContent
	}
	if updates.Timeout > 0 {
		existing.Timeout = updates.Timeout
	}
	existing.RunAsUser = updates.RunAsUser
	existing.Description = updates.Description
	return s.taskRepo.Update(existing)
}

// Delete soft-deletes a task.
func (s *TaskService) Delete(id int64) error {
	_, err := s.taskRepo.GetByID(id)
	if err != nil {
		return errors.New("任务不存在")
	}
	return s.taskRepo.Delete(id)
}

// GetByID returns a task by ID with the creator name filled.
func (s *TaskService) GetByID(id int64) (*model.Task, error) {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	task.CreatorName = s.getUserName(task.CreatorID)
	return task, nil
}

// List returns a paginated list of tasks with creator names.
func (s *TaskService) List(q repository.TaskListQuery) ([]*model.Task, int64, error) {
	items, total, err := s.taskRepo.List(q)
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
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, errors.New("任务不存在")
	}
	if task.Status != 1 {
		return nil, errors.New("任务已禁用")
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

	now := model.LocalTime(time.Now())
	exec := &model.TaskExecution{
		TaskID:      taskID,
		Status:      "pending",
		TargetHosts: string(hostsJSON),
		TotalCount:  len(hostIPs),
		OperatorID:  operatorID,
		StartedAt:   &now,
	}
	if err := s.execRepo.Create(exec); err != nil {
		return nil, fmt.Errorf("创建执行记录失败: %w", err)
	}

	mgr := agentgrpc.GetAgentManager()
	dispatchedCount := 0

	for _, ip := range hostIPs {
		// Find agent by IP
		agentInfo, err := s.agentRepo.GetByAnyIP(ip)
		if err != nil {
			// No agent registered for this IP, create host result as failed
			if err := s.execRepo.CreateHostResult(&model.TaskHostResult{
				ExecutionID: exec.ID,
				HostIP:      ip,
				Status:      "failed",
				Stderr:      "Agent 未注册",
			}); err != nil {
				logger.Warn("创建主机结果记录失败", zap.String("host_ip", ip), zap.Error(err))
			}
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
			if err := s.execRepo.UpdateHostResult(hr); err != nil {
				logger.Warn("创建主机结果记录失败", zap.String("host_ip", ip), zap.Error(err))
			}
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
			if err := s.execRepo.UpdateHostResult(hr); err != nil {
				logger.Warn("创建主机结果记录失败", zap.String("host_ip", ip), zap.Error(err))
			}
			exec.FailCount++
			continue
		}

		dispatchedCount++
	}

	// Update execution status
	if dispatchedCount > 0 {
		exec.Status = "running"
	} else {
		exec.Status = "failed"
		finishedAt := model.LocalTime(time.Now())
		exec.FinishedAt = &finishedAt
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

	return exec, nil
}

// GetExecution returns an execution with host results.
func (s *TaskService) GetExecution(id int64) (*model.TaskExecution, error) {
	exec, err := s.execRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	// Fill task name
	if task, err := s.taskRepo.GetByID(exec.TaskID); err == nil {
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
	// Fill names
	nameMap := make(map[int64]string)
	for _, item := range items {
		if item.OperatorID > 0 {
			if _, ok := nameMap[item.OperatorID]; !ok {
				nameMap[item.OperatorID] = s.getUserName(item.OperatorID)
			}
			item.OperatorName = nameMap[item.OperatorID]
		}
	}
	return items, total, nil
}

// ListAgents returns paginated agent list.
func (s *TaskService) ListAgents(page, size int, status string) ([]*model.AgentInfo, int64, error) {
	return s.agentRepo.List(page, size, status)
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
