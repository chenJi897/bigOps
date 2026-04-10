# BigOps v2.0 开发总结与后续全量待办（详细版）

**更新时间**: 2026-04-10（P0 闭环修复 + P1 稳定化收口完成）  
**说明**: 本文档仅记录“真实已落地代码”和“后续待开发清单”，避免把方案稿当作已实现功能。

---

## 1) 当前真实开发进度（已完成）

### 1.1 任务中心基础能力（已落地）
- 数据模型已创建：
  - `TaskDefinition`
  - `TaskInstance`
  - `TaskStep`
- 前端页面已创建并可访问：
  - `frontend/src/views/TaskTemplates.vue`
  - `frontend/src/views/TaskExecutions.vue`
  - `frontend/src/views/TaskInstanceDetail.vue`
- 导航与路由已接通：
  - `Layout.vue` 已增加任务中心入口
  - 路由层已补齐任务中心页面注册
- 任务模板 CRUD 路径收敛：
  - `Create/Update/Delete/Get/List` 已统一走 `tasks` 兼容链路
  - 已消除 Create 与 Update/Delete 分别落不同表的分叉风险

### 1.2 Agent 技术债治理（已落地）
- 统一日志体系：Agent 侧核心文件从 `log.Printf` 迁移至项目 `zap` logger。
- 统一配置加载：Agent 侧改为走 `internal/pkg/config`，减少散落 `viper` 直连。
- 兼容修复：多处接口签名与仓储调用链修复，保证编译与运行一致。

### 1.3 任务执行实时日志链路（已落地）
- 后端已具备 WebSocket 日志通道能力：
  - `task_handler.go` `WSLogs`
  - `agent_manager` 订阅/发布
  - gRPC `ReportOutput` 转发日志
- 前端详情页已具备实时日志消费能力：
  - 连接状态展示
  - 自动滚动
  - 暂停/继续
  - 清空
  - 重连
  - 导出
  - 日志内容关键字过滤
  - Host 过滤
  - Phase 过滤
  - `stderr` 过滤
- 日志内容增强：
  - 行级展示 `host_ip`
  - 行级展示 `phase`
- 任务收尾增强：
  - 后端在执行完成后实时推送“执行统计日志”（`status/success/fail/duration`）
  - 后端新增进度事件字段：`done_count`、`total_count`、`success_count`、`fail_count`
  - 前端详情页已消费进度事件字段并即时更新进度/成功数

### 1.4 任务详情真实数据化（已落地）
- `TaskInstanceDetail.vue` 已从假数据切换为真实接口：
  - 页面加载即请求 `taskApi.getExecution(id)`
  - 摘要卡片使用真实状态和计数
  - 主机执行表使用真实 `host_results`
- 日志驱动刷新：
  - 收到 WS `finished/error` 事件后自动刷新执行详情

### 1.5 任务执行控制能力（已落地）
- 后端新增执行控制接口：
  - `POST /api/v1/task-executions/:id/cancel`（取消执行）
  - `POST /api/v1/task-executions/:id/retry`（重试失败主机）
- 取消执行行为：
  - 仅允许 `pending/running` 状态取消
  - 批量将未完成主机结果置为 `canceled`
  - 推送取消日志到 WS（`phase=finished`）
- 重试执行行为：
  - 从历史执行中提取 `failed/timeout/canceled` 主机重试
  - 创建新的执行记录并复用既有下发链路
  - 新增 `scope=failed/all` 范围控制（失败主机重试 / 全量重试）
- 前端执行页与详情页已接入取消/重试按钮（含“重试全部”）。
- 任务中心执行类接口错误码已规范化：
  - `404`：资源不存在（任务/执行）
  - `409`：状态冲突（执行中不可重试、状态不允许、任务已禁用）
  - `400`：参数或通用业务错误
- 审计细节增强：
  - 取消执行写入最终状态
  - 重试执行写入 `scope` 与 `new_execution_id`

### 1.6 执行列表真实化与实时刷新（已落地）
- `TaskExecutions.vue` 已从静态 mock 切换为真实接口 `taskApi.executions(...)`。
- 已支持本地搜索（执行 ID / 任务名）。
- 已接入每 5 秒自动刷新，保持执行状态和进度近实时更新。
- 列表操作已接入：
  - 详情跳转
  - 运行中执行可取消
  - 失败/部分失败/已取消执行可重试
