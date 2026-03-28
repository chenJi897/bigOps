# Findings: BigOps 技术发现与经验

## 已解决的技术问题

### GORM 相关
| 问题 | 原因 | 解决方案 |
|------|------|----------|
| 缩写字段列名不匹配 | GORM 默认 snake_case 转换对缩写处理不一致 | 添加 `gorm:"column:xxx"` tag（如 IDC → idc） |
| JSON 字段空值报错 | MySQL JSON 列不接受空字符串 | BeforeSave hook 将 `""` 转为 `"[]"` 或 `"{}"` |
| LocalTime 零值更新丢失 | 构造新对象 Save() 会覆盖零值时间 | 先查 existing 再改字段 |
| 软删除 + uniqueIndex 冲突 | 同步已删除记录时违反唯一索引 | Unscoped 查找已删除记录并恢复 |
| stdout/stderr 并发追加覆盖 | 多 goroutine 同时 Save() 互相覆盖 | 使用 `gorm.Expr("CONCAT(COALESCE(col,''), ?)")` 原子追加 |

### 前端相关
| 问题 | 原因 | 解决方案 |
|------|------|----------|
| keep-alive 缓存失效 | include 匹配的是组件名，不是路由名 | `defineOptions({ name: 'XxxComponent' })` |
| 菜单路径跳转 404 | 前端路径和数据库菜单路径不一致 | 数据库路径含模块前缀（/cmdb/assets），前端跳转需匹配 |
| WebSocket 认证失败 | AuthMiddleware 只读 Header，WS 无法设 Header | 添加 `c.Query("token")` 降级读取 |
| el-tag type 类型报错 | Element Plus TS 类型严格 | 使用 `(xxx as any)` 类型断言 |

### 安全相关
| 问题 | 风险 | 解决方案 |
|------|------|----------|
| 注册/登录无限流 | 暴力注册/破解密码 | Redis IP 限流 + 账号失败锁定 |
| 分页 size 无上限 | 传 size=999999 造成 OOM | `parsePageSize()` 全局限制 max=100 |
| executor cmd.Env 丢失 PATH | 设置 Env 后丢失系统 PATH | `cmd.Env = os.Environ()` 先继承 |
| Casbin 未启用 | 所有认证用户可访问任何 API | 启用中间件 + 白名单 + 启动同步 |
| 关键错误 `_ =` 忽略 | 数据不一致无法排查 | 替换为 `logger.Warn()` |

