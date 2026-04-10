# BigOps 项目进度日志

**最后更新**: 2026-04-10  
**当前阶段**: Phase 5 - 任务执行中心 **P0 已完成**，开始 **P1**

## 2026-04-10（P0 收尾）

### 已完成
- **WebSocket 日志 P0**：统一推送 JSON 契约（`WSExecutionLogEvent`），与前端字段对齐（`content`、`timestamp` Unix、`is_stderr`、`phase` 等）
- **断线重连**：执行处于 `pending`/`running` 时自动指数退避重连；重连请求带 `replay=1` 从数据库按行回放已落库输出（单机可带 `host_ip`）
- **首连去重**：首连使用 `replay=0`，依赖页面已拉取的详情 +实时推流；已结束任务打开「查看日志」时以落库快照为主，不强制建立 WS
- **后端**：`internal/handler/ws_execution_log.go`；`WSLogs` 在订阅后可选 `writeExecutionLogReplay`（上限 5000 行）
- **文档**：`task_plan.md` 同步 Phase 5.2 为 complete、5.3 标注 mostly_complete

### P1 建议方向
- 执行报告（聚合导出、PDF/Markdown 等）
- 执行历史高级筛选、批量操作
- 模板编辑器与参数表单（5.4）
- 统一 TaskInstance / TaskStep 与当前 TaskExecution 主链路的进一步产品化

---

## 历史（2026-04-09）

- Agent 日志与配置规范化、任务中心模型与页面、编译修复等（见 Git 历史与早期会话）

---

**备注**: 规划文件随里程碑更新；以仓库代码与 `task_plan.md` 为准。
