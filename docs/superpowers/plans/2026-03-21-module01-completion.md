# Module 01 收尾 + 前端权限体系 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 完成 Module 01 前后端底座的全部遗留项，建立前端状态管理和权限体系，使系统具备完整的 RBAC 前端落地能力和操作审计能力。

**Architecture:** 分 6 个 Task 按依赖顺序推进：先建立 Pinia 状态管理基座（后续所有改造依赖），然后实现动态路由/菜单（权限可视化），接着做按钮级权限指令，再补全操作审计日志（前后端全链路），最后做布局增强（标签页/面包屑）。每个 Task 都是独立可测试的交付单元。

**Tech Stack:** Go/Gin (backend), Vue 3 + TypeScript + Pinia + Element Plus (frontend), GORM (ORM), Zap (logging)

**依赖关系:**
```
Task 1 (Pinia) ──→ Task 2 (动态路由/菜单) ──→ Task 3 (v-permission)
                                              ↗
Task 4 (审计日志后端) ──→ Task 5 (审计日志前端)
Task 6 (布局增强) — 独立，可并行
```

---

## File Structure

### 新建文件

| 文件 | 职责 |
|------|------|
| `frontend/src/stores/user.ts` | 用户信息 store（token、profile、登录/登出） |
| `frontend/src/stores/permission.ts` | 权限 store（菜单树、动态路由、权限列表） |
| `frontend/src/directives/permission.ts` | v-permission 自定义指令 |
| `frontend/src/views/AuditLogs.vue` | 操作审计日志页面 |
| `backend/internal/model/audit_log.go` | AuditLog GORM 模型 |
| `backend/internal/repository/audit_log_repository.go` | 审计日志数据访问层 |
| `backend/internal/service/audit_log_service.go` | 审计日志业务逻辑 |
| `backend/internal/handler/audit_log_handler.go` | 审计日志 HTTP 处理器 |
| `backend/internal/middleware/audit.go` | 审计日志自动记录中间件 |

### 修改文件

| 文件 | 变更内容 |
|------|----------|
| `frontend/src/router/index.ts` | 改为动态路由注册，静态路由只保留 login + layout 壳 |
| `frontend/src/views/Layout.vue` | 侧边栏从硬编码改为读取 permissionStore 菜单树；增加标签页和面包屑 |
| `frontend/src/views/Login.vue` | 登录后调用 store 动作而非直接操作 localStorage |
| `frontend/src/api/index.ts` | 添加 auditLogApi；拦截器改用 userStore |
| `frontend/src/main.ts` | 注册 v-permission 指令 |
| `backend/api/http/router/router.go` | 注册审计日志路由 |
| `backend/cmd/core/main.go` | AutoMigrate 添加 AuditLog 模型 |

---

## Task 1: Pinia 状态管理基座

**Files:**
- Create: `frontend/src/stores/user.ts`
- Create: `frontend/src/stores/permission.ts`
- Modify: `frontend/src/views/Login.vue`
- Modify: `frontend/src/api/index.ts`

### 目标
把散落在各组件中的 localStorage token 操作和 authApi.getInfo() 调用集中到 Pinia store，为后续动态路由和权限指令提供响应式数据源。

- [ ] **Step 1: 创建 userStore**

```typescript
// frontend/src/stores/user.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { authApi } from '../api'

export interface UserInfo {
  id: number
  username: string
  email: string | null
  phone: string
  real_name: string
  avatar: string
  status: number
}

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const userInfo = ref<UserInfo | null>(null)

  function setToken(t: string) {
    token.value = t
    localStorage.setItem('token', t)
  }

  function clearToken() {
    token.value = ''
    userInfo.value = null
    localStorage.removeItem('token')
  }

  async function fetchUserInfo() {
    const res: any = await authApi.getInfo()
    userInfo.value = res.data
    return res.data
  }

  async function login(username: string, password: string) {
    const res: any = await authApi.login(username, password)
    setToken(res.data.token)
    userInfo.value = res.data.user
    return res.data
  }

  async function logout() {
    try { await authApi.logout() } catch {}
    clearToken()
  }

  return { token, userInfo, setToken, clearToken, fetchUserInfo, login, logout }
})
```

- [ ] **Step 2: 创建 permissionStore 骨架**

