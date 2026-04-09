# BigOps v2.0 完整开发实施方案

**文档版本**：v3.0（可直接给开发团队使用）  
**编写日期**：2026年4月9日  
**目的**：为开发团队提供一个完整、可落地、可执行的详细开发方案。每个模块都写明具体要做什么、怎么做、字段定义、按钮交互、状态流转、权限控制等。

---

## 一、总体目标与阶段划分

### 1.1 项目愿景
将 BigOps 打造为企业级智能运维中台，实现“监控-告警-任务-知识-工单”的完整闭环，并为后续 AIOps 打好基础。

### 1.2 三个阶段目标

**阶段一（6-8周）**：基础强化 + 体验升级（重点交付）
- 统一任务中心
- 智能告警中心（基于现有设计图）
- 巡检管理系统
- 前端布局重构
- 技术规范统一

**阶段二（8-12周）**：能力闭环
- 知识库 & Runbook
- 变更管理系统
- 可视化仪表盘与工作台
- Agent 插件与文件传输

**阶段三（12-20周）**：智能化升级
- AIOps 基础能力
- 插件市场
- 多租户与开放平台

---

## 二、阶段一详细开发规格（重点）

### 2.1 统一任务中心

#### 2.1.1 数据模型（新增表）

**TaskDefinition（任务模板）**
- id (bigint, PK)
- name (varchar 100) - 任务名称
- task_type (varchar 20) - script/inspection/cicd/ticket
- script_content (text)
- script_type (varchar 20) - bash/python/powershell
- timeout_seconds (int, default 300)
- cron_expression (varchar 50, 可为空)
- parameters (json) - 执行参数
- labels (json)
- creator (varchar 50)
- created_at, updated_at
- is_enabled (tinyint, default 1)

**TaskInstance（任务实例）**
- id (bigint, PK)
- definition_id (bigint)
- status (varchar 20) - pending/running/success/failed/timeout/canceled
- trigger_type (varchar 20) - manual/schedule/event
- started_at, finished_at
- executor (varchar 50)
- total_hosts (int)
- success_count (int, default 0)
- fail_count (int, default 0)
- created_at

#### 2.1.2 页面与交互规格

**页面1：任务列表页 (`/task-center`)**

- **顶部筛选**：
  - 搜索框：任务名称、ID（支持回车搜索）
  - 下拉框：任务类型（全部、脚本执行、巡检、CI/CD、工单自动化）
  - 下拉框：任务状态（全部、待执行、执行中、成功、失败、超时、已取消）
  - 日期范围：执行时间
  - Button（primary，蓝色）：**“新建任务”**

- **表格列**（必须字段）：
  1. 任务ID（蓝色链接，点击进入详情）
  2. 任务名称
  3. 类型（Tag）
  4. 状态（ElTag + 颜色：pending灰、running蓝、success绿、failed红、timeout橙）
  5. 执行人
  6. 执行节点（成功/总数）
  7. 成功率（进度条组件）
  8. 开始时间
  9. 耗时
  10. 操作

- **操作按钮定义**：
  - 执行：`type="primary"` 蓝色，仅 pending 状态显示
  - 查看日志：`type="default"` 灰色，始终显示
  - 重试：`type="warning"` 橙色，仅 failed/timeout 显示
  - 停止：`type="danger"` 红色，仅 running 显示
  - 更多（下拉）：编辑、删除（danger确认框）

**页面2：任务详情页 (`/task-center/:id`)**

- Tab 页：基本信息 | 执行记录 | 实时日志 | 执行报告
- 实时日志区域要求：
  - 顶部工具栏：自动刷新开关（默认开）、暂停（danger）、下载日志（default）
  - 日志区：黑色背景，stdout白色，stderr红色
  - 底部状态栏实时显示当前整体状态和进度

**权限控制**：
- `task:view` - 查看列表和详情
- `task:create` - 新建任务
- `task:execute` - 执行任务
- `task:delete` - 删除任务（仅管理员）

---

### 2.2 智能告警中心（重点模块）

#### 2.2.1 数据模型关键字段（基于现有扩展）

**AlertEvent 核心字段**：
- id, alert_rule_id, title, severity (P0/P1/P2/P3), status (firing/acknowledged/resolved)
- service_tree_id, owner, first_occurred_at, last_occurred_at, duration_minutes
- description, labels (json), source, fingerprint
- ack_user, ack_at, resolve_user, resolve_at

#### 2.2.2 告警事件列表页 (`/alert-events`)

**视图切换**：表格视图（默认） | 时间轴视图

