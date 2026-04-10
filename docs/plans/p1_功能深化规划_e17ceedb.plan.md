---
name: P1 功能深化规划
overview: 将 P1 三大模块（告警中心、巡检系统、Golden Signals）从"第一版演示级"升级到"生产可用级"，重点解决数据真实性、逻辑完整性和可扩展性问题。
todos:
  - id: p1-inspect-sync
    content: 巡检执行结果回写：InspectionRecord 状态闭环 + 报告真实数据 + 告警联动修复
    status: completed
  - id: p1-alert-audit
    content: 告警状态变更审计：新增 AlertEventActivity 模型，记录每次状态流转
    status: completed
  - id: p1-alert-convergence
    content: 告警收敛分组改 DB 聚合：SQL GROUP BY 替代内存分组，去掉 1000 条上限
    status: completed
  - id: p1-alert-timeline
    content: 告警时间轴支持多事件关联：同规则同主机的事件合并展示
    status: completed
  - id: p1-golden-real-data
    content: Golden Signals 数据源切换为 Agent 指标（agent_metric_samples）
    status: completed
  - id: p1-slo-config
    content: SLO 可配置：从硬编码改为可查询可修改
    status: completed
  - id: p1-golden-dimension
    content: Golden Signals 维度拆分基于真实数据（ServiceTree/Agent/MetricType）
    status: completed
  - id: p1-root-cause
    content: 根因分析增加变更关联：关联部署/配置变更事件，动态推断根因
    status: completed
isProject: false
---

# P1 功能深化规划

## 现状诊断

通过代码审查，三个 P1 模块的第一版存在以下共性问题：**数据是代理/模拟的，核心逻辑是简化占位的，缺少闭环**。

### 告警中心 — 核心逻辑偏浅

- **收敛分组**：内存聚合，1000 条硬上限，fingerprint 不含 agent_id，大数据量不完整（[alert_service.go](backend/internal/service/alert_service.go) `ListEventGroups`）
- **时间轴**：仅单条事件生命周期（triggered/ack/resolved），不支持同规则同主机的多事件关联时间线
- **根因分析**：写死 `primary_suspect = "rule_agent_hotspot"`，置信度用条数做简单加减，无拓扑/变更关联（`AnalyzeRootCause`）
- **状态机**：核心流转可用（firing→ack→resolved），但无状态变更审计记录，无 `suppressed` 状态

### 巡检系统 — 执行后断链

- **执行引擎**：真实复用了 `TaskService.ExecuteTask`，这部分没问题
- **记录状态不闭环**：`InspectionRecord` 创建后状态永远是 `running`，没有监听任务执行完成并回写 `success`/`failed`（[inspection_service.go](backend/internal/service/inspection_service.go) `ExecutePlan` 第 117-127 行）
- **报告是空壳**：`ReportJSON` 只存了启动时的元信息，不包含实际执行结果（主机输出/成功率/耗时）
- **告警联动只检查初始状态**：`exec.Status == "failed"` 在刚创建时基本不会命中（此时通常是 `pending`/`running`）

### Golden Signals — 数据源是假的

- **四大信号用任务执行记录当代理数据**：可用性 = 任务成功率，延迟 = 任务执行耗时，吞吐 = 每分钟执行数（[monitor_service.go](backend/internal/service/monitor_service.go) `GoldenSignals` 第 454-510 行）
- **SLO 硬编码**：99.9% 可用性、3000ms 延迟，不可配置
- **维度拆分用 TaskID/TaskType/HostIP 做映射**，不是真实的服务/接口/实例

---

## 深化方案

### 模块一：告警中心深化

**目标**：收敛分组支持大数据量 + 时间轴支持多事件 + 根因分析可解释 + 状态变更可追溯

#### 1.1 收敛分组改 DB 聚合

- 在 `alert_event_repository.go` 新增 `GroupByFingerprint` 方法，用 SQL `GROUP BY` 替代内存分组
- fingerprint 改为 DB 计算列或查询时拼接，包含 `rule_id + severity + agent_id`
- 去掉 1000 条硬上限，改为分页

#### 1.2 时间轴支持多事件关联

- `GetEventTimeline` 改为同时拉取同 `rule_id + agent_id` 的近期事件
- 每条事件展开 triggered/ack/resolved 节点，按时间排序合并
- 前端时间轴组件从"单事件三节点"升级为"多事件完整时间线"

#### 1.3 根因分析增加变更关联

