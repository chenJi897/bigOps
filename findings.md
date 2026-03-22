# BigOps - 项目发现与架构决策记录

**最后更新**: 2026-03-22

---

## 1. 架构决策

### 1.1 后端技术栈
| 决策 | 选择 | 原因 |
|------|------|------|
| 语言 | Go 1.26 | 高性能、部署简单 |
| Web 框架 | Gin | 生态成熟、性能好 |
| ORM | GORM | Go 生态最主流 ORM |
| 认证 | JWT (自签发) | 无状态、适合微服务 |
| 权限 | Casbin | 灵活的 RBAC/ABAC 模型 |
| 缓存/黑名单 | Redis | Token 黑名单、后续缓存 |
| 日志 | Zap + lumberjack | 结构化日志 + 文件轮转 |
| 配置 | Viper | 支持多格式、环境变量 |
| API 文档 | Swagger (swaggo) | 自动生成、前端可交互 |

### 1.2 前端技术栈
| 决策 | 选择 | 原因 |
|------|------|------|
| 框架 | Vue 3.5 + TypeScript | 组合式 API、类型安全 |
| 构建 | Vite 8 | 极速 HMR |
| UI 库 | Element Plus | Vue 3 生态最成熟 |
| 状态管理 | Pinia (已使用) | Vue 3 官方推荐 |
| HTTP | Axios | 拦截器、取消请求 |

### 1.3 关键架构决策

**统一使用 GET/POST，移除 PUT/DELETE** (commit 69da01b)
- 原因：简化前端请求、避免某些代理/防火墙对 PUT/DELETE 的限制
- 删除操作使用 `POST /resource/:id/delete`
- 状态更新使用 `POST /resource/:id/status`

**响应约定：业务 code 而非 HTTP 状态码**
- 所有响应 HTTP 200，通过 `code` 字段区分成功/失败
- `code: 0` = 成功，非零 = 业务错误
- 前端拦截器统一处理 `code !== 0`

**Casbin 用户映射使用 username 而非 userID**
- 原因：可读性、调试方便
- 注意：用户名变更需同步 Casbin 策略

**时间字段使用 LocalTime 自定义类型**
- JSON 序列化格式：`"2006-01-02 15:04:05"`
- 避免前端处理 ISO 8601 时区问题
- **重要**：Value() 零值返回 nil，避免 MySQL Error 1292

**云同步架构：SyncRunner + Scheduler** (2026-03-22)
- SyncRunner 是唯一同步入口，手动/定时都调它
- per-account sync.Mutex TryLock 防并发（单进程，不需分布式锁）
- Scheduler 用 Go ticker 每 60s 巡检 enabled 账号
- CloudSyncTask 记录每次同步完整生命周期（独立表，不与 audit_logs 混用）
- 失败不立即重试，等下一个周期

---

## 2. 代码结构与模式

### 2.1 后端分层
```
cmd/core/main.go           → 启动入口（初始化顺序: config → logger → MySQL → AutoMigrate → Casbin → Redis → Router → HTTP）
api/http/router/router.go  → 路由注册中心（单文件）
internal/handler/           → HTTP 处理器（请求绑定 + 响应）
internal/service/           → 业务逻辑层
internal/repository/        → 数据访问层
internal/model/             → GORM 模型
internal/middleware/        → Gin 中间件
internal/pkg/               → 基础设施包
```

### 2.2 代码量统计 (截至 2026-03-22)
- 后端总代码：~6003 行
- 前端总代码：~2686 行
- Handler 文件：6 个 (auth/user/rbac + cloud_account/asset/cloud_sync_task)
- Service 文件：7 个 (auth/role/menu + cloud_account/asset/service_tree + cloud_sync/runner)
- Repository 文件：7 个 (user/role/menu + cloud_account/asset/service_tree/cloud_sync_task)
- Model 文件：9 个 (user/role/menu/local_time + audit_log/service_tree/cloud_account/asset/cloud_sync_task)
- 前端页面：11 个
- API 端点：48 个

### 2.3 单元测试覆盖
| 包 | 状态 |
|----|------|
| internal/pkg/config | PASS (有测试) |
| internal/pkg/crypto | PASS (有测试) |
| internal/pkg/jwt | PASS (有测试) |
| internal/pkg/logger | PASS (有测试) |
| 其他包 | 无测试文件 |

---

## 3. 关键发现

### 3.1 ~~前端硬编码路由问题~~ **已解决** (2026-03-21)
- 已改造为动态路由：登录后调用 `/menus/user` → generateRoutes → addRoute
- Layout.vue 从 permissionStore 渲染侧边栏

### 3.2 ~~Pinia 完全未使用~~ **已解决** (2026-03-21)
- userStore: 管理 token + 用户信息
- permissionStore: 管理菜单 + 权限 + 动态路由

### 3.3 ~~操作审计只有日志无入库~~ **已解决** (2026-03-21)
- AuditLog 模型 + Repository + 中间件自动记录
- 前端审计日志页面 + 筛选

### 3.4 LocalTime 零值陷阱 (2026-03-22 发现)
- 数据库 NULL → Scan 返回零值 → Value() 输出 `0000-00-00` → MySQL Error 1292
- **修复**: Value() 零值返回 nil；所有 Update 必须先查 existing 再改字段
- **规则**: 任何 handler 不能构造新 model 对象直接 Save()，必须经 service 层查出再改

### 3.4 Swagger 定义名称不美观
- swag 生成的类型名称带完整包路径：`github_com_bigops_platform_internal_model.User`
- 不影响功能，但在 Swagger UI 中显示不够友好
- 可通过 `--instanceName` 或类型别名优化（低优先级）

### 3.5 前端缺少 roleApi.updateStatus
- `api/index.ts` 中 roleApi 没有 updateStatus 方法
- 但 `Roles.vue` 页面直接调用了后端接口
- 需要检查前端是否正常工作

### 3.6 部署环境已准备
- `deploy/` 目录有 Docker Compose 配置
- Nginx 配置已准备
- 多模块架构规划 (core/task/monitor/agent/dbmgr)
- 当前只有 core 模块在开发

---

## 4. 依赖版本

### 后端关键依赖
| 依赖 | 版本 |
|------|------|
| go | 1.26.1 |
| gin | v1.10.0 |
| gorm | v1.26.1 |
| casbin | v2.106.0 |
| go-redis | v9.8.0 |
| zap | v1.27.0 |
| viper | v1.20.0 |
| swaggo/swag | v1.16.6 |
| golang-jwt | v5.2.2 |

### 前端关键依赖
| 依赖 | 版本 |
|------|------|
| vue | 3.5.30 |
| vite | 8.0.0 |
| element-plus | 2.13.5 |
| pinia | 3.0.4 |
| vue-router | 4.6.4 |
| axios | 1.13.6 |
| typescript | 5.9.3 |

---

## 5. 风险与注意事项

| 风险 | 级别 | 说明 |
|------|------|------|
| Casbin 中间件未在路由中启用 | 中 | `middleware.CasbinMiddleware()` 存在但未在 router.go 中 Use |
| 无请求频率限制 | 中 | 登录接口无防暴力破解机制 |
| Token 无自动刷新 | 低 | 实施计划中提到但未实现，当前 Token 过期需重新登录 |
| 前端无错误边界 | 低 | 组件异常可能导致白屏 |
| handler 层缺少 user_service | 低 | user_handler 直接调用 repository，跳过了 service 层 |
