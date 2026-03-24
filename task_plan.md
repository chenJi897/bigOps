# BigOps 运维平台 - 项目任务计划

**项目**: BigOps 大运维平台
**启动日期**: 2026-03-17
**当前状态**: Module 01 97% / Module 02 97% / Module 03 工单系统 V1 完成
**最后更新**: 2026-03-25

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

## Phase 2: Module 01 - 前后端底座 `complete`

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
- [x] 前端权限指令 v-permission (`directives/permission.ts`)
- [x] 前端动态路由加载 (`router/index.ts` generateRoutes)

### 2.3 菜单管理 `complete`
- [x] 菜单树形结构查询 `GET /api/v1/menus`
- [x] 用户菜单查询 `GET /api/v1/menus/user`
- [x] 菜单 CRUD 接口
- [x] 前端菜单管理页 (`views/Menus.vue`)
- [x] 前端动态菜单渲染 (Layout 从 permissionStore 加载)

### 2.4 用户管理 `complete`
- [x] 用户列表分页 `GET /api/v1/users`
- [x] 用户启禁用 `POST /api/v1/users/:id/status`
- [x] 用户删除 `POST /api/v1/users/:id/delete`
- [x] 用户编辑（姓名/手机/邮箱/部门）`POST /api/v1/users/:id` `64ffdbc`
- [x] 新建用户时选部门 + 角色 `64ffdbc`
- [x] 前端用户管理页 (`views/Users.vue`)

### 2.45 部门管理 `complete`
- [x] Department 模型 + Repository + Service + Handler `1d0006a`
- [x] 部门 CRUD 6 个 API (list/all/getById/create/update/delete)
- [x] 删除校验：有关联用户时禁止删除
- [x] User 模型新增 department_id
- [x] 前端 Departments.vue + 部门菜单

### 2.5 基础布局 `complete`
- [x] 侧边栏导航 (`views/Layout.vue`)
- [x] 顶部导航栏 + 用户下拉菜单
- [x] 修改密码弹窗
- [x] 面包屑导航
- [x] 标签页多页签 + 右键菜单 `e2b28bc` `88606eb`
- [x] keep-alive 状态保留 `d11b25a`
- [x] 表格体验优化（border 拖拽列宽 + 全局 padding）`e2b28bc`

### 2.6 操作审计日志 `complete`
- [x] 操作日志模型 (AuditLog) `audit_log.go`
- [x] 操作日志中间件 (自动记录写操作) `middleware/audit.go`
- [x] 操作日志查询接口 `GET /api/v1/audit-logs`
- [x] 前端操作日志页面 + 筛选 (`views/AuditLogs.vue`)

### 2.7 Swagger API 文档 `complete`
- [x] 全部 21 个端点 Swagger 注解
- [x] 请求类型导出 + example 标签
- [x] BearerAuth 安全定义
- [x] swag init 生成 docs/ (swagger.json/yaml/docs.go)
- [x] `/swagger/index.html` 路由就绪
- [x] swaggo/swag 升级至 v1.16.6

### 2.8 前端状态管理 `complete`
- [x] Pinia stores 初始化
- [x] 用户信息 store (userStore) `stores/user.ts`
- [x] 权限/菜单 store (permissionStore) `stores/permission.ts`

---

## Phase 5: 工单多级审批与通知中心设计 `in_progress`

### 5.1 目标
- [x] 将现有“工单处理流”升级为“请求/审批/执行”三层模型
- [x] 为资源申请、权限申请、变更申请引入多级审批能力
- [x] 建立统一通知中心，支持站内通知、邮件、Webhook、IM 适配器

### 5.2 设计范围
- [x] 工单类型与审批模板解耦
- [x] 审批阶段、审批实例、审批记录、审批人解析规则建模
- [x] 通过审批后自动生成执行单/资源交付任务
- [x] 通知事件模型与渠道适配器设计

### 5.3 实施顺序建议
- [x] V1: 串行多级审批 + 站内/邮件/Webhook 通知
- [ ] V2: 资源目录化 + 自动执行单 + CMDB/服务树/资源池联动
- [ ] V3: IM Bot 接入（审批提醒、卡片消息、深链）

### 5.4 当前已完成代码
- [x] 设计文档 `docs/08-工单多级审批与通知方案.md`
- [x] 审批与通知基础模型
- [x] 基础 Repository
- [x] `main.go` AutoMigrate 接入
- [x] 请求模板/审批策略 CRUD API
- [x] 请求单创建时初始化审批实例
- [x] 站内通知事件入库与查询接口
- [x] 审批动作流转 API（待审批/通过/拒绝）
- [x] 统一通知发送入口（站内/邮件/Webhook/message-pusher）
- [x] 审批待办前端页面
- [x] 请求模板/审批策略前端管理页
- [x] 通知中心前端接入（头部提醒 + 站内通知抽屉）
- [x] 通知测试发送入口
- [x] 请求单创建页支持 request_template 驱动入口
- [x] 通知事件状态查询与手动重试入口
- [x] 通知失败重试调度器
- [x] 请求模板 schema 可视化编辑器
- [x] 默认管理员菜单 seed（审批/模板/通知入口）
- [x] 工单详情页展示请求表单与审批链
- [x] 通知联调页支持自动刷新、Payload 展开、投递状态明细
- [x] 通知事件/投递状态补充中文摘要、可重试标记与详细响应信息
- [ ] 邮件/Webhook/message-pusher 真实渠道联调

