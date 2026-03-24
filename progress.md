# BigOps - 开发进度日志

---

## Session: 2026-03-23 (同步收尾 + 部门 + 首页改版 + 标签页 + 表格优化)

### 完成事项

**同步流程增强：**
- 云同步离线收敛 — 本地有但云端未返回的资产标记 offline `c5a1954`
- 资产生命周期时间字段 last_sync_at/last_seen_at/offline_at `31b575b`
- 离线资产重新出现自动恢复 online + 变更历史
- 手动编辑资产记录变更历史（diffAsset + change_type=manual）`6c5b79e`
- 阿里云同步补磁盘信息（DescribeDisks API）+ 详细实例日志 `19cbf0a`
- 同步不更新磁盘 fix（diffAsset 补 disk_gb 对比）`a0e6507`
- 删除后重新同步资产丢失 fix（软删除恢复 Unscoped + RestoreSoftDeleted）`a0e6507`

**首页改版 + 统计接口：**
- 统计接口 /stats/summary + /stats/asset-distribution `f1b1144`
- 首页改版为平台总览（6 摘要卡片 + 来源分布 + 服务树 Top 10）`3b027c4`
- 首页卡片/分布行点击跳转 `0d5c605`
- 首页跳转路径修正（/assets → /cmdb/assets 等）`2af5941`

**轻量版部门功能：**
- Department 模型/Repository/Service/Handler 全栈 `1d0006a`
- 部门 CRUD 6 个 API + 删除校验关联用户
- User 模型新增 department_id + 用户编辑功能
- 新建用户时选部门+角色 `64ffdbc`
- Departments.vue 部门管理页面
- 部门菜单修复（parent name system_dir → system）

**标签页 + 表格体验：**
- 标签页多页签 + 自定义标签样式 `e2b28bc`
- keep-alive 状态保留（切 tab 保留搜索/分页）`88606eb`
- 标签右键菜单（关闭当前/其他/左侧/右侧/全部）`88606eb`
- keep-alive 缓存失效修复（componentName 匹配 + defineOptions）`d11b25a`
- 表格 border 拖拽列宽 + 操作列 min-width + 全局 padding

**规则补严：**
- 服务树移动防环校验 `06cd666`

### 新增文件
| 文件 | 说明 |
|------|------|
| `backend/internal/model/department.go` | 部门模型 |
| `backend/internal/repository/department_repository.go` | 部门 CRUD |
| `backend/internal/service/department_service.go` | 部门业务逻辑 |
| `backend/internal/handler/department_handler.go` | 部门 HTTP handler |
| `backend/internal/handler/stats_handler.go` | 统计接口 |
| `frontend/src/views/Departments.vue` | 部门管理页面 |
| `frontend/src/stores/tagsView.ts` | 标签页状态管理 |
| `frontend/src/composables/useTableResize.ts` | 表格列宽持久化 |

### 当前统计
- 后端代码: ~6868 行 (+865)
- 前端代码: ~3196 行 (+510)
- API 端点: 60 个 (+12)
- 前端页面: 12 个 (+1)
- Commits: 14 个

---

## Session: 2026-03-24 (多级审批 + 通知中心方案设计)

### 完成事项

**调研与结论：**
- 调研 Jira Service Management、Freshservice、ServiceNow、蓝鲸 bk-itsm 等模式
- 确认市场主流做法是将“审批”与“执行/处理”分层，而不是所有工单统一审批
- 确认 BigOps 当前实现更接近 Incident 处理流，不足以承载资源申请/变更审批

**设计输出：**
- 形成 BigOps 多级审批架构建议：请求模板、审批模板、审批实例、审批记录、执行单
- 形成通知中心设计：事件总线 + 渠道适配器 + 站内/邮件/Webhook/IM
- 形成 IM 选型建议：Zulip / Mattermost / Matrix-Element

### 当前判断
- 现有 Ticket 可继续承载故障/事件工单
- 资源申请、权限申请、变更单应引入新的审批建模，而不是继续堆在当前状态机上
- 建议优先做 V1: 串行多级审批 + 站内通知 + 邮件/Webhook

---

## Session: 2026-03-24 (多级审批与通知中心 V1 底座开发)

### 完成事项

**文档：**
- 新增设计文档 `docs/08-工单多级审批与通知方案.md`

