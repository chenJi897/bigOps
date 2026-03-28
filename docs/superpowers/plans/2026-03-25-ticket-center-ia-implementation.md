# Ticket Center IA Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将“工单管理”重构为用户视角的信息架构，收敛为“发起工单 / 我的待办 / 我的申请 / 工单模板”四个入口，并移除不合理的二级菜单暴露。

**Architecture:** 复用现有 `TicketList.vue`、`TicketCreate.vue`、`TicketDetail.vue`、`RequestTemplates.vue` 能力，通过菜单 migration、动态路由复用和页面模式注入完成 IA 重构。底层 `ticket_types / request_templates / approval_policies` 保持不变，只调整用户入口、页面文案和模板展示结构。

**Tech Stack:** Vue 3、Vue Router、Pinia、Element Plus、Go、Gin、GORM、MySQL migration

---

## File Structure

### Create

- `backend/migrations/006_refactor_ticket_center_menus.sql`
- `docs/superpowers/specs/2026-03-25-ticket-center-ia-design.md`（已存在，仅引用）

### Modify

- `frontend/src/router/index.ts`
- `frontend/src/views/TicketList.vue`
- `frontend/src/views/TicketCreate.vue`
- `frontend/src/views/RequestTemplates.vue`
- `frontend/src/views/TicketTypes.vue`
- `frontend/src/views/Menus.vue`
- `frontend/src/views/Layout.vue`（仅在菜单显示或 activeMenu 需要配合时修改）
- `frontend/src/api/index.ts`（仅当模板页展示需要补充组合查询时修改）

### Optional Modify

- `frontend/src/stores/viewState.ts`（如新增模板页脏标记或页面模式缓存隔离需要）
- `backend/api/http/router/router.go`（仅在菜单迁移之外需要补额外接口时修改）

### Validation Targets

- `frontend/src/views/TicketDetail.vue`
- `frontend/src/views/ApprovalPolicies.vue`
- `backend/internal/handler/request_template_handler.go`
- `backend/internal/handler/approval_policy_handler.go`

---

### Task 1: 菜单迁移与导航收口

**Files:**
- Create: `backend/migrations/006_refactor_ticket_center_menus.sql`
- Modify: `frontend/src/views/Menus.vue`

- [ ] **Step 1: 写出菜单迁移 SQL**

目标：
- 保留父目录 `ticket_dir`
- 将二级菜单统一迁移为：
  - `ticket_launch` -> `/ticket/create` -> `TicketCreate`
  - `ticket_todo` -> `/ticket/todo` -> `TicketList`
  - `ticket_applied` -> `/ticket/applied` -> `TicketList`
  - `ticket_templates` -> `/ticket/templates` -> `RequestTemplates`
- 将旧菜单 `approval_inbox`、`ticket_types`、`request_templates`、`approval_policies` 设为隐藏或从角色菜单中移除
- 迁移必须是幂等的，并能修正旧环境数据

- [ ] **Step 2: 自查 SQL 是否覆盖旧菜单名**

检查：
- `ticket_list`
- `ticket_create`
- `ticket_detail`
- `approval_inbox`
- `ticket_types`
- `request_templates`
- `approval_policies`

Expected:
- 旧环境升级后不会出现重复入口

- [ ] **Step 3: 更新菜单管理页组件选项**

在 `frontend/src/views/Menus.vue` 中确认以下组件可选：
- `TicketList`
- `TicketCreate`
- `TicketDetail`
- `RequestTemplates`

Expected:
- 菜单管理页能维护新 IA 里的入口组件

- [ ] **Step 4: 手工审查字段收敛**

确认新菜单文案与设计一致：
- 发起工单
- 我的待办
- 我的申请
- 工单模板

- [ ] **Step 5: Commit**

```bash
git add backend/migrations/006_refactor_ticket_center_menus.sql frontend/src/views/Menus.vue
git commit -m "refactor: migrate ticket center menu IA"
```

---

### Task 2: 路由复用与页面模式注入

**Files:**
- Modify: `frontend/src/router/index.ts`

