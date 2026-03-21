# BigOps 运维平台 - 项目任务计划

**项目**: BigOps 大运维平台
**启动日期**: 2026-03-17
**当前状态**: 阶段2 核心模块开发中（Module 01 接近完成）
**最后更新**: 2026-03-21

---

## Phase 1: 基础设施搭建 `complete`

### 1.1 项目初始化 `complete`
- [x] Go 模块初始化 (`go.mod`: github.com/bigops/platform)
- [x] Vue 3 + Vite + TypeScript 前端初始化
- [x] 项目目录结构规范化
- [x] .gitignore 配置

### 1.2 后端基础设施代码 `complete`
- [x] 配置管理 - Viper (`internal/pkg/config/config.go` + 测试)
- [x] 日志系统 - Zap + lumberjack 文件轮转 (`internal/pkg/logger/` + 测试)
- [x] MySQL 连接 - GORM (`internal/pkg/database/mysql.go`)
- [x] Redis 连接 - go-redis (`internal/pkg/database/redis.go`)
- [x] JWT 封装 (`internal/pkg/jwt/jwt.go` + 测试)
- [x] 密码加密 - bcrypt (`internal/pkg/crypto/password.go` + 测试)
- [x] 统一响应封装 (`internal/pkg/response/response.go`)
- [x] 请求参数校验 (`internal/pkg/validator/validator.go`)
- [x] Casbin 权限引擎初始化 (`internal/pkg/casbin/casbin.go`)

### 1.3 前端基础设施代码 `complete`
- [x] Axios 封装 + 请求/响应拦截器 (`src/api/index.ts`)
- [x] Vue Router + 路由守卫 (`src/router/index.ts`)
- [x] Element Plus 集成
- [x] Token 自动注入 (localStorage → Authorization header)
- [x] 401 自动跳转登录页

### 1.4 文档与规范 `complete`
- [x] 技术选型文档 (`docs/01-技术选型方案.md`)
- [x] 模块设计方案 (`docs/02-模块设计方案.md`)
- [x] 数据库设计方案 (`docs/03-数据库设计方案.md`)
- [x] 部署架构方案 (`docs/04-部署架构方案.md`)
- [x] 实施计划 (`docs/05-实施计划.md`)
- [x] 项目目录结构 (`docs/06-项目目录结构.md`)
- [x] CLAUDE.md 开发指南

---

## Phase 2: Module 01 - 前后端底座 `in_progress`

### 2.1 用户认证 `complete`
- [x] User 模型 (`internal/model/user.go`) - 含 LocalTime 自定义时间类型
- [x] 用户注册接口 `POST /api/v1/auth/register`
- [x] 用户登录接口 `POST /api/v1/auth/login` (JWT)
- [x] 用户登出接口 `POST /api/v1/auth/logout` (Redis 黑名单)
- [x] 获取用户信息 `GET /api/v1/auth/info`
- [x] 修改密码 `POST /api/v1/auth/password` (含密码复杂度校验)
- [x] 前端登录页 (`views/Login.vue`) - 登录/注册双模式
- [x] 前端 Token 管理

### 2.2 权限管理 (RBAC) `complete`
- [x] Role 模型 (`internal/model/role.go`)
- [x] Menu 模型 (`internal/model/menu.go`) - 含树形结构
- [x] UserRole 多对多关联模型
- [x] Casbin 集成 + 中间件 (`internal/middleware/casbin.go`)
- [x] 角色 CRUD 接口 (list/get/create/update/delete/status)
- [x] 角色菜单分配 `POST /api/v1/roles/:id/menus`
- [x] 用户角色分配 `POST /api/v1/users/:id/roles`
- [x] 前端角色管理页 (`views/Roles.vue`)
- [ ] **前端权限指令 v-permission** (按钮级权限控制)
- [ ] **前端动态路由加载** (根据用户权限动态生成路由)

### 2.3 菜单管理 `complete`
- [x] 菜单树形结构查询 `GET /api/v1/menus`
- [x] 用户菜单查询 `GET /api/v1/menus/user`
- [x] 菜单 CRUD 接口
- [x] 前端菜单管理页 (`views/Menus.vue`)
- [ ] **前端动态菜单渲染** (从后端 /menus/user 加载，替代硬编码路由)

### 2.4 用户管理 `complete`
- [x] 用户列表分页 `GET /api/v1/users`
- [x] 用户启禁用 `POST /api/v1/users/:id/status`
- [x] 用户删除 `POST /api/v1/users/:id/delete`
- [x] 前端用户管理页 (`views/Users.vue`)

