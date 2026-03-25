package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
	cloudsync "github.com/bigops/platform/internal/service/cloud_sync"
)

var _ model.CloudAccount // swag import

type CloudAccountHandler struct {
	svc *service.CloudAccountService
}

func NewCloudAccountHandler() *CloudAccountHandler {
	return &CloudAccountHandler{svc: service.NewCloudAccountService()}
}

type CreateCloudAccountRequest struct {
	Name          string `json:"name" binding:"required" example:"阿里云生产环境"`
	Provider      string `json:"provider" binding:"required,oneof=aliyun tencent aws" example:"aliyun"`
	AccessKey     string `json:"access_key" binding:"required" example:"LTAI5t..."`
	SecretKey     string `json:"secret_key" binding:"required" example:"xxxxxxxx"`
	Region        string `json:"region" example:"cn-hangzhou,cn-beijing"`
	ServiceTreeID int64  `json:"service_tree_id" example:"1"`
	OwnerIDs      string `json:"owner_ids" example:"[1,5]"`
}

type UpdateCloudAccountRequest struct {
	Name          string `json:"name" binding:"required" example:"阿里云生产环境"`
	Region        string `json:"region" example:"cn-hangzhou,cn-beijing"`
	Status        int8   `json:"status" binding:"oneof=0 1" example:"1"`
	ServiceTreeID int64  `json:"service_tree_id" example:"1"`
	OwnerIDs      string `json:"owner_ids" example:"[1,5]"`
}

type UpdateCloudAccountKeysRequest struct {
	AccessKey string `json:"access_key" binding:"required" example:"LTAI5t..."`
	SecretKey string `json:"secret_key" binding:"required" example:"xxxxxxxx"`
}

// List 云账号列表。
// @Summary 云账号列表
// @Description 分页获取云账号列表（AK/SK 不返回）
// @Tags 云账号
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.CloudAccount}} "云账号列表"
// @Failure 500 {object} response.Response "查询失败"
// @Router /cloud-accounts [get]
func (h *CloudAccountHandler) List(c *gin.Context) {
	page, size := parsePageSize(c)
	accounts, total, err := h.svc.List(page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, accounts, total, page, size)
}

// GetByID 云账号详情。
// @Summary 云账号详情
// @Description 获取云账号详情（AK/SK 不返回）
// @Tags 云账号
// @Produce json
// @Security BearerAuth
// @Param id path int true "云账号ID"
// @Success 200 {object} response.Response{data=model.CloudAccount} "云账号详情"
// @Failure 404 {object} response.Response "云账号不存在"
// @Router /cloud-accounts/{id} [get]
func (h *CloudAccountHandler) GetByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	account, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "云账号不存在")
		return
	}
	response.Success(c, account)
}

// Create 创建云账号。
// @Summary 创建云账号
// @Description 创建云账号，AK/SK 加密存储
// @Tags 云账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateCloudAccountRequest true "创建请求"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /cloud-accounts [post]
func (h *CloudAccountHandler) Create(c *gin.Context) {
	var req CreateCloudAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Create(req.Name, req.Provider, req.AccessKey, req.SecretKey, req.Region, req.ServiceTreeID, req.OwnerIDs); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建云账号", zap.String("operator", getOperator(c)), zap.String("name", req.Name), zap.String("provider", req.Provider))
	c.Set("audit_action", "create")
	c.Set("audit_resource", "cloud_account")
	c.Set("audit_detail", "创建云账号: "+req.Name+" ("+req.Provider+")")
	response.SuccessWithMessage(c, "创建成功", nil)
}

// Update 更新云账号。
// @Summary 更新云账号
// @Description 更新云账号基本信息（不含密钥）
// @Tags 云账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "云账号ID"
// @Param body body UpdateCloudAccountRequest true "更新请求"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /cloud-accounts/{id} [post]
func (h *CloudAccountHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpdateCloudAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Update(id, req.Name, req.Region, req.Status, req.ServiceTreeID, req.OwnerIDs); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新云账号", zap.String("operator", getOperator(c)), zap.Int64("id", id))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "cloud_account")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "更新云账号: "+req.Name)
	response.SuccessWithMessage(c, "更新成功", nil)
}