- 新增 `AlertEventChange` 模型，记录告警前后的变更事件（部署、配置变更、任务执行）
- `AnalyzeRootCause` 查询告警触发前 N 分钟内的关联变更
- 置信度算法引入：关联变更数量、同类历史告警模式匹配
- `primary_suspect` 改为动态推断而非硬编码

#### 1.4 状态变更审计

- 新增 `AlertEventActivity` 模型（event_id, from_status, to_status, operator_id, note, created_at）
- 在 `AcknowledgeEvent`/`ResolveEvent`/`evaluateRule` 中写入状态变更记录
- 前端时间轴组件展示状态变更节点

### 模块二：巡检系统闭环

**目标**：执行结果自动回写 + 报告包含真实数据 + 告警联动检查最终状态

#### 2.1 执行结果回写

- 在 `ExecutePlan` 启动异步 goroutine 轮询任务执行状态
- 或：在 gRPC `checkExecutionCompletion` 中增加钩子，当执行完成时回调巡检服务
- 推荐方案：新增 `InspectionService.SyncRecordStatus(executionID)`，在 `checkExecutionCompletion` 中调用
- 更新 `InspectionRecord` 的 status/finished_at/report_json

#### 2.2 报告包含真实执行结果

- `SyncRecordStatus` 在任务完成时，从 `TaskExecution` + `TaskHostResult` 拉取真实数据
- 生成结构化报告：总主机数、成功数、失败数、每台主机的 stdout/stderr 摘要、耗时
- 报告写入 `ReportJSON`

#### 2.3 告警联动改为最终状态检查

- 将告警创建逻辑从 `ExecutePlan`（创建时）移到 `SyncRecordStatus`（完成时）
- 根据最终 `exec.Status` 判断是否触发告警
- 增加告警类型：部分失败（partial_fail）也可触发

### 模块三：Golden Signals 数据真实化

**目标**：基于 Agent 真实指标数据计算四大信号，而非任务执行代理数据

#### 3.1 数据源切换为 Agent 指标

- 当前 Agent 心跳已经采集了 CPU/内存/磁盘指标，存在 `agent_metric_samples` 表
- `GoldenSignals` 改为基于 `agent_metric_samples` 计算：
  - 可用性 = 窗口内在线 Agent 比例
  - 错误率 = 超阈值指标占比
  - 延迟 = Agent 心跳响应延迟（需在心跳中增加 RTT 字段）
  - 吞吐 = 窗口内总指标采样数 / 时间

#### 3.2 SLO 可配置

- 新增 `SLOConfig` 模型（或在 config.yaml 中配置）
- SLO 目标从硬编码改为可查询/可修改
- 前端增加 SLO 配置入口

#### 3.3 维度拆分基于真实数据

- 服务维度：按 ServiceTree 节点聚合 Agent 指标
- 实例维度：按单 Agent 聚合
- 指标类型维度：按 metric_type (cpu/memory/disk) 分别计算

---

## 推荐执行顺序

```
Phase 1（闭环修复，1-2天）
  ├── 2.1 巡检执行结果回写（最紧急，当前数据断链）
  ├── 2.2 巡检报告真实数据
  └── 2.3 巡检告警联动修复

Phase 2（告警深化，2-3天）
  ├── 1.4 状态变更审计（基础设施，后续都依赖）
  ├── 1.1 收敛分组改 DB 聚合
  └── 1.2 时间轴多事件关联

Phase 3（数据真实化，2-3天）
  ├── 3.1 Golden Signals 切换 Agent 指标数据源
  ├── 3.2 SLO 可配置
  └── 3.3 维度拆分基于真实数据

Phase 4（高级能力，2-3天）
  └── 1.3 根因分析增加变更关联
```

---

## 实施回顾 & 浏览器实测结果（2026-04-10）

> 以下内容基于代码审查 + 浏览器端到端实测（本机后端 localhost:8080 + Vite 5173），非理论判断。

### 已完成并验证通过

