# Progress: BigOps 开发进度日志

## Session: 2026-03-25 (Module 04 + 安全修复)

### 完成事项

**Module 04 任务执行中心 — 全栈实现**
- [x] gRPC Proto 定义 + protoc 代码生成 (agent.proto → agent.pb.go + agent_grpc.pb.go)
- [x] 4 个 GORM 模型: Task, TaskExecution, TaskHostResult, AgentInfo
- [x] 3 个 Repository: TaskRepository, TaskExecutionRepository, AgentRepository
- [x] TaskService: CRUD + ExecuteTask 下发 + GetExecution
- [x] TaskHandler: 10 个 API + WebSocket 实时日志
- [x] gRPC Server: AgentManager 连接管理 + Heartbeat + ReportOutput
- [x] Agent 二进制: gRPC 客户端 + 命令执行器 + 超时控制
- [x] 前端 4 页面: TaskList, TaskCreate, TaskExecution, AgentList
- [x] 菜单自动初始化 (seedTaskMenus, 幂等)
- [x] Swagger 文档更新 (+10 API)

**代码审查修复**
- [x] WebSocket 认证: AuthMiddleware 支持 query param token
- [x] WebSocket 断线重连 (3s 间隔 + destroyed flag)
- [x] executor env 继承: `cmd.Env = os.Environ()` + extras
- [x] ReportOutput 原子追加: GORM CONCAT 表达式
- [x] ListExecutions taskID=0 列出全部
- [x] agent_manager 清理未使用 json.Marshal

**安全加固**
- [x] 启用 Casbin API 权限控制 (白名单 + syncCasbinPolicies)
- [x] task_service.go 5 处 `_ =` → logger.Warn
- [x] UserHandler → UserService 重构 (消除 N+1 部门查询)
- [x] 注册限流: 每 IP 每分钟 3 次
- [x] 登录限流: 每 IP 每分钟 10 次 + 5 次失败锁定 15 分钟
- [x] 分页 size 上限: parsePageSize() 全局 max=100

### Commits
```
2e9ba9c fix: 注册/登录限流 + 登录锁定 + 分页 size 上限
a031299 feat: 启用 Casbin API 权限控制 + 错误处理 + UserService 重构
3ca9694 docs: Swagger 文档更新 — 新增任务中心 10 个 API
57dda89 fix: 代码审查修复 4 项问题
033f7e1 feat: 任务中心菜单自动初始化（幂等 seed）
de7aa92 fix: WebSocket 认证 + 断线重连
a2f73ad feat: Module 04 任务执行中心 — 全栈实现
```

### 文件变更统计
- 新增文件: 20+
- 修改文件: 15+
- 代码增量: ~4,000 行 (Go + Vue)

---

## Next Steps (Phase 5+)
1. Module 05: 监控系统 (Prometheus 集成 + 告警)
2. CMDB 补充: Excel 导入导出 + 批量操作
3. 前端优化: 暗色主题、国际化

---

## Session: 2026-03-25 (工单管理 IA 重构设计)

### 完成事项

**设计收敛：**
- 按用户确认的方向完成“工单管理”信息架构重构设计
- 保留父目录 `工单管理`
- 二级菜单收敛为：
  - 发起工单
  - 我的待办
  - 我的申请
  - 工单模板
- 明确移除左侧主导航：
  - 审批待办
  - 工单类型
  - 请求模板
  - 审批策略
- 明确“工单模板”页面以 `RequestTemplates` 为主体重构，外观参考用户截图
- 明确 `TicketList.vue` 复用为“我的待办 / 我的申请”双模式页面

**文档输出：**
- 新增正式设计文档：
  - `docs/superpowers/specs/2026-03-25-ticket-center-ia-design.md`

### 修改的文件
| 文件 | 操作 |
|------|------|
| `docs/superpowers/specs/2026-03-25-ticket-center-ia-design.md` | 新增工单管理 IA 与页面重构设计 |

### 下一步
1. 完成设计文档 review
2. 让用户确认设计文档
3. 再进入实现计划与代码改造

---

## Session: 2026-03-25 (工单管理 IA 实施计划)

### 完成事项

**计划输出：**
- 基于已确认的 spec，生成可执行实施计划文档
- 计划覆盖：
  - 菜单 migration
  - 路由复用
  - TicketList 双模式
  - TicketCreate 模板单入口
  - RequestTemplates 重构为工单模板
  - 最终收口与验证

**计划文档：**
- `docs/superpowers/plans/2026-03-25-ticket-center-ia-implementation.md`

### 修改的文件
| 文件 | 操作 |
|------|------|
| `docs/superpowers/plans/2026-03-25-ticket-center-ia-implementation.md` | 新增工单管理 IA 重构实施计划 |

