// backend/internal/handler/service_tree_handler.go
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

var _ model.ServiceTree // swag import

type ServiceTreeHandler struct {
	svc *service.ServiceTreeService
}

func NewServiceTreeHandler() *ServiceTreeHandler {
	return &ServiceTreeHandler{svc: service.NewServiceTreeService()}
}

type CreateServiceTreeRequest struct {
	Name        string `json:"name" binding:"required" example:"基础架构"`
	Code        string `json:"code" example:"infra"`
	ParentID    int64  `json:"parent_id" example:"0"`
	Sort        int    `json:"sort" example:"1"`
	Description string `json:"description" example:"基础架构部"`
	OwnerID     int64  `json:"owner_id" example:"1"`
}

type UpdateServiceTreeRequest struct {
	Name        string `json:"name" binding:"required" example:"基础架构"`
	Code        string `json:"code" example:"infra"`
	Sort        int    `json:"sort" example:"1"`
	Description string `json:"description" example:"基础架构部"`
	OwnerID     int64  `json:"owner_id" example:"1"`
}

type MoveServiceTreeRequest struct {
	ParentID int64 `json:"parent_id" example:"0"`
}

// GetTree 获取完整服务树。
// @Summary 获取服务树
// @Description 获取完整的服务树结构
// @Tags 服务树
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]model.ServiceTree} "服务树"
// @Failure 500 {object} response.Response "查询失败"
// @Router /service-trees [get]
func (h *ServiceTreeHandler) GetTree(c *gin.Context) {
	tree, err := h.svc.GetTree()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, tree)
}

// GetByID 获取节点详情。
// @Summary 获取服务树节点详情
// @Description 根据 ID 获取服务树节点
// @Tags 服务树
// @Produce json
// @Security BearerAuth
// @Param id path int true "节点ID"
// @Success 200 {object} response.Response{data=model.ServiceTree} "节点详情"
// @Failure 404 {object} response.Response "节点不存在"
// @Router /service-trees/{id} [get]
func (h *ServiceTreeHandler) GetByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	node, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, 404, "节点不存在")
		return
	}
	response.Success(c, node)
}

// Create 创建节点。
// @Summary 创建服务树节点
// @Description 创建服务树节点，level 自动根据父节点计算
// @Tags 服务树
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateServiceTreeRequest true "创建请求"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /service-trees [post]
func (h *ServiceTreeHandler) Create(c *gin.Context) {
	var req CreateServiceTreeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	node := &model.ServiceTree{
		Name: req.Name, Code: req.Code, ParentID: req.ParentID,
		Sort: req.Sort, Description: req.Description, OwnerID: req.OwnerID,
	}
	if err := h.svc.Create(node); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建服务树节点", zap.String("operator", getOperator(c)), zap.String("name", req.Name))
	c.Set("audit_action", "create")
	c.Set("audit_resource", "service_tree")
	c.Set("audit_detail", "创建服务树节点: "+req.Name)
	response.SuccessWithMessage(c, "创建成功", nil)
}

// Update 更新节点。
// @Summary 更新服务树节点
// @Description 更新服务树节点信息（不改变父节点，移动请用 move 接口）
// @Tags 服务树
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "节点ID"
// @Param body body UpdateServiceTreeRequest true "更新请求"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /service-trees/{id} [post]
func (h *ServiceTreeHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpdateServiceTreeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	node := &model.ServiceTree{
		ID: id, Name: req.Name, Code: req.Code,
		Sort: req.Sort, Description: req.Description, OwnerID: req.OwnerID,
	}
	if err := h.svc.Update(node); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新服务树节点", zap.String("operator", getOperator(c)), zap.Int64("id", id))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "service_tree")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "更新服务树节点: "+req.Name)
	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除节点。
// @Summary 删除服务树节点
// @Description 删除节点，有子节点时不允许删除
// @Tags 服务树
// @Produce json
// @Security BearerAuth
// @Param id path int true "节点ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "存在子节点"
// @Router /service-trees/{id}/delete [post]
func (h *ServiceTreeHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除服务树节点", zap.String("operator", getOperator(c)), zap.Int64("id", id))
	c.Set("audit_action", "delete")
	c.Set("audit_resource", "service_tree")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "删除服务树节点")
	response.SuccessWithMessage(c, "删除成功", nil)
}

// Move 移动节点。
// @Summary 移动服务树节点
// @Description 将节点移动到新的父节点下
// @Tags 服务树
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "节点ID"
// @Param body body MoveServiceTreeRequest true "目标父节点"
// @Success 200 {object} response.Response "移动成功"
// @Failure 400 {object} response.Response "移动失败"
// @Router /service-trees/{id}/move [post]
func (h *ServiceTreeHandler) Move(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req MoveServiceTreeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Move(id, req.ParentID); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("移动服务树节点", zap.String("operator", getOperator(c)), zap.Int64("id", id), zap.Int64("new_parent_id", req.ParentID))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "service_tree")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "移动服务树节点")
	response.SuccessWithMessage(c, "移动成功", nil)
}
