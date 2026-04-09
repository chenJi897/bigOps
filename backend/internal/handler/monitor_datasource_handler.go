package handler

import (

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

type MonitorDatasourceHandler struct {
	svc *service.MonitorDatasourceService
}

func NewMonitorDatasourceHandler() *MonitorDatasourceHandler {
	return &MonitorDatasourceHandler{svc: service.NewMonitorDatasourceService()}
}

// monitorDatasourceRequest 表示创建或更新监控数据源的 payload。
type monitorDatasourceRequest struct {
	// Name 数据源名称。
	Name string `json:"name" binding:"required"`
	// Type 类型（prometheus）。
	Type string `json:"type"`
	// BaseURL 访问地址。
	BaseURL string `json:"base_url" binding:"required"`
	// AccessType 访问方式（proxy/direct）。
	AccessType string `json:"access_type"`
	// AuthType 认证方式（none/basic）。
	AuthType string `json:"auth_type"`
	// Username 认证用户。
	Username string `json:"username"`
	// Password 认证密码。
	Password string `json:"password"`
	// HeadersJSON 自定义 HTTP 头（JSON 字符串）。
	HeadersJSON string `json:"headers_json"`
	// Status 状态（active/inactive）。
	Status string `json:"status"`
}

// List 返回已注册的监控数据源。
// @Summary 数据源列表
// @Description 查询所有监控数据源配置
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]model.MonitorDatasource}
// @Router /monitor/datasources [get]
func (h *MonitorDatasourceHandler) List(c *gin.Context) {
	items, err := h.svc.List()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

// Create 新建监控数据源。
// @Summary 创建数据源
// @Description 仅管理员可操作
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Param body body monitorDatasourceRequest true "数据源请求"
// @Success 200 {object} response.Response{data=model.MonitorDatasource}
// @Router /monitor/datasources [post]
func (h *MonitorDatasourceHandler) Create(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var req monitorDatasourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	item := &model.MonitorDatasource{
		Name:        req.Name,
		Type:        req.Type,
		BaseURL:     req.BaseURL,
		AccessType:  req.AccessType,
		AuthType:    req.AuthType,
		Username:    req.Username,
		Password:    req.Password,
		HeadersJSON: req.HeadersJSON,
		Status:      req.Status,
	}
	if err := h.svc.Create(item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建监控数据源", zap.String("operator", c.GetString("username")), zap.Int64("datasource_id", item.ID), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "创建成功", item)
}

// Update 修改监控数据源配置。
// @Summary 更新数据源
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Param id path int true "数据源 ID"
// @Param body body monitorDatasourceRequest true "数据源请求"
// @Success 200 {object} response.Response{data=model.MonitorDatasource}
// @Router /monitor/datasources/{id} [post]
func (h *MonitorDatasourceHandler) Update(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req monitorDatasourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	item := &model.MonitorDatasource{
		ID:          id,
		Name:        req.Name,
		Type:        req.Type,
		BaseURL:     req.BaseURL,
		AccessType:  req.AccessType,
		AuthType:    req.AuthType,
		Username:    req.Username,
		Password:    req.Password,
		HeadersJSON: req.HeadersJSON,
		Status:      req.Status,
	}
	if err := h.svc.Update(item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新监控数据源", zap.String("operator", c.GetString("username")), zap.Int64("datasource_id", id), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "更新成功", item)
}

// Delete 删除监控数据源。
// @Summary 删除数据源
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Param id path int true "数据源 ID"
// @Success 200 {object} response.Response
// @Router /monitor/datasources/{id}/delete [post]
func (h *MonitorDatasourceHandler) Delete(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除监控数据源", zap.String("operator", c.GetString("username")), zap.Int64("datasource_id", id), zap.String("ip", c.ClientIP()))
	response.SuccessWithMessage(c, "删除成功", nil)
}

// Health 检查数据源健康。
// @Summary 数据源健康检查
// @Tags 监控
// @Produce json
// @Security BearerAuth
// @Param id path int true "数据源 ID"
// @Success 200 {object} response.Response{data=service.MonitorDatasourceHealth}
// @Router /monitor/datasources/{id}/health [get]
func (h *MonitorDatasourceHandler) Health(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	info, err := h.svc.HealthCheck(c.Request.Context(), id)
	if err != nil {
		if info == nil {
			response.Error(c, 400, err.Error())
			return
		}
	}
	response.Success(c, info)
}
