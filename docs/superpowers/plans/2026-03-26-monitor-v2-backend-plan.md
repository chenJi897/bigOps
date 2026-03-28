# Monitor V2/V3 Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan.

**Goal:** 为监控中心 V2/V3 打通后端闭环——告警联动、通知复用、Prometheus 数据源、新查询接口，优先覆盖 backend/ 下模型、仓储、服务、handler。

**Architecture:** 保持现有 Agent 监控链，新增 alert_rule/action 与 alert_event 关联字段后直接走 TicketService/TaskService，所有通知统一交给 NotificationService，Prometheus 功能独立为 datasource + client 层并在 monitor handler 中暴露查询接口。

**Tech Stack:** Go、Gin、GORM、Prometheus HTTP API、NotificationService、Go 标准测试

---

### Task 1: Alert rule + event schema extensions

**Files:**
- Modify: `backend/internal/model/alert_rule.go`
- Modify: `backend/internal/model/alert_event.go`
- Modify: `backend/internal/repository/alert_rule_repository.go`
- Modify: `backend/internal/repository/alert_event_repository.go`
- Modify: `backend/internal/handler/alert_rule_handler.go`

- [ ] **Step 1: Write failing test**  
  Add `TestAlertRuleActionValidation` to `backend/internal/service/alert_service_test.go` that prepares a rule with `Action = "create_ticket"` and expects `validateAlertRule` to accept it.  

```go
func TestAlertRuleActionValidation(t *testing.T) {
  rule := &model.AlertRule{Action: "create_ticket", Name: "a", MetricType: "cpu_usage", Operator: "gt", Threshold: 90, Severity: "critical"}
  if err := validateAlertRule(rule); err != nil {
    t.Fatalf("expected valid action, got %v", err)
  }
}
```

- [ ] **Step 2: Run test to verify it fails**  
  `cd backend && go test ./internal/service -run TestAlertRuleActionValidation` → FAIL because `Action` field and validation are not yet implemented.

- [ ] **Step 3: Write minimal implementation**  
  - Add `Action string 'gorm:"size:32" json:"action"'` plus optional parameters `[TicketTemplateID?]` to `AlertRule`.  
  - Extend `AlertEvent` with `TicketID`, `TaskExecutionID`, `ServiceTreeID`, `OwnerID` columns (default 0).  
  - Update repositories to include new columns when creating/updating/listing.  
  - Enhance handler payload (create/update) to bind new action fields.  
  - Update `validateAlertRule` switch to allow `"notify_only"|"create_ticket"|"execute_task"` and ensure downstream required fields exist (e.g., `TargetTickets` or `TargetTaskID`).  
  - Example snippet:

```go
type AlertRule struct {
  ...
  Action   string `gorm:"size:32" json:"action"`
  TaskID   int64  `json:"task_id"`
  TicketID int64  `json:"ticket_id"`
}
```

- [ ] **Step 4: Run test to verify it passes**  
  `cd backend && go test ./internal/service -run TestAlertRuleActionValidation`

- [ ] **Step 5: Commit**  
  `git add backend/internal/model/alert_rule.go backend/internal/model/alert_event.go backend/internal/repository/alert_* backend/internal/handler/alert_rule_handler.go backend/internal/service/alert_service_test.go && git commit -m "feat(monitor): support alert rule actions"`

---

### Task 2: AlertService actions + Notification reuse

**Files:**
- Modify: `backend/internal/service/alert_service.go`
- Modify: `backend/internal/service/ticket_service.go` (if needed for helper)
- Modify: `backend/internal/service/task_service.go` (if needed)

- [ ] **Step 1: Write failing test**  
  Add `TestAlertServiceTriggerCreatesTicketOrTask` verifying `evaluateRule` assigns `TicketID` or `TaskExecutionID` to the saved `AlertEvent` when a rule has the corresponding action.  

```go
func TestAlertServiceTriggerCreatesTicketOrTask(t *testing.T) {
  rule := &model.AlertRule{Action: "create_ticket", NotifyUserIDs: "[]", ...}
  agent := &model.AgentInfo{AgentID: "test", CPUUsagePct: 99}
  svc := NewAlertService()
  _, err := svc.evaluateRule(agent, rule)
  if err != nil {
    t.Fatalf("unexpected %v", err)
  }
  event, _ := svc.eventRepo.FindOpenByRuleAgent(rule.ID, agent.AgentID)
  if event.TicketID == 0 {
    t.Fatalf("expected ticket created")
  }
}
```

- [ ] **Step 2: Run test to verify it fails**  
  `cd backend && go test ./internal/service -run TestAlertServiceTriggerCreatesTicketOrTask`

- [ ] **Step 3: Write minimal implementation**  
  - Inside `evaluateRule` when creating a new event, after `tx.Create(alertEvent)` call `svc.ticketService.Create` or `svc.taskService.ExecuteTask` depending on `rule.Action`.  
  - Save resulting IDs on `alertEvent` (new fields).  
  - Always invoke `NotificationService.PublishTx` once per business event; reuse existing `NotifyTx` logic for both alerts and CI/CD (see Task 3).  
  - Example snippet:

```go
if rule.Action == "create_ticket" {
  ticket, err := s.ticketService.CreateTicketForAlert(tx, rule, agent)
  alertEvent.TicketID = ticket.ID
}
```

- [ ] **Step 4: Run test to verify it passes**  
  `cd backend && go test ./internal/service -run TestAlertServiceTriggerCreatesTicketOrTask`

- [ ] **Step 5: Commit**  
  `git add backend/internal/service/alert_service.go backend/internal/service/ticket_service.go backend/internal/service/task_service.go && git commit -m "feat(monitor): hook actions into alert service"`

