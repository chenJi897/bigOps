package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/repository"
)

// swag needs model import to resolve annotation types
var _ model.User

// UserHandler 用户管理 HTTP 处理器。
type UserHandler struct {
	userRepo *repository.UserRepository
}

func NewUserHandler() *UserHandler {
	return &UserHandler{userRepo: repository.NewUserRepository()}
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	keyword := c.Query("keyword")
	users, total, err := h.userRepo.List(page, size, keyword)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, users, total, page, size)
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	user, err := h.userRepo.GetByID(id)
	if err != nil {
		response.Error(c, 404, "用户不存在")
		return
	}
	if id == 1 {
		response.Error(c, 400, "不允许禁用管理员")
		return
	}
	user.Status = req.Status
	if err := h.userRepo.Update(user); err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}
	action := "启用"
	if req.Status == 0 {
		action = "禁用"
	}
	logger.Info(fmt.Sprintf("%s用户", action), zap.String("operator", getOperator(c)), zap.Int64("user_id", id), zap.String("username", user.Username))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "user")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", action+"用户: "+user.Username)
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 1 {
		response.Error(c, 400, "不允许删除管理员")
		return
	}
	user, _ := h.userRepo.GetByID(id)
	username := ""
	if user != nil {
		username = user.Username
	}
	if err := h.userRepo.Delete(id); err != nil {
		response.InternalServerError(c, "删除失败")
		return
	}
	logger.Info("删除用户", zap.String("operator", getOperator(c)), zap.Int64("user_id", id), zap.String("username", username))
	c.Set("audit_action", "delete")
	c.Set("audit_resource", "user")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "删除用户: "+username)
	response.SuccessWithMessage(c, "删除成功", nil)
}
