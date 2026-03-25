package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

// swag needs model import to resolve annotation types
var _ model.AuditLog

// AuditLogHandler 审计日志 HTTP 处理器。
type AuditLogHandler struct {
	auditLogService *service.AuditLogService
}

// NewAuditLogHandler 创建 AuditLogHandler 实例。
func NewAuditLogHandler() *AuditLogHandler {
	return &AuditLogHandler{auditLogService: service.NewAuditLogService()}
}

// List 审计日志列表（分页）。
// @Summary 审计日志列表
// @Description 分页获取操作审计日志，支持按用户名、操作类型、资源类型过滤
// @Tags 审计日志
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param username query string false "用户名"
// @Param action query string false "操作类型(create/update/delete/login/logout)"
// @Param resource query string false "资源类型(user/role/menu)"
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.AuditLog}} "审计日志列表"
// @Failure 500 {object} response.Response "查询失败"
// @Router /audit-logs [get]
func (h *AuditLogHandler) List(c *gin.Context) {
	page, size := parsePageSize(c)
	username := c.Query("username")
	action := c.Query("action")
	resource := c.Query("resource")

	logs, total, err := h.auditLogService.List(page, size, username, action, resource)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, logs, total, page, size)
}
