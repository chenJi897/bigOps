# CI/CD 最终版 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把当前 CI/CD 第一批骨架推进成可用闭环，完成项目管理、流水线定义、运行记录、运行详情、审批联动和部署联动。

**Architecture:** 复用现有任务中心负责实际部署执行，复用现有工单/审批负责发布审批，CI/CD 自己只负责项目、流水线、运行记录与状态编排。所有发布过程以 `cicd_pipeline_runs` 为主线，向外关联审批工单和任务执行记录。

**Tech Stack:** Go、Gin、GORM、MySQL、Vue 3、Element Plus、Vue Router、Playwright、Chrome DevTools/CDP

---

## File Structure

### 已有文件继续扩展

- `backend/internal/model/cicd_project.go`
- `backend/internal/model/cicd_pipeline.go`
- `backend/internal/model/cicd_pipeline_run.go`
- `backend/internal/repository/cicd_project_repository.go`
- `backend/internal/repository/cicd_pipeline_repository.go`
- `backend/internal/repository/cicd_pipeline_run_repository.go`
- `backend/internal/service/cicd_service.go`
- `backend/internal/service/approval_service.go`
- `backend/internal/handler/cicd_handler.go`
- `backend/api/http/router/router.go`
- `backend/cmd/core/main.go`
- `backend/migrations/009_seed_cicd_menus.sql`
- `frontend/src/api/index.ts`
- `frontend/src/router/index.ts`
- `frontend/src/views/Menus.vue`
- `frontend/src/views/CicdProjects.vue`
- `frontend/src/views/CicdPipelines.vue`
- `frontend/src/views/CicdRuns.vue`

### 建议新增

- `frontend/src/views/CicdRunDetail.vue`

---

## Task 1: 收口当前第一批骨架

**Files:**
- Modify: `backend/internal/handler/cicd_handler.go`
- Modify: `backend/internal/service/cicd_service.go`
- Modify: `frontend/src/views/CicdProjects.vue`
- Modify: `frontend/src/views/CicdPipelines.vue`
- Modify: `frontend/src/views/CicdRuns.vue`

- [ ] 对齐前后端字段命名：
  - 项目页统一 `repository/default_branch/active`
  - 流水线页统一 `active/status`
  - 运行记录页统一 `triggered_by_name`

- [ ] 确保以下页面都能正常打开且无 console error：
  - `/cicd/projects`
  - `/cicd/pipelines`
  - `/cicd/runs`

- [ ] 确保以下接口都返回稳定结构：
  - `GET /cicd/projects`
  - `GET /cicd/pipelines`
  - `GET /cicd/runs`

- [ ] 运行：
  - `go build ./cmd/core`
  - `npm run build`

---

## Task 2: 流水线列表补齐最近运行信息

**Files:**
- Modify: `backend/internal/service/cicd_service.go`
- Modify: `backend/internal/handler/cicd_handler.go`
- Modify: `frontend/src/views/CicdPipelines.vue`

- [ ] `GET /cicd/pipelines` 返回：
  - `latest_run.id`
  - `latest_run.status`
  - `latest_run.run_number`
  - `latest_run.summary`
  - `latest_run.created_at`

- [ ] 前端流水线页显示：
  - 最近运行状态 tag
  - 最近运行摘要
  - 跳转运行记录入口

- [ ] Playwright 验证：
  - 进入 `/cicd/pipelines`
  - 可见“最近运行”列
  - 触发一次流水线后页面能看到最新运行状态

---

## Task 3: 新增运行详情页

**Files:**
- Create: `frontend/src/views/CicdRunDetail.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/views/Menus.vue`
- Modify: `frontend/src/api/index.ts`
- Modify: `backend/internal/handler/cicd_handler.go`
- Modify: `backend/internal/service/cicd_service.go`

- [ ] 新增页面：
  - `CicdRunDetail.vue`

- [ ] 新增前端入口：
  - 从 `CicdRuns.vue` 行点击或按钮进入详情

- [ ] 后端 `GET /cicd/runs/:id` 返回结构至少包含：
  - `run`
  - `task_execution`
  - `target_hosts_list`
  - `variables`
  - `metadata`
  - `artifact_summary_map`