```typescript
// frontend/src/stores/permission.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { menuApi } from '../api'
import type { RouteRecordRaw } from 'vue-router'

export interface MenuItem {
  id: number
  parent_id: number
  name: string
  title: string
  icon: string
  path: string
  component: string
  api_path: string
  api_method: string
  type: number  // 1=目录 2=菜单 3=按钮
  sort: number
  visible: number
  children?: MenuItem[]
}

export const usePermissionStore = defineStore('permission', () => {
  const menus = ref<MenuItem[]>([])
  const permissions = ref<string[]>([])  // 按钮权限标识列表
  const dynamicRoutes = ref<RouteRecordRaw[]>([])
  const isRoutesGenerated = ref(false)

  async function fetchMenus() {
    const res: any = await menuApi.userMenus()
    menus.value = res.data || []
    // 从菜单树中提取 type=3 的按钮权限
    permissions.value = extractPermissions(menus.value)
    return menus.value
  }

  function extractPermissions(items: MenuItem[]): string[] {
    const perms: string[] = []
    for (const item of items) {
      if (item.type === 3 && item.name) perms.push(item.name)
      if (item.children?.length) perms.push(...extractPermissions(item.children))
    }
    return perms
  }

  function hasPermission(perm: string): boolean {
    return permissions.value.includes(perm)
  }

  function reset() {
    menus.value = []
    permissions.value = []
    dynamicRoutes.value = []
    isRoutesGenerated.value = false
  }

  return { menus, permissions, dynamicRoutes, isRoutesGenerated, fetchMenus, hasPermission, reset }
})
```

- [ ] **Step 3: 改造 Login.vue 使用 userStore**

修改 `frontend/src/views/Login.vue` 中的登录逻辑，将直接调用 `authApi.login` + `localStorage.setItem` 改为调用 `userStore.login()`：

```typescript
// 替换原有登录逻辑
import { useUserStore } from '../stores/user'
const userStore = useUserStore()

async function handleLogin() {
  // ... 表单校验 ...
  const data = await userStore.login(form.value.username, form.value.password)
  ElMessage.success('登录成功')
  router.push('/')
}
```

同样替换注册后的 token 处理。

- [ ] **Step 4: 改造 api/index.ts 拦截器使用 userStore**

```typescript
// 修改请求拦截器，从 userStore 读 token（保持 localStorage 兜底）
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')  // 保持兼容，userStore 写入时同步到 localStorage
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})
```

注意：因为 axios 拦截器在 store 初始化之前注册，保持从 localStorage 读取是正确的做法。userStore.setToken() 已经同步写入 localStorage。

- [ ] **Step 5: 改造 Layout.vue 使用 userStore**

将 `Layout.vue` 中的 `authApi.getInfo()` 调用和 `userInfo` 本地 ref 替换为 userStore：

```typescript
import { useUserStore } from '../stores/user'
const userStore = useUserStore()

onMounted(async () => {
  if (!userStore.userInfo) {
    try { await userStore.fetchUserInfo() }
    catch { router.push('/login') }
  }
})

// 模板中 userInfo?.username → userStore.userInfo?.username
// handleLogout 中调用 userStore.logout()
```

- [ ] **Step 6: 手动验证**

```bash
cd /root/bigOps/frontend && npm run build
```
Expected: 编译通过，无类型错误

- [ ] **Step 7: Commit**

```bash
cd /root/bigOps
git add frontend/src/stores/user.ts frontend/src/stores/permission.ts frontend/src/views/Login.vue frontend/src/views/Layout.vue frontend/src/api/index.ts
git commit -m "feat: 添加 Pinia 状态管理 (userStore + permissionStore)"
```

---

## Task 2: 动态路由与动态菜单

**Files:**
- Modify: `frontend/src/stores/permission.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/views/Layout.vue`

### 目标
登录后从后端 `/menus/user` 获取用户有权限的菜单树，动态生成 Vue Router 路由并渲染侧边栏，替代当前硬编码的 3 个路由。

### 依赖
- Task 1 (Pinia stores 已就位)

- [ ] **Step 1: 定义页面组件映射表**

在 `permission.ts` 中添加组件映射逻辑。前端页面文件使用 Vite 的 `import.meta.glob` 动态加载：

```typescript
// 在 permission.ts 中添加

// 动态导入所有 views 下的 .vue 文件
const viewModules = import.meta.glob('../views/**/*.vue')

function resolveComponent(component: string) {
  // component 值示例: "Layout", "system/Users", "system/Roles"
  const path = `../views/${component}.vue`
  if (viewModules[path]) return viewModules[path]
  console.warn(`Component not found: ${component}`)
  return () => import('../views/404.vue')
}
```

- [ ] **Step 2: 创建 404 页面**

