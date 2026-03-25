package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/repository"
)

const maxPageSize = 100

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
	userID, exists := c.Get("userID")
	if !exists {
		return false
	}
	currentUserID, _ := userID.(int64)
	roleRepo := repository.NewRoleRepository()
	roles, err := roleRepo.GetRolesByUserID(currentUserID)
	if err != nil {
		return false
	}
	for _, role := range roles {
		if role.Name == "admin" {
			return true
		}
	}
	return false
}

func requireAdmin(c *gin.Context) bool {
	if !isAdminUser(c) {
		response.Forbidden(c, "仅管理员可操作")
		return false
	}
	return true
}