### 2.5 基础布局 `partial`
- [x] 侧边栏导航 (`views/Layout.vue`)
- [x] 顶部导航栏 + 用户下拉菜单
- [x] 修改密码弹窗
- [ ] **标签页 (Tabs)** - 多页签导航
- [ ] **面包屑导航**
- [ ] **主题切换** (暗色模式)

### 2.6 操作审计日志 `not_started`
- [ ] 操作日志模型 (AuditLog)
- [ ] 操作日志中间件 (自动记录写操作)
- [ ] 操作日志查询接口
- [ ] 前端操作日志页面 + 筛选

### 2.7 Swagger API 文档 `complete`
- [x] 全部 21 个端点 Swagger 注解
- [x] 请求类型导出 + example 标签
- [x] BearerAuth 安全定义
- [x] swag init 生成 docs/ (swagger.json/yaml/docs.go)
- [x] `/swagger/index.html` 路由就绪
- [x] swaggo/swag 升级至 v1.16.6

### 2.8 前端状态管理 `not_started`
- [ ] Pinia stores 初始化 (目录存在但为空)
- [ ] 用户信息 store (userStore)
- [ ] 权限/菜单 store (permissionStore)

---

## Phase 3: Module 02 - 服务树和 CMDB `not_started`

### 3.1 服务树管理
- [ ] 服务树模型定义 (ServiceTree)
- [ ] 服务树 CRUD 接口
- [ ] 树形结构查询
- [ ] 节点拖拽移动
- [ ] 前端服务树页面 + 树形组件
- [ ] 右键菜单操作

### 3.2 CMDB 资产管理
- [ ] 资产模型定义 (Asset/Host)
- [ ] 资产 CRUD 接口
- [ ] 资产查询和高级筛选
- [ ] 资产导入/导出 (Excel)
- [ ] 资产关联服务树
- [ ] 前端资产列表 + 详情 + 表单页面
- [ ] 批量操作

### 3.3 资产发现
- [ ] SSH 连接封装
- [ ] 资产信息采集 (CPU/内存/磁盘/网络)
- [ ] 定时采集任务 (Asynq)
- [ ] 变更检测 + 变更历史

---

## Phase 4: Module 03 - 自助工单 `not_started`

### 4.1 工单管理
- [ ] 工单模型定义 (Ticket)
- [ ] 工单 CRUD 接口
- [ ] 工单状态机
- [ ] 前端工单页面

### 4.2 工单类型 + 审批流程
- [ ] 工单类型管理
- [ ] 自定义字段配置
- [ ] 审批流程配置
- [ ] 通知功能 (邮件/站内信)

---

## Phase 5: Module 04 - 任务执行中心 `not_started`

- [ ] 任务模型 + CRUD
- [ ] 任务队列 (Asynq)
- [ ] gRPC Server/Client
- [ ] Agent 端开发 (心跳/执行/日志上报)
- [ ] WebSocket 实时日志
- [ ] 前端任务管理 + 脚本编辑器 (Monaco)

---

## Phase 6: Module 05 - 监控平台 `not_started`

- [ ] Prometheus 数据源管理
- [ ] PromQL 查询封装
- [ ] 告警规则管理 + Webhook
- [ ] 监控大盘 (ECharts)

---

## Phase 7: Module 08 - 数据库管理 `not_started`

- [ ] 数据源管理 (连接信息加密)
- [ ] SQL 执行 + 审核
- [ ] 库表结构浏览
- [ ] 前端 SQL 编辑器 (Monaco)

---

## 当前优先级排序

| 优先级 | 任务 | 原因 |
|--------|------|------|
| P0 | 操作审计日志 (2.6) | Module 01 的遗留项，安全合规基础 |
| P0 | Pinia 状态管理 (2.8) | 前端基础，后续所有页面依赖 |
| P1 | 动态菜单 + 动态路由 (2.2/2.3) | 权限系统的前端落地 |
| P1 | v-permission 指令 (2.2) | 按钮级权限控制 |
| P2 | 布局增强 (2.5) | 标签页/面包屑/主题切换 |
| P3 | 服务树 + CMDB (Phase 3) | 运维平台核心数据基础 |
| P4 | 工单系统 (Phase 4) | 运维流程管理 |

---

## Errors Encountered

| Error | Context | Resolution |
|-------|---------|------------|
| swag cannot find model.User | auth_handler.go 未导入 model 包 | 添加 `var _ model.User` 确保导入 |
| swag LeftDelim/RightDelim field error | swag CLI v1.16.6 vs go.mod v1.8.12 不兼容 | 升级 go.mod 中 swaggo/swag 到 v1.16.6 |
| go test ./... pattern error | 在项目根目录执行而非 backend/ | 需在 backend/ 目录下执行 |