```vue
<!-- frontend/src/views/404.vue -->
<template>
  <div style="display: flex; justify-content: center; align-items: center; height: 60vh; flex-direction: column;">
    <h1 style="font-size: 72px; color: #909399; margin-bottom: 16px;">404</h1>
    <p style="color: #909399;">页面不存在</p>
    <el-button type="primary" @click="$router.push('/')" style="margin-top: 16px;">返回首页</el-button>
  </div>
</template>
```

- [ ] **Step 3: 实现 generateRoutes 方法**

在 `permission.ts` 中添加从菜单树生成路由的逻辑：

```typescript
function generateRoutes(menuTree: MenuItem[]): RouteRecordRaw[] {
  const routes: RouteRecordRaw[] = []
  for (const menu of menuTree) {
    // 跳过按钮权限（type=3）和不可见菜单
    if (menu.type === 3 || menu.visible !== 1) continue
    if (!menu.path) continue

    if (menu.type === 1 && menu.children?.length) {
      // 目录：递归子菜单
      const children = generateRoutes(menu.children)
      routes.push(...children)
    } else if (menu.type === 2 && menu.component) {
      // 菜单页面：创建路由记录
      routes.push({
        path: menu.path.startsWith('/') ? menu.path.slice(1) : menu.path,
        name: menu.name,
        component: resolveComponent(menu.component),
        meta: { title: menu.title, icon: menu.icon },
      })
    }
  }
  return routes
}

// 在 store 中暴露 buildRoutes 动作
async function buildRoutes(): Promise<RouteRecordRaw[]> {
  if (isRoutesGenerated.value) return dynamicRoutes.value
  await fetchMenus()
  dynamicRoutes.value = generateRoutes(menus.value)
  isRoutesGenerated.value = true
  return dynamicRoutes.value
}

// 暴露到 return
return { menus, permissions, dynamicRoutes, isRoutesGenerated, fetchMenus, buildRoutes, hasPermission, reset }
```

- [ ] **Step 4: 改造 router/index.ts 为动态注册模式**

```typescript
// frontend/src/router/index.ts
import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

// 静态路由：不需要权限的页面
export const staticRoutes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
  },
  {
    path: '/404',
    name: 'NotFound',
    component: () => import('../views/404.vue'),
  },
]

// Layout 壳路由：动态路由将作为其 children 添加
export const layoutRoute: RouteRecordRaw = {
  path: '/',
  name: 'Layout',
  component: () => import('../views/Layout.vue'),
  meta: { requiresAuth: true },
  children: [],
}

const router = createRouter({
  history: createWebHistory(),
  routes: [...staticRoutes, layoutRoute],
})

// 标记是否已添加动态路由
let dynamicRoutesAdded = false

router.beforeEach(async (to) => {
  const token = localStorage.getItem('token')

  // 未登录 → 去登录页
  if (!token) {
    if (to.path === '/login') return true
    return '/login'
  }

  // 已登录访问登录页 → 重定向首页
  if (to.path === '/login') return '/'

  // 已添加动态路由 → 放行
  if (dynamicRoutesAdded) {
    if (to.matched.length === 0) return '/404'
    return true
  }

  // 首次进入：加载动态路由
  const { usePermissionStore } = await import('../stores/permission')
  const { useUserStore } = await import('../stores/user')
  const permissionStore = usePermissionStore()
  const userStore = useUserStore()

  try {
    if (!userStore.userInfo) await userStore.fetchUserInfo()
    const routes = await permissionStore.buildRoutes()

    // 动态添加子路由到 layout（使用 name 而非 path）
    for (const route of routes) {
      router.addRoute('Layout', route)
    }

    // 添加兜底 404（必须最后添加）
    router.addRoute({ path: '/:pathMatch(.*)*', redirect: '/404' })

    dynamicRoutesAdded = true

    // 如果访问的是根路径，重定向到第一个动态路由
    if (to.path === '/' && routes.length > 0) {
      return { path: '/' + routes[0].path, replace: true }
    }

    // 重新导航到目标路由（确保新路由生效）
    return { ...to, replace: true }
  } catch {
    localStorage.removeItem('token')
    return '/login'
  }
})

// 导出重置函数供登出时调用
export function resetRouter() {
  dynamicRoutesAdded = false
}

export default router
```

- [ ] **Step 5: 改造 Layout.vue 侧边栏为动态菜单**

将硬编码的 `<el-sub-menu>` 替换为从 permissionStore.menus 动态渲染：

