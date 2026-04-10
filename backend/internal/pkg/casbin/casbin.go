// Package casbin 封装 Casbin 权限引擎的初始化和全局访问。
// 使用 RBAC 模型：角色拥有 API 权限，用户通过角色获得权限。
package casbin

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// RBAC 模型定义：
// sub = 角色名, obj = API 路径前缀, act = HTTP 方法（* 表示全部）
// g = 用户-角色映射
// keyMatch 支持 /api/v1/tickets* 匹配 /api/v1/tickets 和 /api/v1/tickets/123/activities
const rbacModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (p.act == "*" || r.act == p.act)) || r.sub == "admin" || g(r.sub, "admin")
`

var globalEnforcer *casbin.Enforcer

// Init 初始化 Casbin 权限引擎，使用 GORM 作为策略存储适配器。
func Init(db *gorm.DB) error {
	// 创建 GORM 适配器，Casbin 策略存储到数据库
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return fmt.Errorf("failed to create casbin adapter: %w", err)
	}

	// 从字符串加载 RBAC 模型
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		return fmt.Errorf("failed to create casbin model: %w", err)
	}

	// 创建 enforcer
	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	// 从数据库加载策略
	if err := enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("failed to load casbin policy: %w", err)
	}

	globalEnforcer = enforcer
	return nil
}

// GetEnforcer 返回全局 Casbin enforcer 实例。
func GetEnforcer() *casbin.Enforcer {
	if globalEnforcer == nil {
		panic("casbin not initialized, call Init() first")
	}
	return globalEnforcer
}
