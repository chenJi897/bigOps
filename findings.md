# BigOps - 项目发现与架构决策记录

**最后更新**: 2026-03-24

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

### 3.7 工单系统当前是“处理流”，不是“审批流”
- 当前代码的 Ticket 流程是固定状态机：`open -> processing -> resolved/rejected -> closed`
- 现有模型没有“审批模板 / 审批阶段 / 审批实例 / 审批记录 / 审批人解析器”结构
- 因此适合故障/事件处理，但不适合资源申请、权限申请、变更审批

### 3.8 市面主流模式：工单与审批/执行解耦
- Jira Service Management、Freshservice、ServiceNow 都将“审批”作为工单工作流中的独立阶段
- 蓝鲸 bk-itsm 代表的开源运维体系，将流程服务与 CMDB/作业平台联动，而非把审批写死在单一状态机中
- 企业级做法通常将“请求单 -> 审批实例 -> 执行单”拆层，而不是所有工单统一走审批

### 3.9 对 BigOps 的设计启示
- 工单应至少拆分为：Incident（事件/故障）、Service Request（服务申请）、Change（变更）
- 多级审批只对 Request/Change 开启，Incident 保持快速处理流
- 通知中心必须独立于审批引擎，审批是业务真相，IM/邮件只是触达渠道
- IM 选型上：Zulip 更偏纯开源，Mattermost 更偏 ChatOps，Matrix/Element 更偏开放协议和自托管主权

### 3.10 已落地的 V1 底座
- 已新增审批与通知相关模型：
  - `RequestTemplate`
  - `ApprovalPolicy`
  - `ApprovalPolicyStage`
  - `ApprovalInstance`
  - `ApprovalRecord`
  - `ExecutionOrder`
  - `NotificationEvent`
  - `NotificationDelivery`
  - `InAppNotification`
- 已新增基础 Repository，用于后续 CRUD 与流程服务实现
- 已将上述模型接入 `cmd/core/main.go` 的 AutoMigrate
- 已新增请求模板 CRUD 与审批策略 CRUD 的后端 API
- 已支持请求单创建时初始化审批实例，并向首批审批人写入站内通知/通知事件
- 已提供站内通知查询、未读统计、标记已读接口
- 当前仍未实现审批动作流转 API、审批待办前端、邮件/Webhook/IM 投递服务，这些属于下一阶段

### 3.11 本轮 review 缺陷已修复
- 工单活动流接口现在复用与详情页一致的访问权限校验，避免通过 `/tickets/:id/activities` 绕过详情权限
- 多资产主资源推导已改为基于显式顺序数组，而不是 `Object.values()` 的数字键顺序
- 工单类型跨标签页刷新从“布尔消费”改为“版本号比较”，避免第一个激活页提前吃掉刷新信号

### 3.12 通知渠道开源方案调研
- GitHub 中文开源里，`songquanpeng/message-pusher` 最适合作为 BigOps 的中国区 IM 聚合出口
- 其价值不在于“直接嵌入”，而在于可作为外部通知网关，让 BigOps 先统一生产事件，再由网关分发到飞书/钉钉/企业微信等渠道
- `feiyu563/PrometheusAlert` 适合借鉴运维告警通知中心的“路由 + 模板 + 多渠道”结构
- `imaegoo/pushoo` / `push-all-in-one` 更适合借鉴 provider 设计，不适合作为 Go 进程内直接依赖

### 3.13 当前通知中心实现原则
- 业务侧只调用 `NotificationService.Publish...`
- `NotificationService` 内部决定要不要写站内通知、邮件、Webhook、message-pusher
- 站内通知落库与事件记录在事务内
- 外部渠道发送在事务提交后异步执行

### 3.14 前端审批管理入口已补齐
- 已新增 `ApprovalInbox.vue`，可消费后端待审批/通过/拒绝 API
- 已新增 `RequestTemplates.vue`，对应请求模板 CRUD
- 已新增 `ApprovalPolicies.vue`，对应审批策略 CRUD
- 已将上述组件加入 `api/index.ts` 与 `router/index.ts` 的动态路由映射
- 管理类接口已补管理员校验，避免前端页面扩大权限面

### 3.15 通知中心当前形态
- 头部已接入站内通知入口，支持未读数、通知列表、单条已读
- 后端新增 `POST /notifications/test`，管理员可测试发送 `in_app/email/webhook/message_pusher`
- 当前外部渠道已具备代码路径，但仍需真实配置后联调

### 3.16 通知可靠性增强已完成
- `notification_delivery` 已新增 `last_attempt_at / next_retry_at`
- 已新增通知重试调度器，按配置周期扫描 `pending/failed` 投递
- 已新增通知事件查询与手动重试接口，便于管理员在联调期定位问题

### 3.16 请求单创建页已从“只认工单类型”升级为“双入口”
- `TicketCreate.vue` 当前支持同时从 `ticket_type` 与 `request_template` 进入
- 请求模板入口会携带 `request_template_id` 与 `ticket_kind`
- 当前已支持根据 `form_schema` 动态渲染最小字段集：`text/textarea/number/select/switch`