- [ ] **Step 1: 为 TicketList 设计双模式路由**

目标：
- `/ticket/todo`
- `/ticket/applied`

两者都复用 `TicketList.vue`，但通过 `meta` 注入模式：
- `ticketMode: 'todo'`
- `ticketMode: 'applied'`

- [ ] **Step 2: 保留 TicketDetail companion route**

确认：
- `TicketDetail` 继续作为隐藏 companion route
- `activeMenu` 应根据当前列表页回落

- [ ] **Step 3: 校正 TicketCreate 路由定位**

目标：
- `TicketCreate` 从“隐藏 companion route”升级为显式菜单页
- 但兼容原有 `/ticket/create` 使用方式

- [ ] **Step 4: 调整 RequestTemplates 路由语义**

目标：
- 让 `RequestTemplates` 对应主菜单“工单模板”
- 若需要，更新 title / activeMenu

- [ ] **Step 5: 运行前端构建检查路由类型**

Run:
```bash
cd /data/bigOps/frontend && npm run build
```

Expected:
- 构建通过，无路由类型错误

- [ ] **Step 6: Commit**

```bash
git add frontend/src/router/index.ts
git commit -m "refactor: align ticket center routes with new IA"
```

---

### Task 3: 重构 TicketList 为“我的待办 / 我的申请”双模式

**Files:**
- Modify: `frontend/src/views/TicketList.vue`

- [ ] **Step 1: 增加页面模式解析**

目标：
- 从 `route.path` 或 `route.meta.ticketMode` 解析当前模式
- 模式值：
  - `todo`
  - `applied`
  - 兼容旧列表入口时的默认模式

- [ ] **Step 2: 固定 scope 注入**

规则：
- `todo` -> `scope = my_assigned`
- `applied` -> `scope = my_created`

Expected:
- 这两种模式下不再依赖用户手动切 tabs

- [ ] **Step 3: 隐藏 scope tabs**

目标：
- 在 `todo` / `applied` 模式下不渲染当前 scope tabs 区块
- 仅保留筛选条件和列表

- [ ] **Step 4: 调整页面标题与按钮文案**

目标：
- 标题改为：
  - 我的待办
  - 我的申请
- 右上角按钮统一改为“发起工单”

- [ ] **Step 5: 验证 keep-alive 不串状态**

手工检查：
- 切换“我的待办”和“我的申请”后，筛选条件和分页不应错误串联

- [ ] **Step 6: 运行前端构建**

Run:
```bash
cd /data/bigOps/frontend && npm run build
```

Expected:
- 构建通过

- [ ] **Step 7: Commit**

```bash
git add frontend/src/views/TicketList.vue
git commit -m "refactor: split ticket list into todo and applied modes"
```

---

### Task 4: 发起工单页改为模板单入口

**Files:**
- Modify: `frontend/src/views/TicketCreate.vue`

- [ ] **Step 1: 移除工单类型卡片入口**

目标：
- Step 1 只展示模板卡片
- 不再渲染工单类型卡片列表

- [ ] **Step 2: 文案统一改为“发起工单”**

目标：
- 页面标题
- 按钮文案
- 成功提示
- 返回文案

- [ ] **Step 3: 保留底层映射**

要求：
- 模板选中后仍正确写入：
  - `request_template_id`
  - `ticket_kind`
  - `type_id`

- [ ] **Step 4: 验证动态表单仍可正常提交**

检查：
- `extra_fields.request_form`
- 资源选择
- 提交成功后跳详情

- [ ] **Step 5: 运行前端构建**

Run:
```bash
cd /data/bigOps/frontend && npm run build
```