---

### Task 3: NotificationService reuse for CI/CD

**Files:**
- Modify: `backend/internal/service/cicd_service.go`

- [ ] **Step 1: Write failing test**  
  `TestCICDServiceNotification` in `backend/internal/service/cicd_service_test.go` that triggers a pipeline and asserts `NotificationService.PublishTx` was invoked with `biz_type="cicd_pipeline"`.  

```go
func TestCICDServiceNotification(t *testing.T) {
  svc := NewCICDService()
  svc.triggerNotificationForRun( ... )
  if !svc.notifySvc.WasCalledWith("pipeline_failed") { t.Fatal("missing notify") }
}
```

- [ ] **Step 2: Run test to verify it fails**
  `cd backend && go test ./internal/service -run TestCICDServiceNotification`

- [ ] **Step 3: Write minimal implementation**
  - Add private helper `triggerNotificationForRun` that reuses `NotificationService.PublishTx` and is called after run creation/update (success/failure).  
  - Replace existing ad-hoc log/alert messages with NotificationService events (e.g., `pipeline_succeeded`, `pipeline_failed`).  

```go
func (s *CICDService) notifyPipeline(run *model.CICDPipelineRun, level string) {
  s.notifySvc.PublishTx(tx, NotificationPublishRequest{ EventType: level, ... })
}
```

- [ ] **Step 4: Run test to verify it passes**
  `cd backend && go test ./internal/service -run TestCICDServiceNotification`

- [ ] **Step 5: Commit**
  `git add backend/internal/service/cicd_service.go backend/internal/service/cicd_service_test.go && git commit -m "chore(ci): reuse notification service"`

---

### Task 4: Prometheus datasource CRUD + health

**Files:**
- Create: `backend/internal/model/monitor_datasource.go`
- Create: `backend/internal/repository/monitor_datasource_repository.go`
- Create: `backend/internal/service/monitor_datasource_service.go`
- Create: `backend/internal/service/prometheus_client.go`
- Create: `backend/internal/handler/monitor_datasource_handler.go`
- Modify: `backend/api/http/router/router.go`
- Modify: `backend/cmd/core/main.go`
- Modify: `backend/migrations/` (add new migration script)

- [ ] **Step 1: Write failing test**
  Create `backend/internal/service/monitor_datasource_service_test.go` with `TestCreateDatasource` that POSTs a datasource and fails because handler not implemented.  

- [ ] **Step 2: Run test to verify it fails**
  `cd backend && go test ./internal/service -run TestCreateDatasource`

- [ ] **Step 3: Write minimal implementation**
  - Define `MonitorDatasource` GORM model with fields `Name`, `Type`, `BaseURL`, `AccessType`, `AuthType`, `Username`, `Password`, `HeadersJSON`, `Status`.  
  - Implement repository CRUD + health (e.g., `Ping` verifying `GET /api/v1/status`).  
  - `PrometheusClient` wraps HTTP client for `/api/v1/label/__name__/values` etc.  
  - Handler exposes endpoints for list/create/update/delete plus health-check.  
  - Register router group `/monitor/datasources` guarded by auth.  
  - Add SQL migration near `backend/migrations/010...?` to create table.  

- [ ] **Step 4: Run test to verify it passes**
  `cd backend && go test ./internal/service -run TestCreateDatasource`

- [ ] **Step 5: Commit**
  `git add backend/internal/model/monitor_datasource.go backend/internal/repository/monitor_datasource_repository.go backend/internal/service/monitor_datasource_*.go backend/internal/handler/monitor_datasource_handler.go backend/api/http/router/router.go backend/cmd/core/main.go backend/migrations/010_monitor_datasource.sql && git commit -m "feat(monitor): add prometheus datasource"`

---

### Task 5: Prometheus query endpoints

**Files:**
- Modify: `backend/internal/handler/monitor_handler.go`
- Modify: `backend/internal/service/monitor_service.go`
- Modify: `backend/internal/service/prometheus_client.go`
- Modify: `backend/api/http/router/router.go`

- [ ] **Step 1: Write failing test**
  `TestMonitorPromQLQuery` that POSTs a query string expecting an error because endpoint missing.  

- [ ] **Step 2: Run test to verify it fails**
  `cd backend && go test ./internal/service -run TestMonitorPromQLQuery`

- [ ] **Step 3: Write minimal implementation**
  - Add handler methods `Query` and `QueryRange` that decode PromQL request, forward to `PrometheusClient.Query` / `QueryRange`, and return JSON response or error.  
  - In `MonitorService`, add `QueryPrometheus(datasourceID int64, query string, ts time.Time)` that loads datasource, instantiates Prometheus client, calls `RangeQuery`.  
  - Add router entries: `POST /monitor/query`, `POST /monitor/query-range`.  

- [ ] **Step 4: Run test to verify it passes**
  `cd backend && go test ./internal/service -run TestMonitorPromQLQuery`

- [ ] **Step 5: Commit**
  `git add backend/internal/handler/monitor_handler.go backend/internal/service/monitor_service.go backend/internal/service/prometheus_client.go backend/api/http/router/router.go && git commit -m "feat(monitor): add prometheus query API"`

---

### Task 6: Build + sanity tests

- [ ] **Step 1: Run go build**  
  `cd backend && go build ./cmd/core`
  Expected: build succeeds without errors.

- [ ] **Step 2: Run targeted tests**  
  `cd backend && go test ./internal/service/...`
  Expected: all new unit tests pass.

- [ ] **Step 3: Commit (if additional adjustments were necessary)**  
  `git add . && git commit -m "chore(monitor): finalize backend V2/V3 bits"`

