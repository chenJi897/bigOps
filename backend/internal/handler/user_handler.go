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

// swag needs model import to resolve annotation types
var _ model.User

// UserHandler 用户管理 HTTP 处理器。
type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler() *UserHandler {
	return &UserHandler{svc: service.NewUserService()}
}

// List 用户列表（分页）。
// @Summary 用户列表
// @Description 分页获取用户列表，支持关键字模糊搜索（用户名/邮箱/姓名）
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param keyword query string false "关键字（用户名/邮箱/姓名）"
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.User}} "用户列表"
// @Failure 500 {object} response.Response "查询失败"
// @Router /users [get]
func (h *UserHandler) List(c *gin.Context) {
	page, size := parsePageSize(c)
	keyword := c.Query("keyword")
	users, total, err := h.svc.List(page, size, keyword)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, users, total, page, size)
}

type UpdateUserRequest struct {
	RealName     string `json:"real_name" example:"张三"`
	Phone        string `json:"phone" example:"13800138000"`
	Email        string `json:"email" example:"test@example.com"`
	DepartmentID int64  `json:"department_id" example:"1"`
}

// Update 更新用户信息。
// @Summary 更新用户信息
// @Description 更新用户姓名、手机、邮箱、部门
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param body body UpdateUserRequest true "更新请求"
// @Success 200 {object} response.Response
// @Router /users/{id} [post]
func (h *UserHandler) Update(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	params := service.UpdateUserParams{
		RealName:     req.RealName,
		Phone:        req.Phone,
		Email:        req.Email,
		DepartmentID: req.DepartmentID,
	}
	if err := h.svc.Update(id, params); err != nil {
		response.Error(c, 404, err.Error())
		return
	}
	logger.Info("更新用户信息", zap.String("operator", getOperator(c)), zap.Int64("user_id", id))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "user")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", fmt.Sprintf("更新用户信息: ID %d", id))
	response.SuccessWithMessage(c, "更新成功", nil)
}

// UpdateUserStatusRequest 更新用户状态请求参数。
type UpdateUserStatusRequest struct {
	Status int8 `json:"status" binding:"oneof=0 1" example:"1" enums:"0,1"` // 0:禁用 1:启用
}

// UpdateStatus 启用/禁用用户。
// @Summary 启用/禁用用户
// @Description 更新用户状态，0=禁用 1=启用，不允许禁用管理员（ID=1）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param body body UpdateUserStatusRequest true "状态请求"
// @Success 200 {object} response.Response "启用/禁用成功"
// @Failure 400 {object} response.Response "不允许禁用管理员"
// @Failure 404 {object} response.Response "用户不存在"
// @Router /users/{id}/status [post]
func (h *UserHandler) UpdateStatus(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	username, err := h.svc.UpdateStatus(id, req.Status)
	if err != nil {
		if err.Error() == "不允许禁用管理员" {
			response.Error(c, 400, err.Error())
		} else {
			response.Error(c, 404, err.Error())
		}
		return
	}
	action := "启用"
	if req.Status == 0 {
		action = "禁用"
	}
	logger.Info(fmt.Sprintf("%s用户", action), zap.String("operator", getOperator(c)), zap.Int64("user_id", id), zap.String("username", username))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "user")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", action+"用户: "+username)
	response.SuccessWithMessage(c, action+"成功", nil)
}

// Delete 删除用户。
// @Summary 删除用户
// @Description 软删除用户，不允许删除管理员（ID=1）
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "不允许删除管理员"
// @Failure 500 {object} response.Response "删除失败"
// @Router /users/{id}/delete [post]
func (h *UserHandler) Delete(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	username, err := h.svc.Delete(id)
	if err != nil {
		if err.Error() == "不允许删除管理员" {
			response.Error(c, 400, err.Error())
		} else {
			response.InternalServerError(c, "删除失败")
		}
		return
	}
	logger.Info("删除用户", zap.String("operator", getOperator(c)), zap.Int64("user_id", id), zap.String("username", username))
	c.Set("audit_action", "delete")
	c.Set("audit_resource", "user")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "删除用户: "+username)
	response.SuccessWithMessage(c, "删除成功", nil)
}

type SetDepartmentRequest struct {
	DepartmentID int64 `json:"department_id" example:"1"`
}

// SetDepartment 设置用户部门。
// @Summary 设置用户部门
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param body body SetDepartmentRequest true "部门请求"
// @Success 200 {object} response.Response
// @Router /users/{id}/department [post]
func (h *UserHandler) SetDepartment(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req SetDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	username, err := h.svc.SetDepartment(id, req.DepartmentID)
	if err != nil {
		response.Error(c, 404, err.Error())
		return
	}
	logger.Info("设置用户部门", zap.String("operator", getOperator(c)), zap.Int64("user_id", id), zap.Int64("dept_id", req.DepartmentID))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "user")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", fmt.Sprintf("设置用户 %s 部门 ID: %d", username, req.DepartmentID))
	response.SuccessWithMessage(c, "设置成功", nil)
}