### 架构模式
| 发现 | 说明 |
|------|------|
| 云同步架构 | SyncRunner 统一入口 + Scheduler 60s 巡检 + per-account mutex |
| Agent 通信 | gRPC 双向流心跳 → Server 通过 HeartbeatResponse.Task 下发任务 |
| WebSocket 日志 | AgentManager pub/sub 模式，channel 满时非阻塞跳过 |
| Casbin 同步时机 | SetMenus() 和 SetUserRoles() 时自动同步 + 启动时全量同步 |
| 菜单驱动路由 | 后端菜单树 → 前端 generateRoutes → 动态路由 + companion 隐藏路由 |
| 工单管理 IA | 用户视角入口应收敛为“发起工单 / 我的待办 / 我的申请 / 工单模板”，不应直接暴露工单类型、请求模板、审批策略等底层概念 |
| TicketList 复用策略 | 同一页面通过 route meta 注入 `ticketMode` 复用成“我的待办 / 我的申请”两种固定视角，避免 tabs 混用多种角色视图 |
| 工单模板定位 | 用户看到的“工单模板”应以 `RequestTemplate` 为主实体，`TicketType` 退居底层规则支撑，`ApprovalPolicy` 作为模板配置入口 |
| 监控模块 V1 路线 | 先以现有 Agent 体系做“内建监控闭环”（指标上报 -> 指标落库 -> 告警规则 -> 站内通知 -> 大盘展示），Prometheus 作为后续兼容能力而非第一落点 |
| Agent 指标上报 | 复用现有 gRPC Heartbeat 流最省成本，直接在心跳包里增加 CPU/内存/磁盘使用率字段即可，不必单独新开上报协议 |
| 告警模型边界 | 告警规则与告警事件应独立落库，通知只作为副作用；规则只负责判定，事件负责状态流转（firing/resolved） |
| 监控最终版 UI 结构 | 监控页最实用的信息密度是“总览卡片 + Agent 列表 + 最近告警 + 趋势抽屉”，而不是一开始就上 PromQL 大盘编辑器 |
| 监控数据新鲜度 | `last_collected_at` 需要直接暴露给前端，否则用户无法判断当前大盘是不是旧数据 |
| stale agent 收敛 | 只靠 gRPC 断链不足以保证离线状态准确，监控查询和告警巡检前都应补一次 `MarkStaleOffline` |
| 告警闭环 | 监控系统的“最终版”至少要覆盖：规则启停、立即巡检、事件确认、事件关闭、恢复通知，而不只是 CRUD |
| Playwright root 环境 | `@playwright/mcp` 官方支持 `--no-sandbox` / `PLAYWRIGHT_MCP_NO_SANDBOX`。root 下浏览器启动失败时，应先关闭 sandbox；若仍被当前沙箱限制，则需在沙箱外执行 Playwright 验证 |
| 全菜单烟测基线 | 当前 `admin` 视角下 16 个可见菜单页均可正常打开，且没有 console error / failed response，可作为后续回归的基线 |
| 工单发起阻塞点 | 当前工单前端已经去掉 `type_id` 输入，但后端 `CreateTicketRequest.TypeID` 仍保留 `required` 校验，导致真实发起工单流程失败 |
| 审批待办按钮交互 | `ApprovalInbox` 行点击与操作按钮如果不做 `.stop`，会在审批确认前被整行导航打断，导致“点通过却没真正审批” |
| 审批流事务边界 | 审批记录更新后若用事务外连接重新读取当前阶段记录，会读到旧状态，导致审批永远卡在当前阶段；阶段推进判断必须使用同一个 `tx` |
| 普通账户基线 | `yunwei` 当前可见 7 个菜单页均可正常打开，监控与工单相关页面浏览正常 |
| CI/CD 第一批落点 | 第一批最划算的闭环不是先做完整构建引擎，而是先做 `项目管理 + 流水线定义 + 手动触发运行记录`，执行复用任务中心，审批复用工单模板 |
| CI/CD 页面契约 | 前端页面当前使用 `repository/default_branch/active/schedule` 这套字段命名，后端需要做兼容映射，否则页面会显示空值或提交失败 |
| CI/CD 最小运行闭环 | 即使没有配置部署任务和审批模板，流水线也应该能生成 `PipelineRun(status=created)`，这样前端可以先看到“触发记录”，后续再逐步接入真实执行 |
| CI/CD 最终闭环 | “像回事”的最低标准不是有项目/流水线 CRUD，而是：`手动触发 -> 运行记录 -> 运行详情 -> 审批通过 -> 自动部署 -> 状态回写` |
| 运行详情形态 | 对当前后台来说，`运行详情抽屉` 比单独详情页更快落地，也足够承载发布记录、任务执行和主机结果 |
| 执行状态回写 | `PipelineRun` 不能只在触发时写一次状态，必须在查询运行列表/详情时根据 `TaskExecution` 最新状态同步，否则前端会永远看到过时的 `running` |
| CI/CD V3 触发方式 | `webhook` 不应依赖登录态，直接公开入口更符合发布系统习惯；但必须至少有 `pipeline code` 这一层显式路由标识 |
| 回滚的现实边界 | 当前回滚本质是“基于历史运行再触发一条 rollback 类型运行”，真正是否回退到旧版本，取决于部署任务脚本是否消费了传入的 CI/CD 环境变量 |
| CI/CD 阶段真相 | 对 BigOps 这类运维平台，最小可用 CI/CD 不该只有 `run.status`，而要把 `build / approval / deploy` 三段状态一起落到运行快照里，否则前端永远只能看到一个黑盒状态 |
| build 阶段复用任务中心 | 现有任务中心足够承载 build 阶段，没必要另造执行器；关键是用 `ConfigJSON.build_hosts` 区分构建主机与部署主机 |
| 运行详情错位根因 | 即使 `syncRunExecutionStatus` 已实现，如果 `GetRunDetail` 直接绕过它、只按 `TaskExecution` 算总状态，也会出现“总状态已成功但阶段卡在 build/running”的假象 |
| Webhook 入口契约 | 后端当前真实 Webhook 路由是 `/api/v1/cicd/webhook/:pipelineCode`，不是“项目编码 + 流水线编码”双段式；页面文案和预览地址必须与此一致，否则用户会直接配错 |
| build 成功后的重复推进风险 | 构建完成进入审批后，如果不清掉活动中的 build 执行引用，后续每次查详情都有机会再次推进状态机，导致重复建审批单或重复触发部署 |
| 站内通知体验缺口 | 仅提供单条 `mark read` 的通知抽屉，用户感知就是“只是展示”；通知中心至少要具备 `未读/全部` 过滤、`全部已读`、`清空已读` 三件套，才像成熟产品 |
| 站内通知跳转价值 | `biz_type + biz_id` 如果不用起来，通知只能读不能处理；把通知直接跳到工单、执行记录、告警页，才能形成闭环 |
| 通知配置分层 | 通知系统不能只有“管理员全局配置”或“业务规则选渠道”其中一层；完整产品至少要拆成：管理员全局渠道配置、业务对象选渠道、用户个人通知偏好三层 |
| 告警通知延迟根因 | 仅调用 `PublishTx()` 会落库但不立即发，真正发送要等调度器扫描；告警这类强时效事件应在事务提交后立即 `DispatchEventAsync()` |
| message-pusher 路线 | 对 BigOps 来说，比自研钉钉/飞书/企微三套 adapter 更合适的是：自研通知中心 + `message-pusher` 聚合出口，业务层只关心渠道选择，不关心平台细节 |
| 首页权限展示原则 | 首页不能只做全局统计墙；更合理的是“个人工作台 + 按权限显示的平台总览”，普通用户看与自己有关的数据和自己能进的模块，管理员看完整平台视角 |
| Agent 身份策略 | 用 `hostname_ip` 作为默认 agent_id 很容易在重装、换 IP、并行进程时冲突；更稳的正式方案应是“固定 id 或首次生成 UUID 并落本地 state file” |
| 监控 0 值根因 | `0.0%` 不一定是采集代码坏了，也可能是旧 agent、旧二进制、错误身份策略导致数据被覆盖或混写；必须同时查采集器、gRPC 连接和 agent_id 冲突 |
| 监控中心 V2 最小可用形态 | `AgentDetail + AlertEvents + 数据源管理 + PromQL 查询台` 这 4 个页面补齐后，监控中心才从“大盘”升级为“值守入口” |
| 监控中心 V3 值守形态 | 对运维平台来说，比“更多图表”更值钱的是 `服务树聚合 + 负责人聚合 + 告警静默 + OnCall`，这样值守和归责才完整 |
| AlertRule 配置漏字段风险 | 前端接了 `oncall_schedule_id` 并不代表链路通了；如果 handler 不接这个字段，页面保存会静默丢配置，属于典型“表单看着能填、后端实际没落库”问题 |
| 迁移文件必要性 | 仅依赖 `AutoMigrate` 会让新环境部署和排障不可追溯；像 `alert_silences / oncall_schedules` 这类新表应补版本化 SQL 迁移，避免运行时偷偷改库 |

## 代码模式参考

### 后端分层标准流程
```
1. model/xxx.go        — GORM 模型 + TableName() + BeforeSave()
2. repository/xxx.go   — DB 操作 + 分页 + 批量查询
3. service/xxx.go      — 业务逻辑 + 批量填充关联名称
4. handler/xxx.go      — 参数绑定 + Swagger + 审计日志 + response
5. router.go           — 注册路由 (GET 读 / POST 写)
6. main.go             — AutoMigrate + seed + 启动
```

### 前端页面标准模式
```vue
<script setup lang="ts">
defineOptions({ name: 'XxxPage' })  // keep-alive 必须
// imports, refs, fetchData, handlers
onMounted(() => { fetchData() })
</script>
<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 筛选 → 表格 → 分页 -->
    </el-card>
  </div>
</template>
<style scoped>
.page { padding: 20px; }
</style>
```
