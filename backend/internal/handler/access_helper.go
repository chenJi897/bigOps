package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/repository"
)

const maxPageSize = 100

// parsePathID 从 URL 路径参数解析 int64 ID，解析失败返回 400 并中止请求。
func parsePathID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "无效的 "+name+" 参数")
		return 0, false
	}
	return id, true
}

// parsePageSize 解析分页参数并限制 size 上限。
func parsePageSize(c *gin.Context) (page, size int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ = strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	return
}

func isAdminUser(c *gin.Context) bool {
	// 同一请求内缓存结果，避免重复查库
	if cached, exists := c.Get("_isAdmin"); exists {
		return cached.(bool)
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.Set("_isAdmin", false)
		return false
	}
	currentUserID, _ := userID.(int64)
	roleRepo := repository.NewRoleRepository()
	roles, err := roleRepo.GetRolesByUserID(currentUserID)
	if err != nil {
		c.Set("_isAdmin", false)
		return false
	}
	for _, role := range roles {
		if role.Name == "admin" {
			c.Set("_isAdmin", true)
			return true
		}
	}
	c.Set("_isAdmin", false)
	return false
}

func requireAdmin(c *gin.Context) bool {
	if !isAdminUser(c) {
		response.Forbidden(c, "仅管理员可操作")
		return false
	}
	return true
}
