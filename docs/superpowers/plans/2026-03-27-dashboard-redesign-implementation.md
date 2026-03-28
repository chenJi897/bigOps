# 首页仪表盘改版 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把首页 Dashboard 改造成“个人工作台 + 按权限显示的平台总览”双层结构，并确保普通用户只看到自己有权限的内容。

**Architecture:** 后端新增 `dashboard/personal` 返回个人工作台摘要，前端 `Dashboard.vue` 负责根据 `permissionStore.menus` 做模块显隐。平台总览继续复用现有统计接口，监控摘要继续复用 `monitor/summary`。个人区与平台区在同一页面分层展示。

**Tech Stack:** Go、Gin、GORM、Vue 3、Pinia、Element Plus、Playwright、Chrome DevTools/CDP

---

### Task 1: 新增个人工作台后端摘要接口

**Files:**
- Modify: `backend/internal/handler/stats_handler.go`
- Modify: `backend/api/http/router/router.go`
- Test: `backend/internal/handler/stats_handler.go`（先用构建和 API 验证）

- [ ] **Step 1: 新增个人摘要响应结构**
- [ ] **Step 2: 在 handler 中根据当前登录用户统计：**
  - 我的待办工单
  - 我的申请工单
  - 我的相关告警
  - 我的任务执行
  - 我的流水线运行
  - 我负责的资产
- [ ] **Step 3: 注册路由**
  - `GET /api/v1/dashboard/personal`
- [ ] **Step 4: 运行编译**
  - `cd backend && go build ./cmd/core`

### Task 2: 首页前端信息架构重构

**Files:**
- Modify: `frontend/src/views/Dashboard.vue`
- Modify: `frontend/src/api/index.ts`
- Modify: `frontend/src/stores/permission.ts`（如需补工具函数）

- [ ] **Step 1: 新增 dashboardApi.personal()**
- [ ] **Step 2: Dashboard 改为同时加载：**
  - `statsApi.summary()`
  - `statsApi.assetDistribution()`
  - `monitorApi.summary()`
  - `dashboardApi.personal()`
- [ ] **Step 3: 新增模块权限 computed**
  - `canViewCMDB`
  - `canViewMonitor`
  - `canViewTicket`
  - `canViewTask`
  - `canViewCICD`
  - `canViewSystem`
- [ ] **Step 4: 页面布局改为：**
  - 欢迎区
  - 快捷入口
  - 个人工作台
  - 平台总览
  - 分布与排行
- [ ] **Step 5: 普通用户隐藏无权限模块**

### Task 3: 快捷入口与跳转整理

**Files:**
- Modify: `frontend/src/views/Dashboard.vue`

- [ ] **Step 1: 快捷入口按权限显示**
- [ ] **Step 2: 管理员显示更多平台级入口**
- [ ] **Step 3: 普通用户优先显示：**
  - 我的待办
  - 我的申请
  - 告警事件
  - 任务执行
  - 运行记录

### Task 4: 视觉与交互优化

**Files:**
- Modify: `frontend/src/views/Dashboard.vue`

- [ ] **Step 1: 首页标题文案按角色区分**
  - admin: 平台总览
  - 普通用户: 工作台
- [ ] **Step 2: 个人工作台卡片区增加状态强调**
- [ ] **Step 3: 平台总览卡片区降低 CRUD 感，提升信息密度与层次**
- [ ] **Step 4: 保留分布区，但只对有权限用户显示**

### Task 5: 权限与角色验收

**Files:**
- Verify only

- [ ] **Step 1: admin 登录验收**
  - 完整首页展示
- [ ] **Step 2: yunwei 登录验收**
  - 只显示有权限模块
  - 个人工作台为主
- [ ] **Step 3: 确认无权限模块不显示**

### Task 6: 浏览器验收

**Files:**
- Verify only

- [ ] **Step 1: `go build ./cmd/core`**
- [ ] **Step 2: `npm run build`**
- [ ] **Step 3: Playwright**
  - admin 首页
  - 普通用户首页
- [ ] **Step 4: Chrome DevTools / CDP**
  - 路由存在
  - 关键文本命中
  - `console error = 0`
  - `failed response = 0`