```vue
<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '../stores/user'
import { usePermissionStore } from '../stores/permission'
import { resetRouter } from '../router'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const permissionStore = usePermissionStore()
const isCollapse = ref(false)

// 修改密码
const pwdVisible = ref(false)
const pwdForm = ref({ old_password: '', new_password: '', confirm_password: '' })

// 只显示 type=1(目录) 和 type=2(菜单) 且 visible=1 的菜单项
const visibleMenus = computed(() => {
  return permissionStore.menus.filter(m => m.visible === 1 && m.type !== 3)
})

async function handleLogout() {
  try {
    await ElMessageBox.confirm('确定退出登录？', '提示', { type: 'warning' })
    await userStore.logout()
    permissionStore.reset()
    resetRouter()
    router.push('/login')
  } catch {}
}

// ... 保留 openPwdDialog, submitPwd 不变，submitPwd 中改用 userStore.logout() ...
</script>

<template>
  <!-- 侧边栏菜单改为动态渲染 -->
  <el-menu :default-active="route.path" router :collapse="isCollapse"
           background-color="#304156" text-color="#bfcbd9" active-text-color="#409eff">
    <template v-for="menu in visibleMenus" :key="menu.id">
      <!-- 有子菜单的目录 -->
      <el-sub-menu v-if="menu.children?.length" :index="menu.path || String(menu.id)">
        <template #title>
          <el-icon v-if="menu.icon"><component :is="menu.icon" /></el-icon>
          <span>{{ menu.title }}</span>
        </template>
        <template v-for="child in menu.children" :key="child.id">
          <el-menu-item v-if="child.type !== 3 && child.visible === 1" :index="child.path">
            <el-icon v-if="child.icon"><component :is="child.icon" /></el-icon>
            <span>{{ child.title }}</span>
          </el-menu-item>
        </template>
      </el-sub-menu>
      <!-- 无子菜单的页面 -->
      <el-menu-item v-else :index="menu.path">
        <el-icon v-if="menu.icon"><component :is="menu.icon" /></el-icon>
        <span>{{ menu.title }}</span>
      </el-menu-item>
    </template>
  </el-menu>
</template>
```

- [ ] **Step 6: 验证编译**

```bash
cd /root/bigOps/frontend && npm run build
```
Expected: 编译通过

- [ ] **Step 7: Commit**

```bash
cd /root/bigOps
git add frontend/src/stores/permission.ts frontend/src/router/index.ts frontend/src/views/Layout.vue frontend/src/views/404.vue
git commit -m "feat: 实现动态路由和动态菜单渲染"
```

---

## Task 3: v-permission 按钮级权限指令

**Files:**
- Create: `frontend/src/directives/permission.ts`
- Modify: `frontend/src/main.ts`

### 目标
实现 `v-permission="'user:delete'"` 指令，当用户没有对应权限时自动隐藏按钮/元素。

### 依赖
- Task 1 (permissionStore 已有 hasPermission 方法)

- [ ] **Step 1: 创建 permission 指令**

```typescript
// frontend/src/directives/permission.ts
import type { Directive, DirectiveBinding } from 'vue'
import { usePermissionStore } from '../stores/permission'

export const permission: Directive = {
  mounted(el: HTMLElement, binding: DirectiveBinding<string | string[]>) {
    const permissionStore = usePermissionStore()
    const value = binding.value

    if (!value) return

    const perms = Array.isArray(value) ? value : [value]
    const hasPermission = perms.some(p => permissionStore.hasPermission(p))

    if (!hasPermission) {
      el.parentNode?.removeChild(el)
    }
  },
}
```

- [ ] **Step 2: 在 main.ts 中注册指令**

在 `frontend/src/main.ts` 中添加：

```typescript
import { permission } from './directives/permission'

// 在 app.mount('#app') 之前
app.directive('permission', permission)
```

- [ ] **Step 3: 在现有页面添加使用示例**

在 `Users.vue` 的删除按钮上添加权限控制示例：

```vue
<el-button v-permission="'user:delete'" type="danger" size="small" @click="handleDelete(row)">删除</el-button>
```

注意：这是示例用法，需要在菜单管理中添加对应的 type=3 按钮权限记录才会生效。当前如果 permissionStore.permissions 为空，所有按钮都会被隐藏。因此需要判断：如果用户是 admin 角色或未配置任何按钮权限，则默认显示所有按钮。

改进 permission 指令：

```typescript
export const permission: Directive = {
  mounted(el: HTMLElement, binding: DirectiveBinding<string | string[]>) {
    const permissionStore = usePermissionStore()
    const value = binding.value

    if (!value) return

    // 如果没有配置任何按钮权限（系统未初始化权限），不做限制
    if (permissionStore.permissions.length === 0) return

    const perms = Array.isArray(value) ? value : [value]
    const hasPermission = perms.some(p => permissionStore.hasPermission(p))

    if (!hasPermission) {
      el.parentNode?.removeChild(el)
    }
  },
}
```