**后端模型：**
- 新增 `RequestTemplate`
- 新增 `ApprovalPolicy` / `ApprovalPolicyStage`
- 新增 `ApprovalInstance` / `ApprovalRecord`
- 新增 `ExecutionOrder`
- 新增 `NotificationEvent` / `NotificationDelivery` / `InAppNotification`

**后端 Repository：**
- 新增请求模板 Repository
- 新增审批策略 Repository
- 新增审批实例 Repository
- 新增执行单 Repository
- 新增通知 Repository

**基础接入：**
- `cmd/core/main.go` AutoMigrate 已接入上述模型
- `go build ./cmd/core` 通过

### 修改的文件
| 文件 | 操作 |
|------|------|
| `docs/08-工单多级审批与通知方案.md` | 新增设计文档 |
| `backend/internal/model/request_template.go` | 新增请求模板模型 |
| `backend/internal/model/approval_policy.go` | 新增审批策略与阶段模型 |
| `backend/internal/model/approval_instance.go` | 新增审批实例与记录模型 |
| `backend/internal/model/execution_order.go` | 新增执行单模型 |
| `backend/internal/model/notification.go` | 新增通知相关模型 |
| `backend/internal/repository/request_template_repository.go` | 新增请求模板 Repository |
| `backend/internal/repository/approval_policy_repository.go` | 新增审批策略 Repository |
| `backend/internal/repository/approval_instance_repository.go` | 新增审批实例 Repository |
| `backend/internal/repository/execution_order_repository.go` | 新增执行单 Repository |
| `backend/internal/repository/notification_repository.go` | 新增通知 Repository |
| `backend/cmd/core/main.go` | AutoMigrate 接入审批与通知模型 |

### 下一步
- 实现请求模板 CRUD
- 实现审批策略 CRUD
- 实现请求单发起与审批实例初始化
- 实现站内通知与事件投递服务

---

## Session: 2026-03-24 (review 修复 + 审批配置 API)

### 完成事项

**review 缺陷修复：**
- 工单活动流接口补上与详情一致的权限校验
- 多资产选择改为显式顺序数组，避免主资源被数字 ID 排序污染
- 工单类型缓存刷新机制改为版本号，而不是布尔消费

**继续开发：**
- 新增请求模板 Service + Handler + Router
- 新增审批策略 Service + Handler + Router
- 后端现已具备请求模板 CRUD / 审批策略 CRUD 基础 API

### 修改的文件
| 文件 | 操作 |
|------|------|
| `backend/internal/handler/ticket_handler.go` | 修复活动流权限绕过 |
| `frontend/src/stores/viewState.ts` | 脏标记升级为版本号 |
| `frontend/src/views/TicketList.vue` | 修复工单类型刷新信号消费 |
| `frontend/src/views/TicketTypes.vue` | 修复工单类型刷新信号消费 |
| `frontend/src/views/TicketCreate.vue` | 修复多资产顺序与主资源推导 |
| `backend/internal/service/request_template_service.go` | 新增请求模板服务 |
| `backend/internal/service/approval_policy_service.go` | 新增审批策略服务 |
| `backend/internal/handler/request_template_handler.go` | 新增请求模板接口 |
| `backend/internal/handler/approval_policy_handler.go` | 新增审批策略接口 |
| `backend/api/http/router/router.go` | 注册请求模板/审批策略路由 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 后端编译 | `go build ./cmd/core` | 编译通过 | 通过 | ✓ |
| 前端构建 | `npm run build` | 构建通过 | 通过 | ✓ |

---

## Session: 2026-03-24 (多 agent 收尾：详情展示 + 通知可观测性)

### 完成事项

**工单详情：**
- `TicketDetail.vue` 新增请求表单展示区，直接渲染 `extra_fields.request_form`
- 右侧补充单据类型与请求模板信息
- 审批记录改为中文状态展示，详情页对请求/审批单更易读

**通知中心：**
- `NotificationConsole.vue` 新增手动刷新、15 秒自动刷新、最后刷新时间、Payload 展开与投递明细
- 通知事件与投递记录新增 `status_summary / can_retry`
- `notification_service.go` 记录更详细的投递结果，包含 SMTP 成功目标、Webhook / message-pusher HTTP 状态与响应体摘要
- 通知事件状态细化为 `pending / retrying / failed / partial_failed / sent`
- 管理员手动重试接口返回本次触发的投递数量