### 3.17 默认菜单入口已补种子
- 已新增 `004_seed_ticket_approval_menus.sql`
- 默认管理员菜单会包含：
  - 工单列表
  - 审批待办
  - 工单类型
  - 请求模板
  - 审批策略
  - 通知联调
- 这些菜单项的路径和组件名已与当前前端 `viewModules` 对齐

### 3.18 请求单详情展示策略
- `TicketDetail.vue` 现在直接从 `ticket.extra_fields.request_form` 渲染请求表单，不额外引入新的详情接口
- 这条策略的好处是保持后端接口稳定，创建页和详情页通过 `extra_fields` 这一事实来源对齐
- 当前展示层对 `boolean/number/string/array/object` 都做了兜底格式化，适合 V1 的动态表单最小集

### 3.19 通知可观测性增强
- `notification_events` / `notification_deliveries` 的列表接口现在会补充：
  - `status_summary`
  - `can_retry`
- `notification_service.go` 已按渠道返回更有意义的 `response` 文本：
  - SMTP 成功会记录目标邮箱
  - Webhook / message-pusher 会记录 HTTP 状态码和有限长度响应体
- 事件状态不再只区分 `pending/sent/partial_failed`，当前已细化为：
  - `pending`
  - `retrying`
  - `failed`
  - `partial_failed`
  - `sent`
- 这使得前端联调页不需要新增接口，就能定位“还在重试”“彻底失败”“部分成功”三类常见问题

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

---

## 6. 2026-03-24 前端重构分析（gin-vue-admin）

### 6.1 需求理解
- 用户希望先基于 `backend/docs/swagger.yaml` 分析现有项目，再考虑使用 `gin-vue-admin` 重构前端。
- 本次重点不是替换后端，而是判断 BigOps 现有后端与 `gin-vue-admin` 前端骨架的适配方式、重构边界和风险点。

### 6.2 真实项目状态
- 后端真实暴露接口远多于早期阶段，`router.go` 已覆盖：认证、用户、角色、菜单、部门、审计、服务树、资产、云账号、同步日志、统计、工单类型、工单。
- 当前前端已有 16 个页面、13 组 API 封装、动态路由、标签页、keep-alive、按钮级权限指令，说明这不是“从 0 到 1 的后台前端”，而是“结构化重构”。
- 统一响应格式是 `code/message/data`，并且业务错误也走 HTTP 200；前端请求层基于这个约定做统一拦截。
- 认证头使用 `Authorization: Bearer <token>`，并非项目外部模板常见的其他自定义头风格。

### 6.3 Swagger 与真实接口的偏差
- `backend/docs/swagger.yaml` 当前有 56 个接口操作。
- handler 上的 `@Router` 注解已有 71 个接口操作。
- 差异正好是工单与工单类型的 15 个操作，说明 Swagger 未重新生成或未纳入最新增量。
- 结论：重构时不能只以 Swagger 为准，必须以 `router.go` + handler 为真实来源。

### 6.4 前端架构现状
- 菜单、路由、按钮权限都由后端 `menus` 表驱动：`type=1` 目录、`type=2` 页面、`type=3` 按钮。
- 动态路由生成逻辑较轻：后端 `component` 字段直接映射本地 `views/*.vue`。
- 侧边栏渲染只显式支持两层菜单；虽然路由生成递归处理 children，但目录节点会被展平。
- 当前页面内存在路由命名不统一问题：菜单种子是 `/users`、`/roles`、`/assets`，但页面跳转中混用了 `/system/users`、`/cmdb/assets`、`/cmdb/cloud-accounts`。

### 6.5 与 gin-vue-admin 的适配判断
- 适配点：
  - 双方前端技术栈都在 Vue 3 + Vite + Element Plus + Pinia 范围内，迁移成本可控。
  - 双方都使用动态路由、菜单权限、按钮权限、统一请求封装，心智模型接近。
- 冲突点：
  - BigOps 现有接口命名、响应结构、登录态读取、菜单结构字段，与 `gin-vue-admin` 原生前后端约定并不完全一致。
  - BigOps 当前按钮权限来源于 `menus.type=3 + name`；`gin-vue-admin` 官方文档中的按钮权限实现依赖其自身的 `v-auth/useBtnAuth` 体系。
  - BigOps 当前 Casbin 策略会随着菜单 APIPath/APIMethod 同步，但实际路由中尚未挂载 Casbin 中间件，前端“可见性权限”和后端“可执行权限”并未完全闭环。
- 结论：
  - **应该借用 `gin-vue-admin` 的前端骨架、布局组织、表格/表单范式、权限壳层。**
  - **不应该直接照搬 `gin-vue-admin` 的后端 API 契约、权限模型和菜单接口返回格式。**