- 列表统计卡片（成功/运行中/失败/总数）改为基于真实数据计算。
- 刷新策略优化：
  - 首次进入做全量拉取
  - 定时刷新改为仅刷新 `pending/running` 活跃执行，降低全量列表请求开销
  - 页面从后台切回前台（`visibilitychange`）会立即触发一次刷新，缩短状态滞后窗口
  - 增加刷新重入保护（避免并发重复请求），并在冲突场景做延迟补刷
  - 新增自动刷新控制：
    - 开关：开启/关闭自动刷新
    - 间隔：`5s / 10s / 30s`
    - 配置持久化到 `localStorage`，页面重进后保持上次选择
  - 新增刷新可观测性：
    - 页面显示“刷新中”状态提示
    - 页面显示“上次刷新时间”便于确认数据新鲜度

### 1.7 权限与联调治理（已落地）
- 已移除联调用的临时白名单放行。
- Casbin matcher 已调整，确保 `admin` 用户或 `admin` 角色拥有全权限。

### 1.8 编译状态（最新）
- 后端：`go build -o core ./cmd/core` 通过  
- 前端：`npm run build` 通过

### 1.9 详情页统计增强（已落地）
- 在执行详情“进度概览”中新增：
  - 成功率（%）
  - 失败率（%）
  - 平均耗时（ms，按主机结果计算）
- 统计值由真实接口字段动态计算，和执行状态保持一致。

### 1.10 P1 第一批交付（已落地）
- 智能告警中心：
  - 后端新增告警收敛分组接口：`GET /api/v1/alert-events/groups`（支持 `window_minutes`）
  - 后端新增告警时间轴接口：`GET /api/v1/alert-events/:id/timeline`
  - 前端 `AlertEvents.vue` 新增“事件视图 / 收敛分组”切换与时间轴抽屉
- Golden Signals：
  - 后端新增聚合接口：`GET /api/v1/monitor/golden-signals?minutes=60`
  - 前端 `MonitorDashboard.vue` 新增四大信号卡片（可用性/错误率/延迟/吞吐）
- 巡检系统：
  - 新增模型：`InspectionTemplate / InspectionPlan / InspectionRecord`
  - 新增接口：模板 CRUD、计划 CRUD、立即执行、执行记录查询
  - 新增页面：`frontend/src/views/InspectionCenter.vue`（模板、计划、记录三栏）
- 构建验证：
  - 后端 `go test ./internal/service ./internal/handler ./internal/repository` 通过
  - 前端 `npm run build` 通过

### 1.11 P1 第二批交付（已落地）
- 巡检计划 cron 调度器：
  - 新增 `backend/internal/service/inspection_scheduler.go`
  - 已在 `backend/cmd/core/main.go` 接入启动/停止生命周期
  - 支持读取启用计划并按 `cron_expr` 自动触发 `ExecutePlan`
- 告警根因分析面板：
  - 后端新增接口：`GET /api/v1/alert-events/:id/root-cause`
  - 前端 `AlertEvents.vue` 新增“根因”抽屉，展示置信度和证据链
- Golden Signals 时间窗口筛选：
  - 前端 `MonitorDashboard.vue` 新增 30/60/180 分钟窗口切换
  - 后端复用 `minutes` 参数做动态计算
- 验证结果：
  - 后端编译测试通过（含 `cmd/core`）
  - 前端构建通过

### 1.12 P0 闭环修复 + P1 稳定化收口（本轮新增，已落地）
- P0 闭环修复（任务中心）：
  - 修复侧边栏重复“任务中心”入口问题（`Layout.vue` 菜单去重）
  - 修复 `/task/templates` 空白与旧 mock 页面误导问题（路由映射改为真实任务管理页）
  - 修复任务中心直达页路由守卫绕过导致的动态路由未注入问题
  - 浏览器实测：任务创建、任务编辑、任务列表检索均可用
- P1 稳定化（全部完成）：
  - 补接口自动化测试（后端 service 层新增单测）：
    - `backend/internal/service/monitor_service_test.go`
    - `backend/internal/service/inspection_service_test.go`
  - 仪表盘维度命名映射（Golden Signals）：
    - 后端维度结果新增 `dimension_name`
    - 修复 `dimension=interface` 逻辑（不再错误落到 operator）
    - 前端维度表新增“维度名称”列，维度类型中文化显示
  - 巡检报告导出文件（JSON/CSV）：
    - 后端新增导出接口：`GET /api/v1/inspection/records/:id/report/export?format=json|csv`
    - 前端巡检中心新增按钮：`导出JSON` / `导出CSV`
    - 浏览器实测已成功下载：
      - `inspection-record-1.json`
      - `inspection-record-1.csv`