### 修改的文件
| 文件 | 操作 |
|------|------|
| `frontend/src/views/TicketDetail.vue` | 新增请求表单展示与审批状态中文化 |
| `frontend/src/views/NotificationConsole.vue` | 增强联调可观测性与自动刷新 |
| `backend/internal/model/notification.go` | 新增状态摘要与可重试字段 |
| `backend/internal/repository/notification_repository.go` | 新增按状态查询投递方法 |
| `backend/internal/service/notification_service.go` | 完善投递结果、错误分类、状态汇总与重试逻辑 |
| `backend/internal/handler/notification_handler.go` | 重试接口返回投递数量 |
| `task_plan.md` | 更新 Phase 5 已完成项 |
| `findings.md` | 记录详情展示和通知可观测性设计决策 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 后端编译 | `go build ./cmd/core` | 编译通过 | 通过 | ✓ |
| 前端构建 | `npm run build` | 构建通过 | 通过 | ✓ |

---

## Session: 2026-03-24 (审批实例初始化 + 站内通知 API)

### 完成事项

**请求单创建链路：**
- `Ticket` 模型新增 `ticket_kind / request_template_id / approval_status / approval_instance_id / execution_status`
- 创建工单时若带 `request_template_id`，会预校验审批模板
- 若模板绑定审批策略，则自动初始化审批实例和第一阶段审批记录

**通知中心：**
- 新增 `NotificationService`
- 请求单进入审批时写入 `notification_events` / `notification_deliveries` / `in_app_notifications`
- 新增站内通知接口：
  - `GET /notifications/in-app`
  - `GET /notifications/in-app/unread-count`
  - `POST /notifications/in-app/:id/read`

### 修改的文件
| 文件 | 操作 |
|------|------|
| `backend/internal/model/ticket.go` | 扩展请求/审批/执行字段 |
| `backend/internal/service/ticket_service.go` | 创建工单时初始化审批实例 |
| `backend/internal/service/notification_service.go` | 新增通知服务 |
| `backend/internal/repository/role_repository.go` | 新增按角色查用户 |
| `backend/internal/repository/notification_repository.go` | 新增站内通知读状态查询/更新 |
| `backend/internal/handler/ticket_handler.go` | 创建请求支持 request_template_id |
| `backend/internal/handler/notification_handler.go` | 新增站内通知接口 |
| `backend/api/http/router/router.go` | 注册通知接口 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 后端编译 | `go build ./cmd/core` | 编译通过 | 通过 | ✓ |

---

## Session: 2026-03-24 (审批动作流转 + 统一通知发送入口)

### 完成事项

**审批动作 API：**
- 新增审批待办列表
- 新增审批通过
- 新增审批拒绝
- 审批通过后可推进到下一阶段，或结束审批实例

**通知中心：**
- 业务侧统一走 `NotificationService.PublishTx(...)`
- 事务内落库：`notification_events / notification_deliveries / in_app_notifications`
- 事务后异步分发外部渠道
- 已支持渠道骨架：
  - `in_app`
  - `email`
  - `webhook`
  - `message_pusher`

**外部渠道调研：**
- 记录 `message-pusher` 作为中国区 IM 聚合出口的接入方向

### 修改的文件
| 文件 | 操作 |
|------|------|
| `backend/internal/service/approval_service.go` | 新增审批待办/通过/拒绝流转 |
| `backend/internal/handler/approval_handler.go` | 新增审批动作接口 |
| `backend/internal/service/notification_service.go` | 统一通知发送入口 + 外部渠道适配 |
| `backend/internal/repository/notification_repository.go` | 新增事件/投递读取与更新方法 |
| `backend/internal/pkg/config/config.go` | 新增通知配置结构 |
| `backend/config/config.yaml` | 新增通知配置示例 |
| `backend/config/config.example.yaml` | 新增通知配置示例 |
| `backend/api/http/router/router.go` | 注册审批与通知接口 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 后端编译 | `go build ./cmd/core` | 编译通过 | 通过 | ✓ |

---

## Session: 2026-03-24 (审批待办前端 + 配置页接入)

### 完成事项

