package handler

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/repository"
	"github.com/bigops/platform/internal/service"
)

var _ model.Ticket         // swag
var _ model.TicketActivity // swag

type TicketHandler struct {
	svc         *service.TicketService
	approvalSvc *service.ApprovalService
}

func NewTicketHandler() *TicketHandler {
	return &TicketHandler{
		svc:         service.NewTicketService(),
		approvalSvc: service.NewApprovalService(),
	}
}

type CreateTicketRequest struct {
	Title             string  `json:"title" binding:"required" example:"服务器磁盘告警"`
	TypeID            int64   `json:"type_id" binding:"required" example:"1"`
	Priority          string  `json:"priority" example:"high"`
	Description       string  `json:"description" example:"磁盘使用率超过90%"`
	ResourceType      string  `json:"resource_type" example:"asset"`
	ResourceID        int64   `json:"resource_id" example:"1"`
	ResourceIDs       []int64 `json:"resource_ids"`
	HandleDeptID      int64   `json:"handle_dept_id"`
	AssigneeID        int64   `json:"assignee_id"`
	RequestTemplateID int64   `json:"request_template_id"`
	TicketKind        string  `json:"ticket_kind"`
	ExtraFields       map[string]interface{} `json:"extra_fields"`
}

type ProcessTicketRequest struct {
	Action  string `json:"action" binding:"required" example:"resolve"` // resolve/reject
	Content string `json:"content" example:"已处理完成"`
}

type CloseTicketRequest struct {
	Resolution string `json:"resolution" binding:"required" example:"fixed"` // fixed/wontfix/duplicate/invalid/workaround
	Note       string `json:"note" example:"清理了日志文件"`
}

type AssignTicketRequest struct {
	AssigneeID int64 `json:"assignee_id" binding:"required" example:"3"`
}

type TransferTicketRequest struct {
	AssigneeID int64  `json:"assignee_id" binding:"required" example:"5"`
	Content    string `json:"content" example:"转交给更熟悉的同事"`
}

type CommentRequest struct {
	Content string `json:"content" binding:"required" example:"已定位问题"`
}

// List 工单列表。
// @Summary 工单列表
// @Tags 工单管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param status query string false "状态"
// @Param priority query string false "优先级"
// @Param type_id query int false "工单类型ID"
// @Param source query string false "来源"
// @Param scope query string false "范围" Enums(my_created,my_assigned,my_dept,all)
// @Param keyword query string false "关键字"
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.Ticket}}
// @Router /tickets [get]
func (h *TicketHandler) List(c *gin.Context) {
	page, size := parsePageSize(c)
	typeID, _ := strconv.ParseInt(c.Query("type_id"), 10, 64)
	creatorID, _ := strconv.ParseInt(c.Query("creator_id"), 10, 64)
	assigneeID, _ := strconv.ParseInt(c.Query("assignee_id"), 10, 64)
	handleDeptID, _ := strconv.ParseInt(c.Query("handle_dept_id"), 10, 64)
	serviceTreeID, _ := strconv.ParseInt(c.Query("service_tree_id"), 10, 64)

	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	// 获取用户部门
	var currentDeptID int64
	userRepo := repository.NewUserRepository()
	if user, err := userRepo.GetByID(currentUserID); err == nil {
		currentDeptID = user.DepartmentID
	}

	q := repository.TicketListQuery{
		Page:          page,
		Size:          size,
		Status:        c.Query("status"),
		Priority:      c.Query("priority"),
		TypeID:        typeID,
		Source:        c.Query("source"),
		ResourceType:  c.Query("resource_type"),
		CreatorID:     creatorID,
		AssigneeID:    assigneeID,
		HandleDeptID:  handleDeptID,
		ServiceTreeID: serviceTreeID,
		Keyword:       c.Query("keyword"),
		Scope:         c.DefaultQuery("scope", "all"),
		IsAdmin:       isAdminUser(c),
		CurrentUserID: currentUserID,
		CurrentDeptID: currentDeptID,
	}

	items, total, err := h.svc.List(q)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

// GetByID 工单详情。
// @Summary 工单详情
// @Tags 工单管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "工单ID"
// @Success 200 {object} response.Response{data=model.Ticket}
// @Router /tickets/{id} [get]
func (h *TicketHandler) GetByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ticket, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "工单不存在")
		return
	}
	if !canViewTicket(c, ticket) {
		response.Forbidden(c, "无权查看该工单")
		return
	}
	response.Success(c, ticket)
}

// ApprovalInstance 工单审批链详情。
// @Summary 工单审批链
// @Tags 工单管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "工单ID"
// @Success 200 {object} response.Response
// @Router /tickets/{id}/approval-instance [get]
func (h *TicketHandler) ApprovalInstance(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ticket, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "工单不存在")
		return
	}
	if !canViewTicket(c, ticket) {
		response.Forbidden(c, "无权查看该工单审批链")
		return
	}
	if ticket.ApprovalInstanceID == 0 {
		response.Success(c, nil)
		return
	}
	instance, err := h.approvalSvc.GetByTicketID(id)
	if err != nil {
		response.Error(c, 404, "审批实例不存在")
		return
	}
	response.Success(c, instance)
}

