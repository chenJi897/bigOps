package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	agentgrpc "github.com/bigops/platform/internal/grpc"
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/pkg/safego"
	"github.com/bigops/platform/internal/repository"
	"github.com/bigops/platform/internal/service"
)

var _ model.Task           // swag
var _ model.TaskExecution  // swag
var _ model.TaskHostResult // swag
var _ model.AgentInfo      // swag

// TaskHandler 任务管理 HTTP 处理器。
type TaskHandler struct {
	svc *service.TaskService
}

// NewTaskHandler 创建 TaskHandler 实例。
func NewTaskHandler() *TaskHandler {
	return &TaskHandler{
		svc: service.NewTaskService(),
	}
}

// ---------- Request types ----------

// CreateTaskRequest 创建任务请求参数。
type CreateTaskRequest struct {
	Name            string `json:"name" binding:"required" example:"磁盘清理脚本"`
	TaskType        string `json:"task_type" example:"script"`
	ScriptType      string `json:"script_type" example:"bash"`
	ScriptContent   string `json:"script_content" example:"df -h"`
	Timeout         int    `json:"timeout" example:"60"`
	RunAsUser       string `json:"run_as_user" example:"root"`
	Description     string `json:"description" example:"清理临时文件"`
	RiskLevel       string `json:"risk_level" example:"low"`
	RequireApproval int8   `json:"require_approval" example:"0"`
}

// UpdateTaskRequest 更新任务请求参数。
type UpdateTaskRequest struct {
	Name            string `json:"name" example:"磁盘清理脚本"`
	TaskType        string `json:"task_type" example:"script"`
	ScriptType      string `json:"script_type" example:"bash"`
	ScriptContent   string `json:"script_content" example:"df -h"`
	Timeout         int    `json:"timeout" example:"60"`
	RunAsUser       string `json:"run_as_user" example:"root"`
	Description     string `json:"description" example:"清理临时文件"`
	RiskLevel       string `json:"risk_level" example:"low"`
	RequireApproval int8   `json:"require_approval" example:"0"`
}

// ExecuteTaskRequest 执行任务请求参数。
type ExecuteTaskRequest struct {
	HostIPs []string `json:"host_ips" binding:"required" example:"10.0.0.1,10.0.0.2"`
}

// ---------- Handlers ----------

// List 任务列表。
// @Summary 任务列表
// @Tags 任务管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param keyword query string false "关键字"
// @Param task_type query string false "任务类型"
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.Task}}
// @Router /tasks [get]
func (h *TaskHandler) List(c *gin.Context) {
	page, size := parsePageSize(c)

	q := repository.TaskListQuery{
		Page:     page,
		PageSize: size,
		Keyword:  c.Query("keyword"),
		TaskType: c.Query("task_type"),
		Status:   -1,
	}
	if rawStatus := c.Query("status"); rawStatus != "" {
		if v, err := strconv.Atoi(rawStatus); err == nil {
			q.Status = v
		}
	}

	items, total, err := h.svc.List(q)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

// GetByID 任务详情。
// @Summary 任务详情
// @Tags 任务管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response{data=model.Task}
// @Router /tasks/{id} [get]
func (h *TaskHandler) GetByID(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	task, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "任务不存在")
		return
	}
	response.Success(c, task)
}

// Create 创建任务。
// @Summary 创建任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateTaskRequest true "创建请求"
// @Success 200 {object} response.Response{data=model.Task}
// @Router /tasks [post]
func (h *TaskHandler) Create(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	operatorName := getOperator(c)

	task := &model.Task{
		Name:            req.Name,
		TaskType:        req.TaskType,
		ScriptType:      req.ScriptType,
		ScriptContent:   req.ScriptContent,
		Timeout:         req.Timeout,
		RunAsUser:       req.RunAsUser,
		Description:     req.Description,
		RiskLevel:       req.RiskLevel,
		RequireApproval: req.RequireApproval,
		CreatorID:       operatorID,
	}

	if err := h.svc.Create(task); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	logger.Info("创建任务", zap.String("operator", operatorName), zap.String("task_name", task.Name))
	c.Set("audit_action", "create")
	c.Set("audit_resource", "task")
	c.Set("audit_detail", "创建任务: "+task.Name)
	response.SuccessWithMessage(c, "创建成功", task)
}

