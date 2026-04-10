# BigOps 运维平台 - 项目任务计划（更新版）

**项目**: BigOps 大运维平台 v2.0  
**启动日期**: 2026-04-09（本轮重点开发）  
**当前状态**: Phase 5（任务执行中心）P0 已收尾，进入 P1  
**最后更新**: 2026-04-10  

---

## Phase 5: Module 05 - 任务执行中心 `in_progress`

### 5.1 任务中心基础架构 `complete`
- [x] TaskDefinition, TaskInstance, TaskStep 模型（支持版本、变量、步骤编排）
- [x] TaskRepository + TaskInstanceRepository + TaskStepRepository
- [x] TaskService（创建、执行、列表、统计）
- [x] TaskHandler（CRUD、执行、列表、WebSocket日志）
- [x] 前端页面（TaskList / TaskExecutions / TaskInstanceDetail 等）
- [x] 任务中心侧栏菜单（含「执行记录」启动种子 `bootstrap.EnsureTaskExecutionsMenu`）
- [x] 路由与动态菜单集成

### 5.2 WebSocket实时日志 `complete`（P0 已交付）
- [x] WSLogs Handler（Upgrade、读写泵、心跳）
- [x] 日志格式标准化：统一 JSON 事件 `WSExecutionLogEvent`（`execution_id`, `timestamp`, `content`, `is_stderr`, `phase`, `host_ip`, 进度字段等）
- [x] 与执行 ID 绑定：`SubscribeLogs(executionID)` + gRPC `PublishLog` 推流
- [x] 前端日志抽屉：时间戳解析、按主机过滤、阶段筛选（含 `replay` / `captured`）、自动滚动
- [x] 断线重连：指数退避（执行仍为 pending/running 时）；手动「重连」对运行中任务使用 `replay=1` 从 DB 回放已落库输出
- [x] 查询参数：`replay=1` 启用回放，`host_ip` 限定单机；默认 `replay=0` 避免重复回放

### 5.3 任务执行引擎 `mostly_complete`
- [x] Agent 对接（gRPC 心跳 + 任务下发 + ReportOutput 流）
- [x] 执行结果异步入库（主机 stdout/stderr 追加、终态与耗时）
- [x] 执行状态机（`task_execution_fsm.go`：pending/running/终态）
- [x] 超时控制与进程组清理（Agent executor）
- [ ] 执行报告生成（导出/邮件/归档）→ **划入 P1/P2**

### 5.4 前端交互优化 `not_started`
- [ ] 任务模板编辑器（Monaco）
- [ ] 执行参数表单动态生成
- [ ] 执行历史高级筛选与批量导出
- [ ] 实时日志终端主题/搜索高亮增强

---

## 当前优先级

1. ~~**P0**: 任务中心闭环（执行记录、详情、重试、WS 日志契约、回放与重连、菜单种子）~~ **已完成**
2. **P1**: 执行报告与可观测性增强、历史筛选/导出、与统一实例模型的进一步对齐
3. **P2**: 登录态与路由体验优化
4. **P3**: 智能告警中心深化

---

## 本轮开发记录（2026-04-10）

- P0 收尾：WebSocket 统一事件体、可选 DB 回放、前端重连与单机过滤、已结束任务仅展示落库日志（不强制长连 WS）
- 后端：`internal/handler/ws_execution_log.go`，`WSLogs` 支持 `replay`、`host_ip`
- 前端：`TaskInstanceDetail.vue` 重连与 `resolveWsURL` 查询参数

---

**更新原则**：每次里程碑后更新本文件并落代码，不再仅做描述。
