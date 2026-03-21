package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/service"
)

// AuditLog 审计日志中间件，在 handler 执行后读取 c.Get 中的审计标记并异步写入数据库。
func AuditLog() gin.HandlerFunc {
	auditLogService := service.NewAuditLogService()

	return func(c *gin.Context) {
		c.Next()

		// 读取审计标记
		action, exists := c.Get("audit_action")
		if !exists {
			return
		}

		resource, _ := c.Get("audit_resource")
		detail, _ := c.Get("audit_detail")

		var userID int64
		if uid, ok := c.Get("userID"); ok {
			switch v := uid.(type) {
			case int64:
				userID = v
			case int:
				userID = int64(v)
			}
		}

		username := ""
		if u, ok := c.Get("username"); ok {
			username = u.(string)
		}

		var rid int64
		if resourceID, ok := c.Get("audit_resource_id"); ok {
			switch v := resourceID.(type) {
			case int64:
				rid = v
			case int:
				rid = int64(v)
			}
		}

		actionStr, _ := action.(string)
		resourceStr, _ := resource.(string)
		detailStr, _ := detail.(string)

		log := &model.AuditLog{
			UserID:     userID,
			Username:   username,
			Action:     actionStr,
			Resource:   resourceStr,
			ResourceID: rid,
			Detail:     detailStr,
			IP:         c.ClientIP(),
			StatusCode: c.Writer.Status(),
		}

		auditLogService.Record(log)
	}
}