- [ ] **Step 4: 验证编译**

```bash
cd /root/bigOps/frontend && npm run build
```
Expected: 编译通过

- [ ] **Step 5: Commit**

```bash
cd /root/bigOps
git add frontend/src/directives/permission.ts frontend/src/main.ts
git commit -m "feat: 添加 v-permission 按钮级权限指令"
```

---

## Task 4: 操作审计日志后端

**Files:**
- Create: `backend/internal/model/audit_log.go`
- Create: `backend/internal/repository/audit_log_repository.go`
- Create: `backend/internal/service/audit_log_service.go`
- Create: `backend/internal/handler/audit_log_handler.go`
- Create: `backend/internal/middleware/audit.go`
- Modify: `backend/api/http/router/router.go`
- Modify: `backend/cmd/core/main.go`

### 目标
将当前散落在各 handler 中的 `logger.Info("xxx操作", ...)` 审计日志入库，并提供查询接口。通过中间件自动捕获写操作，无需手动在每个 handler 中重复记录。

### 依赖
- 无（纯后端，可与 Task 1-3 并行）

- [ ] **Step 1: 创建 AuditLog 模型**

```go
// backend/internal/model/audit_log.go
package model

// AuditLog 操作审计日志模型，对应 audit_logs 表。
type AuditLog struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"index;not null" json:"user_id"`
	Username  string    `gorm:"size:50;index;not null" json:"username"`
	Action    string    `gorm:"size:50;index;not null" json:"action"`    // login/logout/create/update/delete
	Resource  string    `gorm:"size:50;index" json:"resource"`           // user/role/menu
	ResourceID int64   `gorm:"default:0" json:"resource_id"`
	Detail    string    `gorm:"type:text" json:"detail"`                 // 操作详情描述
	IP        string    `gorm:"size:50" json:"ip"`
	UserAgent string    `gorm:"size:255" json:"user_agent"`
	Status    int8      `gorm:"default:1;not null" json:"status"`        // 1=成功 0=失败
	CreatedAt LocalTime `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
```

- [ ] **Step 2: 创建 AuditLog Repository**

```go
// backend/internal/repository/audit_log_repository.go
package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type AuditLogRepository struct{}

func NewAuditLogRepository() *AuditLogRepository {
	return &AuditLogRepository{}
}

func (r *AuditLogRepository) Create(log *model.AuditLog) error {
	return database.GetDB().Create(log).Error
}

func (r *AuditLogRepository) List(page, size int, username, action, resource string) ([]*model.AuditLog, int64, error) {
	var logs []*model.AuditLog
	var total int64

	db := database.GetDB().Model(&model.AuditLog{})

	if username != "" {
		db = db.Where("username LIKE ?", "%"+username+"%")
	}
	if action != "" {
		db = db.Where("action = ?", action)
	}
	if resource != "" {
		db = db.Where("resource = ?", resource)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
```

- [ ] **Step 3: 创建 AuditLog Service**

```go
// backend/internal/service/audit_log_service.go
package service

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type AuditLogService struct {
	repo *repository.AuditLogRepository
}

func NewAuditLogService() *AuditLogService {
	return &AuditLogService{repo: repository.NewAuditLogRepository()}
}

func (s *AuditLogService) Record(userID int64, username, action, resource string, resourceID int64, detail, ip, userAgent string, status int8) {
	log := &model.AuditLog{
		UserID:     userID,
		Username:   username,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Detail:     detail,
		IP:         ip,
		UserAgent:  userAgent,
		Status:     status,
	}
	// 异步写入，不阻塞请求
	go func() {
		_ = s.repo.Create(log)
	}()
}

func (s *AuditLogService) List(page, size int, username, action, resource string) ([]*model.AuditLog, int64, error) {
	return s.repo.List(page, size, username, action, resource)
}
```

- [ ] **Step 4: 创建 AuditLog Handler**

```go
// backend/internal/handler/audit_log_handler.go
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/service"
)

type AuditLogHandler struct {
	auditService *service.AuditLogService
}

func NewAuditLogHandler() *AuditLogHandler {
	return &AuditLogHandler{auditService: service.NewAuditLogService()}
}

// List 查询操作审计日志。
// @Summary 操作审计日志列表
// @Description 分页查询操作审计日志，支持按用户名、操作类型、资源类型筛选
// @Tags 审计日志
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param username query string false "用户名（模糊搜索）"
// @Param action query string false "操作类型" Enums(login,logout,create,update,delete)
// @Param resource query string false "资源类型" Enums(user,role,menu)
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.AuditLog}} "审计日志列表"
// @Failure 500 {object} response.Response "查询失败"
// @Router /audit-logs [get]
func (h *AuditLogHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	username := c.Query("username")
	action := c.Query("action")
	resource := c.Query("resource")

	logs, total, err := h.auditService.List(page, size, username, action, resource)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, logs, total, page, size)
}
```