### 下一步
1. 选择执行方式
2. 进入实现阶段

---

## Session: 2026-03-25 (工单管理 IA 重构实现)

### 完成事项

**菜单与路由：**
- 新增 `007_refactor_ticket_center_menus.sql`
- 工单父目录标题改为“工单管理”
- 新 IA 菜单收敛为：
  - 发起工单
  - 我的待办
  - 我的申请
  - 工单模板
- 动态路由支持：
  - `/ticket/create`
  - `/ticket/todo`
  - `/ticket/applied`
  - `/ticket/templates`
- `ApprovalPolicies` 改为依赖 `RequestTemplates` 入口补隐藏路由，不再依赖 `TicketTypes`

**页面改造：**
- `TicketList.vue` 重构为双模式页面：
  - `todo -> my_assigned`
  - `applied -> my_created`
- 移除列表页“审批待办”按钮
- 顶部按钮统一改成“发起工单”
- `TicketCreate.vue` 改成模板单入口，不再展示工单类型卡片
- `RequestTemplates.vue` 重构为“工单模板”页：
  - 列表字段改成模板管理视角
  - 增加刷新按钮
  - 增加审批策略入口
  - 增加启用开关

**配套修复：**
- `Layout.vue` 根据工单详情来源高亮“我的待办 / 我的申请”
- `request_template` 后端更新逻辑支持 `status=0`，避免模板只能启用不能停用
- Casbin 模型继续采用 `keyMatch`，确保 `/api/v1/tickets*`、`/api/v1/request-templates*` 这类策略能覆盖集合和子路径
- `Menus.vue` 补齐工单与任务中心组件选项，并修复类型标签的空 type 警告

### 修改的文件
| 文件 | 操作 |
|------|------|
| `backend/migrations/007_refactor_ticket_center_menus.sql` | 新增工单管理 IA 菜单迁移 |
| `backend/internal/handler/request_template_handler.go` | 支持模板状态显式更新 |
| `backend/internal/service/request_template_service.go` | 支持模板禁用状态落库 |
| `frontend/src/router/index.ts` | 重构工单 IA 动态路由与隐藏路由 |
| `frontend/src/views/Layout.vue` | 工单详情 activeMenu 回落优化 |
| `frontend/src/views/Menus.vue` | 补齐组件选项与标签修复 |
| `frontend/src/views/TicketList.vue` | 双模式待办/申请列表 |
| `frontend/src/views/TicketCreate.vue` | 模板单入口发起工单 |
| `frontend/src/views/RequestTemplates.vue` | 重构为工单模板页 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 后端编译 | `go build ./cmd/core` | 编译通过 | 通过 | ✓ |
| 前端构建 | `npm run build` | 构建通过 | 通过 | ✓ |

---

## Session: 2026-03-27（首页 Dashboard 重构）

### 完成事项

- 新增 `GET /api/v1/dashboard/personal`
- 首页重构为：
  - 欢迎区 / 快捷入口
  - 个人工作台
  - 按权限显示的平台总览
  - 资源分布
- 普通用户根据菜单权限动态隐藏无权限模块
- 管理员保留完整平台视角

### 验证结果

- `go build ./cmd/core` ✓
- `npm run build` ✓
- Playwright：
  - admin 首页显示 `个人工作台 + 平台总览`
  - yunwei 首页显示 `个人工作台 + 平台总览`
  - 普通用户不显示自己无权限的系统类模块卡片
- Chrome DevTools / CDP：
  - 关键文本命中正常
  - `consoleErrors = 0`
  - `failedResponses = 0`

## Session: 2026-03-26 (CI/CD 增强收口)

### 完成事项

- **CI/CD 阶段化增强**
  - `build -> approval -> deploy` 三段状态已真正进入运行主链
  - `build_task_id` 不再是摆设，会先触发构建任务，再自动推进到审批或部署
  - `StartApprovedRunsByTicketID` 只负责从审批阶段进入部署阶段
- **Webhook 配置贯通**
  - 流水线创建/更新已支持：
    - `build_hosts`
    - `variables`
    - `webhook_enabled`
    - `webhook_secret`
  - Webhook 继续以 `pipeline code` 作为公开触发标识
  - 已校验 `X-BigOps-Webhook-Secret` / `X-Webhook-Secret`
- **运行详情增强**
  - 运行记录详情抽屉新增 `Build / Approval / Deploy` 三段视图
  - 支持跳转到：
    - 审批单 `/ticket/detail/:id`
    - 任务执行 `/task/execution/:id`
  - 构建主机、环境变量、Webhook 配置在流水线表单中可直接维护
- **前端页面收口**
  - 恢复并重建了 `CicdPipelines.vue`
  - 修正空值显示 `0`
  - 修正 Webhook 预览地址与后端真实路由一致