- [ ] 页面展示：
  - 基本运行信息
  - 审批工单链接
  - 任务执行链接
  - 状态摘要
  - 错误信息

- [ ] Playwright 验证：
  - 进入运行记录页
  - 打开详情
  - 可见运行状态与摘要

---

## Task 4: 审批通过后自动触发部署任务

**Files:**
- Modify: `backend/internal/service/approval_service.go`
- Modify: `backend/internal/service/cicd_service.go`
- Modify: `backend/internal/repository/cicd_pipeline_run_repository.go`

- [ ] 新增运行记录回查能力：
  - 按 `approval_ticket_id + status=waiting_approval` 查询运行记录

- [ ] 在审批服务最终 `approval_approved` 分支中：
  - 调用 `CICDService.StartApprovedRunsByTicketID(ticketID)`

- [ ] `StartApprovedRunsByTicketID` 逻辑：
  - 查找等待审批的 `PipelineRun`
  - 找到对应 `Pipeline`
  - 若配置 `deploy_task_id + target_hosts`
    - 调用 `TaskService.ExecuteTask`
    - 回写 `task_execution_id`
    - `run.status = running`
  - 若未配置部署任务
    - `run.status = success`
    - `summary = 审批已通过，未配置部署任务或目标主机`
  - 若执行失败
    - `run.status = failed`
    - `error_message`

- [ ] API 验证：
  - 创建带审批模板的流水线
  - 触发流水线
  - 一级审批通过
  - 二级审批通过
  - 查询 `GET /cicd/runs/:id`
  - 断言 `task_execution_id > 0` 或状态正确回写

---

## Task 5: 任务执行状态回写到流水线运行

**Files:**
- Modify: `backend/internal/service/cicd_service.go`
- Modify: `backend/internal/repository/cicd_pipeline_run_repository.go`
- Modify: `frontend/src/views/CicdRuns.vue`
- Modify: `frontend/src/views/CicdRunDetail.vue`

- [ ] 增加按 `task_execution_id` 查询运行记录能力

- [ ] 设计状态映射：
  - `TaskExecution.running -> PipelineRun.running`
  - `TaskExecution.success -> PipelineRun.success`
  - `TaskExecution.failed/partial_fail -> PipelineRun.failed`

- [ ] 查询运行列表和运行详情时，动态补齐或同步：
  - `status`
  - `result`
  - `summary`
  - `duration_seconds`

- [ ] 前端展示：
  - 运行详情页显示任务执行状态
  - 失败时显示错误摘要

---

## Task 6: 测试方案

**Files:**
- Reuse existing browser scripts under `/tmp`
- Create if needed: `frontend/scripts/playwright_cicd_full_flow.cjs`
- Create if needed: `frontend/scripts/chrome_devtools_cicd_verify.cjs`

- [ ] **接口测试**
  - 创建项目
  - 创建流水线
  - 触发无审批流水线
  - 触发有审批流水线
  - 查询运行列表
  - 查询运行详情
  - 审批通过后自动部署

- [ ] **Playwright 测试**
  - admin 登录
  - 创建项目
  - 创建流水线
  - 触发流水线
  - 打开运行记录
  - 打开运行详情
  - 审批通过后再次查看运行详情

- [ ] **Chrome DevTools/CDP 测试**
  - 登录页 DOM 正常
  - CI/CD 路由存在
  - `/cicd/projects` 文本命中
  - `/cicd/pipelines` 文本命中
  - `/cicd/runs` 文本命中
  - 无 console error
  - 无 failed response

- [ ] **角色测试**
  - `admin`：完整操作
  - `ops`：页面可见与查看

---

## Task 7: 收尾与验收

**Files:**
- Modify: `progress.md`
- Modify: `findings.md`
- Modify: `task_plan.md`

- [ ] 运行：
  - `go build ./cmd/core`
  - `npm run build`

- [ ] 浏览器验收：
  - Playwright 通过
  - Chrome DevTools 通过

- [ ] 记录：
  - 更新 `progress.md`
  - 更新 `findings.md`
  - 更新 `task_plan.md`

- [ ] 若有测试账号或测试模板：
  - 明确保留还是清理