// Update 更新任务。
// @Summary 更新任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "任务ID"
// @Param body body UpdateTaskRequest true "更新请求"
// @Success 200 {object} response.Response
// @Router /tasks/{id} [post]
func (h *TaskHandler) Update(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	operatorName := getOperator(c)
	updates := &model.Task{
		Name:            req.Name,
		TaskType:        req.TaskType,
		ScriptType:      req.ScriptType,
		ScriptContent:   req.ScriptContent,
		Timeout:         req.Timeout,
		RunAsUser:       req.RunAsUser,
		Description:     req.Description,
		RiskLevel:       req.RiskLevel,
		RequireApproval: req.RequireApproval,
	}

	if err := h.svc.Update(id, updates); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	logger.Info("更新任务", zap.String("operator", operatorName), zap.Int64("task_id", id))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "task")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "更新任务")
	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除任务。
// @Summary 删除任务
// @Tags 任务管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response
// @Router /tasks/{id}/delete [post]
func (h *TaskHandler) Delete(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	operatorName := getOperator(c)

	if err := h.svc.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	logger.Info("删除任务", zap.String("operator", operatorName), zap.Int64("task_id", id))
	c.Set("audit_action", "delete")
	c.Set("audit_resource", "task")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "删除任务")
	response.SuccessWithMessage(c, "删除成功", nil)
}

// Execute 执行任务。
// @Summary 执行任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "任务ID"
// @Param body body ExecuteTaskRequest true "执行请求"
// @Success 200 {object} response.Response{data=model.TaskExecution}
// @Router /tasks/{id}/execute [post]
func (h *TaskHandler) Execute(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req ExecuteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	operatorName := getOperator(c)

	exec, err := h.svc.ExecuteTask(id, req.HostIPs, operatorID)
	if err != nil {
		response.Error(c, classifyTaskExecErrorCode(err), err.Error())
		return
	}

	logger.Info("执行任务", zap.String("operator", operatorName), zap.Int64("task_id", id), zap.Int64("execution_id", exec.ID))
	c.Set("audit_action", "execute")
	c.Set("audit_resource", "task")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "执行任务")
	response.SuccessWithMessage(c, "执行已下发", exec)
}

// GetExecution 执行详情。
// @Summary 执行详情
// @Tags 任务管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "执行ID"
// @Success 200 {object} response.Response{data=model.TaskExecution}
// @Router /task-executions/{id} [get]
func (h *TaskHandler) GetExecution(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	exec, err := h.svc.GetExecution(id)
	if err != nil {
		response.Error(c, 404, "执行记录不存在")
		return
	}
	response.Success(c, exec)
}

// ListExecutions 执行记录列表。
// @Summary 执行记录列表
// @Tags 任务管理
// @Produce json
// @Security BearerAuth
// @Param task_id query int false "任务ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.TaskExecution}}
// @Router /task-executions [get]
func (h *TaskHandler) ListExecutions(c *gin.Context) {
	taskID, _ := strconv.ParseInt(c.Query("task_id"), 10, 64)
	page, size := parsePageSize(c)

	items, total, err := h.svc.ListExecutions(taskID, page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

// CancelExecution 取消执行。
// @Summary 取消执行
// @Tags 任务管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "执行ID"
// @Success 200 {object} response.Response{data=model.TaskExecution}
// @Router /task-executions/{id}/cancel [post]
func (h *TaskHandler) CancelExecution(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	operatorName := getOperator(c)

	exec, err := h.svc.CancelExecution(id, operatorID)
	if err != nil {
		response.Error(c, classifyTaskExecErrorCode(err), err.Error())
		return
	}
	logger.Info("取消执行", zap.String("operator", operatorName), zap.Int64("execution_id", id))
	c.Set("audit_action", "cancel")
	c.Set("audit_resource", "task_execution")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "取消任务执行(status="+exec.Status+")")
	response.SuccessWithMessage(c, "取消成功", exec)
}

// RetryExecution 重试执行。
// @Summary 重试执行
// @Tags 任务管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "执行ID"
// @Param scope query string false "重试范围: failed/all" default(failed)
// @Success 200 {object} response.Response{data=model.TaskExecution}
// @Router /task-executions/{id}/retry [post]
func (h *TaskHandler) RetryExecution(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	operatorName := getOperator(c)

	scope := c.DefaultQuery("scope", "failed")
	if scope != "failed" && scope != "all" {
		response.BadRequest(c, "无效的重试范围，支持: failed/all")
		return
	}
	var retryBody struct {
		HostIPs []string `json:"host_ips"`
	}
	if c.Request.Body != nil {
		_ = json.NewDecoder(c.Request.Body).Decode(&retryBody)
	}
	exec, err := h.svc.RetryExecution(id, operatorID, scope, retryBody.HostIPs)
	if err != nil {
		response.Error(c, classifyTaskExecErrorCode(err), err.Error())
		return
	}
	hostAudit := strings.Join(retryBody.HostIPs, ",")
	if hostAudit == "" {
		hostAudit = "*"
	}
	logger.Info("重试执行", zap.String("operator", operatorName), zap.Int64("from_execution_id", id), zap.Int64("new_execution_id", exec.ID), zap.String("host_ips", hostAudit))
	c.Set("audit_action", "retry")
	c.Set("audit_resource", "task_execution")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "重试任务执行(scope="+scope+",host_ips="+hostAudit+",new_execution_id="+strconv.FormatInt(exec.ID, 10)+")")
	response.SuccessWithMessage(c, "重试已创建", exec)
}

