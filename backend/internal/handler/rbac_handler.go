package handler

import (
	"fmt"

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

// CreateRoleRequest 创建角色请求参数。
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=50" example:"editor"`
	DisplayName string `json:"display_name" binding:"required" example:"编辑员"`
	Description string `json:"description" example:"内容编辑角色"`
	Sort        int    `json:"sort" example:"1"`
}

// Create 创建角色。
// @Summary 创建角色
// @Description 创建新角色，角色名不可重复
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateRoleRequest true "创建角色请求"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误/角色名已存在"
// @Router /roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.roleService.Create(req.Name, req.DisplayName, req.Description, req.Sort); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建角色", zap.String("operator", getOperator(c)), zap.String("role", req.Name), zap.String("display_name", req.DisplayName))
	c.Set("audit_action", "create")
	c.Set("audit_resource", "role")
	c.Set("audit_detail", "创建角色: "+req.Name)
	response.SuccessWithMessage(c, "创建成功", nil)
}

// UpdateRoleRequest 更新角色请求参数。
type UpdateRoleRequest struct {
	DisplayName string `json:"display_name" binding:"required" example:"编辑员"`
	Description string `json:"description" example:"内容编辑角色"`
	Sort        int    `json:"sort" example:"1"`
}

// Update 更新角色。
// @Summary 更新角色
// @Description 更新角色显示名、描述、排序
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "角色ID"
// @Param body body UpdateRoleRequest true "更新角色请求"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 404 {object} response.Response "角色不存在"
// @Router /roles/{id} [post]
func (h *RoleHandler) Update(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req UpdateRoleRequest
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
	c.Set("audit_action", "update")
	c.Set("audit_resource", "role")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "更新角色: "+req.DisplayName)
	response.SuccessWithMessage(c, "更新成功", nil)
}

// UpdateRoleStatusRequest 更新角色状态请求参数。
type UpdateRoleStatusRequest struct {
	Status int8 `json:"status" binding:"oneof=0 1" example:"1" enums:"0,1"` // 0:禁用 1:启用
}

// UpdateStatus 启用/禁用角色。
// @Summary 启用/禁用角色
// @Description 更新角色状态，0=禁用 1=启用，不允许禁用管理员角色
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "角色ID"
// @Param body body UpdateRoleStatusRequest true "状态请求"
// @Success 200 {object} response.Response "启用/禁用成功"
// @Failure 400 {object} response.Response "不允许禁用管理员角色"
// @Failure 404 {object} response.Response "角色不存在"
// @Router /roles/{id}/status [post]
func (h *RoleHandler) UpdateStatus(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req UpdateRoleStatusRequest
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
	c.Set("audit_action", "update")
	c.Set("audit_resource", "role")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", action+"角色: "+role.Name)
	response.SuccessWithMessage(c, action+"成功", nil)
}

// Delete 删除角色。
// @Summary 删除角色
// @Description 删除角色，管理员角色不可删除
// @Tags 角色管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "不允许删除管理员角色"
// @Router /roles/{id}/delete [post]
func (h *RoleHandler) Delete(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
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
	c.Set("audit_action", "delete")
	c.Set("audit_resource", "role")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "删除角色: "+roleName)
	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetByID 获取角色详情。
// @Summary 获取角色详情
// @Description 根据角色 ID 获取角色详细信息
// @Tags 角色管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response{data=model.Role} "角色详情"
// @Failure 404 {object} response.Response "角色不存在"
// @Router /roles/{id} [get]
func (h *RoleHandler) GetByID(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	role, err := h.roleService.GetByID(id)
	if err != nil {
		response.Error(c, 404, "角色不存在")
		return
	}
	response.Success(c, role)
}

// List 角色列表。
// @Summary 角色列表
// @Description 分页获取角色列表
// @Tags 角色管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.Role}} "角色列表"
// @Failure 500 {object} response.Response "查询失败"
// @Router /roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	page, size := parsePageSize(c)
	roles, total, err := h.roleService.List(page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, roles, total, page, size)
}

// SetMenusRequest 设置角色菜单权限请求参数。
type SetMenusRequest struct {
	MenuIDs []int64 `json:"menu_ids" example:"1,2,3"`
}

// SetMenus 设置角色菜单权限。
// @Summary 设置角色菜单权限
// @Description 为角色分配菜单权限，传入菜单 ID 列表
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "角色ID"
// @Param body body SetMenusRequest true "菜单ID列表"
// @Success 200 {object} response.Response "设置成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /roles/{id}/menus [post]
func (h *RoleHandler) SetMenus(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req SetMenusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.roleService.SetMenus(id, req.MenuIDs); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("设置角色菜单", zap.String("operator", getOperator(c)), zap.Int64("role_id", id), zap.Int("menu_count", len(req.MenuIDs)))
	c.Set("audit_action", "update")
	c.Set("audit_resource", "role")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "设置角色菜单")
	response.SuccessWithMessage(c, "设置成功", nil)
}

