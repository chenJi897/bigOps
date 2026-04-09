package handler

import (

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

var _ model.ApprovalPolicy // swag

type ApprovalPolicyHandler struct {
	svc *service.ApprovalPolicyService
}

func NewApprovalPolicyHandler() *ApprovalPolicyHandler {
	return &ApprovalPolicyHandler{svc: service.NewApprovalPolicyService()}
}

type ApprovalPolicyStageRequest struct {
	StageNo        int    `json:"stage_no" binding:"required"`
	Name           string `json:"name" binding:"required"`
	StageType      string `json:"stage_type"`
	ApproverType   string `json:"approver_type" binding:"required"`
	ApproverConfig string `json:"approver_config"`
	PassRule       string `json:"pass_rule"`
	TimeoutHours   int    `json:"timeout_hours"`
	Required       int8   `json:"required"`
	Sort           int    `json:"sort"`
}

type UpsertApprovalPolicyRequest struct {
	Name        string                       `json:"name" binding:"required"`
	Code        string                       `json:"code" binding:"required"`
	Description string                       `json:"description"`
	Scope       string                       `json:"scope"`
	Enabled     int8                         `json:"enabled"`
	Stages      []ApprovalPolicyStageRequest `json:"stages" binding:"required"`
}

// List godoc
// @Summary 获取审批策略列表
// @Tags 审批策略
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Response{data=[]model.ApprovalPolicy}
// @Router /approval-policies [get]
func (h *ApprovalPolicyHandler) List(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	items, err := h.svc.List()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

// GetByID godoc
// @Summary 获取审批策略详情
// @Tags 审批策略
// @Security BearerAuth
// @Produce json
// @Param id path int true "策略ID"
// @Success 200 {object} response.Response{data=model.ApprovalPolicy}
// @Router /approval-policies/{id} [get]
func (h *ApprovalPolicyHandler) GetByID(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	item, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "审批策略不存在")
		return
	}
	response.Success(c, item)
}

// Create godoc
// @Summary 创建审批策略
// @Tags 审批策略
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body UpsertApprovalPolicyRequest true "审批策略信息"
// @Success 200 {object} response.Response
// @Router /approval-policies [post]
func (h *ApprovalPolicyHandler) Create(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var req UpsertApprovalPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	item, stages := buildApprovalPolicyInput(req)
	item.Enabled = 1
	if err := h.svc.Create(item, stages); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建审批策略", zap.String("operator", getOperator(c)), zap.String("name", req.Name))
	c.Set("audit_action", "create")
	c.Set("audit_resource", "approval_policy")
	c.Set("audit_resource_id", item.ID)
	c.Set("audit_detail", "创建审批策略: "+req.Name)
	response.SuccessWithMessage(c, "创建成功", item)
}

// Update godoc
// @Summary 更新审批策略
// @Tags 审批策略
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "策略ID"
// @Param body body UpsertApprovalPolicyRequest true "审批策略信息"
// @Success 200 {object} response.Response
// @Router /approval-policies/{id} [post]
func (h *ApprovalPolicyHandler) Update(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req UpsertApprovalPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	item, stages := buildApprovalPolicyInput(req)
	item.Enabled = req.Enabled
	if err := h.svc.Update(id, item, stages); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新审批策略", zap.String("operator", getOperator(c)), zap.Int64("id", id), zap.String("name", req.Name))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "approval_policy")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "更新审批策略: "+req.Name)
	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete godoc
// @Summary 删除审批策略
// @Tags 审批策略
// @Security BearerAuth
// @Produce json
// @Param id path int true "策略ID"
// @Success 200 {object} response.Response
// @Router /approval-policies/{id}/delete [post]
func (h *ApprovalPolicyHandler) Delete(c *gin.Context) {
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
	logger.Info("删除审批策略", zap.String("operator", getOperator(c)), zap.Int64("id", id))
	c.Set("audit_action", "delete")
	c.Set("audit_resource", "approval_policy")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "删除审批策略")
	response.SuccessWithMessage(c, "删除成功", nil)
}

func buildApprovalPolicyInput(req UpsertApprovalPolicyRequest) (*model.ApprovalPolicy, []model.ApprovalPolicyStage) {
	item := &model.ApprovalPolicy{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Scope:       req.Scope,
	}
	stages := make([]model.ApprovalPolicyStage, 0, len(req.Stages))
	for _, stage := range req.Stages {
		stages = append(stages, model.ApprovalPolicyStage{
			StageNo:        stage.StageNo,
			Name:           stage.Name,
			StageType:      stage.StageType,
			ApproverType:   stage.ApproverType,
			ApproverConfig: stage.ApproverConfig,
			PassRule:       stage.PassRule,
			TimeoutHours:   stage.TimeoutHours,
			Required:       stage.Required,
			Sort:           stage.Sort,
		})
	}
	return item, stages
}