// Create 创建工单。
// @Summary 创建工单
// @Tags 工单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateTicketRequest true "创建请求"
// @Success 200 {object} response.Response{data=model.Ticket}
// @Router /tickets [post]
func (h *TicketHandler) Create(c *gin.Context) {
	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	operatorName := getOperator(c)

	ticket := &model.Ticket{
		Title: req.Title, TypeID: req.TypeID, Priority: req.Priority,
		Description: req.Description, ResourceType: req.ResourceType,
		ResourceID: req.ResourceID, HandleDeptID: req.HandleDeptID,
		AssigneeID: req.AssigneeID, RequestTemplateID: req.RequestTemplateID,
		TicketKind: req.TicketKind,
	}
	extraFields := req.ExtraFields
	if extraFields == nil {
		extraFields = map[string]interface{}{}
	}
	if req.ResourceType == "asset" && len(req.ResourceIDs) > 0 {
		if ticket.ResourceID == 0 {
			ticket.ResourceID = req.ResourceIDs[0]
		}
		extraFields["resource_ids"] = req.ResourceIDs
	}
	if len(extraFields) > 0 {
		if data, err := json.Marshal(extraFields); err == nil {
			ticket.ExtraFields = string(data)
		}
	}

	if err := h.svc.Create(ticket, operatorID, operatorName); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	logger.Info("创建工单", zap.String("operator", operatorName), zap.String("ticket_no", ticket.TicketNo))
	c.Set("audit_action", "create")
	c.Set("audit_resource", "ticket")
	c.Set("audit_detail", "创建工单: "+ticket.TicketNo+" "+ticket.Title)
	response.SuccessWithMessage(c, "创建成功", ticket)
}

// Assign 分配处理人。
// @Summary 分配处理人
// @Tags 工单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "工单ID"
// @Param body body AssignTicketRequest true "分配请求"
// @Success 200 {object} response.Response
// @Router /tickets/{id}/assign [post]
func (h *TicketHandler) Assign(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req AssignTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	if err := h.svc.Assign(id, req.AssigneeID, operatorID); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	c.Set("audit_action", "update")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "分配工单处理人")
	response.SuccessWithMessage(c, "分配成功", nil)
}

// Process 处理工单（解决/驳回）。
// @Summary 处理工单
// @Tags 工单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "工单ID"
// @Param body body ProcessTicketRequest true "处理请求"
// @Success 200 {object} response.Response
// @Router /tickets/{id}/process [post]
func (h *TicketHandler) Process(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req ProcessTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	if err := h.svc.Process(id, req.Action, req.Content, operatorID); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	c.Set("audit_action", "update")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "处理工单: "+req.Action)
	response.SuccessWithMessage(c, "处理成功", nil)
}

// Close 关闭工单。
// @Summary 关闭工单
// @Tags 工单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "工单ID"
// @Param body body CloseTicketRequest true "关闭请求"
// @Success 200 {object} response.Response
// @Router /tickets/{id}/close [post]
func (h *TicketHandler) Close(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req CloseTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	if err := h.svc.Close(id, req.Resolution, req.Note, operatorID); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	c.Set("audit_action", "update")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "关闭工单: "+req.Resolution)
	response.SuccessWithMessage(c, "关闭成功", nil)
}

// Reopen 重新打开工单。
// @Summary 重新打开工单
// @Tags 工单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "工单ID"
// @Param body body CommentRequest true "重开原因"
// @Success 200 {object} response.Response
// @Router /tickets/{id}/reopen [post]
func (h *TicketHandler) Reopen(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	if err := h.svc.Reopen(id, req.Content, operatorID); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	c.Set("audit_action", "update")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "重新打开工单")
	response.SuccessWithMessage(c, "已重新打开", nil)
}

// Comment 添加评论。
// @Summary 添加评论
// @Tags 工单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "工单ID"
// @Param body body CommentRequest true "评论"
// @Success 200 {object} response.Response
// @Router /tickets/{id}/comment [post]
func (h *TicketHandler) Comment(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	if err := h.svc.Comment(id, req.Content, operatorID); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "评论成功", nil)
}

// Transfer 转交工单。
// @Summary 转交工单
// @Tags 工单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "工单ID"
// @Param body body TransferTicketRequest true "转交请求"
// @Success 200 {object} response.Response
// @Router /tickets/{id}/transfer [post]
func (h *TicketHandler) Transfer(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req TransferTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	if err := h.svc.Transfer(id, req.AssigneeID, req.Content, operatorID); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	c.Set("audit_action", "update")
	c.Set("audit_resource", "ticket")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "转交工单")
	response.SuccessWithMessage(c, "转交成功", nil)
}

// Activities 工单活动流。
// @Summary 工单活动流
// @Tags 工单管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "工单ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(50)
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.TicketActivity}}
// @Router /tickets/{id}/activities [get]
func (h *TicketHandler) Activities(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ticket, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "工单不存在")
		return
	}
	if !canViewTicket(c, ticket) {
		response.Forbidden(c, "无权查看该工单活动")
		return
	}
	page, size := parsePageSize(c)
	items, total, err := h.svc.GetActivities(id, page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

func canViewTicket(c *gin.Context, ticket *model.Ticket) bool {
	if isAdminUser(c) {
		return true
	}
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	// 创建人或处理人可查看
	if ticket.CreatorID == currentUserID || ticket.AssigneeID == currentUserID {
		return true
	}
	// 审批人可查看（当前工单有审批链且用户是审批人之一）
	if ticket.ApprovalInstanceID > 0 {
		var count int64
		database.GetDB().Model(&model.ApprovalRecord{}).
			Where("instance_id = ? AND approver_id = ?", ticket.ApprovalInstanceID, currentUserID).
			Count(&count)
		if count > 0 {
			return true
		}
	}
	return false
}