- **后端状态机修复**
  - `GetRunDetail` 改为走 `syncRunExecutionStatus`
  - 修复运行详情里“总状态已变更，但阶段字段仍停留在 build/running”的错位
  - 修复 build 成功后重复推进阶段、重复建审批单的风险

### 验证结果

- 后端：
  - `go test ./internal/service -run 'TestParsePipelineRuntimeConfig|TestBuildTaskEnv' -count=1` ✓
  - `go test ./internal/service ./internal/handler ./internal/repository -count=1` ✓
  - `go build ./cmd/core` ✓
- 前端：
  - `npm run build` ✓
- API：
  - 新建增强流水线 `enhance-1774502130` 成功
  - 触发后运行记录 `#12` 成功进入：
    - `build_status = success`
    - `deploy_status = success`
    - `current_stage = done`
    - `summary = 部署完成：成功 1 / 总数 1`
- Playwright：
  - `admin` 打开 `/cicd/pipelines`、编辑弹窗、`/cicd/runs?pipeline_id=12`、运行详情成功
  - `yunwei` 打开 `/cicd/runs` 成功，未出现无权限提示
  - 产物：
    - `/tmp/cicd-enhanced-playwright.json`
    - `/tmp/cicd-enhanced-playwright.png`
- Chrome DevTools / CDP：
  - 成功登录并进入：
    - `/cicd/pipelines`
    - `/cicd/runs?pipeline_id=12`
  - 命中关键文本：
    - `CI/CD 流水线`
    - `构建主机`
    - `环境变量`
    - `Webhook`
    - `CI/CD 运行记录`
    - `Build / Approval / Deploy`
    - `构建阶段完成`
    - `部署完成：成功 1 / 总数 1`
  - `console error = 0`
  - `failedResponses = 0`
  - 产物：
    - `/tmp/chrome-devtools-cicd-final.json`
    - `/tmp/chrome-devtools-cicd-final.png`

## Session: 2026-03-26 (站内通知中心增强)

### 完成事项

- 后端新增批量通知操作：
  - `POST /api/v1/notifications/in-app/read-all`
  - `POST /api/v1/notifications/in-app/clear-read`
- 前端通知抽屉从“只展示”升级成可操作通知中心：
  - `未读 / 全部` 过滤
  - `全部已读`
  - `清空已读`
  - 点击通知后自动标记已读，并按 `biz_type / biz_id` 跳转到业务页面
- 跳转规则已支持：
  - `ticket -> /ticket/detail/:id`
  - `task_execution/execution -> /task/execution/:id`
  - `alert_event -> /monitor/alert-rules`

### 验证结果

- 后端：
  - `go build ./cmd/core` ✓
- 前端：
  - `npm run build` ✓
- API：
  - 发送站内通知后，未读数从 `22` 变为 `0`
  - `read-all` 返回 `已全部标记为已读`
  - `clear-read` 返回 `已清空已读通知`
- 浏览器：
  - Playwright 命中：
    - `站内通知`
    - `未读`
    - `全部`
    - `全部已读`
    - `清空已读`
  - Chrome DevTools / CDP 命中同样关键文本，且 `consoleErrors = 0`、`failedResponses = 0`

## Session: 2026-03-26 (监控中心 V2/V3 第一批)

### 完成事项

- **Agent 身份收口**
  - Agent 不再默认使用 `hostname_ip` 作为身份
  - 支持：
    - `agent.id`
    - `agent.state_file`
    - 首次启动自动生成 UUID 并持久化
  - 相关文件：
    - `backend/cmd/agent/identity.go`
    - `backend/cmd/agent/main.go`
    - `backend/cmd/agent/identity_test.go`
- **监控链修复**
  - 修复因旧 agent / 旧身份冲突导致监控中心指标长期显示 `0.0%`
  - 当前正式 agent 已稳定上报：
    - CPU
    - Memory
    - Disk
- **监控前端新增页面**
  - `AgentDetail.vue`
  - `AlertEvents.vue`
  - `MonitorDatasources.vue`
  - `MonitorQuery.vue`
  - `AlertSilences.vue`
  - `OnCallSchedules.vue`
- **监控前端接入**
  - `router/index.ts` 已注册监控隐藏路由
  - `api/index.ts` 已补监控数据源和 PromQL 查询 API
  - `Menus.vue` 已补监控组件选项
  - `MonitorDashboard.vue` 已增加：
    - Agent 详情入口
    - 告警事件中心入口
    - 数据源页入口
    - PromQL 查询台入口
    - 服务树聚合监控
    - 负责人聚合监控
    - 告警静默 / OnCall 入口
  - `AlertRules.vue` 已增加：
    - 告警事件中心入口
    - 数据源入口
    - 查询台入口
    - 告警动作字段展示与配置
    - OnCall / 服务树 / 负责人 范围配置
    - 告警静默 / OnCall 入口
    - 修复任务下拉选择
  - `AgentDetail.vue` 已增加：
    - 服务树归属
    - 负责人归属
    - 资产来源
