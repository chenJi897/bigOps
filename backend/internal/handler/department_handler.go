package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

var _ model.Department // swag import

type DepartmentHandler struct {
	svc *service.DepartmentService
}

func NewDepartmentHandler() *DepartmentHandler {
	return &DepartmentHandler{svc: service.NewDepartmentService()}
}

type CreateDepartmentRequest struct {
	Name        string `json:"name" binding:"required" example:"运维部"`
	Code        string `json:"code" example:"ops"`
	Description string `json:"description" example:"负责基础设施运维"`
	ManagerID   int64  `json:"manager_id" example:"1"`
	Sort        int    `json:"sort" example:"1"`
}

// List 部门列表。
// @Summary 部门列表
// @Tags 部门管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.Department}}
// @Router /departments [get]
func (h *DepartmentHandler) List(c *gin.Context) {
	page, size := parsePageSize(c)
	departments, total, err := h.svc.List(page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, departments, total, page, size)
}

// GetAll 全量部门列表（下拉选择用）。
// @Summary 全量部门列表
// @Tags 部门管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]model.Department}
// @Router /departments/all [get]
func (h *DepartmentHandler) GetAll(c *gin.Context) {
	departments, err := h.svc.GetAll()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, departments)
}

// GetByID 部门详情。
// @Summary 部门详情
// @Tags 部门管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "部门ID"
// @Success 200 {object} response.Response{data=model.Department}
// @Router /departments/{id} [get]
func (h *DepartmentHandler) GetByID(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	dept, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "部门不存在")
		return
	}
	response.Success(c, dept)
}

// Create 创建部门。
// @Summary 创建部门
// @Tags 部门管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateDepartmentRequest true "创建请求"
// @Success 200 {object} response.Response
// @Router /departments [post]
func (h *DepartmentHandler) Create(c *gin.Context) {
	var req CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	dept := &model.Department{
		Name: req.Name, Code: req.Code, Description: req.Description,
		ManagerID: req.ManagerID, Sort: req.Sort, Status: 1,
	}
	if err := h.svc.Create(dept); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建部门", zap.String("operator", getOperator(c)), zap.String("name", req.Name))
	c.Set("audit_action", "create")
	c.Set("audit_resource", "department")
	c.Set("audit_detail", "创建部门: "+req.Name)
	response.SuccessWithMessage(c, "创建成功", nil)
}

// Update 更新部门。
// @Summary 更新部门
// @Tags 部门管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "部门ID"
// @Param body body CreateDepartmentRequest true "更新请求"
// @Success 200 {object} response.Response
// @Router /departments/{id} [post]
func (h *DepartmentHandler) Update(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	dept := &model.Department{
		Name: req.Name, Code: req.Code, Description: req.Description,
		ManagerID: req.ManagerID, Sort: req.Sort,
	}
	if err := h.svc.Update(id, dept); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新部门", zap.String("operator", getOperator(c)), zap.Int64("id", id))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "department")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "更新部门: "+req.Name)
	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除部门。
// @Summary 删除部门
// @Tags 部门管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "部门ID"
// @Success 200 {object} response.Response
// @Router /departments/{id}/delete [post]
func (h *DepartmentHandler) Delete(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除部门", zap.String("operator", getOperator(c)), zap.Int64("id", id))
	c.Set("audit_action", "delete")
	c.Set("audit_resource", "department")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", fmt.Sprintf("删除部门 ID: %d", id))
	response.SuccessWithMessage(c, "删除成功", nil)
}
