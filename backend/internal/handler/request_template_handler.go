package handler

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

var _ model.RequestTemplate // swag

type RequestTemplateHandler struct {
	svc *service.RequestTemplateService
}

func NewRequestTemplateHandler() *RequestTemplateHandler {
	return &RequestTemplateHandler{svc: service.NewRequestTemplateService()}
}

type UpsertRequestTemplateRequest struct {
	Name              string `json:"name" binding:"required"`
	Code              string `json:"code"`
	Category          string `json:"category"`
	ProjectName       string `json:"project_name"`
	EnvironmentName   string `json:"environment_name"`
	Description       string `json:"description"`
	Icon              string `json:"icon"`
	TypeID            int64  `json:"type_id"`
	FormSchema        string `json:"form_schema"`
	ApprovalPolicyID  int64  `json:"approval_policy_id"`
	NodesJSON         string `json:"nodes_json"`
	ExecutionTemplate string `json:"execution_template"`
	TicketKind        string `json:"ticket_kind"`
	AutoCreateOrder   int8   `json:"auto_create_order"`
	NotifyApplicant   int8   `json:"notify_applicant"`
	NotifyChannels    []string `json:"notify_channels"`
	Sort              int    `json:"sort"`
	Status            *int8  `json:"status"`
}

func (h *RequestTemplateHandler) List(c *gin.Context) {
	enabledOnly := c.DefaultQuery("enabled_only", "0") == "1"
	items, err := h.svc.List(enabledOnly)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

func (h *RequestTemplateHandler) GetByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	item, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "请求模板不存在")
		return
	}
	response.Success(c, item)
}

func (h *RequestTemplateHandler) Create(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var req UpsertRequestTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	status := int8(1)
	if req.Status != nil {
		status = *req.Status
	}
	notifyChannels, _ := json.Marshal(req.NotifyChannels)
	item := &model.RequestTemplate{
		Name:              req.Name,
		Code:              req.Code,
		Category:          req.Category,
		ProjectName:       req.ProjectName,
		EnvironmentName:   req.EnvironmentName,
		Description:       req.Description,
		Icon:              req.Icon,
		TypeID:            req.TypeID,
		FormSchema:        req.FormSchema,
		ApprovalPolicyID:  req.ApprovalPolicyID,
		NodesJSON:         req.NodesJSON,
		ExecutionTemplate: req.ExecutionTemplate,
		TicketKind:        req.TicketKind,
		AutoCreateOrder:   req.AutoCreateOrder,
		NotifyApplicant:   req.NotifyApplicant,
		NotifyChannels:    string(notifyChannels),
		Sort:              req.Sort,
		Status:            status,
	}
	if item.AutoCreateOrder == 0 {
		item.AutoCreateOrder = 1
	}
	if item.NotifyApplicant == 0 {
		item.NotifyApplicant = 1
	}
	if err := h.svc.Create(item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建请求模板", zap.String("operator", getOperator(c)), zap.String("name", req.Name))
	c.Set("audit_action", "create")
	c.Set("audit_resource", "request_template")
	c.Set("audit_resource_id", item.ID)
	c.Set("audit_detail", "创建请求模板: "+req.Name)
	response.SuccessWithMessage(c, "创建成功", item)
}

func (h *RequestTemplateHandler) Update(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpsertRequestTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	existing, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "请求模板不存在")
		return
	}
	status := existing.Status
	if req.Status != nil {
		status = *req.Status
	}
	notifyChannels, _ := json.Marshal(req.NotifyChannels)
	item := &model.RequestTemplate{
		Name:              req.Name,
		Code:              req.Code,
		Category:          req.Category,
		ProjectName:       req.ProjectName,
		EnvironmentName:   req.EnvironmentName,
		Description:       req.Description,
		Icon:              req.Icon,
		TypeID:            req.TypeID,
		FormSchema:        req.FormSchema,
		ApprovalPolicyID:  req.ApprovalPolicyID,
		NodesJSON:         req.NodesJSON,
		ExecutionTemplate: req.ExecutionTemplate,
		TicketKind:        req.TicketKind,
		AutoCreateOrder:   req.AutoCreateOrder,
		NotifyApplicant:   req.NotifyApplicant,
		NotifyChannels:    string(notifyChannels),
		Sort:              req.Sort,
		Status:            status,
	}
	if err := h.svc.Update(id, item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新请求模板", zap.String("operator", getOperator(c)), zap.Int64("id", id), zap.String("name", req.Name))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "request_template")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "更新请求模板: "+req.Name)
	response.SuccessWithMessage(c, "更新成功", nil)
}

func (h *RequestTemplateHandler) Delete(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除请求模板", zap.String("operator", getOperator(c)), zap.Int64("id", id))
	c.Set("audit_action", "delete")
	c.Set("audit_resource", "request_template")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "删除请求模板")
	response.SuccessWithMessage(c, "删除成功", nil)
}