- **监控后端第一批已落树**
  - Prometheus 数据源模型 / 仓储 / 服务 / Handler
  - 告警动作：
    - `notify_only`
    - `create_ticket`
    - `execute_task`
  - 告警联动工单 / 修复任务
- **监控 V3 第二批已落树**
  - `AlertSilence`
  - `OnCallSchedule`
  - `/monitor/aggregates/service-trees`
  - `/monitor/aggregates/owners`
  - `/alert-silences`
  - `/oncall-schedules`
  - `alert_rule_handler` 已接收 `oncall_schedule_id`
  - 监控相关 Handler 的 Swagger 注释已补齐到代码层
  - 新增迁移文件：`backend/migrations/012_create_alert_ops_tables.sql`
  - 新增菜单迁移：`backend/migrations/013_extend_monitor_ops_menus.sql`
- **通知中心分层收口**
  - 管理员页：`NotificationConsole.vue` 升级为“通知配置中心”
  - 个人页：新增 `MyNotificationSettings.vue`
  - 后端新增：
    - `GET/POST /api/v1/notifications/config`
    - `GET/POST /api/v1/notifications/preferences`
  - 业务线通知渠道已接入：
    - 监控告警：规则级 `notify_channels`
    - CI/CD：流水线级 `notify_channels`
    - 工单：模板级 `notify_channels`
  - 通知配置从 YAML 手工维护推进为页面化配置
  - 告警通知改为立即异步投递，不再只依赖重试调度器扫描

### 验证结果

- `go test ./cmd/agent -run TestResolveAgentID -count=1` ✓
- `go build ./cmd/core` ✓
- `go test ./internal/service -run 'TestEvaluateRule|TestParsePipelineRuntimeConfig|TestBuildTaskEnv' -count=1` ✓
- `npm run build` ✓
- Playwright 监控 V2/V3 页面烟测：
  - `/monitor/dashboard` ✓
  - `/monitor/alerts` ✓
  - `/monitor/datasources` ✓
  - `/monitor/query` ✓
  - `/monitor/agents/:agent_id` ✓

### 补充结果

- **Prometheus 数据源联调**
  - 使用本地 mock Prometheus（19090）验证：
    - 数据源创建成功
    - 健康检查成功
    - `/api/v1/monitor/query` 返回 vector 结果
    - `/api/v1/monitor/query-range` 返回 matrix 结果
- **告警自动化联调**
  - `execute_task` 规则触发后，`alert_event.task_execution_id` 已正常回写
  - `create_ticket` 规则触发后，`alert_event.ticket_id` 已正常回写
  - 清理旧的脏测试规则后，`evaluate` 结果已恢复：
    - `error_count = 0`
- **菜单接入**
  - 新增监控菜单 migration：
    - `alert_events`
    - `monitor_datasources`
    - `monitor_query`
  - 已执行到 `bigops2`
  - 侧边栏 Playwright 烟测通过：
    - `监控大盘`
    - `告警规则`
    - `告警事件`
    - `监控数据源`
    - `PromQL 查询台`
- **Agent 身份收口**
  - 当前正式 agent 已切到 UUID 身份：
    - `81fa4fbb-8a2b-4938-a368-e46ff57679c5`
  - 当前监控指标恢复正常：
    - CPU `5.57%`
    - Memory `72.89%`
    - Disk `93.13%`

---

## Session: 2026-03-25 (Module 05 监控系统 V1)

### 完成事项

**Agent 指标链路：**
- 扩展 `agent.proto` Heartbeat，支持上报：
  - CPU 使用率
  - 内存使用率
  - 磁盘使用率
- Agent 本地新增资源采集器
- gRPC Server 心跳接收时同时更新 `agent_infos` 当前指标并落历史采样

**后端监控与告警：**
- 新增模型：
  - `AgentMetricSample`
  - `AlertRule`
  - `AlertEvent`
- 新增 Repository：
  - `agent_metric_sample_repository.go`
  - `alert_rule_repository.go`
  - `alert_event_repository.go`
- 新增服务：
  - `monitor_service.go`
  - `alert_service.go`
- 新增接口：
  - `/api/v1/monitor/summary`
  - `/api/v1/monitor/agents`
  - `/api/v1/monitor/agents/:agent_id/trends`
  - `/api/v1/alert-rules`
  - `/api/v1/alert-events`
- 告警事件触发时复用现有 `NotificationService` 发站内通知
- 启动时接入 `AlertScheduler`