- [ ] **Step 5: 创建审计中间件**

```go
// backend/internal/middleware/audit.go
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/service"
)

var auditService = service.NewAuditLogService()

// AuditLog 记录写操作的审计日志。
// 通过在 handler 中设置 c.Set("audit_action", "create") 等来触发记录。
func AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 只记录设置了 audit 标记的请求
		action, exists := c.Get("audit_action")
		if !exists {
			return
		}

		resource, _ := c.Get("audit_resource")
		resourceID, _ := c.Get("audit_resource_id")
		detail, _ := c.Get("audit_detail")
		status := int8(1)
		if len(c.Errors) > 0 {
			status = 0
		}

		userID := int64(0)
		if id, ok := c.Get("userID"); ok {
			userID = id.(int64)
		}
		username := ""
		if u, ok := c.Get("username"); ok {
			username = u.(string)
		}

		rid := int64(0)
		if resourceID != nil {
			switch v := resourceID.(type) {
			case int64:
				rid = v
			case int:
				rid = int64(v)
			}
		}

		detailStr := ""
		if detail != nil {
			detailStr, _ = detail.(string)
		}

		auditService.Record(
			userID, username,
			action.(string),
			stringOrEmpty(resource),
			rid, detailStr,
			c.ClientIP(), c.Request.UserAgent(),
			status,
		)
	}
}

func stringOrEmpty(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
```

- [ ] **Step 6: 注册路由和 AutoMigrate**

修改 `router.go`，在 authGroup 中添加审计日志路由和中间件：

```go
// 在 authGroup.Use(middleware.AuthMiddleware()) 后添加审计中间件
authGroup.Use(middleware.AuditLog())

// 在菜单管理路由后添加：
// --- 审计日志 ---
auditLogHandler := handler.NewAuditLogHandler()
authGroup.GET("/audit-logs", auditLogHandler.List)
```

修改 `main.go` AutoMigrate，添加 AuditLog：

```go
if err := database.GetDB().AutoMigrate(&model.User{}, &model.Role{}, &model.Menu{}, &model.UserRole{}, &model.AuditLog{}); err != nil {
```

- [ ] **Step 7: 在关键 handler 中设置审计标记**

在现有 handler 的写操作中，在成功执行业务逻辑之后、调用 `response.Success*()` 之前，添加 `c.Set()` 调用。审计中间件在 `c.Next()` 返回后读取这些值。

**各 handler 审计标记映射表：**

| Handler 方法 | action | resource | detail 模板 |
|--------------|--------|----------|-------------|
| `AuthHandler.Register` | `create` | `user` | `"注册用户: "+username` |
| `AuthHandler.Login` | `login` | `user` | `"用户登录: "+username` |
| `AuthHandler.Logout` | `logout` | `user` | `"用户登出"` |
| `AuthHandler.ChangePassword` | `update` | `user` | `"修改密码"` |
| `UserHandler.UpdateStatus` | `update` | `user` | `"启用/禁用用户: "+username` |
| `UserHandler.Delete` | `delete` | `user` | `"删除用户: "+username` |
| `RoleHandler.Create` | `create` | `role` | `"创建角色: "+name` |
| `RoleHandler.Update` | `update` | `role` | `"更新角色: "+displayName` |
| `RoleHandler.UpdateStatus` | `update` | `role` | `"启用/禁用角色: "+name` |
| `RoleHandler.Delete` | `delete` | `role` | `"删除角色: "+name` |
| `RoleHandler.SetMenus` | `update` | `role` | `"设置角色菜单"` |
| `RoleHandler.SetUserRoles` | `update` | `user` | `"设置用户角色"` |
| `MenuHandler.Create` | `create` | `menu` | `"创建菜单: "+title` |
| `MenuHandler.Update` | `update` | `menu` | `"更新菜单: "+title` |
| `MenuHandler.Delete` | `delete` | `menu` | `"删除菜单"` |

示例（以 user_handler.go Delete 为例）：

