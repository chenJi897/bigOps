# BigOps 全代码深度审计 — Bug 与逻辑漏洞清单

> 审计范围: 39 model / 38 repository / 30+ service / 28 handler / 6 middleware / 51 前端页面  
> 审计时间: 2026-04-11

---

## 严重程度分级

| 级别 | 说明 |
|------|------|
| **P0-致命** | 数据损坏、安全漏洞、核心流程阻断 |
| **P1-高危** | 业务逻辑错误、状态不一致、可被利用 |
| **P2-中危** | 边界场景处理不当、性能隐患 |
| **P3-低危** | 代码规范、可维护性、潜在风险 |

---

## P0 — 致命问题

### BUG-001: 任务审批门禁可被绕过
**文件**: `backend/internal/service/task_service.go:197-201`  
**问题**: 审批门禁检查 `FindLatestApprovedByTask` 只查最新一条 approved 记录，不验证该审批是否对应当前执行的主机范围。一旦某次审批通过，后续任何人都可以对该任务执行任意主机，审批形同虚设。  
**影响**: 高危任务审批可被永久绕过  
**修复建议**: 审批通过后设为「已消费」状态，每次执行消耗一条审批记录；或在审批记录中绑定 host_ips，执行时校验匹配。

### BUG-002: RateLimit Incr + Expire 非原子操作（竞态条件）
**文件**: `backend/internal/middleware/rate_limit.go:31-39`  
**问题**: `Incr` 和 `Expire` 是两个独立的 Redis 命令。如果程序在 `Incr` 之后、`Expire` 之前崩溃，该 key 将永不过期，导致该 IP/用户被永久限流。  
**影响**: 生产环境下用户可能被永久锁定  
**修复建议**: 使用 Redis Lua 脚本或 `SET NX EX` + `INCR` 的原子操作方式。

### BUG-003: Casbin 白名单过于宽泛 — 任意认证用户可读取所有用户信息
**文件**: `backend/internal/middleware/casbin.go:41-42`  
**问题**: `casbinPrefixWhitelistGET` 包含 `/api/v1/users/` 和 `/api/v1/users`，意味着所有认证用户都能 GET 访问用户列表和任意用户详情，绕过角色权限。  
**影响**: 普通用户可以查看所有用户信息（邮箱、手机号等）  
**修复建议**: 从 GET 白名单中移除 `/api/v1/users`，仅保留 `/api/v1/auth/info` 供当前用户获取自身信息。

---

## P1 — 高危问题

### BUG-004: TaskListQuery 状态过滤逻辑错误
**文件**: `backend/internal/repository/task_repository.go:67-69`  
**问题**: `if q.Status == 0 || q.Status == 1` 会让 `Status=0`（禁用）始终被过滤。当用户不传 status 参数时，handler 传入 `Status: -1`，但如果传入 `Status=0` 想筛禁用任务，会与 Go 的零值冲突。实际上 `Status=0` 是合法的禁用状态，但代码逻辑把它和"查全部"混淆了。  
**影响**: 无法正确筛选禁用状态的任务  
**修复建议**: 使用 `*int` 指针类型或专门的 `-1` 哨兵值明确区分。

### BUG-005: 告警事件统计只基于当前页而非全量
**文件**: `frontend/src/views/AlertEvents.vue:41-49`  
**问题**: `stats` 的 `firing/acknowledged/resolved/suppressed` 计数是基于 `events.value`（当前页数据），而 `total` 来自后端分页的全量计数。当事件数大于 pageSize 时，显示的状态统计只反映当前页数据，与 total 不匹配。  
**影响**: 用户看到的告警统计数据不准确  
**修复建议**: 后端增加按状态聚合的 count 接口，或在返回列表时附带 status_counts。

### BUG-006: 巡检自动重试可能导致无限递归
**文件**: `backend/internal/service/inspection_service.go` (`autoRetryIfAllowed`)  
**问题**: `autoRetryIfAllowed` 在 `SyncRecordStatus` 中被调用，创建新的执行记录。新执行完成后又会触发 `SyncRecordStatus`，再次调用 `autoRetryIfAllowed`。虽然有 `RetryCount >= MaxRetries` 的上限检查，但如果 `MaxRetries` 被设为很大的值（如 999），会产生大量重试执行。  
**影响**: 可能产生大量无效执行，耗尽系统资源  
**修复建议**: 增加全局重试频率限制（如同一模板每小时最多重试 N 次），并对 MaxRetries 设置硬上限（如最大 5）。