**前端页面：**
- 新增 `MonitorDashboard.vue`
- 新增 `AlertRules.vue`
- 新增监控前端 API：
  - `monitorApi`
  - `alertRuleApi`
- 菜单/路由已补 `MonitorDashboard`、`AlertRules` 组件识别
- 新增 `008_seed_monitor_menus.sql`

### 修改的文件
| 文件 | 操作 |
|------|------|
| `backend/proto/agent.proto` | 扩展 Agent 心跳指标字段 |
| `backend/internal/agent/metrics.go` | 新增 Agent 本地运行指标采集 |
| `backend/internal/agent/client.go` | 心跳上报指标 |
| `backend/internal/grpc/server.go` | 接收心跳指标并落历史采样 |
| `backend/internal/model/agent_info.go` | 新增当前指标字段 |
| `backend/internal/model/agent_metric_sample.go` | 新增指标采样模型 |
| `backend/internal/model/alert_rule.go` | 新增告警规则模型 |
| `backend/internal/model/alert_event.go` | 新增告警事件模型 |
| `backend/internal/repository/agent_metric_sample_repository.go` | 指标采样仓储 |
| `backend/internal/repository/alert_rule_repository.go` | 告警规则仓储 |
| `backend/internal/repository/alert_event_repository.go` | 告警事件仓储 |
| `backend/internal/service/monitor_service.go` | 监控查询服务 |
| `backend/internal/service/alert_service.go` | 告警判定/通知/调度器 |

---

## Session: 2026-03-25 (上下文压缩检查点)

### 当前状态

- 已确认后续开发主线切换到 **Module 05 监控系统**
- 用户已明确：
  - 工单系统视为完成，不再继续扩展
  - `Ctrl+C` 退出问题用户已自行修复，不再处理
  - 后续开发要求直接推进、尽量并行、以最终版为目标
- 已确认浏览器调试能力可用：
  - Chrome DevTools 风格排查
  - Playwright 浏览器自动化

### 监控模块当前结论

- 监控 V1 主链已经落地：
  - Agent 心跳指标上报
  - 指标落库
  - 告警规则/事件模型
  - 站内通知复用
  - 监控大盘与告警规则前端页面
- 当前还未完成最终版联调，关键剩余工作：
  1. 联调 Agent 指标持续落库与趋势接口
  2. 验证告警规则触发、恢复、通知闭环
  3. 收口监控页面体验并完成菜单/API 实测

### 下一步

1. 实测 monitor API 与数据库落库
2. 用浏览器工具复现并检查监控页面实际行为
3. 补齐最终版缺口后统一执行构建与联调验收

---

## Session: 2026-03-26 (Module 05 监控系统最终版收口)

### 完成事项

**后端增强：**
- `AgentRepository` 新增监控场景专用列表查询，支持 `status/keyword` 过滤与监控排序
- `MonitorService` 补充：
  - 启用规则总数
  - 最近采样时间
  - 最近告警摘要
  - 告警状态分布 / 级别分布
  - Top Agent 排序与裁剪
  - 监控查询前自动执行 stale agent offline 收敛
- `AlertService` 补充：
  - 规则字段标准化与校验
  - 规则重名校验
  - 巡检结果统计（triggered/resolved/updated/error）
  - 告警恢复时发送恢复通知
  - 调度器改为适配新的巡检返回结构
- `AlertRuleHandler` / `router.go` 补充：
  - `POST /api/v1/alert-rules/evaluate`
  - 告警事件关键字筛选

**前端最终版：**
- `MonitorDashboard.vue` 重构为值守视角页面：
  - 顶部总览卡片
  - 自动刷新
  - Agent 关键字 / 状态筛选
  - CPU / 内存 / 磁盘 Top
  - 最近告警面板
  - Agent 趋势抽屉（CPU / 内存 / 磁盘趋势）
- `AlertRules.vue` 重构为完整告警台：
  - 规则筛选
  - 启停开关
  - 立即巡检
  - 告警事件筛选
  - 事件确认 / 关闭
- `frontend/src/api/index.ts` 对齐新增监控 / 告警接口能力

**运行态验证：**
- `go build ./cmd/core` 通过
- `npm run build` 通过
- 使用 `timeout 5 ./core` 验证：
  - 最新后端可启动
  - `AutoMigrate` 会创建：
    - `agent_metric_samples`
    - `alert_rules`
    - `alert_events`
  - 监控 / 告警路由均已注册
- 使用管理员账号补做鉴权接口验证：
  - `POST /api/v1/auth/login`
  - `GET /api/v1/monitor/summary`
  - `GET /api/v1/monitor/agents`
  - `POST /api/v1/alert-rules/evaluate`
  - `GET /api/v1/alert-events`
  均返回成功

