package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

// RoleHandler 角色管理 HTTP 处理器。
type RoleHandler struct {
	roleService *service.RoleService
}

func NewRoleHandler() *RoleHandler {
	return &RoleHandler{roleService: service.NewRoleService()}
}

type createRoleRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=50"`
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
}

// Create 创建角色。POST /api/v1/roles
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
	response.SuccessWithMessage(c, "创建成功", nil)
}

type updateRoleRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Status      int8   `json:"status"`
}

// Update 更新角色。PUT /api/v1/roles/:id
func (h *RoleHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.roleService.Update(id, req.DisplayName, req.Description, req.Sort, req.Status); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除角色。DELETE /api/v1/roles/:id
func (h *RoleHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.roleService.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetByID 获取角色详情。GET /api/v1/roles/:id
func (h *RoleHandler) GetByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	role, err := h.roleService.GetByID(id)
	if err != nil {
		response.Error(c, 404, "角色不存在")
		return
	}
	response.Success(c, role)
}

// List 角色列表。GET /api/v1/roles
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

// SetMenus 设置角色菜单权限。PUT /api/v1/roles/:id/menus
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
	response.SuccessWithMessage(c, "设置成功", nil)
}

type setUserRolesRequest struct {
	RoleIDs []int64 `json:"role_ids"`
}

// SetUserRoles 设置用户角色。PUT /api/v1/users/:id/roles
func (h *RoleHandler) SetUserRoles(c *gin.Context) {
	userID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req setUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	// TODO: 应查询用户名，这里先用 ID 字符串简化处理
	if err := h.roleService.SetUserRoles(userID, req.RoleIDs, c.Param("id")); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "设置成功", nil)
}

// GetUserRoles 获取用户角色。GET /api/v1/users/:id/roles
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

// Create 创建菜单。POST /api/v1/menus
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
	response.SuccessWithMessage(c, "创建成功", nil)
}

// Update 更新菜单。PUT /api/v1/menus/:id
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
	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除菜单。DELETE /api/v1/menus/:id
func (h *MenuHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.menuService.Delete(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetTree 获取完整菜单树。GET /api/v1/menus
func (h *MenuHandler) GetTree(c *gin.Context) {
	tree, err := h.menuService.GetTree()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Success(c, tree)
}

// GetUserMenus 获取当前用户的菜单树。GET /api/v1/menus/user
// 管理员返回全部菜单，普通用户返回其角色关联的菜单。
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