### BUG-007: 工单评论未校验内容为空
**文件**: `backend/internal/service/ticket_service.go:459-475`  
**问题**: `Comment` 方法没有校验 `content` 是否为空。用户可以提交空评论，产生无意义的 activity 记录。  
**影响**: 产生垃圾数据  
**修复建议**: 在 handler 或 service 层增加 `content` 非空校验。

### BUG-008: 任务执行不检查 DeletedAt（软删除的任务仍可执行）
**文件**: `backend/internal/service/task_service.go:193-194`  
**问题**: `GetTask` 使用 GORM 的 `First`，GORM 默认会过滤软删除记录，但 `executeTask` 没有额外校验。如果存在缓存或直接传 ID 的场景，需确认 GORM 的 soft delete scope 是否在所有路径生效。  
**影响**: 风险较低但需确认  
**修复建议**: 在 `executeTask` 中显式检查 `task.DeletedAt` 是否有值。

---

## P2 — 中危问题

### BUG-009: 登录成功未清除 IP 级别限流计数
**文件**: `backend/internal/middleware/rate_limit.go:93-98`  
**问题**: 登录成功时只清除了账号级别的失败计数（`login:fail:{username}`），但未清除 IP 级别的请求计数（`ratelimit:login:ip:{ip}`）。如果一个 IP 短时间内多次登录（即使都成功），IP 级别计数仍会累积，可能导致后续登录被限流。  
**影响**: 共享 IP 环境下（如公司出口 IP）可能误伤  
**修复建议**: 登录成功后也清除 IP 级别的 rateLimit key，或改用滑动窗口。

### BUG-010: TaskExecution 缺少 DeletedAt 软删除
**文件**: `backend/internal/model/task_execution.go`  
**问题**: `TaskExecution` 没有 `gorm.DeletedAt` 字段，意味着执行记录不支持软删除。虽然执行记录通常不需要删除，但与 `Task`（有 DeletedAt）不一致，且没有级联清理机制。当任务被软删除后，其执行记录仍然可被查询，但关联的任务名称将查询失败（返回空）。  
**影响**: 数据一致性问题  
**修复建议**: 执行记录不建议软删除，但应在查询时处理任务已删除的情况（已有 fallback，影响不大）。

### BUG-011: 密码复杂度检查不要求特殊字符
**文件**: `backend/internal/service/auth_service.go:186-218`  
**问题**: `checkPasswordComplexity` 只要求大写、小写、数字，不要求特殊字符。使用手册中写的"必须包含大小写字母、数字、特殊字符"与实际实现不一致。  
**影响**: 文档与实现不符，安全策略弱于预期  
**修复建议**: 要么在代码中加上特殊字符检查，要么修改文档。

### BUG-012: Casbin 策略同步不支持热更新
**文件**: `backend/cmd/core/main.go:247-299`  
**问题**: `syncCasbinPolicies` 只在启动时执行一次。如果运行时通过 API 修改了角色-菜单绑定关系，Casbin 策略不会自动刷新，直到服务重启。  
**影响**: 角色权限修改需要重启服务才能生效  
**修复建议**: 在角色/菜单变更的 service 层增加 `enforcer.LoadPolicy()` 或增量更新调用。

### BUG-013: 告警规则阈值校验对 agent_offline 类型不合理
**文件**: `backend/internal/service/alert_service.go:888-889`  
**问题**: `agent_offline` 类型的阈值为 0 或 1（布尔型），但校验逻辑 `item.Threshold < 0 || item.Threshold > 100` 对 agent_offline 类型跳过了（有前置判断）。然而创建规则时，用户可以为 `agent_offline` 设置任意阈值（如 50），语义不合理。  
**影响**: 用户困惑，可能配置无效规则  
**修复建议**: 对 `agent_offline` 类型强制阈值为 1。