// UpdateKeys 更新密钥。
// @Summary 更新云账号密钥
// @Description 更新云账号的 AK/SK
// @Tags 云账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "云账号ID"
// @Param body body UpdateCloudAccountKeysRequest true "密钥请求"
// @Success 200 {object} response.Response "密钥更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /cloud-accounts/{id}/keys [post]
func (h *CloudAccountHandler) UpdateKeys(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpdateCloudAccountKeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.UpdateKeys(id, req.AccessKey, req.SecretKey); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新云账号密钥", zap.String("operator", getOperator(c)), zap.Int64("id", id))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "cloud_account")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "更新云账号密钥")
	response.SuccessWithMessage(c, "密钥更新成功", nil)
}

// Delete 删除云账号。
// @Summary 删除云账号
// @Description 软删除云账号
// @Tags 云账号
// @Produce json
// @Security BearerAuth
// @Param id path int true "云账号ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "删除失败"
// @Router /cloud-accounts/{id}/delete [post]
func (h *CloudAccountHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除云账号", zap.String("operator", getOperator(c)), zap.Int64("id", id))
	c.Set("audit_action", "delete")
	c.Set("audit_resource", "cloud_account")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "删除云账号")
	response.SuccessWithMessage(c, "删除成功", nil)
}

// Sync 触发云账号同步。
// @Summary 触发云资产同步
// @Description 手动触发从云端同步主机资产到本地
// @Tags 云账号
// @Produce json
// @Security BearerAuth
// @Param id path int true "云账号ID"
// @Success 200 {object} response.Response "同步完成"
// @Failure 400 {object} response.Response "同步失败"
// @Router /cloud-accounts/{id}/sync [post]
func (h *CloudAccountHandler) Sync(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	// 获取操作人信息
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	operatorName := getOperator(c)

	task, err := cloudsync.RunSync(id, "manual", operatorID, operatorName)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	msg := fmt.Sprintf("同步完成: 新增 %d, 更新 %d, 无变化 %d, 离线 %d, 总计 %d",
		task.CreatedCount, task.UpdatedCount, task.UnchangedCount, task.OfflineCount, task.TotalCount)
	c.Set("audit_action", "update")
	c.Set("audit_resource", "cloud_account")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", msg)
	response.SuccessWithMessage(c, msg, task)
}

type UpdateSyncConfigRequest struct {
	SyncEnabled  bool `json:"sync_enabled"`
	SyncInterval int  `json:"sync_interval" example:"30"` // 分钟: 0/10/30/60/1440
}

// UpdateSyncConfig 更新云账号同步调度配置。
// @Summary 更新同步调度配置
// @Description 设置云账号的定时同步开关和周期
// @Tags 云账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "云账号ID"
// @Param body body UpdateSyncConfigRequest true "同步配置"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /cloud-accounts/{id}/sync-config [post]
func (h *CloudAccountHandler) UpdateSyncConfig(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpdateSyncConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	validIntervals := map[int]bool{0: true, 10: true, 30: true, 60: true, 1440: true}
	if !validIntervals[req.SyncInterval] {
		response.BadRequest(c, "同步周期只支持 0/10/30/60/1440 分钟")
		return
	}
	if err := h.svc.UpdateSyncConfig(id, req.SyncEnabled, req.SyncInterval); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新同步配置", zap.String("operator", getOperator(c)), zap.Int64("account_id", id),
		zap.Bool("enabled", req.SyncEnabled), zap.Int("interval", req.SyncInterval))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "cloud_account")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", fmt.Sprintf("更新同步配置: enabled=%v, interval=%d min", req.SyncEnabled, req.SyncInterval))
	response.SuccessWithMessage(c, "更新成功", nil)
}