---

## Phase 3: Module 02 - 服务树和 CMDB `complete`

### 3.1 服务树管理 `complete`
- [x] 服务树模型定义 (ServiceTree) `9a68da6`
- [x] 服务树 CRUD 接口 (6 个 API) + 资产数量统计 API
- [x] 树形结构查询
- [x] 节点拖拽移动
- [x] 前端服务树页面 + 树形组件 + 节点资产数量 badge
- [x] 右侧联动资产列表（递归查子节点）
- [x] 删除节点前校验关联资产
- [x] 跳转到资产页自动筛选

### 3.2 云账号管理 `complete`
- [x] 云账号模型 (CloudAccount) + AES-GCM 加密 AK/SK `50c7c19`
- [x] 云账号 CRUD 接口 (7 个 API，含同步)
- [x] 前端云账号管理页面 (`views/CloudAccounts.vue`)
- [x] 同步详细日志 + 阿里云 endpoint 可配置
- [x] last_sync_message 改为 TEXT 类型
- [x] 云账号关联服务树 — 同步资产自动归属指定节点 `7b94db2`

### 3.5 定时同步系统 `complete`
- [x] CloudSyncTask 同步任务记录模型 `a1ac2ab`
- [x] CloudSyncTask Repository (CRUD + 分页)
- [x] CloudAccount 新增 sync_enabled / sync_interval 字段
- [x] SyncRunner 统一同步执行器 (per-account mutex 防重复)
- [x] 重构 handler Sync 方法，改为调用 RunSync()
- [x] Scheduler 定时调度器 (60s 巡检 + Go ticker)
- [x] 同步日志查询 API (3 个新端点)
- [x] 前端同步配置 (编辑表单) + 同步记录抽屉
- [x] main.go 集成调度器启动

### 3.3 CMDB 资产管理 `complete`
- [x] 资产模型定义 (Asset + AssetChange) `25450d9`
- [x] 资产 CRUD 接口 (6 个 API)
- [x] 资产查询和多条件筛选 (status/service_tree_id/source/keyword/recursive)
- [x] 资产关联服务树 (关联查询节点名称 + 表单选择 + 列表展示)
- [x] 前端资产列表 + 详情抽屉 + 表单 (`views/Assets.vue`)
- [ ] 资产导入/导出 (Excel)
- [ ] 批量操作

### 3.4 阿里云 ECS 同步 `complete`
- [x] CloudProvider 接口 + AliyunProvider 实现 `569a06f`
- [x] 同步接口 POST /api/v1/cloud-accounts/:id/sync
- [x] 按 cloud_instance_id upsert + diff 变更记录
- [x] MockProvider 集成测试 (4 场景通过) `60ff9d1`
- [x] DescribeDisks API 获取磁盘信息 `19cbf0a`
- [x] 离线收敛（云端未返回 → offline）`c5a1954`
- [x] 离线恢复（云端再次返回 → online + 变更历史）`31b575b`
- [x] 软删除资产恢复（界面删后再同步）`a0e6507`

### 3.5 资产变更历史 `complete`
- [x] AssetChange 模型 + 查询接口 `f6dc0af`
- [x] 云同步时自动 diff 并记录
- [x] 手动编辑时自动 diff 并记录 `6c5b79e`
- [x] 前端变更历史 tab (资产详情抽屉内)

### 3.6 统计接口 + 首页 `complete`
- [x] /stats/summary 平台摘要（资产/云账号/服务树/用户/部门）`f1b1144`
- [x] /stats/asset-distribution 资产分布（状态/来源/Top 10）`f1b1144`
- [x] 首页改版平台总览（摘要卡片 + 分布条形图）`3b027c4`
- [x] 首页卡片点击跳转对应功能页 `0d5c605`

---

## Phase 4: Module 03 - 自助工单 `complete`

### 4.1 工单管理 `complete`
- [x] 工单模型定义 (Ticket) — 含状态机 + 通用资源关联
- [x] 工单 CRUD 接口 — 创建/列表/详情/状态流转/活动流
- [x] 工单状态机 — open/processing/resolved/closed/cancelled
- [x] 前端工单页面 — TicketList / TicketCreate / TicketDetail
- [x] 工单类型管理 — TicketTypes CRUD + 前端管理页
- [x] 多资产关联 — 弹窗选择器 + 多选 + 服务树联动