| 能力 | 接口/页面 | 浏览器实测 |
|------|-----------|-----------|
| 告警分组 DB 聚合 | `GET /alert-events/groups` → `GroupByFingerprint` SQL GROUP BY | 200，前端「收敛分组」tab 可见 |
| 时间轴 + 关联事件 + activity 审计 | `GET /alert-events/:id/timeline` | 200，弹窗展示 related_triggered / related_resolved / ack / resolved 节点 |
| 根因分析（动态推断 + 近期任务执行变更线索） | `GET /alert-events/:id/root-cause` + `/context` | 200，弹窗展示规则/主机/指标/置信度/关联工单 |
| Ack / Resolve 写 activity 审计 | `POST /alert-events/:id/ack` / `/resolve` | 写入 `alert_event_activities` 表，timeline 合并展示 |
| 自动恢复路径写 activity | 告警巡检 evaluateRule 内 | 代码确认，note = "指标自动恢复" |
| Golden Signals 基于 agent_metric_samples | `GET /monitor/golden-signals?minutes=60` | 200，卡片展示可用性/错误率/延迟/吞吐 |
| SLO 从 config.yaml 读取 | `slo.target_availability` / `slo_target_latency_ms` | 200，前端展示 SLO 目标值 |
| Golden Signals 维度切换 | `GET /monitor/golden-signals/dimensions?dimension=service` | 200，下拉切换服务/接口/实例会重发请求 |
| 巡检 SyncRecordStatus 代码 | `inspection_service.go` 约 133-219 行 | 函数存在，含完整报告拼装与告警联动 |
| 巡检「立即执行」 | `POST /inspection/plans/:id/run` | 200，执行记录立即出现 |
| 巡检「查看报告」 | `GET /inspection/records/:id/report` | 200，弹窗展示 JSON 报告 |

### 已有代码但存在差距（需修复）

| 问题 | 影响 | 修复建议 |
|------|------|----------|
| **巡检 SyncRecordStatus 未接线** | `main.go` 中 `OnExecutionComplete` 注册了回调，但如果 gRPC Agent 不在线，`checkExecutionCompletion` 不会触发 → 记录长期 `running` | 增加定时轮询兜底：`InspectionScheduler` 定期扫描 `running` 记录并主动 `SyncRecordStatus` |
| **告警分组 windowMinutes 未生效** | `ListEventGroups` 签名有 `windowMinutes` 参数，但函数体内未使用，前端传参无效 | 在 repo SQL 增加 `WHERE triggered_at >= NOW() - INTERVAL ? MINUTE` |
| **告警分组 first_triggered / duration 字段未填** | 前端表格里时间列始终空 | 服务层从 repo 的 `MIN/MAX(triggered_at)` 映射到 group 结构体 |
| **Golden Signals 延迟指标不准** | `avg_latency_ms` 实际用 CPU usage 均值代理，不是真实请求延迟 | 心跳增加 RTT 字段，或用新的 `latency` 类型指标 |
| **Golden Signals 服务维度 = IP** | 「服务维度」底层按 IP 分组，不是 ServiceTree 名称 | repo 层 JOIN `agent_infos.service_tree_id` → `service_trees.name` |
| **根因分析无 CI/CD 血缘** | 仅统计近期任务执行次数，无流水线部署事件 | 查 `cicd_pipeline_runs` 表关联到告警主机 |
| **activity 审计 Create 错误被忽略** | `activityRepo.Create` 返回的 err 被 `_` 丢弃 | 改为 warn 日志，不影响主流程 |
| **前端巡检报告只有 JSON 原文** | 用户需要从 `pre` 块自己看 `host_results` | 做表格化展示（主机/状态/exit_code/耗时/输出摘要） |
| **前端巡检执行记录无轮询** | 执行后需手动刷新才能看到终态 | 新建执行后启动 5s 轮询，直到状态非 `running` |

---

## P1+ 功能建议（后续可做的方向）

### 方向一：告警中心增强

#### 1.5 告警抑制（Suppression）
- 新增 `suppressed` 状态，当父告警（如主机宕机）存在时，自动抑制其子告警（如磁盘/CPU）
- 在 `evaluateRule` 触发前检查是否存在高优先级关联告警
- 价值：减少告警风暴，让值班人员聚焦真正的根因

#### 1.6 告警通知升级（Escalation）
- 告警触发后 N 分钟无人 ack → 自动升级到下一级 OnCall 值班人
- 结合已有的 `OnCallSchedule` 和 `NotifyGroup`，形成 L1→L2→L3 升级链
- 价值：避免告警被遗漏，关键告警必有人处理

#### 1.7 告警关联拓扑视图
- 前端增加「影响面」视图：以当前告警主机为中心，展示同 ServiceTree 下其它主机的健康状态
- 调用已有的 `monitor/aggregates/service-trees` 接口，叠加告警数据
- 价值：快速判断是单机问题还是集群性故障

#### 1.8 告警评论与协作
- `AlertEventActivity` 增加 `action` 字段（ack/resolve/comment/assign）
- 支持值班人员在告警上添加评论和处置记录
- 前端时间轴里展示评论节点
- 价值：告警处置过程可追溯，交接班有据可查