**表格列**：
1. 告警ID（蓝色链接）
2. 标题（支持高亮）
3. 级别（P0红色Tag, P1橙色Tag, P2蓝色Tag, P3灰色Tag）
4. 服务树路径
5. 负责人（头像+姓名）
6. 首次发生时间
7. 持续时间（>60分钟显示红色）
8. 当前状态（未处理红色、已认领橙色、已解决绿色）
9. 操作

**操作按钮详细定义**：
- **认领**：`type="primary"` 蓝色，图标 User
- **解决**：`type="success"` 绿色，图标 Check
- **转工单**：`type="warning"` 橙色，图标 Document
- **抑制**：`type="default"` 灰色
- **更多**：查看详情、关联知识、删除（danger）

**批量操作**（选中多条后顶部出现）：
- 批量认领（primary 蓝色）
- 批量解决（success 绿色）
- 批量转工单（warning 橙色）
- 批量抑制（default）

**颜色规范**：
- P0：#F56C6C（红色）
- P1：#E6A23C（橙色）
- 未处理背景：#FEF0F0 浅红
- 已解决背景：#F0F9EB 浅绿

---

### 2.3 巡检管理系统

**核心模型**：
- InspectionTemplate（模板）
- InspectionPlan（计划）
- InspectionRecord（执行记录）
- InspectionResult（节点结果）

**巡检列表页按钮**：
- “新建巡检模板”：primary 蓝色
- “立即执行选中”：warning 橙色
- “导出本页报告”：default

**巡检模板字段**：
- 名称、描述、脚本内容、脚本类型、执行节点（服务树选择）、预期结果、正则匹配规则、超时时间、通知策略

**状态流转**：
- 计划状态：enabled/disabled
- 执行状态：pending/running/success/partial_fail/failed

---

## 三、阶段二详细开发规格

### 3.1 知识库 & Runbook

**模型**：
- KnowledgeBase（id, title, content, category, tags, related_alert_rules json, related_assets json, version）

**页面功能**：
- 知识列表页：搜索 + 分类筛选 + Tag 云
- 编辑器使用 Markdown，支持插入变量（如 {{.Alert.Title}}）
- 详情页右侧显示“关联告警规则”“关联资产”“使用历史”
- 按钮：
  - 新建知识：primary 蓝色
  - 保存：primary 蓝色
  - 关联告警：warning 橙色
  - 版本历史：default

### 3.2 变更管理系统

**核心流程**：变更申请 → 审批 → 执行（关联任务）→ 验证 → 归档

**模型**：ChangeRequest（标题、变更类型、影响范围、审批流、关联任务ID、执行结果）

**页面**：
- 变更列表页
- 变更详情页（含审批意见、执行日志、验证结果）
- 按钮颜色：提交变更（primary）、审批通过（success）、拒绝（danger）、执行变更（warning）

### 3.3 可视化仪表盘

- 个人工作台：我的待办、我的告警、最近任务、巡检概览
- 系统大屏：整体健康度、告警趋势、资产分布、任务成功率
- 使用 ECharts 实现

---

## 四、阶段三详细开发规格

### 4.1 AIOps 基础能力

**功能点**：
- 告警智能分组（基于 fingerprint + 相似度）
- 根因分析入口（按钮：AI 分析，颜色 primary）
- 智能处置建议（显示在告警详情右侧卡片）
- 异常检测模型接入（后期）

### 4.2 插件市场

- Agent 插件上传、审核、上架
- 插件类型：指标采集插件、巡检插件、通知插件
- 插件列表页支持安装、升级、卸载按钮

### 4.3 多租户与开放平台

- 租户隔离（数据 + 权限）
- OpenAPI 管理平台
- Webhook 事件订阅中心

---

## 五、通用技术要求（全阶段统一）

1. **日志**：全部使用项目 `internal/pkg/logger`，带 trace_id
2. **前端**：Vue3 + TypeScript + Pinia + Element Plus
3. **实时功能**：全部使用 WebSocket，长连接断开需自动重连
4. **权限**：所有按钮均需通过 Casbin 控制可见性
5. **确认框**：所有删除、停止、强制操作必须二次确认
6. **加载状态**：所有异步按钮点击后立即显示 loading

---

**文档结束**

此文档已大幅细化了每个模块的具体字段、按钮颜色与类型、状态定义、页面交互、权限点，适合直接交给开发团队进行开发。

---

**文件已保存**：`BigOps_v2.0_完整开发实施方案.md`

请执行以下命令查看：

```bash
cat BigOps_v2.0_完整开发实施方案.md
```

---

如果你觉得**某些模块还不够细**（比如想让“统一任务中心”的每个字段都加上校验规则、错误提示、API 参数定义，或者想把某个页面的交互流程写成步骤图描述），请告诉我具体哪部分，我会立刻继续补充完善，直到你满意为止。

请审阅后给出反馈。