```go
func (h *UserHandler) Delete(c *gin.Context) {
    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    if id == 1 {
        response.Error(c, 400, "不允许删除管理员")
        return
    }
    user, _ := h.userRepo.GetByID(id)
    username := ""
    if user != nil {
        username = user.Username
    }
    if err := h.userRepo.Delete(id); err != nil {
        response.InternalServerError(c, "删除失败")
        return
    }
    // 审计标记 — 在成功操作后、响应前设置
    c.Set("audit_action", "delete")
    c.Set("audit_resource", "user")
    c.Set("audit_resource_id", id)  // int64 类型
    c.Set("audit_detail", "删除用户: "+username)

    logger.Info("删除用户", zap.String("operator", getOperator(c)), zap.Int64("user_id", id), zap.String("username", username))
    response.SuccessWithMessage(c, "删除成功", nil)
}
```

注意：Login 和 Register 是公开路由，没有经过 AuthMiddleware，context 中没有 `userID` 和 `username`。审计中间件已处理这种情况（userID 默认 0，username 默认空字符串）。对于 Login，可以在登录成功后手动 `c.Set("username", req.Username)`。

- [ ] **Step 8: 验证编译**

```bash
cd /root/bigOps/backend && go build ./cmd/core/
```
Expected: 编译通过

- [ ] **Step 9: 更新 Swagger 文档**

```bash
export PATH=$PATH:$(go env GOPATH)/bin
cd /root/bigOps/backend && swag init -g cmd/core/main.go -o docs --parseDependency --parseInternal
```
Expected: 生成成功，包含 `/audit-logs` 端点

- [ ] **Step 10: Commit**

```bash
cd /root/bigOps
git add backend/internal/model/audit_log.go backend/internal/repository/audit_log_repository.go backend/internal/service/audit_log_service.go backend/internal/handler/audit_log_handler.go backend/internal/middleware/audit.go backend/api/http/router/router.go backend/cmd/core/main.go backend/docs/
git commit -m "feat: 添加操作审计日志（模型+中间件+查询接口）"
```

---

## Task 5: 审计日志前端页面

**Files:**
- Create: `frontend/src/views/AuditLogs.vue`
- Modify: `frontend/src/api/index.ts`

### 目标
创建审计日志查看页面，支持按用户名、操作类型、资源类型筛选，分页展示。

### 依赖
- Task 4 (后端接口就绪)

- [ ] **Step 1: 添加 auditLogApi**

在 `frontend/src/api/index.ts` 中添加：

```typescript
// 审计日志
export const auditLogApi = {
  list: (params: { page?: number; size?: number; username?: string; action?: string; resource?: string }) =>
    api.get('/audit-logs', { params }),
}
```

- [ ] **Step 2: 创建 AuditLogs.vue**

```vue
<!-- frontend/src/views/AuditLogs.vue -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { auditLogApi } from '../api'

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const query = ref({ page: 1, size: 20, username: '', action: '', resource: '' })

const actionOptions = [
  { label: '全部', value: '' },
  { label: '登录', value: 'login' },
  { label: '登出', value: 'logout' },
  { label: '创建', value: 'create' },
  { label: '更新', value: 'update' },
  { label: '删除', value: 'delete' },
]
const resourceOptions = [
  { label: '全部', value: '' },
  { label: '用户', value: 'user' },
  { label: '角色', value: 'role' },
  { label: '菜单', value: 'menu' },
]

async function fetchData() {
  loading.value = true
  try {
    const res: any = await auditLogApi.list(query.value)
    list.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  query.value.page = 1
  fetchData()
}

function handlePageChange(page: number) {
  query.value.page = page
  fetchData()
}

onMounted(fetchData)
</script>

<template>
  <div>
    <el-card shadow="never" style="margin-bottom: 16px;">
      <el-form inline>
        <el-form-item label="用户名">
          <el-input v-model="query.username" placeholder="搜索用户名" clearable @clear="handleSearch" />
        </el-form-item>
        <el-form-item label="操作类型">
          <el-select v-model="query.action" @change="handleSearch" style="width: 120px;">
            <el-option v-for="o in actionOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="资源类型">
          <el-select v-model="query.resource" @change="handleSearch" style="width: 120px;">
            <el-option v-for="o in resourceOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="用户" width="120" />
        <el-table-column prop="action" label="操作" width="100">
          <template #default="{ row }">
            <el-tag :type="row.action === 'delete' ? 'danger' : row.action === 'create' ? 'success' : 'info'" size="small">
              {{ row.action }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="resource" label="资源" width="100" />
        <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP" width="140" />
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
              {{ row.status === 1 ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" width="180" />
      </el-table>
      <el-pagination
        style="margin-top: 16px; justify-content: flex-end;"
        :current-page="query.page"
        :page-size="query.size"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="handlePageChange"
      />
    </el-card>
  </div>
</template>
```

- [ ] **Step 3: 验证编译**

