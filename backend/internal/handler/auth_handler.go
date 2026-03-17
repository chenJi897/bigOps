// Package handler 提供 HTTP 请求处理器，负责接收请求、调用服务、返回响应。
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

// AuthHandler 认证相关的 HTTP 处理器。
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler 创建 AuthHandler 实例。
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authService: service.NewAuthService(),
	}
}

// registerRequest 注册请求参数。
type registerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=50"`
	Email    string `json:"email" binding:"omitempty,email"`
}

// Register 用户注册接口。
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.authService.Register(req.Username, req.Password, req.Email); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.SuccessWithMessage(c, "注册成功", nil)
}

// loginRequest 登录请求参数。
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 用户登录接口。
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	result, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		response.Error(c, 401, err.Error())
		return
	}

	response.Success(c, result)
}

// Logout 用户登出接口。
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// 从 Header 提取 token
	authHeader := c.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		response.BadRequest(c, "token 格式错误")
		return
	}

	if err := h.authService.Logout(parts[1]); err != nil {
		response.InternalServerError(c, "登出失败")
		return
	}

	response.SuccessWithMessage(c, "登出成功", nil)
}

// GetUserInfo 获取当前登录用户信息接口。
// GET /api/v1/auth/info
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "用户未认证")
		return
	}

	user, err := h.authService.GetUserInfo(userID.(int64))
	if err != nil {
		response.Error(c, 404, err.Error())
		return
	}

	response.Success(c, user)
}

// changePasswordRequest 修改密码请求参数。
type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=50"`
}

// ChangePassword 修改密码接口。
// PUT /api/v1/auth/password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "用户未认证")
		return
	}

	if err := h.authService.ChangePassword(userID.(int64), req.OldPassword, req.NewPassword); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.SuccessWithMessage(c, "密码修改成功", nil)
}
