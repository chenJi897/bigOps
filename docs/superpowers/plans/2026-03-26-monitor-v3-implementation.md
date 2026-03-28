# 监控中心 V2/V3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan.

**Goal:** 把监控中心从 V1 的“总览 + 规则 + 事件”推进到 V2/V3，完成对象钻取、告警处置、通知联动、自动化联动以及 Prometheus 数据源接入。

**Architecture:** 以现有 Agent 监控链为主线，新增对象详情与事件处置；并行引入 Prometheus 数据源和查询能力。通知、工单、任务执行全部复用已有模块。

**Tech Stack:** Go、Gin、GORM、Vue 3、Element Plus、Playwright、Chrome DevTools/CDP、Prometheus HTTP API

---

## Task 1: 修复 Agent 指标链稳定性

**Files**
- Modify: `backend/internal/agent/client.go`
- Modify: `backend/internal/grpc/server.go`
- Modify: `backend/internal/repository/agent_repository.go`

- [ ] 排查并修复多进程使用同一 `agent_id` 时的覆盖问题
- [ ] 确保当前版本 Agent 上报的 CPU / Memory / Disk 稳定落库
- [ ] 趋势查询默认返回最近时间段内的最新点，而不是只看到最早的旧 0 值

---

## Task 2: Agent 详情页

**Files**
- Create: `frontend/src/views/AgentDetail.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/api/index.ts`
- Modify: `backend/internal/service/monitor_service.go`
- Modify: `backend/internal/handler/monitor_handler.go`
- Modify: `backend/api/http/router/router.go`

- [ ] 新增 `GET /api/v1/monitor/agents/:agent_id`
- [ ] 新增 `GET /api/v1/monitor/agents/:agent_id/alerts`
- [ ] 新增 `GET /api/v1/monitor/agents/:agent_id/task-executions`
- [ ] 前端支持从监控大盘进入 Agent 详情

---

## Task 3: 告警事件中心增强

**Files**
- Create: `frontend/src/views/AlertEvents.vue`
- Modify: `frontend/src/views/AlertRules.vue`
- Modify: `frontend/src/api/index.ts`
- Modify: `frontend/src/router/index.ts`

- [ ] 独立告警事件中心页
- [ ] 支持按状态 / 级别 / 主机 / 规则 / 关键字筛选
- [ ] 支持批量 `ack / resolve`
- [ ] 支持跳转工单 / 任务执行

---

## Task 4: 告警联动工单与任务执行

**Files**
- Modify: `backend/internal/model/alert_event.go`
- Modify: `backend/internal/repository/alert_event_repository.go`
- Modify: `backend/internal/service/alert_service.go`
- Modify: `backend/internal/handler/alert_rule_handler.go`

- [ ] 为事件补齐关联字段：
  - `ticket_id`
  - `task_execution_id`
  - `service_tree_id`
  - `owner_id`
- [ ] 告警规则支持动作：
  - `notify_only`
  - `create_ticket`
  - `execute_task`
- [ ] 触发告警时自动创建工单或执行修复任务

---

## Task 5: 通知联动增强

**Files**
- Modify: `backend/internal/service/alert_service.go`
- Modify: `backend/internal/service/cicd_service.go`
- Modify: `frontend/src/views/NotificationConsole.vue`

- [x] 告警触发 / 恢复复用通知中心多渠道
- [x] CI/CD 部署成功 / 失败复用通知中心多渠道
- [x] 通知联调页补充为“通知配置中心”
- [x] 新增“我的通知设置”页，支持个人渠道和订阅业务类型
- [x] 告警通知改为事务提交后立即异步投递

---

## Task 6: Prometheus 数据源

**Files**
- Create: `backend/internal/model/monitor_datasource.go`
- Create: `backend/internal/repository/monitor_datasource_repository.go`
- Create: `backend/internal/service/monitor_datasource_service.go`
- Create: `backend/internal/service/prometheus_client.go`
- Create: `backend/internal/handler/monitor_datasource_handler.go`
- Modify: `backend/api/http/router/router.go`
- Modify: `backend/cmd/core/main.go`

- [ ] 数据源 CRUD
- [ ] 健康检查
- [ ] 支持 `prometheus` 类型

---

## Task 7: PromQL 查询台

**Files**
- Create: `frontend/src/views/MonitorDatasources.vue`
- Create: `frontend/src/views/MonitorQuery.vue`
- Modify: `frontend/src/api/index.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/views/Menus.vue`
- Modify: `backend/internal/handler/monitor_handler.go`
- Modify: `backend/internal/service/monitor_service.go`

- [ ] 新增查询接口：
  - `POST /api/v1/monitor/query`
  - `POST /api/v1/monitor/query-range`
- [ ] 前端支持：
  - 数据源选择
  - PromQL 输入
  - 时间范围
  - 图表结果

---

## Task 8: 验证

- [ ] `go build ./cmd/core`
- [ ] `npm run build`
- [ ] Playwright：
  - 监控大盘
  - Agent 详情
  - 告警事件中心
  - 数据源管理
  - PromQL 查询台
- [ ] Chrome DevTools / CDP：
  - 路由存在
  - 关键文本命中
  - `console error = 0`
  - `failed response = 0`