// SetUserRolesRequest 设置用户角色请求参数。
type SetUserRolesRequest struct {
	RoleIDs []int64 `json:"role_ids" example:"1,2"`
}

// SetUserRoles 设置用户角色。
// @Summary 设置用户角色
// @Description 为用户分配角色，传入角色 ID 列表
// @Tags 用户角色
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param body body SetUserRolesRequest true "角色ID列表"
// @Success 200 {object} response.Response "设置成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 404 {object} response.Response "用户不存在"
// @Router /users/{id}/roles [post]
func (h *RoleHandler) SetUserRoles(c *gin.Context) {
	userID, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req SetUserRolesRequest
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
	c.Set("audit_action", "update")
	c.Set("audit_resource", "user")
	c.Set("audit_resource_id", userID)
	c.Set("audit_detail", "设置用户角色")
	response.SuccessWithMessage(c, "设置成功", nil)
}

// GetUserRoles 获取用户角色。
// @Summary 获取用户角色
// @Description 获取指定用户的角色列表
// @Tags 用户角色
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=[]model.Role} "角色列表"
// @Failure 500 {object} response.Response "查询失败"
// @Router /users/{id}/roles [get]
func (h *RoleHandler) GetUserRoles(c *gin.Context) {
	userID, ok := parsePathID(c, "id")
	if !ok {
		return
	}
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

// CreateMenuRequest 创建/更新菜单请求参数。
type CreateMenuRequest struct {
	ParentID  int64  `json:"parent_id" example:"0"`
	Name      string `json:"name" binding:"required" example:"system"`
	Title     string `json:"title" binding:"required" example:"系统管理"`
	Icon      string `json:"icon" example:"setting"`
	Path      string `json:"path" example:"/system"`
	Component string `json:"component" example:"Layout"`
	APIPath   string `json:"api_path" example:"/api/v1/users"`
	APIMethod string `json:"api_method" example:"GET"`
	Type      int8   `json:"type" binding:"required,oneof=1 2 3" enums:"1,2,3" example:"1"` // 1:目录 2:菜单 3:按钮/权限
	Sort      int    `json:"sort" example:"1"`
	Visible   int8   `json:"visible" enums:"0,1" example:"1"` // 0:隐藏 1:显示
}

// Create 创建菜单。
// @Summary 创建菜单
// @Description 创建菜单/目录/权限按钮，type: 1=目录 2=菜单 3=按钮权限
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateMenuRequest true "创建菜单请求"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /menus [post]
func (h *MenuHandler) Create(c *gin.Context) {
	var req CreateMenuRequest
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
	c.Set("audit_action", "create")
	c.Set("audit_resource", "menu")
	c.Set("audit_detail", "创建菜单: "+req.Title)
	response.SuccessWithMessage(c, "创建成功", nil)
}

// Update 更新菜单。
// @Summary 更新菜单
// @Description 更新菜单信息
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "菜单ID"
// @Param body body CreateMenuRequest true "更新菜单请求"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /menus/{id} [post]
func (h *MenuHandler) Update(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req CreateMenuRequest
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
	c.Set("audit_action", "update")
	c.Set("audit_resource", "menu")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "更新菜单: "+req.Title)
	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除菜单。
// @Summary 删除菜单
// @Description 删除菜单/目录/权限按钮
// @Tags 菜单管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "菜单ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "删除失败"
// @Router /menus/{id}/delete [post]
func (h *MenuHandler) Delete(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.menuService.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("删除菜单", zap.String("operator", getOperator(c)), zap.Int64("menu_id", id))
	c.Set("audit_action", "delete")
	c.Set("audit_resource", "menu")
	c.Set("audit_resource_id", id)
	c.Set("audit_detail", "删除菜单")
	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetTree 获取完整菜单树。
// @Summary 获取完整菜单树
// @Description 获取所有菜单的树形结构
// @Tags 菜单管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]model.Menu} "菜单树"
// @Failure 500 {object} response.Response "查询失败"
// @Router /menus [get]
func (h *MenuHandler) GetTree(c *gin.Context) {
	tree, err := h.menuService.GetTree()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, tree)
}

// GetUserMenus 获取当前用户的菜单树。
// @Summary 获取当前用户菜单
// @Description 获取当前登录用户有权限访问的菜单树，管理员返回所有菜单
// @Tags 菜单管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]model.Menu} "用户菜单树"
// @Failure 401 {object} response.Response "用户未认证"
// @Failure 500 {object} response.Response "查询失败"
// @Router /menus/user [get]
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