**前端页面：**
- 新增审批待办页 `ApprovalInbox.vue`
- 新增请求模板管理页 `RequestTemplates.vue`
- 新增审批策略管理页 `ApprovalPolicies.vue`

**前端接入：**
- `api/index.ts` 新增 approval/requestTemplate/approvalPolicy/notification API 封装
- `router/index.ts` 新增组件映射与隐藏 companion routes
- `Menus.vue` 新增可选页面组件项

**权限修正：**
- 后端 `request_template` 与 `approval_policy` 相关接口新增管理员校验

### 修改的文件
| 文件 | 操作 |
|------|------|
| `backend/internal/handler/access_helper.go` | 新增管理员判定工具 |
| `backend/internal/handler/request_template_handler.go` | 新增管理员校验 |
| `backend/internal/handler/approval_policy_handler.go` | 新增管理员校验 |
| `frontend/src/api/index.ts` | 新增审批/模板/通知 API |
| `frontend/src/router/index.ts` | 新增新页面组件映射与隐藏路由 |
| `frontend/src/views/ApprovalInbox.vue` | 新增审批待办页 |
| `frontend/src/views/RequestTemplates.vue` | 新增请求模板页 |
| `frontend/src/views/ApprovalPolicies.vue` | 新增审批策略页 |
| `frontend/src/views/Menus.vue` | 新增组件选项 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 后端编译 | `go build ./cmd/core` | 编译通过 | 通过 | ✓ |
| 前端构建 | `npm run build` | 构建通过 | 通过 | ✓ |

---

## Session: 2026-03-24 (通知中心前端接入 + 测试发送入口)

### 完成事项

**前端：**
- `Layout.vue` 接入头部通知铃铛
- 支持未读数展示
- 支持站内通知抽屉查看
- 支持单条通知点击已读
- 新增 `NotificationConsole.vue` 作为管理员联调入口

**后端：**
- 新增 `POST /notifications/test`
- `NotificationService` 增加非事务入口 `Publish(...)`
- 支持通过统一发送入口测试 `in_app/email/webhook/message_pusher`

### 修改的文件
| 文件 | 操作 |
|------|------|
| `backend/internal/service/notification_service.go` | 新增统一测试发送入口 |
| `backend/internal/handler/notification_handler.go` | 新增通知测试发送接口 |
| `backend/api/http/router/router.go` | 注册 `/notifications/test` |
| `frontend/src/views/Layout.vue` | 新增头部通知中心 UI |
| `frontend/src/views/NotificationConsole.vue` | 新增通知联调页面 |
| `frontend/src/api/index.ts` | 新增通知测试发送 API |
| `frontend/src/router/index.ts` | 新增通知联调页面路由映射 |
| `frontend/src/views/Menus.vue` | 新增 `NotificationConsole` 组件选项 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 后端编译 | `go build ./cmd/core` | 编译通过 | 通过 | ✓ |
| 前端构建 | `npm run build` | 构建通过 | 通过 | ✓ |

---

## Session: 2026-03-24 (请求单创建页支持 request_template)

### 完成事项

**前端：**
- `TicketCreate.vue` 第一步选择卡片支持“工单类型 + 请求模板”双入口
- 请求模板进入时会写入 `request_template_id` 和 `ticket_kind`
- 提交请求单时后端可直接初始化审批实例

### 修改的文件
| 文件 | 操作 |
|------|------|
| `frontend/src/views/TicketCreate.vue` | 支持 request_template 双入口 |
| `frontend/src/api/index.ts` | requestTemplateApi 用于创建页数据加载 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 前端构建 | `npm run build` | 构建通过 | 通过 | ✓ |

---

## Session: 2026-03-24 (通知重试策略 + 管理可视化)

### 完成事项

**后端：**
- `notification_delivery` 新增重试时间字段
- `NotificationService` 新增事件列表、手动重试、到期重试扫描
- 新增通知重试调度器，并在 `main.go` 启动
- 新增接口：
  - `GET /notifications/events`
  - `POST /notifications/events/:id/retry`

**前端：**
- `NotificationConsole.vue` 增加事件列表
- 显示每个事件的投递状态、重试次数
- 支持手动重试某个事件

