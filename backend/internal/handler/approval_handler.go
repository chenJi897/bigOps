package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

type ApprovalHandler struct {
	svc *service.ApprovalService
}

func NewApprovalHandler() *ApprovalHandler {
	return &ApprovalHandler{svc: service.NewApprovalService()}
}

type ApprovalActionRequest struct {
	Comment string `json:"comment"`
}

func (h *ApprovalHandler) Pending(c *gin.Context) {
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	items, err := h.svc.ListPendingByApproverID(currentUserID)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, items)
}

func (h *ApprovalHandler) Approve(c *gin.Context) {
	instanceID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	var req ApprovalActionRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.BadRequest(c, "参数错误")
		return
	}
	if err := h.svc.Approve(instanceID, currentUserID, req.Comment); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	c.Set("audit_action", "update")
	c.Set("audit_resource", "approval_instance")
	c.Set("audit_resource_id", instanceID)
	c.Set("audit_detail", "审批通过")
	response.SuccessWithMessage(c, "审批通过", nil)
}

func (h *ApprovalHandler) Reject(c *gin.Context) {
	instanceID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID, _ := c.Get("userID")
	currentUserID, _ := userID.(int64)
	var req ApprovalActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	if err := h.svc.Reject(instanceID, currentUserID, req.Comment); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	c.Set("audit_action", "update")
	c.Set("audit_resource", "approval_instance")
	c.Set("audit_resource_id", instanceID)
	c.Set("audit_detail", "审批拒绝")
	response.SuccessWithMessage(c, "审批已拒绝", nil)
}