### 6.6 推荐策略
| 决策 | Rationale |
|------|-----------|
| 保留 BigOps 后端，单独重构前端 | 现有领域能力已成型，改后端只会扩大范围 |
| 以 BigOps 接口为主，做 gin-vue-admin 前端适配层 | 比反向改造后端接口更稳，风险更低 |
| 先统一 IA/菜单路径，再迁页面 | 当前 `/system/*`、`/cmdb/*`、扁平路径混用，会放大重构成本 |
| 先迁登录/布局/RBAC/系统管理，再迁 CMDB/工单 | 这是前端骨架与业务页的自然依赖顺序 |

### 6.7 外部参考（已核对）
- gin-vue-admin 官方仓库：最新 release 为 `v2.7.8`，日期为 `2026-03-09`
- 官方前端文档说明其前端仍基于 Vue 3、Vite、Element Plus、Pinia，并使用动态路由与按钮权限体系

### 6.8 本轮问题记录
| Issue | Resolution |
|-------|------------|
| 误以为菜单 handler 位于独立 `menu_handler.go` | 实际实现位于 `internal/handler/rbac_handler.go`，已修正检索路径 |
| shell 检索 migration 时未正确转义反引号 | 改为安全模式检索，避免被 bash 当作命令替换 |

---

## 7. 2026-03-24 工单前端 Bug 修复记录

### 7.1 创建工单无法选择主机资产
- 现象：创建工单时选择“主机资产”，下拉框打开后没有候选数据，必须依赖远程搜索触发。
- 根因：资产选项仅在 `remote-method` 执行时加载，没有“下拉打开即加载默认候选”的兜底逻辑。
- 修复：
  - 切换 `resource_type=asset` 时预加载资产列表
  - 打开资产下拉时，如果当前没有候选项，则自动加载第一页资产
  - 远程搜索保留，可继续按主机名/IP 过滤

### 7.2 工单列表点击后 404
- 现象：工单中心点击行跳转到 `/ticket/detail/:id`，页面 404。
- 根因：菜单配置路径是 `/ticket/detail`，动态路由按原路径注册，没有参数位；实际导航却带 `/:id`。
- 修复：
  - 在前端动态路由生成时，对 `TicketDetail` 组件自动规范为 `/ticket/detail/:id?`
  - 保留菜单原始路径作为 `activeMenu`，保证侧边栏激活态正常

### 7.3 进入工单详情后提示“工单不存在”
- 现象：直接进入不带参数的“工单详情”页面，会请求 `id=NaN` 并提示不存在。
- 根因：详情页默认假设 `route.params.id` 一定存在，没有对空参数做兜底。
- 修复：
  - 无 `id` 或 `id` 非法时不再发起接口请求
  - 改为展示空态提示“请从工单列表选择具体工单”

---

## 8. 2026-03-24 工单创建页资源选择交互改造

### 8.1 交互调整
- 将“关联资源 -> 主机资产”从远程搜索下拉，改为弹窗资产选择器。
- 选择器采用“左侧服务树 + 右侧资产表 + 搜索 + 分页”的结构，更适合资产规模稍大的场景。
- 当前实现保留“云账号”作为独立资源类型。
- “服务树”不再作为工单创建页里的独立资源类型，因为它已经成为资产选择器内的过滤维度。

### 8.2 设计理由
- 主机资产本质上是结构化对象，不适合纯下拉搜索；用户通常需要结合服务树定位，再从候选列表中确认。
- 服务树单独作为资源类型意义较弱，因为工单最终更常关联到具体资产，而不是抽象节点。
- 当前方案与 CMDB 的信息架构更一致：先按服务树筛，再选主机。

### 8.3 实现方式
- 工单创建页新增资产选择弹窗，包含：
  - 左侧服务树筛选与节点计数
  - 右侧资产表格
  - 关键字搜索
  - 当前选中资产摘要
- 表单主区域不再直接显示下拉，而是展示“已选资产卡片 + 选择/清空按钮”。

---

## 9. 2026-03-24 工单关联主机资产多选支持

### 9.1 约束确认
- 原始后端模型只支持单个 `resource_id` / `resource_name`。
- 用户要求“选择主机资产时自动弹窗”以及“支持多选和全选”。
- 直接只改前端不可行，否则提交时会丢失多选结果。

### 9.2 实际方案
- 保持数据库表结构不变。
- 新增策略：
  - `tickets.resource_id` 仍保存第一台主机，兼容现有资源关联和自动分派逻辑。
  - 多个已选主机的 ID 与摘要信息写入 `tickets.extra_fields` JSON。
- 这样可以做到：
  - 前端支持多选/全选
  - 后端无需新增表
  - 旧逻辑不被破坏

### 9.3 当前行为
- 选择资源类型为“主机资产”时，资产选择弹窗会自动打开。
- 弹窗表格支持多选，表头自带当前页全选。
- 创建工单后，详情页会展示完整的“关联主机列表”，不再只显示一台。