### 修改的文件
| 文件 | 操作 |
|------|------|
| `backend/internal/model/notification.go` | 新增重试时间字段 |
| `backend/internal/repository/notification_repository.go` | 新增事件列表/重试查询 |
| `backend/internal/service/notification_service.go` | 新增重试逻辑与调度器 |
| `backend/internal/handler/notification_handler.go` | 新增事件列表与重试接口 |
| `backend/api/http/router/router.go` | 注册通知事件接口 |
| `backend/cmd/core/main.go` | 启动通知重试调度器 |
| `frontend/src/api/index.ts` | 新增通知事件查询/重试 API |
| `frontend/src/views/NotificationConsole.vue` | 新增通知事件状态展示与重试按钮 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 后端编译 | `go build ./cmd/core` | 编译通过 | 通过 | ✓ |
| 前端构建 | `npm run build` | 构建通过 | 通过 | ✓ |

---

## Session: 2026-03-24 (请求模板 schema builder + 默认菜单 seed)

### 完成事项

**请求模板：**
- `RequestTemplates.vue` 新增 schema 可视化编辑器
- 支持字段类型：
  - text
  - textarea
  - number
  - select
  - switch
- 保留原始 JSON 文本编辑能力
- 请求模板现已强制绑定底层 `ticket_type`

**请求单创建：**
- `TicketCreate.vue` 已能根据 `request_template.form_schema` 渲染最小动态表单
- 提交时把 `request_form` 数据写入 `extra_fields`

**菜单种子：**
- 新增 `backend/migrations/004_seed_ticket_approval_menus.sql`
- 为默认管理员补齐工单中心、审批待办、请求模板、审批策略、通知联调等入口

### 修改的文件
| 文件 | 操作 |
|------|------|
| `backend/internal/model/request_template.go` | 新增 type_id/type_name |
| `backend/internal/service/request_template_service.go` | 校验并回填 ticket_type |
| `backend/internal/handler/request_template_handler.go` | 支持提交 type_id |
| `backend/internal/service/ticket_service.go` | request_template 回填底层 type_id |
| `backend/internal/handler/ticket_handler.go` | 支持 extra_fields 提交 |
| `frontend/src/stores/viewState.ts` | 新增 requestTemplateVersion |
| `frontend/src/views/RequestTemplates.vue` | 新增 schema builder |
| `frontend/src/views/TicketCreate.vue` | 动态表单渲染与 extra_fields 提交 |
| `backend/migrations/004_seed_ticket_approval_menus.sql` | 新增默认菜单种子 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 前端构建 | `npm run build` | 构建通过 | 通过 | ✓ |

---

## Session: 2026-03-22 (定时同步系统 + Bug 修复 + 云账号关联服务树)

### 完成事项

**云账号定时同步系统（9 个子任务全部完成）：** `a1ac2ab`
- CloudSyncTask 模型 — 记录每次同步完整生命周期
- CloudSyncTask Repository — CRUD + 分页 + 按账号/状态筛选
- CloudAccount 新增 sync_enabled / sync_interval 字段
- SyncRunner 统一同步执行器 — 从 handler 抽取 200 行逻辑，per-account mutex 防重复
- 重构 handler Sync 方法 → 调用 RunSync()
- Scheduler 定时调度器 — 60s 巡检 + Go ticker + 优雅停止
- 同步日志查询 API (GET /sync-tasks + GET /cloud-accounts/:id/sync-tasks)
- 同步配置接口 (POST /cloud-accounts/:id/sync-config)
- 前端: 编辑表单同步开关 + 周期选择 + 同步记录抽屉 + 定时同步列
- main.go 集成: AutoMigrate CloudSyncTask + 启动 Scheduler

**云账号关联服务树：** `7b94db2`
- CloudAccount 新增 service_tree_id 字段
- 新增/编辑表单加 el-tree-select 服务树选择器
- SyncRunner 同步新资产时自动设置 service_tree_id
- 列表新增"所属服务"列，显示完整路径

**Bug 修复：**
- 菜单更新排序 Error 1292 `045d3ef` — handler 构造新对象零值 CreatedAt → service 层先查 existing
- LocalTime 零值写入 0000-00-00 `76b766a` — Value() 零值返回 nil + 修复数据库 NULL 数据
- 系统管理菜单权限失效 `50afceb` — Layout.vue 硬编码 + 静态路由 → 全部走动态路由
- 系统管理菜单被软删除 — UPDATE menus SET deleted_at = NULL WHERE id = 1
- 资产所属服务显示完整路径 `bba7186`
- ServiceTree badge 递归计数 + 防止点击节点收起 `e7a426e`

