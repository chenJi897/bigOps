// Package handler 提供 HTTP 请求处理器，负责接收请求、调用服务、返回响应。
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/middleware"
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

// swag needs model import to resolve annotation types
var _ model.User

// AuthHandler 认证相关的 HTTP 处理器。
type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{authService: service.NewAuthService()}
}

// RegisterRequest 用户注册请求参数。
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50" example:"johndoe"`
	Password string `json:"password" binding:"required,min=6,max=50" example:"Pass123456"`
	Email    string `json:"email" binding:"omitempty,email" example:"john@example.com"`
}

// Register 用户注册。
// @Summary 用户注册
// @Description 注册新用户账号，密码需包含大小写字母和数字，长度 8-50 位
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body RegisterRequest true "注册请求"
// @Success 200 {object} response.Response "注册成功"
// @Failure 400 {object} response.Response "参数错误/用户名已存在"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
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
	c.Set("audit_action", "create")
	c.Set("audit_resource", "user")
	c.Set("audit_detail", "注册用户: "+req.Username)
	c.Set("username", req.Username)
	response.SuccessWithMessage(c, "注册成功", nil)
}

// LoginRequest 用户登录请求参数。
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"Admin123"`
}

// Login 用户登录。
// @Summary 用户登录
// @Description 使用用户名密码登录，返回 JWT token 和用户信息
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body LoginRequest true "登录请求"
// @Success 200 {object} response.Response{data=service.LoginResult} "登录成功"
// @Failure 401 {object} response.Response "用户名或密码错误/账号已被禁用"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	c.Set("login_username", req.Username)

	// 检查账号是否被锁定
	if middleware.IsAccountLocked(req.Username) {
		c.Set("login_failed", true)
		logger.Warn("登录被锁定", zap.String("username", req.Username), zap.String("ip", c.ClientIP()))
		response.Error(c, 401, "登录失败次数过多，账号已锁定 15 分钟")
		return
	}

	result, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		c.Set("login_failed", true)
		logger.Warn("登录失败", zap.String("username", req.Username), zap.String("ip", c.ClientIP()), zap.String("error", err.Error()))
		response.Error(c, 401, err.Error())
		return
	}
	logger.Info("用户登录", zap.String("username", req.Username), zap.String("ip", c.ClientIP()))
	c.Set("audit_action", "login")
	c.Set("audit_resource", "user")
	c.Set("audit_detail", "用户登录: "+req.Username)
	c.Set("username", req.Username)
	response.Success(c, result)
}

// Logout 用户登出。
// @Summary 用户登出
// @Description 使当前 token 失效，加入 Redis 黑名单
// @Tags 认证
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response "登出成功"
// @Failure 400 {object} response.Response "token 格式错误"
// @Router /auth/logout [post]
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
	c.Set("audit_action", "logout")
	c.Set("audit_resource", "user")
	c.Set("audit_detail", "用户登出")
	response.SuccessWithMessage(c, "登出成功", nil)
}

// GetUserInfo 获取当前用户信息。
// @Summary 获取当前登录用户信息
// @Description 根据 token 获取当前登录用户的详细信息
// @Tags 认证
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=model.User} "用户信息"
// @Failure 401 {object} response.Response "用户未认证"
// @Failure 404 {object} response.Response "用户不存在"
// @Router /auth/info [get]
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

// ChangePasswordRequest 修改密码请求参数。
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required" example:"OldPass123"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=50" example:"NewPass456"`
}

// ChangePassword 修改密码。
// @Summary 修改当前用户密码
// @Description 验证旧密码后修改为新密码，新密码需包含大小写字母和数字，长度 8-50 位
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body ChangePasswordRequest true "修改密码请求"
// @Success 200 {object} response.Response "密码修改成功"
// @Failure 400 {object} response.Response "参数错误/原密码错误/密码复杂度不够"
// @Router /auth/password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
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
	c.Set("audit_action", "update")
	c.Set("audit_resource", "user")
	c.Set("audit_detail", "修改密码")
	response.SuccessWithMessage(c, "密码修改成功", nil)
}