- 本轮验证：
  - 后端：`go test ./internal/handler ./internal/service ./api/http/router -v` 通过
  - 前端：`npm run build` 通过
  - Playwright：P0 + P1关键链路页面点击回归通过

---

## 2) 当前未完成功能总览（按优先级）

> P0 = 必须先做（核心主链路）；P1 = 关键业务能力；P2 = 增强能力；P3 = 开源竞争力与生态建设。

### 2.1 P0：任务中心主链路收口（必须先完成）
- [x] 任务模板 CRUD 与执行链路统一到同一数据模型（去兼容分叉）
- [x] 执行列表页实时刷新（状态/进度/成功失败数）
- [x] 执行详情页主机维度状态实时刷新（不仅日志，表格也实时）
- [x] 执行失败重试（按主机重试、全量重试）
- [x] 执行终止/取消能力（running -> canceled）
- [x] 幂等与并发控制（重复点击执行、防重入，前端刷新链路已做）
- [x] 执行审计链路闭环（谁在何时执行了什么）

### 2.2 P1：智能告警中心重构
- [x] 告警收敛算法（按规则、资源、时间窗口聚合，第一版）
- [x] 告警事件时间轴视图（触发/确认/恢复）
- [x] 告警根因分析第一版（规则关联 + 证据链展示，第一版）
- [x] 告警详情页增强（上下文、关联任务/工单、建议动作）
- [x] 告警静默策略和生效范围细化（规则/主机/服务树/负责人维度，第一版）

### 2.3 P1：巡检管理系统
- [x] 巡检模板管理（第一版）
- [x] 巡检计划管理（一次性/周期，第一版）
- [x] 巡检执行引擎（复用任务中心，第一版）
- [x] 巡检 cron 调度器（按计划自动触发，第一版）
- [x] 巡检报告输出（摘要、明细、趋势对比，第一版）
- [x] 巡检结果与告警联动（异常触发告警，第一版）

### 2.4 P1：SRE Golden Signals 仪表盘
- [x] 统一指标模型：可用性、错误率、延迟、吞吐（第一版）
- [x] 服务/接口/实例三级看板（第一版）
- [x] 时间范围筛选（第一版，按窗口分钟切换）
- [x] SLA/SLO 计算与超标提示（第一版）
- [x] 大盘组件化（可复用图表卡片，第一版）

### 2.5 P2：AIOps 能力建设
- [ ] 根因分析引擎 v2（多信号关联）
- [ ] 预测性告警（时序趋势 + 异常阈值）
- [ ] 智能建议生成（处置建议、回滚建议、检查建议）
- [ ] 告警去重与噪声抑制策略

### 2.6 P2：可视化工作流编排
- [ ] 节点模型定义（脚本、审批、等待、分支、并行）
- [ ] 画布设计与节点编辑器
- [ ] 工作流执行器（状态机）
- [ ] 工作流实例追踪与回放

### 2.7 P3：开源竞争力落地
- [ ] 最小可运行 Demo（1 键启动）
- [ ] 文档体系重构（安装、架构、开发、贡献）
- [ ] 示例项目与脚手架
- [ ] 插件化能力（告警插件、任务插件）
- [ ] 开放 API 与 SDK（先 Go/TS）
- [ ] 社区运营资产（Roadmap、RFC、Issue 模板）

---

## 2.8 P1 仍可继续增强的功能（建议优先）

> 说明：以下为在“P1 已基本完成”基础上的增量增强项，优先选择“低改造、高收益”的能力继续打磨。

### A. 任务表单类型约束与脚本编辑体验（高优先）
- [ ] `task_type` 与 `script_type` 前后端联动约束（禁止冲突组合）
- [ ] `bash/python` 默认模板与片段库（Snippets）
- [ ] 脚本提交前基础静态校验（空脚本、危险命令提示、类型冲突拦截）
- [ ] 编辑器组件化（`ScriptEditor`）并支持按语言切换高亮

### B. 执行报告产品化（高优先）
- [ ] 在执行详情增加“可读报告视图”（摘要卡 + 主机结果 + 错误 Top）
- [ ] 导出增强：除 JSON/CSV 外支持 Markdown（可选 PDF）
- [ ] 报告包含执行上下文（操作人、目标主机、开始/结束、耗时分布）

### C. 巡检计划运维能力（中高优先）
- [ ] 计划批量启停、批量删除
- [ ] 计划冲突检测（同模板同时间窗提示）
- [ ] 失败自动重试策略（按模板可配置）
- [ ] 最近 N 次执行健康度趋势与异常波动提示