### 新增文件
| 文件 | 说明 |
|------|------|
| `backend/internal/model/cloud_sync_task.go` | 同步任务模型 |
| `backend/internal/repository/cloud_sync_task_repository.go` | 同步任务 CRUD |
| `backend/internal/service/cloud_sync/runner.go` | 统一同步执行器 |
| `backend/internal/service/cloud_sync/scheduler.go` | 定时调度器 |
| `backend/internal/handler/cloud_sync_task_handler.go` | 同步日志查询 API |

### 当前统计
- 后端代码: ~6003 行
- 前端代码: ~2686 行
- API 端点: 48 个
- 前端页面: 11 个
- 新增 API: 4 个 (sync-config, sync-tasks x2, cloud-accounts 关联服务树)

### 下一步工作
- 见文末 Next Steps

---

## Session: 2026-03-21 (项目状态审计 + Swagger 完善)

### 完成事项

**Swagger API 文档完善**
- 为全部 21 个 API 端点添加了 Swagger 注解
- 导出所有请求类型 (RegisterRequest, LoginRequest 等) 并添加 example 标签
- 添加 BearerAuth 安全定义到 main.go
- 解决 swag CLI v1.16.6 与 go.mod v1.8.12 不兼容问题 (升级 swaggo/swag)
- 解决 swag 跨包类型引用问题 (model.User → 通过 `var _ model.User` 导入)
- Model LocalTime 字段添加 `swaggertype:"string"` 标签
- 验证 `go build ./cmd/core/` 编译通过
- 生成 docs/ 目录: docs.go, swagger.json, swagger.yaml

**项目状态审计**
- 扫描全部后端代码: 2785 行 (13 个包, 21 个文件)
- 扫描全部前端代码: 729 行 (5 个页面, 1 个 API 模块)
- 运行后端测试: 4 个包 PASS (config/crypto/jwt/logger)
- 核对实施计划文档与实际代码完成度
- 创建 planning-with-files 计划文件体系

### 修改的文件
| 文件 | 操作 |
|------|------|
| `backend/internal/handler/auth_handler.go` | 添加 Swagger 注解 + 导出请求类型 + model 导入 |
| `backend/internal/handler/user_handler.go` | 添加 Swagger 注解 + 导出请求类型 + model 导入 |
| `backend/internal/handler/rbac_handler.go` | 添加 Swagger 注解 + 导出请求类型 |
| `backend/cmd/core/main.go` | 添加 securityDefinitions + docs 导入 |
| `backend/internal/model/menu.go` | LocalTime 字段添加 swaggertype 标签 |
| `backend/go.mod` / `backend/go.sum` | 升级 swaggo/swag v1.8.12 → v1.16.6 |
| `backend/docs/*` | 新生成 swagger 文档 |

### 当前阻塞项
- 无

### 下一步工作
- P0: 操作审计日志 (model + repository + handler + 前端页面)
- P0: Pinia stores 初始化 (userStore + permissionStore)
- P1: 前端动态菜单 + 动态路由改造
- P1: v-permission 权限指令

---

## 项目总体进度

```
Phase 1: 基础设施搭建     [████████████████████] 100%
Phase 2: Module 01 底座    [███████████████████░]  97%
  - 认证                   [████████████████████] 100%
  - RBAC + 权限指令         [████████████████████] 100%
  - 菜单管理 + 动态菜单     [████████████████████] 100%
  - 用户管理 + 编辑         [████████████████████] 100%  ← 新增编辑/部门
  - 部门管理               [████████████████████] 100%  ← NEW
  - 布局 + 标签页           [███████████████████░]  95%  ← 多页签+keep-alive
  - 审计日志               [████████████████████] 100%
  - Swagger 文档           [████████████████████] 100%
  - Pinia 状态管理          [████████████████████] 100%
Phase 3: Module 02 CMDB    [███████████████████░]  97%
  - 服务树                 [████████████████████] 100%  ← 防环校验
  - 云账号                 [████████████████████] 100%
  - 定时同步系统            [████████████████████] 100%
  - 离线收敛 + 恢复         [████████████████████] 100%  ← NEW
  - 资产管理               [████████████████░░░░]  85%  (缺 Excel 导入导出/批量)
  - 阿里云同步             [████████████████████] 100%  ← 补磁盘
  - 变更历史               [████████████████████] 100%  ← 手动编辑也记录
  - 统计接口               [████████████████████] 100%  ← NEW
  - 首页总览               [████████████████████] 100%  ← NEW
Phase 4: Module 03 工单    [░░░░░░░░░░░░░░░░░░░░]   0%
Phase 5: Module 04 任务    [░░░░░░░░░░░░░░░░░░░░]   0%
Phase 6: Module 05 监控    [░░░░░░░░░░░░░░░░░░░░]   0%
Phase 7: Module 08 数据库  [░░░░░░░░░░░░░░░░░░░░]   0%
```

