package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

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

// List 用户列表（分页）。GET /api/v1/users
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

type updateStatusRequest struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

// UpdateStatus 启用/禁用用户。POST /api/v1/users/:id/status
func (h *UserHandler) UpdateStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	user, err := h.userRepo.GetByID(id)
	if err != nil {
		response.Error(c, 404, "用户不存在")
		return
	}
	user.Status = req.Status
	if err := h.userRepo.Update(user); err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}
	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除用户。POST /api/v1/users/:id/delete
func (h *UserHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 1 {
		response.Error(c, 400, "不允许删除管理员")
		return
	}
	if err := h.userRepo.Delete(id); err != nil {
		response.InternalServerError(c, "删除失败")
		return
	}
	response.SuccessWithMessage(c, "删除成功", nil)
}