func classifyTaskExecErrorCode(err error) int {
	if err == nil {
		return 0
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "不存在"):
		return http.StatusNotFound
	case strings.Contains(msg, "进行中"), strings.Contains(msg, "不允许"), strings.Contains(msg, "已禁用"):
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}

// ListAgents Agent 列表。
// @Summary Agent 列表
// @Tags 任务管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param status query string false "状态"
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.AgentInfo}}
// @Router /agents [get]
func (h *TaskHandler) ListAgents(c *gin.Context) {
	page, size := parsePageSize(c)
	status := c.Query("status")

	items, total, err := h.svc.ListAgents(page, size, status)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

// ---------- WebSocket ----------

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSLogs 任务执行日志 WebSocket。
// @Summary 执行日志 WebSocket
// @Tags 任务管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "执行ID"
// @Router /ws/task-executions/{id}/logs [get]
func (h *TaskHandler) WSLogs(c *gin.Context) {
	executionID, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	if executionID == 0 {
		response.BadRequest(c, "执行ID无效")
		return
	}

	replay := c.DefaultQuery("replay", "0") == "1"
	hostIPFilter := strings.TrimSpace(c.Query("host_ip"))

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Warn("WebSocket upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	mgr := agentgrpc.GetAgentManager()
	logCh := mgr.SubscribeLogs(executionID)
	defer mgr.UnsubscribeLogs(executionID, logCh)

	if replay {
		if err := h.writeExecutionLogReplay(conn, executionID, hostIPFilter); err != nil {
			logger.Warn("ws execution log replay", zap.Int64("execution_id", executionID), zap.Error(err))
		}
	}

	// Read pump: detect client disconnect
	done := make(chan struct{})
	go func() {
		defer safego.Recover("ws-read-pump")
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Write pump: send log lines to client
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case line, ok := <-logCh:
			if !ok {
				return
			}
			ev := wsEventFromLogLine(executionID, line)
			if err := conn.WriteJSON(ev); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

// ExportExecutionReport 导出执行报告（Markdown / JSON）。
// @Summary 导出执行报告
// @Tags 任务管理
// @Produce text/markdown
// @Security BearerAuth
// @Param id path int true "执行ID"
// @Param format query string false "格式: markdown/json" default(markdown)
// @Router /task-executions/{id}/report [get]
func (h *TaskHandler) ExportExecutionReport(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	exec, err := h.svc.GetExecution(id)
	if err != nil {
		response.Error(c, 404, "执行记录不存在")
		return
	}

	format := c.DefaultQuery("format", "markdown")
	if format == "json" {
		response.Success(c, exec)
		return
	}

	md := h.svc.GenerateMarkdownReport(exec)
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=execution-"+strconv.FormatInt(id, 10)+"-report.md")
	c.String(http.StatusOK, md)
}
