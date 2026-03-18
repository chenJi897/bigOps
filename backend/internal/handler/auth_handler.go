// Package handler 提供 HTTP 请求处理器，负责接收请求、调用服务、返回响应。
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

// AuthHandler 认证相关的 HTTP 处理器。
type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{authService: service.NewAuthService()}
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=50"`
	Email    string `json:"email" binding:"omitempty,email"`
}

// Register 用户注册。
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.authService.Register(req.Username, req.Password, req.Email); err != nil {
		logger.Warn("注册失败", zap.String("username", req.Username), zap.String("ip", c.ClientIP()), zap.String("error", err.Error()))
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("用户注册", zap.String("username", req.Username), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "注册成功", nil)
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 用户登录。
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	result, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		logger.Warn("登录失败", zap.String("username", req.Username), zap.String("ip", c.ClientIP()), zap.String("error", err.Error()))
		response.Error(c, 401, err.Error())
		return
	}
	logger.Info("用户登录", zap.String("username", req.Username), zap.String("ip", c.ClientIP()))
	response.Success(c, result)
}

// Logout 用户登出。
func (h *AuthHandler) Logout(c *gin.Context) {
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
	logger.Info("用户登出", zap.String("operator", getOperator(c)), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "登出成功", nil)
}

// GetUserInfo 获取当前用户信息。
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

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=50"`
}

// ChangePassword 修改密码。
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
	logger.Info("修改密码", zap.String("operator", getOperator(c)), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "密码修改成功", nil)
}
