package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

var _ model.TicketType // swag

type TicketTypeHandler struct {
	svc *service.TicketTypeService
}

func NewTicketTypeHandler() *TicketTypeHandler {
	return &TicketTypeHandler{svc: service.NewTicketTypeService()}
}

type CreateTicketTypeRequest struct {
	Name            string `json:"name" binding:"required" example:"故障报修"`
	Code            string `json:"code" example:"incident"`
	Icon            string `json:"icon" example:"Warning"`
	Description     string `json:"description"`
	HandleDeptID    int64  `json:"handle_dept_id"`
	DefaultAssignee int64  `json:"default_assignee"`
	Priority        string `json:"priority" example:"medium"`
	AutoAssignRule  string `json:"auto_assign_rule" example:"resource_owner"`
	Sort            int    `json:"sort"`
}

// List 工单类型列表。
// @Summary 工单类型列表
// @Tags 工单类型
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.TicketType}}
// @Router /ticket-types [get]
func (h *TicketTypeHandler) List(c *gin.Context) {
	page, size := parsePageSize(c)
	items, total, err := h.svc.List(page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, items, total, page, size)
}

// GetAll 全量工单类型。
// @Summary 全量工单类型
// @Tags 工单类型
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]model.TicketType}
// @Router /ticket-types/all [get]
func (h *TicketTypeHandler) GetAll(c *gin.Context) {
	items, err := h.svc.GetAll()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

// Create 创建工单类型。
// @Summary 创建工单类型
// @Tags 工单类型
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateTicketTypeRequest true "创建请求"
// @Success 200 {object} response.Response
// @Router /ticket-types [post]
func (h *TicketTypeHandler) Create(c *gin.Context) {
	var req CreateTicketTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	tt := &model.TicketType{
		Name: req.Name, Code: req.Code, Icon: req.Icon, Description: req.Description,
		HandleDeptID: req.HandleDeptID, DefaultAssignee: req.DefaultAssignee,
		Priority: req.Priority, AutoAssignRule: req.AutoAssignRule, Sort: req.Sort, Status: 1,
	}
	if tt.AutoAssignRule == "" {
		tt.AutoAssignRule = "manual"
	}
	if tt.Priority == "" {
		tt.Priority = "medium"
	}
	if err := h.svc.Create(tt); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建工单类型", zap.String("operator", getOperator(c)), zap.String("name", req.Name))
	c.Set("audit_action", "create")
	c.Set("audit_resource", "ticket_type")
	c.Set("audit_detail", "创建工单类型: "+req.Name)
	response.SuccessWithMessage(c, "创建成功", nil)
}

// Update 更新工单类型。
// @Summary 更新工单类型
// @Tags 工单类型
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID"
// @Param body body CreateTicketTypeRequest true "更新请求"
// @Success 200 {object} response.Response
// @Router /ticket-types/{id} [post]
func (h *TicketTypeHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req CreateTicketTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	tt := &model.TicketType{
		Name: req.Name, Code: req.Code, Icon: req.Icon, Description: req.Description,
		HandleDeptID: req.HandleDeptID, DefaultAssignee: req.DefaultAssignee,
		Priority: req.Priority, AutoAssignRule: req.AutoAssignRule, Sort: req.Sort,
	}
	if err := h.svc.Update(id, tt); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新工单类型", zap.String("operator", getOperator(c)), zap.Int64("id", id))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "ticket_type")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "更新工单类型: "+req.Name)
	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除工单类型。
// @Summary 删除工单类型
// @Tags 工单类型
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID"
// @Success 200 {object} response.Response
// @Router /ticket-types/{id}/delete [post]
func (h *TicketTypeHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除工单类型", zap.String("operator", getOperator(c)), zap.Int64("id", id))
	c.Set("audit_action", "delete")
	c.Set("audit_resource", "ticket_type")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", fmt.Sprintf("删除工单类型 ID: %d", id))
	response.SuccessWithMessage(c, "删除成功", nil)
}
