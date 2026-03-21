// backend/internal/handler/asset_handler.go
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/repository"
	"github.com/bigops/platform/internal/service"
)

var _ model.Asset       // swag import
var _ model.AssetChange // swag import

type AssetHandler struct {
	svc *service.AssetService
}

func NewAssetHandler() *AssetHandler {
	return &AssetHandler{svc: service.NewAssetService()}
}

type CreateAssetRequest struct {
	Hostname      string `json:"hostname" binding:"required" example:"web-server-01"`
	IP            string `json:"ip" binding:"required" example:"10.0.1.1"`
	InnerIP       string `json:"inner_ip" example:"192.168.1.1"`
	OS            string `json:"os" example:"CentOS"`
	OSVersion     string `json:"os_version" example:"7.9"`
	CPUCores      int    `json:"cpu_cores" example:"4"`
	MemoryMB      int    `json:"memory_mb" example:"8192"`
	DiskGB        int    `json:"disk_gb" example:"200"`
	Status        string `json:"status" example:"online" enums:"online,offline"`
	AssetType     string `json:"asset_type" example:"server" enums:"server,network"`
	ServiceTreeID int64  `json:"service_tree_id" example:"1"`
	IDC           string `json:"idc" example:"杭州"`
	SN            string `json:"sn" example:"SN12345"`
	Tags          string `json:"tags" example:"[\"production\",\"web\"]"`
	Remark        string `json:"remark" example:"Web 前端服务器"`
}

// List 资产列表。
// @Summary 资产列表
// @Description 分页获取主机资产列表，支持多条件筛选
// @Tags 资产管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param status query string false "状态" Enums(online,offline)
// @Param service_tree_id query int false "服务树节点ID"
// @Param source query string false "来源" Enums(manual,aliyun,tencent,aws)
// @Param keyword query string false "搜索关键字（主机名/IP）"
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.Asset}} "资产列表"
// @Failure 500 {object} response.Response "查询失败"
// @Router /assets [get]
func (h *AssetHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	serviceTreeID, _ := strconv.ParseInt(c.Query("service_tree_id"), 10, 64)

	q := repository.AssetListQuery{
		Page:          page,
		Size:          size,
		Status:        c.Query("status"),
		ServiceTreeID: serviceTreeID,
		Source:        c.Query("source"),
		Keyword:       c.Query("keyword"),
	}
	assets, total, err := h.svc.List(q)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, assets, total, page, size)
}

// GetByID 资产详情。
// @Summary 资产详情
// @Description 获取主机资产详情
// @Tags 资产管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "资产ID"
// @Success 200 {object} response.Response{data=model.Asset} "资产详情"
// @Failure 404 {object} response.Response "资产不存在"
// @Router /assets/{id} [get]
func (h *AssetHandler) GetByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	asset, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "资产不存在")
		return
	}
	response.Success(c, asset)
}

// Create 创建资产。
// @Summary 创建资产
// @Description 手动创建主机资产
// @Tags 资产管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateAssetRequest true "创建请求"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误/主机名已存在"
// @Router /assets [post]
func (h *AssetHandler) Create(c *gin.Context) {
	var req CreateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	asset := &model.Asset{
		Hostname: req.Hostname, IP: req.IP, InnerIP: req.InnerIP,
		OS: req.OS, OSVersion: req.OSVersion, CPUCores: req.CPUCores,
		MemoryMB: req.MemoryMB, DiskGB: req.DiskGB, Status: req.Status,
		AssetType: req.AssetType, ServiceTreeID: req.ServiceTreeID,
		IDC: req.IDC, SN: req.SN, Tags: req.Tags, Remark: req.Remark,
		Source: "manual",
	}
	if err := h.svc.Create(asset); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建资产", zap.String("operator", getOperator(c)), zap.String("hostname", req.Hostname))
	c.Set("audit_action", "create")
	c.Set("audit_resource", "asset")
	c.Set("audit_detail", "创建资产: "+req.Hostname)
	response.SuccessWithMessage(c, "创建成功", nil)
}

// Update 更新资产。
// @Summary 更新资产
// @Description 更新主机资产信息
// @Tags 资产管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资产ID"
// @Param body body CreateAssetRequest true "更新请求"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /assets/{id} [post]
func (h *AssetHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req CreateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	asset := &model.Asset{
		ID: id, Hostname: req.Hostname, IP: req.IP, InnerIP: req.InnerIP,
		OS: req.OS, OSVersion: req.OSVersion, CPUCores: req.CPUCores,
		MemoryMB: req.MemoryMB, DiskGB: req.DiskGB, Status: req.Status,
		AssetType: req.AssetType, ServiceTreeID: req.ServiceTreeID,
		IDC: req.IDC, SN: req.SN, Tags: req.Tags, Remark: req.Remark,
	}
	if err := h.svc.Update(asset); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新资产", zap.String("operator", getOperator(c)), zap.Int64("id", id))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "asset")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "更新资产: "+req.Hostname)
	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除资产。
// @Summary 删除资产
// @Description 软删除主机资产
// @Tags 资产管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "资产ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "删除失败"
// @Router /assets/{id}/delete [post]
func (h *AssetHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除资产", zap.String("operator", getOperator(c)), zap.Int64("id", id))
	c.Set("audit_action", "delete")
	c.Set("audit_resource", "asset")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "删除资产")
	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetChanges 资产变更历史。
// @Summary 资产变更历史
// @Description 获取指定资产的变更历史记录
// @Tags 资产管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "资产ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.AssetChange}} "变更历史"
// @Failure 500 {object} response.Response "查询失败"
// @Router /assets/{id}/changes [get]
func (h *AssetHandler) GetChanges(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	changeRepo := repository.NewAssetChangeRepository()
	changes, total, err := changeRepo.ListByAssetID(id, page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, changes, total, page, size)
}