```bash
cd /root/bigOps/frontend && npm run build
```
Expected: 编译通过

注意：AuditLogs.vue 页面路由将通过动态菜单系统注册（需要在菜单管理中添加对应菜单记录），不需要手动在 router/index.ts 中添加静态路由。

- [ ] **Step 4: Commit**

```bash
cd /root/bigOps
git add frontend/src/api/index.ts frontend/src/views/AuditLogs.vue
git commit -m "feat: 添加审计日志前端页面"
```

---

## Task 6: 布局增强（标签页 + 面包屑）

**Files:**
- Modify: `frontend/src/views/Layout.vue`

### 目标
在 Layout 中增加标签页导航（打开的页面标签）和面包屑路径提示，提升多页面操作体验。

### 依赖
- Task 2 (动态路由 meta.title 就绪)

- [ ] **Step 1: 在 Layout.vue 中添加标签页和面包屑**

在 `<el-header>` 和 `<el-main>` 之间插入标签栏。使用 Vue Router 的 `afterEach` 钩子自动收集已访问路由：

```vue
<script setup lang="ts">
// 在现有 imports 后添加
import { ref, computed, watch } from 'vue'

// 标签页数据
interface TabItem {
  path: string
  title: string
}

const visitedTabs = ref<TabItem[]>([])
const activeTab = ref('')

// 监听路由变化，添加标签
watch(() => route.path, (path) => {
  activeTab.value = path
  const title = (route.meta?.title as string) || route.name as string || path
  if (!visitedTabs.value.find(t => t.path === path)) {
    visitedTabs.value.push({ path, title })
  }
}, { immediate: true })

function handleTabClick(tab: any) {
  router.push(tab.props.name)
}

function handleTabRemove(path: string) {
  const idx = visitedTabs.value.findIndex(t => t.path === path)
  if (idx === -1) return
  visitedTabs.value.splice(idx, 1)
  // 如果关闭的是当前标签，跳转到最后一个标签
  if (path === activeTab.value && visitedTabs.value.length) {
    router.push(visitedTabs.value[visitedTabs.value.length - 1].path)
  }
}

// 面包屑
const breadcrumbs = computed(() => {
  return route.matched
    .filter(r => r.meta?.title)
    .map(r => ({ title: r.meta.title as string, path: r.path }))
})
</script>
```

在 template 中 `<el-header>` 之后、`<el-main>` 之前添加：

```vue
<!-- 标签页导航 -->
<div class="tabs-bar" style="background: #fff; padding: 4px 16px 0; border-bottom: 1px solid #f0f0f0;">
  <el-tabs v-model="activeTab" type="card" closable
           @tab-click="handleTabClick" @tab-remove="handleTabRemove">
    <el-tab-pane v-for="tab in visitedTabs" :key="tab.path" :label="tab.title" :name="tab.path" />
  </el-tabs>
</div>
<!-- 面包屑 -->
<div style="padding: 12px 16px 0; background: #f0f2f5;" v-if="breadcrumbs.length">
  <el-breadcrumb separator="/">
    <el-breadcrumb-item v-for="item in breadcrumbs" :key="item.path">{{ item.title }}</el-breadcrumb-item>
  </el-breadcrumb>
</div>
```

- [ ] **Step 2: 验证编译**

```bash
cd /root/bigOps/frontend && npm run build
```
Expected: 编译通过

- [ ] **Step 3: Commit**

```bash
cd /root/bigOps
git add frontend/src/views/Layout.vue
git commit -m "feat: Layout 添加标签页导航和面包屑"
```

---

## 执行顺序总结

```
Step 1: Task 1 (Pinia)           → 基础，所有前端改造依赖
Step 2: Task 4 (审计日志后端)      → 可与 Task 1 并行（纯后端）
Step 3: Task 2 (动态路由/菜单)     → 依赖 Task 1
Step 4: Task 3 (v-permission)     → 依赖 Task 1
Step 5: Task 5 (审计日志前端)      → 依赖 Task 4
Step 6: Task 6 (布局增强)         → 依赖 Task 2
```

**可并行的组合：**
- Task 1 + Task 4 (前后端各自独立)
- Task 3 + Task 5 + Task 6 (互不依赖，都依赖前序任务)

---

## 完成后的状态

Module 01 前后端底座将达到 **100% 完成**：
- Pinia 状态管理就绪
- 动态路由 + 动态菜单渲染
- 按钮级 v-permission 权限指令
- 操作审计日志全链路（后端记录 + 前端查看）
- 标签页 + 面包屑导航
- **随时可进入 Module 02（服务树/CMDB）开发**
