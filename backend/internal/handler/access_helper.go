package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/repository"
)

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
