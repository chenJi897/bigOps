package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/pkg/logger"
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

// Pending godoc
// @Summary 获取我的待审批列表
// @Tags 审批
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Response{data=[]model.ApprovalInstance}
// @Router /approval-instances/pending [get]
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

// Approve godoc
// @Summary 审批通过
// @Tags 审批
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "审批实例ID"
// @Param body body ApprovalActionRequest false "审批意见"
// @Success 200 {object} response.Response
// @Router /approval-instances/{id}/approve [post]
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
	logger.Info("审批通过", zap.String("operator", c.GetString("username")), zap.Int64("instance_id", instanceID), zap.String("ip", c.ClientIP()))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "approval_instance")
	c.Set("audit_resource_id", instanceID)
	c.Set("audit_detail", "审批通过")
	response.SuccessWithMessage(c, "审批通过", nil)
}

// Reject godoc
// @Summary 审批拒绝
// @Tags 审批
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "审批实例ID"
// @Param body body ApprovalActionRequest true "拒绝意见"
// @Success 200 {object} response.Response
// @Router /approval-instances/{id}/reject [post]
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
	logger.Info("审批拒绝", zap.String("operator", c.GetString("username")), zap.Int64("instance_id", instanceID), zap.String("ip", c.ClientIP()))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "approval_instance")
	c.Set("audit_resource_id", instanceID)
	c.Set("audit_detail", "审批拒绝")
	response.SuccessWithMessage(c, "审批已拒绝", nil)
}