### D. 告警事件工作流深化（中优先）
- [ ] 指派后的 SLA 计时与超时提醒
- [ ] 告警处理链路审计（指派、转派、确认、恢复）可追溯
- [ ] 静默到期自动恢复并通知
- [ ] 告警与任务/工单双向跳转及关联展示

### E. Golden Signals 可信度与可解释性（中优先）
- [ ] 指标卡增加“数据来源/计算口径”说明
- [ ] 维度筛选增强（服务/实例/时间窗 + 环比）
- [ ] 对代理计算指标标注可信度等级，逐步替换为真实来源

### F. P1 收官质量门禁（必须）
- [ ] 自动化回归：任务执行主链路（创建→执行→日志→重试）
- [ ] 自动化回归：巡检主链路（计划→执行→报告导出）
- [ ] 自动化回归：告警主链路（触发→分组→指派→详情）
- [ ] 文档中的“已完成”条目均附可复现验证命令与结果

---

## 3) 后续详细开发清单（可直接执行）

## 3.1 任务中心（P0）详细拆解

- **后端**
  - [x] 统一 `TaskDefinition` / `TaskExecution` / `TaskInstance` 的最终模型，移除临时 Compat 路径（以 `Task(tasks)` + `TaskExecution` 为主链路，`TaskDefinition/TaskInstance` 标注为预留且不参与双写；移除仓储 Compat 命名）
  - [x] `ExecuteTask` 增加状态流转规范：`pending -> running -> success/partial_fail/failed/canceled`（新增 `task_execution_fsm.go` 并在服务层/取消链路对齐）
  - [x] 增加取消执行接口：`POST /task-executions/:id/cancel`
  - [x] 增加重试接口：`POST /task-executions/:id/retry`
  - [x] 增加执行进度推送事件（done_count、total_count）
  - [x] 完善 host_result 持久化字段（开始/结束时间、耗时、错误摘要；running 写 started_at；finished/error 写 finished_at+duration_ms+error_summary；cancel 批量补齐 finished_at+duration_ms）

- **前端**
  - [x] `TaskExecutions.vue` 接入增量刷新（轮询或 WS）
  - [x] `TaskInstanceDetail.vue` 主机表行状态随 WS 实时更新
  - [x] 新增“取消执行”“重试失败主机”“重试全部”按钮
  - [x] 新增执行统计卡（成功率、失败率、平均耗时）
  - [x] 新增日志搜索关键字过滤（内容级）

- **验收标准**
  - [x] 用户从模板发起执行后，5 秒内可在列表看到 `running`（执行成功下发后即置 running；列表页定时刷新 active 执行）
  - [x] 执行结束后，列表和详情在 2 秒内一致（finished/error WS 事件触发详情刷新；列表页 active 增量刷新）
  - [x] 失败主机可单独重试且有独立审计记录（`retry` 支持 body 传 `host_ips`；审计 detail 记录 host_ips + new_execution_id；前端详情页提供“重试本机”按钮）

## 3.2 智能告警中心（P1）详细拆解

- **后端**
  - [ ] 告警收敛服务：支持时间窗口 + 主体键聚合
  - [ ] 根因分析服务：输入告警事件，输出关联资源链路
  - [ ] 告警状态机完善：`firing/ack/resolved/suppressed`
  - [ ] 告警时间轴查询接口

- **前端**
  - [ ] 告警列表支持收敛组展开
  - [ ] 告警详情新增时间轴组件
  - [ ] 根因面板（展示推断结果与置信度）
  - [ ] 批量操作（确认、恢复、静默）

- **验收标准**
  - [ ] 同类抖动告警收敛率可观测
  - [ ] 详情页可完整追踪事件生命周期

## 3.3 巡检系统（P1）详细拆解

- **后端**
  - [ ] 巡检模板 CRUD
  - [ ] 巡检计划调度器（cron）
  - [ ] 巡检任务执行器复用任务中心引擎
  - [ ] 巡检报告生成服务（JSON + 可导出）

- **前端**
  - [ ] 巡检模板页
  - [ ] 巡检计划页
  - [ ] 巡检执行记录页
  - [ ] 巡检报告详情页

- **验收标准**
  - [ ] 能创建周期巡检并自动触发
  - [ ] 报告可查看历史趋势

## 3.4 Golden Signals（P1）详细拆解

- **后端**
  - [ ] 指标聚合接口（服务级/接口级/实例级）
  - [ ] SLA/SLO 计算接口
  - [ ] 指标缓存优化（Redis）

- **前端**
  - [ ] Golden Signals 大盘页面
  - [ ] 四大信号图表组件
  - [ ] 服务筛选与时间范围控件

