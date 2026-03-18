package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/repository"
	"github.com/bigops/platform/internal/service"
)

// RoleHandler 角色管理 HTTP 处理器。
type RoleHandler struct {
	roleService *service.RoleService
	userRepo    *repository.UserRepository
}

func NewRoleHandler() *RoleHandler {
	return &RoleHandler{
		roleService: service.NewRoleService(),
		userRepo:    repository.NewUserRepository(),
	}
}

// getOperator 从 Context 获取操作人用户名。
func getOperator(c *gin.Context) string {
	if u, ok := c.Get("username"); ok {
		return u.(string)
	}
	return "unknown"
}

type createRoleRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=50"`
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
}

// Create 创建角色。
func (h *RoleHandler) Create(c *gin.Context) {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.roleService.Create(req.Name, req.DisplayName, req.Description, req.Sort); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建角色", zap.String("operator", getOperator(c)), zap.String("role", req.Name), zap.String("display_name", req.DisplayName))
	response.SuccessWithMessage(c, "创建成功", nil)
}

type updateRoleRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
}

// Update 更新角色。
func (h *RoleHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	role, err := h.roleService.GetByID(id)
	if err != nil {
		response.Error(c, 404, "角色不存在")
		return
	}
	if err := h.roleService.Update(id, req.DisplayName, req.Description, req.Sort, role.Status); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新角色", zap.String("operator", getOperator(c)), zap.Int64("role_id", id), zap.String("display_name", req.DisplayName))
	response.SuccessWithMessage(c, "更新成功", nil)
}

type updateRoleStatusRequest struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

// UpdateStatus 启用/禁用角色。
func (h *RoleHandler) UpdateStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req updateRoleStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	role, err := h.roleService.GetByID(id)
	if err != nil {
		response.Error(c, 404, "角色不存在")
		return
	}
	if role.Name == "admin" {
		response.Error(c, 400, "不允许禁用管理员角色")
		return
	}
	if err := h.roleService.Update(id, role.DisplayName, role.Description, role.Sort, req.Status); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	action := "启用"
	if req.Status == 0 {
		action = "禁用"
	}
	logger.Info(fmt.Sprintf("%s角色", action), zap.String("operator", getOperator(c)), zap.Int64("role_id", id), zap.String("role", role.Name))
	response.SuccessWithMessage(c, action+"成功", nil)
}

// Delete 删除角色。
func (h *RoleHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	role, _ := h.roleService.GetByID(id)
	if err := h.roleService.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	roleName := ""
	if role != nil {
		roleName = role.Name
	}
	logger.Info("删除角色", zap.String("operator", getOperator(c)), zap.Int64("role_id", id), zap.String("role", roleName))
	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetByID 获取角色详情。
func (h *RoleHandler) GetByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	role, err := h.roleService.GetByID(id)
	if err != nil {
		response.Error(c, 404, "角色不存在")
		return
	}
	response.Success(c, role)
}

// List 角色列表。
func (h *RoleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	roles, total, err := h.roleService.List(page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, roles, total, page, size)
}

type setMenusRequest struct {
	MenuIDs []int64 `json:"menu_ids"`
}

// SetMenus 设置角色菜单权限。
func (h *RoleHandler) SetMenus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req setMenusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.roleService.SetMenus(id, req.MenuIDs); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("设置角色菜单", zap.String("operator", getOperator(c)), zap.Int64("role_id", id), zap.Int("menu_count", len(req.MenuIDs)))
	response.SuccessWithMessage(c, "设置成功", nil)
}

type setUserRolesRequest struct {
	RoleIDs []int64 `json:"role_ids"`
}

// SetUserRoles 设置用户角色。
func (h *RoleHandler) SetUserRoles(c *gin.Context) {
	userID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req setUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	// 查询真实用户名用于 Casbin 映射
	username := ""
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		response.Error(c, 404, "用户不存在")
		return
	}
	username = user.Username

	if err := h.roleService.SetUserRoles(userID, req.RoleIDs, username); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("设置用户角色", zap.String("operator", getOperator(c)), zap.Int64("user_id", userID), zap.String("username", username), zap.Int("role_count", len(req.RoleIDs)))
	response.SuccessWithMessage(c, "设置成功", nil)
}

// GetUserRoles 获取用户角色。
func (h *RoleHandler) GetUserRoles(c *gin.Context) {
	userID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	roles, err := h.roleService.GetUserRoles(userID)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, roles)
}

// MenuHandler 菜单管理 HTTP 处理器。
type MenuHandler struct {
	menuService *service.MenuService
	roleService *service.RoleService
}

func NewMenuHandler() *MenuHandler {
	return &MenuHandler{
		menuService: service.NewMenuService(),
		roleService: service.NewRoleService(),
	}
}

type createMenuRequest struct {
	ParentID  int64  `json:"parent_id"`
	Name      string `json:"name" binding:"required"`
	Title     string `json:"title" binding:"required"`
	Icon      string `json:"icon"`
	Path      string `json:"path"`
	Component string `json:"component"`
	APIPath   string `json:"api_path"`
	APIMethod string `json:"api_method"`
	Type      int8   `json:"type" binding:"required,oneof=1 2 3"`
	Sort      int    `json:"sort"`
	Visible   int8   `json:"visible"`
}

// Create 创建菜单。
func (h *MenuHandler) Create(c *gin.Context) {
	var req createMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	menu := &model.Menu{
		ParentID: req.ParentID, Name: req.Name, Title: req.Title,
		Icon: req.Icon, Path: req.Path, Component: req.Component,
		APIPath: req.APIPath, APIMethod: req.APIMethod,
		Type: req.Type, Sort: req.Sort, Visible: req.Visible, Status: 1,
	}
	if err := h.menuService.Create(menu); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建菜单", zap.String("operator", getOperator(c)), zap.String("menu", req.Name), zap.String("title", req.Title))
	response.SuccessWithMessage(c, "创建成功", nil)
}

// Update 更新菜单。
func (h *MenuHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req createMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	menu := &model.Menu{
		ID: id, ParentID: req.ParentID, Name: req.Name, Title: req.Title,
		Icon: req.Icon, Path: req.Path, Component: req.Component,
		APIPath: req.APIPath, APIMethod: req.APIMethod,
		Type: req.Type, Sort: req.Sort, Visible: req.Visible,
	}
	if err := h.menuService.Update(menu); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("更新菜单", zap.String("operator", getOperator(c)), zap.Int64("menu_id", id), zap.String("title", req.Title))
	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除菜单。
func (h *MenuHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.menuService.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除菜单", zap.String("operator", getOperator(c)), zap.Int64("menu_id", id))
	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetTree 获取完整菜单树。
func (h *MenuHandler) GetTree(c *gin.Context) {
	tree, err := h.menuService.GetTree()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, tree)
}

// GetUserMenus 获取当前用户的菜单树。
func (h *MenuHandler) GetUserMenus(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "用户未认证")
		return
	}

	menuIDs, isAdmin, err := h.roleService.GetMenuIDsByUserID(userID.(int64))
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	if isAdmin {
		tree, err := h.menuService.GetTree()
		if err != nil {
			response.InternalServerError(c, "查询菜单失败")
			return
		}
		response.Success(c, tree)
		return
	}

	tree, err := h.menuService.GetTreeByIDs(menuIDs)
	if err != nil {
		response.InternalServerError(c, "查询菜单失败")
		return
	}
	response.Success(c, tree)
}