Expected:
- 构建通过

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/TicketCreate.vue
git commit -m "refactor: make ticket launch template-driven"
```

---

### Task 5: 重构 RequestTemplates 为“工单模板”

**Files:**
- Modify: `frontend/src/views/RequestTemplates.vue`
- Modify: `frontend/src/api/index.ts`（仅当需要补组合查询）

- [ ] **Step 1: 调整页面标题与顶部按钮**

目标：
- 页面标题改为“工单模板”
- 顶部按钮保留：
  - 刷新
  - 新增工单模板
- 审批策略入口降级为页面内按钮，不作为左侧菜单

- [ ] **Step 2: 重构列表字段**

目标列：
- ID
- 模板名
- 所属类别
- 节点列表
- 启用
- 备注
- 更新时间
- 操作

移除列：
- 编码
- 排序
- 执行模板
- 单据类型
- 绑定类型

- [ ] **Step 3: 实现“节点列表”摘要**

目标：
- 根据绑定的 `approval_policy.stages` 生成摘要
- 样式示例：
  - `发起 => 一级审批 => 二级审批`
- 未绑定策略时展示：
  - `发起 => 处理`

- [ ] **Step 4: 增加启用开关**

目标：
- 模板列表里直接显示启用状态
- 若后端已有更新接口复用现有 update

- [ ] **Step 5: 保留现有模板编辑能力**

要求：
- 弹窗中仍支持：
  - 表单 schema
  - 分类
  - 绑定底层类型
  - 绑定审批策略

- [ ] **Step 6: 运行前端构建**

Run:
```bash
cd /data/bigOps/frontend && npm run build
```

Expected:
- 构建通过

- [ ] **Step 7: Commit**

```bash
git add frontend/src/views/RequestTemplates.vue frontend/src/api/index.ts
git commit -m "refactor: turn request templates into ticket templates page"
```

---

### Task 6: 收口旧入口与验证主流程

**Files:**
- Modify: `frontend/src/views/TicketTypes.vue`
- Modify: `frontend/src/views/Layout.vue`（如菜单渲染、activeMenu 或标签页需要配合）
- Optional Modify: `frontend/src/stores/viewState.ts`

- [ ] **Step 1: 从工单管理主流程移除旧入口跳转**

目标：
- `TicketTypes.vue` 不再承担主导航入口角色
- `ApprovalInbox.vue` 不再作为工单管理主入口

- [ ] **Step 2: 保留详情和通知审批入口**

检查：
- `TicketDetail.vue` 审批信息区仍正常
- 站内通知入口不受影响

- [ ] **Step 3: 验证 4 个新入口行为**

手工检查：
- 发起工单
- 我的待办
- 我的申请
- 工单模板

- [ ] **Step 4: 验证 admin 与普通用户**

检查：
- admin 菜单和页面正常
- 普通用户菜单和页面正常
- 普通用户不会看到已移除的旧二级菜单

- [ ] **Step 5: 最终构建验证**

Run:
```bash
cd /data/bigOps/backend && go build ./cmd/core
cd /data/bigOps/frontend && npm run build
```

Expected:
- 后端编译通过
- 前端构建通过

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/TicketTypes.vue frontend/src/views/Layout.vue frontend/src/stores/viewState.ts
git commit -m "refactor: finalize ticket center IA cleanup"
```

---

## Implementation Notes

### 路径与菜单命名建议

- 发起工单：`/ticket/create`
- 我的待办：`/ticket/todo`
- 我的申请：`/ticket/applied`
- 工单模板：`/ticket/templates`

### 风险控制

1. `TicketList.vue` 双模式复用时，注意 keep-alive 状态隔离。
2. `RequestTemplates.vue` 改造成模板页时，不要顺手重写整个编辑器。
3. 菜单 migration 必须覆盖旧环境，不能只插新记录。
4. `TicketDetail` 仍需保持从列表和通知跳转可用。

### Verification Checklist

- [ ] 左侧“工单管理”仅保留 4 个子菜单
- [ ] 发起工单首屏只有模板入口
- [ ] 我的待办没有 scope tabs，固定我处理的
- [ ] 我的申请没有 scope tabs，固定我创建的
- [ ] 工单模板列表不显示编码、排序
- [ ] 审批待办不再出现在左侧菜单
- [ ] 工单详情审批区正常
- [ ] `go build ./cmd/core` 通过
- [ ] `npm run build` 通过