### 尚未完成

- Playwright 在当前 root 环境下受到 Chromium sandbox 限制，未完成浏览器自动化回归

---

## Session: 2026-03-26 (浏览器联调环境修复)

### 完成事项

- 通过官方 `@playwright/mcp` 文档定位 root 环境问题，确认官方支持：
  - `--no-sandbox`
  - `PLAYWRIGHT_MCP_NO_SANDBOX`
- 已修改 `/root/.codex/config.toml` 中的 Playwright MCP 启动参数：
  - `--headless`
  - `--no-sandbox`
- 已在沙箱外用原生 Playwright 脚本验证浏览器可启动并可正常访问页面内容

### 当前结论

- 当前会话里的 MCP Playwright transport 因手动重启已断开，需要新会话/重连后才能直接继续用 `browser_*` 工具
- 但 Playwright 运行时方案本身已验证可用，后续前端联调可以继续用 Playwright 做真实测试

---

## Session: 2026-03-26 (双浏览器工具验收)

### 完成事项

- 已使用 **Playwright** 真实登录并验证：
  - 监控大盘 `/monitor/dashboard`
  - 告警规则 `/monitor/alert-rules`
  - 无 console error
  - 无 4xx/5xx failed response
- 已使用 **Chrome DevTools Protocol** 真实登录并验证：
  - 登录页 DOM / placeholder / button 结构
  - Vue Router 中存在监控相关动态路由
  - 监控大盘命中：
    - `监控中心`
    - `Agent 实时列表`
    - `最近告警`
    - `热点 Top`
  - 告警规则命中：
    - `告警规则`
    - `告警事件`
    - `立即巡检`
    - `新增规则`
  - 无 console error
  - 无 JS exception
  - 无 failed response

### 产物

- Playwright 截图：
  - `/tmp/monitor-pages-check.png`
- Chrome DevTools 截图：
  - `/tmp/chrome-devtools-monitor-check.png`

---

## Session: 2026-03-26 (全流程烟测 + 关键深测)

### 完成事项

**Playwright 全菜单烟测：**
- 以 `admin` 账号登录后，自动遍历当前可见菜单路由
- 实测通过页面共 16 个，全部通过：
  - `服务树`
  - `云账号`
  - `主机资产`
  - `发起工单`
  - `我的待办`
  - `我的申请`
  - `工单模板`
  - `任务管理`
  - `Agent 管理`
  - `用户管理`
  - `角色管理`
  - `菜单管理`
  - `审计日志`
  - `部门管理`
  - `监控大盘`
  - `告警规则`
- 本轮烟测：
  - `totalRoutes=16`
  - `passedRoutes=16`
  - `failedRoutes=0`
  - `globalConsoleErrors=0`

**Chrome DevTools 关键页面复核：**
- 登录页 DOM、按钮、placeholder 正常
- 登录后路由表包含监控页路由
- 监控大盘和告警规则页的关键文本全部命中
- `consoleErrors=0`
- `jsExceptions=0`
- `failedResponses=0`

**关键深流程：**
- `任务创建`：成功
  - 新建任务 `E2E任务...`
  - 创建后正确回到 `任务管理` 列表
- `告警规则新增`：成功
  - 已通过 API 清理测试规则，避免污染环境
- `工单发起`：失败
  - 前端选模板并填写标题/描述后，页面返回：
    - `参数错误: Key: 'CreateTicketRequest.TypeID' Error:Field validation for 'TypeID' failed on the 'required' tag`
  - 说明当前工单系统前端已经去掉了 `type_id` 输入，但后端创建工单接口仍然强依赖 `TypeID`

### 产物

- Playwright 烟测结果：
  - `/tmp/playwright-full-smoke.json`
- Playwright 深测结果：
  - `/tmp/playwright-deep-flows.json`
- Chrome DevTools 复核结果：
  - `/tmp/chrome-devtools-smoke.json`

### 当前结论

- 大多数模块页面级逻辑已可正常访问与请求
- 当前最明确的业务级阻塞点是：
  - `发起工单` 流程仍被后端 `TypeID required` 拦住

---

## Session: 2026-03-26 (工单发起修复 + 普通账户 + 多级审批实测)

### 完成事项

**缺陷修复：**
- 修复 `CreateTicketRequest.TypeID` 仍为必填导致真实发起工单失败的问题
  - 文件：`backend/internal/handler/ticket_handler.go`
- 修复审批待办页按钮事件冒泡，避免点击“通过/拒绝”时被整行跳转打断
  - 文件：`frontend/src/views/ApprovalInbox.vue`