### BUG-014: inferRiskLevel 逻辑顺序问题
**文件**: `backend/internal/service/task_service.go:71-93`  
**问题**: `inferRiskLevel` 中，如果脚本包含 `service`（非 `systemctl restart`），也会被标记为 `high`。例如脚本中有 `echo "this is a service description"` 就会误判为高风险。`service` 这个关键词过于宽泛。  
**影响**: 大量普通脚本被误标为高风险  
**修复建议**: 使用更精确的模式匹配，如 `\bservice\s+\w+\s+(start|stop|restart)\b`。

---

## P3 — 低危/改进

### BUG-015: 前端 TaskTemplates 新增审批按钮但 TaskList 页面未同步
**文件**: `frontend/src/views/TaskTemplates.vue` vs `frontend/src/views/TaskList.vue`  
**问题**: 风险等级列和审批按钮只加在了 `TaskTemplates.vue`，但路由映射中 `TaskTemplates` 实际使用的是 `TaskList.vue`（`viewModules.TaskTemplates` → `TaskList.vue`）。两个文件功能重复，可能导致用户看到的页面没有审批功能。  
**影响**: 审批功能可能在生产中不可见  
**修复建议**: 确认路由最终指向哪个文件，统一功能。

### BUG-016: Service 层频繁 New 实例（无单例）
**文件**: 多处 service/handler  
**问题**: 每次请求都 `NewXxxService()` 创建新实例，内部又 `NewXxxRepository()` 创建新仓库。虽然当前实现足够轻量（无连接池状态），但随着服务增长，`NewAlertService()` 内部会递归创建 `NewNotificationService()`、`NewTicketService()`、`NewTaskService()` 等，产生大量临时对象。  
**影响**: 内存/GC 压力随请求量线性增长  
**修复建议**: 对核心 service 实现单例或依赖注入。

### BUG-017: LocalTime JSON 反序列化可能丢失时区
**文件**: `backend/internal/model/local_time.go`  
**问题**: 需确认 `LocalTime` 的 UnmarshalJSON 是否正确处理了时区。如果前端传 UTC 时间、后端期望本地时间，可能出现时间偏移。  
**影响**: 跨时区场景下时间显示可能偏差  

### BUG-018: 告警事件 Topology 视图 N+1 查询
**文件**: `backend/internal/service/alert_service.go:1089-1124`  
**问题**: `TopologyView` 对每个 asset 查 agent，再对每个 agent 查 firing events，产生 N×M 次 DB 查询。资产数量较多时性能极差。  
**影响**: 大规模部署下拓扑视图响应缓慢  
**修复建议**: 批量查询 agents 和 events，在内存中关联。

### BUG-019: 前端 API 层缺少请求去重/防抖
**文件**: `frontend/src/api/index.ts`  
**问题**: 没有请求去重机制。用户快速双击"执行"按钮会发出两次请求，可能创建两条执行记录。  
**影响**: 重复操作导致脏数据  
**修复建议**: 在关键写操作按钮上加 loading 态（已部分实现）+ 后端幂等键。

### BUG-020: WebSocket 连接无心跳超时关闭
**文件**: `backend/internal/handler/task_handler.go:459-523`  
**问题**: WebSocket 写 pump 有 30 秒 ping，但读 pump 没有设置 `SetReadDeadline`。如果客户端网络断开但 TCP 连接未 RST，服务端会永远持有该连接的 goroutine 和 channel。  
**影响**: 长时间运行后 goroutine 泄漏  
**修复建议**: 在读 pump 中 `SetReadDeadline(time.Now().Add(60s))`，超时则关闭。

---

## 总结

| 级别 | 数量 | 关键问题 |
|------|------|----------|
| P0 致命 | 3 | 审批绕过、限流竞态、权限泄露 |
| P1 高危 | 5 | 统计不准、无限重试、状态过滤、空评论 |
| P2 中危 | 6 | IP限流残留、密码策略不符、Casbin不热更新、误判高风险 |
| P3 低危 | 6 | 路由映射冲突、N+1查询、无单例、goroutine泄漏 |
| **合计** | **20** | |

建议优先修复 P0 和 P1，其中 BUG-001（审批绕过）和 BUG-003（权限泄露）应立即处理。