### 4.2 工单类型 + 审批流程 `complete`
- [x] 工单类型管理 — TicketType CRUD + 缓存版本号
- [x] 请求模板 — RequestTemplate + schema 可视化编辑器
- [x] 审批策略 — ApprovalPolicy + 串行多级审批阶段
- [x] 审批实例初始化 — 创建请求单时自动初始化审批
- [x] 审批动作流转 — 待办列表/通过/拒绝 + 阶段推进
- [x] 通知中心 — 站内通知 + 邮件/Webhook/message-pusher 骨架
- [x] 通知重试调度器 + 手动重试入口
- [x] 前端审批待办/请求模板/审批策略/通知联调页面

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

## Special Track: gin-vue-admin 前端重构分析 `complete`

- [x] 对照 `backend/docs/swagger.yaml` 与 `backend/api/http/router/router.go` 梳理真实后端能力
- [x] 盘点现有前端路由、菜单、权限、请求封装与页面边界
- [x] 评估 `gin-vue-admin` 现有前端骨架与 BigOps 的兼容点/冲突点
- [x] 形成推荐迁移策略与实施顺序

### 当前结论
- **推荐**：保留 BigOps 现有 Gin 后端，只重构前端壳层与页面组织；不要把 `gin-vue-admin` 的后端模型一并引入。
- **原因**：BigOps 已有完整认证、RBAC、菜单树、CMDB、工单、统计与同步能力，接口风格也已稳定。
- **关键风险**：`swagger.yaml` 当前落后于真实路由，缺少工单与工单类型 15 个操作；不能单靠 Swagger 驱动重构。
- **关键风险**：当前菜单路径规范不统一，页面里存在 `/system/*`、`/cmdb/*` 与扁平 `/users`、`/assets` 混用，迁移前必须先统一 IA。

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
| ~~P1~~ | ~~Module 03 工单系统 (Phase 4)~~ | ~~已完成~~ |
| P1 | 邮件/Webhook/message-pusher 真实渠道联调 | 通知中心外部渠道验证 |
| P2 | 资产导入/导出 Excel (3.3) | CMDB 数据迁移必备 |
| P2 | 资产批量操作 (3.3) | 运维效率 |
| P2 | 腾讯云/AWS 同步 | 多云支持 |
| P3 | Module 04 任务执行中心 (Phase 5) | 自动化运维 |

---

## Errors Encountered

| Error | Context | Resolution |
|-------|---------|------------|
| swag cannot find model.User | auth_handler.go 未导入 model 包 | 添加 `var _ model.User` 确保导入 |
| swag LeftDelim/RightDelim field error | swag CLI v1.16.6 vs go.mod v1.8.12 不兼容 | 升级 go.mod 中 swaggo/swag 到 v1.16.6 |
| go test ./... pattern error | 在项目根目录执行而非 backend/ | 需在 backend/ 目录下执行 |
| AES key invalid hex | config.yaml 中 CHANGE_ME_ 前缀含非 hex 字符 | 替换为合法 64 位 hex key |
| Asset IDC → id_c 列名 | GORM 自动 snake_case 转换 | 添加 `gorm:"column:idc"` tag |
| Asset tags 空字符串 MySQL JSON 报错 | `gorm:"type:json"` 不允许空字符串 | BeforeSave hook 转为 "[]" |
| Sync upsert 覆盖 CreatedAt/Tags | 用新对象 Save 导致零值覆盖 | 改为在 existing 上更新字段 |
| last_sync_message Data too long | varchar(500) 放不下 SDK 错误信息 | 改为 TEXT 类型 |
| ServiceTree code 唯一索引空串冲突 | 多个空 code 违反 unique | uniqueIndex → 普通 index |
| Vue Router parent "/" not found | layoutRoute 没有 name 属性 | 添加 `name: 'Layout'` |
| 子路由 path 以 / 开头 warning | 作为 Layout 子路由不能带前导 / | generateRoutes 去掉前导 / |
| 前端 sync_status 字段不显示 | 后端 JSON 是 last_sync_status | 前端字段名改为 last_sync_status |
| 菜单更新排序 Error 1292 | handler 构造新 Menu{} 零值 CreatedAt | service 层先查 existing 再更新字段 |
| LocalTime 零值 0000-00-00 | 数据库 NULL → Scan 零值 → Value 输出 0000-00-00 | Value() 零值返回 nil |
| 系统管理菜单权限失效 | Layout.vue 硬编码菜单 + 静态路由 | 移除硬编码，全部走动态路由 |
| 同步不更新磁盘 | diffAsset 没对比 DiskGB | 补 disk_gb 对比 |
| 删除后重新同步资产丢失 | hostname uniqueIndex + 软删除冲突 | Unscoped 查找 + RestoreSoftDeleted |
| 首页跳转 404 | 路径写 /assets 但实际是 /cmdb/assets | 修正为数据库实际菜单路径 |
| 部门菜单不显示 | SQL 查 system_dir 但实际是 system | 修正 parent name |
| keep-alive 缓存失效 | include 用路由名而非组件名 | componentName + defineOptions |
| 阿里云 DescribeDisks 磁盘为 0 | DescribeInstances 不返回磁盘 | 新增 DescribeDisks API 调用 |