---

## Next Steps

| 优先级 | 任务 | 说明 |
|--------|------|------|
| P1 | Module 03 工单系统 | 工单类型/创建/流转/审批 — 平台最缺的运维流程能力 |
| P2 | 资产导入/导出 Excel | excelize 后端 + 前端上传/下载，CMDB 数据迁移必备 |
| P2 | 资产批量操作 | 批量修改服务树/批量删除 |
| P2 | 腾讯云/AWS 同步 | 扩展 CloudProvider 接口 |
| P3 | Module 04 任务执行中心 | Agent + gRPC + 远程执行 |

---

## Session: 2026-03-24 (项目分析 + gin-vue-admin 前端重构评估)

### 完成事项

**后端能力盘点：**
- 对照 `backend/docs/swagger.yaml` 与 `backend/api/http/router/router.go`
- 确认后端已覆盖认证、RBAC、菜单、部门、审计、CMDB、统计、工单类型、工单
- 统计得到 Swagger 56 个操作、handler 注解 71 个操作
- 识别 Swagger 缺失工单相关 15 个操作，文档已落后于真实接口

**现有前端架构盘点：**
- 检查 `frontend/src/api/index.ts`，确认 13 组 API 模块、统一 Bearer Token 与 `code/message/data` 拦截
- 检查 `frontend/src/router/index.ts` 与 `stores/permission.ts`，确认动态菜单与按钮权限来自后端 `menus` 表
- 检查 `views/Layout.vue`，确认当前侧边栏只显式渲染两层菜单
- 检查 `views/Dashboard.vue` 与菜单 seed SQL，确认项目内部存在 `/system/*`、`/cmdb/*`、扁平路径混用

**gin-vue-admin 适配评估：**
- 核对官方仓库与文档，确认当前前端仍是 Vue 3 + Vite + Element Plus + Pinia 路线
- 核对其动态路由与按钮权限文档，确认可以借用前端骨架，但不能直接套 BigOps 当前接口契约
- 输出建议：保留 BigOps 后端，仅重构前端壳层和页面组织

### 文件修改
| 文件 | 操作 |
|------|------|
| `task_plan.md` | 更新专项分析结论 |
| `findings.md` | 记录 gin-vue-admin 迁移分析与风险 |
| `progress.md` | 记录本次分析过程 |

### Error Log
| Timestamp | Error | Attempt | Resolution |
|-----------|-------|---------|------------|
| 2026-03-24 | 误读 `menu_handler.go` 文件路径 | 1 | 改为在 `rbac_handler.go` 中定位 `MenuHandler` |
| 2026-03-24 | shell 检索 migration 时触发反引号命令替换 | 1 | 调整检索表达式，避免 bash 解释 SQL 片段 |

### 5-Question Reboot Check
| Question | Answer |
|----------|--------|
| Where am I? | 已完成项目分析与 gin-vue-admin 适配评估 |
| Where am I going? | 若继续实施，下一步应先统一菜单 IA，再搭建新前端骨架 |
| What's the goal? | 用更标准的后台前端骨架重构 BigOps 前端，但不破坏现有后端能力 |
| What have I learned? | Swagger 落后、路由规范不统一、前端可借 gin-vue-admin 骨架但不能直接套其后端契约 |
| What have I done? | 完成接口、路由、权限、页面与外部框架适配分析 |

---

## Session: 2026-03-24 (工单创建/详情页 Bug 修复)

### 完成事项