### 方向二：巡检系统深化

#### 2.4 巡检报告结构化展示
- 前端报告弹窗从 JSON 原文改为分区展示：摘要卡片（成功率/耗时/总计）+ 主机结果表格（status/exit_code/duration/stdout 截断）
- 失败主机标红置顶
- 价值：用户不需要看 JSON，一眼判断巡检结果

#### 2.5 巡检对比（Diff）
- 同一模板的两次执行结果 side-by-side 对比
- 标注新增失败主机、恢复主机、持续异常主机
- 价值：发现趋势恶化或改善

#### 2.6 巡检 SLA 统计
- 按模板统计近 7/30 天巡检成功率趋势（复用 `TemplateTrend` 接口增强）
- 前端「查看趋势」展示折线图 + SLA 达标线
- 价值：巡检质量可量化考核

#### 2.7 巡检结果自动修复（Remediation）
- 巡检失败时，如果模板配置了「修复任务 ID」，自动触发修复任务执行
- 修复后再次巡检验证
- 价值：从「发现问题」走向「自动修复”

### 方向三：Golden Signals & 监控深化

#### 3.4 SLO 前端配置入口
- 新增 SLO 管理页面（或在监控大盘头部增加「SLO 设置」按钮）
- 前端可修改 `target_availability` 和 `target_latency_ms`
- 后端增加 `POST /monitor/slo-config` 接口（写 config 或独立表）
- 价值：运维人员自助调整 SLO 目标

#### 3.5 指标异常检测
- 基于 `agent_metric_samples` 的历史数据，计算基线 + 标准差
- 当新样本偏离基线超过 N 倍标准差时，标记为「异常」
- 在监控大盘上用不同颜色标注异常 Agent
- 价值：从阈值告警升级到智能异常发现

#### 3.6 Agent 心跳 RTT 指标
- 心跳上报增加 `rtt_ms` 字段（Agent 发送到 gRPC Server 接收的时间差）
- Golden Signals 的「延迟」从 CPU 代理改为真实 RTT
- 价值：延迟指标有真实物理含义

#### 3.7 容量预测
- 对磁盘、内存使用率做线性回归预测
- 在监控大盘增加「预测耗尽时间」列
- 告警规则支持「预测 N 天后超阈值则提前告警」
- 价值：从被动运维到主动预防

### 方向四：跨模块联动

#### 4.1 告警 → 工单自动闭环
- 告警触发时自动创建工单（已有部分实现），工单关闭后自动 resolve 告警
- 工单处理过程中的评论同步到告警 timeline
- 价值：告警和工单双向联动，不再是孤岛

#### 4.2 巡检 → 告警 → 工单全链路
- 巡检失败 → 创建告警事件 → 自动建工单 → 指派责任人
- 全链路可追溯：从巡检记录能跳到告警事件，从告警事件能跳到工单
- 价值：问题发现到处理全闭环

#### 4.3 变更风险评估
- 执行任务或触发流水线前，基于历史数据评估风险等级
- 展示「上次同类变更后是否产生告警」「影响主机数」等
- 价值：变更前有依据，降低线上事故率

---

## 推荐执行顺序（P1+ 阶段）

```
Phase 5（修复差距，1天）
  ├── 巡检 SyncRecordStatus 接线兜底（定时轮询 running 记录）
  ├── 告警分组 windowMinutes 生效 + 时间字段回填
  └── activity 审计 error 日志化

Phase 6（体验提升，2天）
  ├── 2.4 巡检报告结构化展示（表格化主机结果）
  ├── 前端巡检执行后轮询终态
  └── 1.8 告警评论与协作

Phase 7（数据准确性，2天）
  ├── 3.6 Agent 心跳 RTT（Golden Signals 延迟真实化）
  ├── Golden Signals 服务维度 JOIN ServiceTree
  └── 根因分析关联 CI/CD 部署事件

Phase 8（高级能力，3-5天）
  ├── 1.5 告警抑制（Suppression）
  ├── 1.6 告警通知升级（Escalation）
  ├── 3.4 SLO 前端配置入口
  └── 3.5 指标异常检测

Phase 9（跨模块联动，3-5天）
  ├── 4.1 告警 → 工单自动闭环
  ├── 4.2 巡检 → 告警 → 工单全链路
  └── 2.7 巡检结果自动修复

Phase 10（前瞻性，按需）
  ├── 3.7 容量预测
  ├── 2.5 巡检对比（Diff）
  ├── 1.7 告警关联拓扑视图
  └── 4.3 变更风险评估
```