- **验收标准**
  - [ ] 页面 3 秒内可完成首屏渲染
  - [ ] 任意时间窗口可正确计算四大信号

## 3.5 AIOps（P2）详细拆解

- **后端**
  - [ ] 预测告警策略服务
  - [ ] 建议生成服务（规则模板 + 历史案例）
  - [ ] 根因引擎 v2（多源数据融合）

- **前端**
  - [ ] AIOps 分析页
  - [ ] 告警建议卡片
  - [ ] 置信度与解释信息展示

- **验收标准**
  - [ ] 预测结果可回溯到输入数据
  - [ ] 建议内容可关联执行动作

## 3.6 开源竞争力（P3）详细拆解

- **工程与文档**
  - [ ] `docker-compose` 一键启动
  - [ ] README 重写（5 分钟跑通）
  - [ ] 架构图与模块图
  - [ ] 贡献指南与代码规范

- **生态与产品**
  - [ ] 插件机制 MVP
  - [ ] SDK（Go/TS）最小版本
  - [ ] Demo 场景脚本（任务、告警、巡检）

- **验收标准**
  - [ ] 新用户 30 分钟内完成本地体验
  - [ ] 社区 PR 可按模板顺畅贡献

---

## 4) 推荐执行顺序（里程碑）

### M1（当前进行中）
- 完成任务中心主链路收口（P0）
- 目标：任务执行“可用、可看、可追溯、可重试、可取消”

### M2
- 完成告警中心 + 巡检系统第一版（P1）
- 目标：具备稳定运营闭环（发现 -> 处置 -> 复盘）

### M3
- 完成 Golden Signals + AIOps 第一版（P1 + P2）
- 目标：从“被动运维”升级到“可预测运维”

### M4
- 完成开源竞争力工程化落地（P3）
- 目标：可快速安装、可对外演示、可社区协作

---

## 5) 风险与依赖（必须提前准备）

- [ ] 数据模型收敛风险：`TaskDefinition` 与旧 `Task` 并存会持续拖慢开发
- [ ] 接口契约风险：前后端字段命名不统一会导致频繁兼容代码
- [ ] 权限策略风险：新模块接口持续增加，需同步 Casbin 策略治理
- [ ] 实时链路风险：WS 推送与数据库最终一致性可能出现短时偏差
- [ ] 观测风险：缺少链路追踪与埋点会影响故障定位效率

---

## 6) 下一步立即执行项（下一个开发循环）

- [x] 增加一组端到端自动化回归用例（创建/执行/查看/重试/取消）
- [x] P1 第一版进入稳定化：补接口自动化测试 + 仪表盘维度命名映射 + 巡检报告导出文件

---

## 8) 已提供的最小回归脚本

- 脚本路径：`backend/scripts/task_center_smoke.sh`
- 覆盖流程：
  - 登录
  - 创建任务
  - 执行任务
  - 查询执行详情
  - 按状态自动取消（可开关）
  - 按状态自动重试（可开关）
- 运行示例：
  - `BASE_URL="http://127.0.0.1:8080" USERNAME="e2e_admin" PASSWORD="Admin123" HOST_IPS="127.0.0.1" ./backend/scripts/task_center_smoke.sh`
- 可选参数：
  - `MODE=quick`：仅执行 登录/创建/执行/详情，不做取消和重试
  - `MODE=full`：完整执行流程（默认）
  - `DO_CANCEL=0` 跳过取消步骤
  - `DO_RETRY=0` 跳过重试步骤
  - `RETRY_SCOPE=failed|all` 控制重试范围
  - `LOGIN_PATH` 自定义登录路径（默认 `/auth/login`）
  - `AUTH_HEADER_PREFIX` 自定义认证头前缀（默认 `Bearer`）
- 增强能力：
  - 脚本输出增加步骤级失败定位：`[FAILED][step] ...`，可快速判断失败点
- 当前环境实跑结果（本次）：
  - 使用 `admin` 账号实跑：
    - `MODE=full DO_RETRY=0` 已跑通（登录/创建/执行/详情）
    - 重启后端到最新代码后，`MODE=full RETRY_SCOPE=all` 已完整跑通（含“重试全部”）

---

## 7) 当前结论

- 已完成：P0 主链路闭环（含任务中心可用性修复）+ P1 第一版与稳定化收口（告警、巡检、Golden Signals、导出能力、自动化测试补齐）。
- 未完成：P2/P3 方向（AIOps 深化、可视化工作流、开源工程化）仍待推进。
- 当前建议：进入 P2（AIOps v2）并保持“每完成一组功能即浏览器回归 + 文档同步”节奏。