**工单创建页：**
- 修复“关联资源 -> 主机资产”下拉无数据问题
- 切换到 `asset` 类型时自动预加载第一页资产
- 下拉展开时如果尚未加载候选项，则自动补拉资产列表

**工单详情路由：**
- 修复 `TicketDetail` 动态路由不支持 `/:id` 参数导致的 404
- 为详情页路由加上可选参数，兼容 `/ticket/detail` 与 `/ticket/detail/:id`
- 增加 `activeMenu` 元信息，保证详情页侧边栏高亮正常

**工单详情页：**
- 修复无 `id` 时仍然请求详情接口的问题
- 无参数时展示空态提示，而不是报“工单不存在”

**验证：**
- 前端 `npm run build` 通过
- 首次在沙箱内构建时遇到 `getaddrinfo EAI_AGAIN localhost`
- 已在非沙箱环境重跑并确认构建成功

### 修改的文件
| 文件 | 操作 |
|------|------|
| `frontend/src/views/TicketCreate.vue` | 资产下拉预加载与展开加载 |
| `frontend/src/router/index.ts` | `TicketDetail` 路由参数兼容与 `activeMenu` 注入 |
| `frontend/src/views/Layout.vue` | 侧边栏激活路径改为优先使用 `route.meta.activeMenu` |
| `frontend/src/views/TicketDetail.vue` | 无效/缺失 `id` 的空态处理 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 前端构建 | `npm run build` | TS 检查与 Vite 构建成功 | 非沙箱环境下成功构建 | ✓ |

### Error Log
| Timestamp | Error | Attempt | Resolution |
|-----------|-------|---------|------------|
| 2026-03-24 | `npm run build` 在沙箱内报 `getaddrinfo EAI_AGAIN localhost` | 1 | 在非沙箱环境重跑，同一命令通过 |

---

## Session: 2026-03-24 (工单创建页资产选择器改造)

### 完成事项

**交互改造：**
- 将工单创建页“主机资产”从远程下拉改为弹窗式资产选择器
- 选择器采用左侧服务树、右侧资产表、顶部搜索、底部分页与已选摘要
- 资源类型中移除“服务树”，避免与资产选择器左侧树重复

**页面表现：**
- 表单中新增已选资产卡片，展示主机名、IP、状态、服务树路径与系统信息
- 支持打开弹窗选择、双击资产快速确认、清空已选资产
- 服务树节点显示递归资产数量，便于快速定位

**验证：**
- 前端 `npm run build` 通过

### 修改的文件
| 文件 | 操作 |
|------|------|
| `frontend/src/views/TicketCreate.vue` | 将资产下拉替换为弹窗选择器并移除服务树资源类型 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 前端构建 | `npm run build` | TS 检查与 Vite 构建成功 | 成功 | ✓ |

---

## Session: 2026-03-24 (工单关联主机多选 + 自动弹窗)

### 完成事项

**前端：**
- 选择资源类型为“主机资产”时自动弹出资产选择器
- 资产表格改为多选模式，支持表头当前页全选
- 已选资产在表单中展示为摘要卡片，而不是单个下拉值
- 详情页新增“关联主机”区域，展示所有已选资产

**后端：**
- 创建工单请求新增 `resource_ids`
- 多个主机资产写入 `tickets.extra_fields`
- `resource_id` 仍保留第一台主机，保证旧逻辑兼容
- 自动填充 `resource_name` 时：
  - 单资产显示主机名/IP
  - 多资产显示“X 台主机资产”

**验证：**
- `go build ./cmd/core` 通过
- `npm run build` 通过

### 修改的文件
| 文件 | 操作 |
|------|------|
| `backend/internal/repository/asset_repository.go` | 新增批量查询资产方法 |
| `backend/internal/handler/ticket_handler.go` | 创建工单支持 `resource_ids` |
| `backend/internal/service/ticket_service.go` | 多资产写入 `extra_fields` 并生成资源摘要 |
| `frontend/src/views/TicketCreate.vue` | 自动弹窗 + 多选 + 当前页全选 |
| `frontend/src/views/TicketDetail.vue` | 展示关联主机列表 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 后端编译 | `go build ./cmd/core` | 编译通过 | 通过 | ✓ |
| 前端构建 | `npm run build` | 构建通过 | 通过 | ✓ |