- 修复审批流事务可见性问题：
  - 审批记录在事务中更新后，阶段通过判断必须走同一个 `tx`
  - 否则会读到旧 `pending` 记录，导致永远卡在当前阶段
  - 文件：`backend/internal/service/approval_service.go`

**浏览器实测：**
- `admin`：
  - 全菜单烟测通过
  - Playwright 与 Chrome DevTools 双证据已跑通
- `yunwei`：
  - Playwright 烟测通过，当前可见 7 个菜单页全部通过
  - Chrome DevTools 复核通过：
    - `/ticket/applied`
    - `/monitor/dashboard`
- 关键深流程：
  - `发起工单`：已修复并通过
  - `任务创建`：通过
  - `告警规则新增/删除`：通过

**多级审批实测：**
- 新建测试账号：
  - `e2e_approver_090729`
- 新建测试模板：
  - `E2E两级审批模板-e2e_approver_090729`
- 两级审批链路已通过：
  1. `yunwei` 发起请求工单
  2. `admin` 在 `/approval/inbox` 审批通过一级
  3. `e2e_approver_090729` 在 `/approval/inbox` 审批通过二级
  4. `yunwei` 回看工单详情，审批状态为 `审批通过`

### Test Results
| Test | Result |
|------|--------|
| `go build ./cmd/core` | 通过 |
| `npm run build` | 通过 |
| `admin` Playwright 全菜单烟测 | 16/16 通过 |
| `admin` Chrome DevTools 关键流 | 通过 |
| `yunwei` Playwright 普通账户烟测 | 7/7 通过 |
| `yunwei` Chrome DevTools 普通账户复核 | 通过 |
| 两级审批端到端 | 通过 |

---

## Session: 2026-03-26 (Module 06 CI/CD 第一批实现)

### 完成事项

**后端：**
- 新增 CI/CD 模型：
  - `CICDProject`
  - `CICDPipeline`
  - `CICDPipelineRun`
- 新增 Repository：
  - `cicd_project_repository.go`
  - `cicd_pipeline_repository.go`
  - `cicd_pipeline_run_repository.go`
- 新增 Service：
  - `cicd_service.go`
- 新增 Handler：
  - `cicd_handler.go`
- 新增接口：
  - `GET /api/v1/cicd/projects`
  - `POST /api/v1/cicd/projects`
  - `POST /api/v1/cicd/projects/:id`
  - `POST /api/v1/cicd/projects/:id/status`
  - `POST /api/v1/cicd/projects/:id/delete`
  - `GET /api/v1/cicd/pipelines`
  - `POST /api/v1/cicd/pipelines`
  - `POST /api/v1/cicd/pipelines/:id`
  - `POST /api/v1/cicd/pipelines/:id/trigger`
  - `POST /api/v1/cicd/pipelines/:id/delete`
  - `GET /api/v1/cicd/runs`
- 运行触发已复用任务中心与工单模板：
  - 无审批模板时可直接生成 `PipelineRun`
  - 配置部署任务 + 目标主机时可复用 `TaskService.ExecuteTask`
  - 配置审批模板时可复用工单审批，生成发布审批单
- `main.go` 已接入 AutoMigrate
- 新增菜单 migration：
  - `backend/migrations/009_seed_cicd_menus.sql`
  - 已执行到 `bigops2`

**前端：**
- 新增页面：
  - `CicdProjects.vue`
  - `CicdPipelines.vue`
- 新增 API：
  - `cicdProjectApi`
  - `cicdPipelineApi`
- 路由与菜单组件映射已接入：
  - `CicdProjects`
  - `CicdPipelines`

**验证：**
- `go build ./cmd/core` 通过
- `npm run build` 通过
- API 实测：
  - 项目创建成功
  - 流水线创建成功
  - 流水线触发成功
  - `GET /api/v1/cicd/runs` 可查询到运行记录
- Playwright 页面实测：
  - `CI/CD 项目` 页面可打开
  - `新增项目` 可成功创建项目
  - `CI/CD 流水线` 页面可打开

### 当前状态

- CI/CD 已不再是空白模块，第一批主骨架已落地
- 下一步优先级：
  1. 流水线运行历史页
  2. 流水线页面接入运行记录与触发结果展示
  3. 发布审批通过后自动触发部署任务

---

## Session: 2026-03-26 (Module 06 CI/CD 最终闭环)

### 完成事项

**后端闭环：**
- `GET /cicd/runs/:id` 已可返回：
  - `run`
  - `task_execution`
  - `target_hosts_list`
  - `variables`
  - `metadata`
  - `artifact_summary_map`
- 审批通过后自动部署已打通：
  - `approval_service.go` 在最终 `approval_approved` 分支回调 `CICDService.StartApprovedRunsByTicketID`
  - `StartApprovedRunsByTicketID` 会按 `approval_ticket_id` 找到等待审批的 `PipelineRun`
  - 自动触发 `deploy_task_id`
  - 回写 `task_execution_id`
  - 更新 `summary`
