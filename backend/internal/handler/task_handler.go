package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	agentgrpc "github.com/bigops/platform/internal/grpc"
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/repository"
	"github.com/bigops/platform/internal/service"
)

var _ model.Task          // swag
var _ model.TaskExecution // swag
var _ model.TaskHostResult // swag
var _ model.AgentInfo     // swag

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
	Name          string `json:"name" binding:"required" example:"磁盘清理脚本"`
	TaskType      string `json:"task_type" example:"shell"`
	ScriptType    string `json:"script_type" example:"bash"`
	ScriptContent string `json:"script_content" example:"df -h"`
	Timeout       int    `json:"timeout" example:"60"`
	RunAsUser     string `json:"run_as_user" example:"root"`
	Description   string `json:"description" example:"清理临时文件"`
}

// UpdateTaskRequest 更新任务请求参数。
type UpdateTaskRequest struct {
	Name          string `json:"name" example:"磁盘清理脚本"`
	TaskType      string `json:"task_type" example:"shell"`
	ScriptType    string `json:"script_type" example:"bash"`
	ScriptContent string `json:"script_content" example:"df -h"`
	Timeout       int    `json:"timeout" example:"60"`
	RunAsUser     string `json:"run_as_user" example:"root"`
	Description   string `json:"description" example:"清理临时文件"`
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	q := repository.TaskListQuery{
		Page:     page,
		Size:     size,
		Keyword:  c.Query("keyword"),
		TaskType: c.Query("task_type"),
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
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
		Name:          req.Name,
		TaskType:      req.TaskType,
		ScriptType:    req.ScriptType,
		ScriptContent: req.ScriptContent,
		Timeout:       req.Timeout,
		RunAsUser:     req.RunAsUser,
		Description:   req.Description,
		CreatorID:     operatorID,
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	operatorName := getOperator(c)
	updates := &model.Task{
		Name:          req.Name,
		TaskType:      req.TaskType,
		ScriptType:    req.ScriptType,
		ScriptContent: req.ScriptContent,
		Timeout:       req.Timeout,
		RunAsUser:     req.RunAsUser,
		Description:   req.Description,
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
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
		response.Error(c, 400, err.Error())
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	items, total, err := h.svc.ListExecutions(taskID, page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
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
	executionID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if executionID == 0 {
		response.BadRequest(c, "执行ID无效")
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Warn("WebSocket upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	mgr := agentgrpc.GetAgentManager()
	logCh := mgr.SubscribeLogs(executionID)
	defer mgr.UnsubscribeLogs(executionID, logCh)

	// Read pump: detect client disconnect
	done := make(chan struct{})
	go func() {
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
			if err := conn.WriteJSON(line); err != nil {
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
