package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/repository"
)

// UserHandler 用户管理 HTTP 处理器。
type UserHandler struct {
	userRepo *repository.UserRepository
}

func NewUserHandler() *UserHandler {
	return &UserHandler{userRepo: repository.NewUserRepository()}
}

// List 用户列表（分页）。
func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	users, total, err := h.userRepo.List(page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, users, total, page, size)
}

type updateUserStatusRequest struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

// UpdateStatus 启用/禁用用户。
func (h *UserHandler) UpdateStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req updateUserStatusRequest
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
	response.SuccessWithMessage(c, action+"成功", nil)
}

// Delete 删除用户。
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
	response.SuccessWithMessage(c, "删除成功", nil)
}
