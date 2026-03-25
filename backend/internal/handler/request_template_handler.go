package handler

import (
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
	Code              string `json:"code" binding:"required"`
	Category          string `json:"category"`
	Description       string `json:"description"`
	Icon              string `json:"icon"`
	TypeID            int64  `json:"type_id"`
	FormSchema        string `json:"form_schema"`
	ApprovalPolicyID  int64  `json:"approval_policy_id"`
	ExecutionTemplate string `json:"execution_template"`
	TicketKind        string `json:"ticket_kind"`
	AutoCreateOrder   int8   `json:"auto_create_order"`
	NotifyApplicant   int8   `json:"notify_applicant"`
	Sort              int    `json:"sort"`
	Status            int8   `json:"status"`
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
	item := &model.RequestTemplate{
		Name:              req.Name,
		Code:              req.Code,
		Category:          req.Category,
		Description:       req.Description,
		Icon:              req.Icon,
		TypeID:            req.TypeID,
		FormSchema:        req.FormSchema,
		ApprovalPolicyID:  req.ApprovalPolicyID,
		ExecutionTemplate: req.ExecutionTemplate,
		TicketKind:        req.TicketKind,
		AutoCreateOrder:   req.AutoCreateOrder,
		NotifyApplicant:   req.NotifyApplicant,
		Sort:              req.Sort,
		Status:            1,
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
	item := &model.RequestTemplate{
		Name:              req.Name,
		Code:              req.Code,
		Category:          req.Category,
		Description:       req.Description,
		Icon:              req.Icon,
		TypeID:            req.TypeID,
		FormSchema:        req.FormSchema,
		ApprovalPolicyID:  req.ApprovalPolicyID,
		ExecutionTemplate: req.ExecutionTemplate,
		TicketKind:        req.TicketKind,
		AutoCreateOrder:   req.AutoCreateOrder,
		NotifyApplicant:   req.NotifyApplicant,
		Sort:              req.Sort,
		Status:            req.Status,
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