- 任务执行结果回写已接上：
  - `PipelineRun <- TaskExecution` 状态映射
  - `success / failed / canceled / running`
  - 运行详情会动态同步最终状态

**前端闭环：**
- `CicdPipelines.vue`
  - 最近运行状态、时间、摘要、详情抽屉
- `CicdRuns.vue`
  - 运行列表
  - 详情抽屉
  - 任务执行与主机结果展示
- `CicdRuns` 菜单入口已加入并执行 migration

**关键验证：**
- `go build ./cmd/core` 通过
- `npm run build` 通过
- 浏览器验收：
  - `运行记录` 页可正常打开
  - `运行详情` 抽屉可正常打开
  - `流水线列表` 中“最近运行”可显示成功/失败/审批中等状态
- 接口闭环实测：
  - 创建带审批模板 + 部署任务的流水线
  - 触发后生成发布审批工单
  - 两级审批通过
  - 自动触发部署任务
  - `task_execution_id` 已回写
  - 最新代码下 `GET /cicd/runs/:id` 返回：
    - `run_status = success`
    - `task_execution_status = success`
    - `summary = 部署完成：成功 1 / 总数 1`

### 当前结论

- CI/CD 已从“管理页骨架”推进到“可用闭环”
- 当前已具备：
  - 项目管理
  - 流水线定义
  - 手动触发
  - 运行记录
  - 运行详情
  - 发布审批联动
  - 审批通过自动部署
  - 任务执行结果回写

### 补充结果

- `运行记录` 页面打不开的问题已修复：
  - 原因：`CicdRuns.vue` 错误导入了不存在的 `cicdRunApi`
  - 修复后页面可正常打开，且无 console error / failed response
- `CicdRuns` 与 `CicdPipelines` 现已支持运行详情抽屉：
  - 基本信息
  - 摘要 / 错误
  - 任务执行
  - 主机结果
- `PipelineRun <- TaskExecution` 已实现状态动态同步：
  - `success`
  - `failed`
  - `running`
  - `canceled`
- `审批拒绝 -> PipelineRun.failed` 已补齐
- `webhook / retry / rollback` 能力已补齐：
  - `POST /api/v1/cicd/webhook/:code`
  - `POST /api/v1/cicd/runs/:id/retry`
  - `POST /api/v1/cicd/runs/:id/rollback`
- 部署任务已支持接收 CI/CD 环境变量：
  - `CICD_RUN_ID`
  - `CICD_PIPELINE_ID`
  - `CICD_PROJECT_NAME`
  - `CICD_BRANCH`
  - `CICD_COMMIT_ID`
  - `CICD_TRIGGER_TYPE`
  - `CICD_SOURCE_RUN_ID`

### 最终验证

- API 闭环验证：
  - 带审批模板的流水线触发后生成审批工单
  - 两级审批通过后自动触发部署任务
  - `GET /cicd/runs/:id` 返回：
    - `run_status = success`
    - `task_execution_status = success`
    - `summary = 部署完成：成功 1 / 总数 1`
  - 审批拒绝后：
    - `run_status = failed`
    - `result = rejected`
    - `summary = 审批被拒绝：...`
- 浏览器验收：
  - Playwright：
    - `CI/CD 项目`
    - `流水线管理`
    - `运行记录`
    - `运行详情`
  - Chrome DevTools/CDP：
    - `/cicd/projects`
    - `/cicd/pipelines`
    - `/cicd/runs`
    - 页面关键文本命中正常
- 额外验证：
  - `webhook` 路由已存在，命中时不再返回 404，而是正常返回业务错误 `流水线不存在`
| `backend/internal/handler/monitor_handler.go` | 监控接口 |
| `backend/internal/handler/alert_rule_handler.go` | 告警规则/事件接口 |
| `backend/api/http/router/router.go` | 注册监控与告警路由 |
| `backend/cmd/core/main.go` | AutoMigrate 与告警调度器接入 |
| `backend/migrations/008_seed_monitor_menus.sql` | 监控模块菜单迁移 |
| `frontend/src/api/index.ts` | 新增监控 API |
| `frontend/src/views/MonitorDashboard.vue` | 新增监控大盘页 |
| `frontend/src/views/AlertRules.vue` | 新增告警规则页 |
| `frontend/src/router/index.ts` | 接入监控页面组件映射 |
| `frontend/src/views/Menus.vue` | 补充监控组件选项 |

### Test Results
| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| 后端编译 | `go build ./cmd/core` | 编译通过 | 通过 | ✓ |
| 前端构建 | `npm run build` | 构建通过 | 通过 | ✓ |
