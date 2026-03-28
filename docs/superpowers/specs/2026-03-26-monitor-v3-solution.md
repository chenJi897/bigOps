# 监控中心 V2/V3 方案

**日期**: 2026-03-26  
**状态**: 可执行设计  
**目标模块**: Module 05 监控中心 V2/V3

---

## 一、目标

把当前仅有 `Agent 总览 + 告警规则 + 告警事件` 的监控模块推进到真正可处理、可联动、可扩展的运维监控中心。

目标拆成两级：

### V2

1. 提供 `Agent 详情页`
2. 提供 `告警事件中心`
3. 支持 `告警 -> 工单 / 任务执行` 跳转与联动
4. 补齐通知渠道复用：站内信、Webhook、Message Pusher
5. 按服务树 / 负责人聚合主机监控

### V3

1. 支持 `Prometheus 数据源` 接入
2. 支持 `PromQL 查询 / 范围查询`
3. 支持 `告警自动建单`
4. 支持 `告警自动执行修复任务`
5. 监控中心形成 `看见问题 -> 处理问题 -> 留痕追踪` 的闭环

---

## 二、设计原则

### 2.1 双数据源，不推翻现有 Agent 路线

监控中心不只支持 Agent 上报，也不能放弃当前已经做好的 Agent 链路。

统一做成两条来源：

- `agent`：主机资源数据、任务执行上下文、轻量 CMDB 联动
- `prometheus`：应用 / 中间件 / K8s / Exporter 数据

### 2.2 监控不是大盘，是处置中心

监控中心必须能落到：

- 告警事件
- 工单
- 任务执行
- 服务树 / 负责人

不做“只有图，没有动作”的页面。

### 2.3 不重复造轮子

- 通知复用 `NotificationService`
- 自动修复复用 `TaskService`
- 自动建单复用 `TicketService`
- 资源归属复用 `Asset / ServiceTree / Department`

---

## 三、V2 方案

### 3.1 Agent 详情页

新增 `AgentDetail.vue`，承载：

- 基本信息：主机名、IP、版本、系统、最后心跳
- 当前资源：CPU / 内存 / 磁盘 / 最近采样时间
- 最近 3 小时趋势
- 最近告警事件
- 最近任务执行
- 归属信息：
  - 资产
  - 服务树
  - 负责人

### 3.2 告警事件中心

把现有 `AlertRules.vue` 下半区升级成真正的事件台：

- 筛选：
  - 状态
  - 级别
  - 规则
  - 主机 / 关键字
  - 服务树
  - 负责人
- 操作：
  - 确认 `ack`
  - 关闭 `resolve`
  - 批量确认
  - 批量关闭
  - 创建工单
  - 执行修复任务

### 3.3 告警通知增强

已有通知中心能力直接复用到告警和 CI/CD：

- `alert_firing`
- `alert_resolved`
- `pipeline_succeeded`
- `pipeline_failed`
- `execution_failed`

渠道：

- `in_app`
- `webhook`
- `message_pusher`
- `email`

通知配置分三层：

- 管理员全局渠道配置：`NotificationConsole`
- 业务对象选择渠道：
  - 告警规则 `notify_channels`
  - CI/CD 流水线 `notify_channels`
  - 工单模板 `notify_channels`
- 用户个人通知偏好：`MyNotificationSettings`

这意味着：

- 告警可以选择发到哪些渠道
- 渠道连接参数由管理员统一维护
- 用户可自行关闭某些业务类型或个人渠道

### 3.4 聚合维度

监控中心 V2 必须支持：

- 按 `服务树` 聚合
- 按 `负责人` 聚合
- 按 `在线/离线` 聚合

这样页面不再只是“主机表格”，而是“业务视角 + 运维视角”双入口。

---

## 四、V3 方案

### 4.1 Prometheus 数据源

新增数据源模型：

- `monitor_datasources`

字段建议：

- `name`
- `type`：当前先支持 `prometheus`
- `base_url`
- `access_type`
- `auth_type`
- `username`
- `password`
- `headers_json`
- `status`

### 4.2 查询能力

新增查询接口：

- `GET /api/v1/monitor/datasources`
- `POST /api/v1/monitor/datasources`
- `POST /api/v1/monitor/datasources/:id`
- `POST /api/v1/monitor/datasources/:id/delete`
- `POST /api/v1/monitor/query`
- `POST /api/v1/monitor/query-range`

首期只做：

- 即时查询
- 范围查询
- 基础错误透传

不做复杂查询编排器。

### 4.3 Prometheus 与 Agent 融合展示

在监控中心做双入口：

- `主机监控`
- `Prometheus 监控`

Prometheus 数据先服务于：

- 应用指标
- Exporter 指标
- K8s 指标

页面能力：

- 预置查询卡片
- 查询结果图表
- 将某条查询保存到监控面板

### 4.4 告警自动化闭环

新增规则动作：

- `create_ticket`
- `execute_task`
- `notify_only`

典型流程：

```text
触发告警
-> 生成 AlertEvent
-> 发送通知
-> 如果规则配置了 create_ticket
   -> 创建工单
-> 如果规则配置了 execute_task
   -> 执行修复任务
-> 回写 event note / execution_id / ticket_id
```

---

## 五、页面结构

### 5.1 菜单结构

`监控中心`

二级菜单：

- 监控大盘
- Agent 详情（隐藏路由）
- 告警规则
- 告警事件中心
- 数据源管理
- PromQL 查询台

### 5.2 页面职责

#### `MonitorDashboard.vue`

- 总览
- 热点 Top
- 最近告警
- 跳转 Agent 详情 / 告警中心

#### `AgentDetail.vue`

- 单主机详情与趋势
- 关联告警
- 关联任务执行
- 关联资产/服务树/负责人

#### `AlertRules.vue`

- 规则管理
- 规则动作配置
- 通知对象配置

#### `AlertEvents.vue`

- 告警事件列表
- 批量处置
- 跳工单 / 跳任务

#### `MonitorDatasources.vue`

- 数据源 CRUD
- 健康检查

#### `MonitorQuery.vue`

- PromQL 输入
- 时间范围
- 图表结果

---

## 六、关键实现点

### 6.1 Agent 详情接口

建议新增：

- `GET /api/v1/monitor/agents/:agent_id`
- `GET /api/v1/monitor/agents/:agent_id/alerts`
- `GET /api/v1/monitor/agents/:agent_id/task-executions`

### 6.2 告警联动字段

为 `alert_events` 增加业务关联字段：

- `ticket_id`
- `task_execution_id`
- `service_tree_id`
- `owner_id`

### 6.3 数据源抽象

不要把 Prometheus 查询直接塞进 `MonitorService`。

建议新增：

- `monitor_datasource_service.go`
- `prometheus_client.go`

让 `PrometheusClient` 只负责查询，`MonitorService` 负责聚合监控业务。

### 6.4 通知复用

CI/CD 与监控统一复用：

- `NotificationPublishRequest`
- `NotificationConsole`
- `message_pusher`
- `webhook`

---

## 七、验收标准

### V2

- 能从监控大盘跳到 Agent 详情
- 能在告警中心批量确认 / 关闭
- 告警能创建工单
- 告警能触发任务执行
- 告警通知能通过站内信 + Webhook + Message Pusher 发出

### V3

- 可新增 Prometheus 数据源
- 可执行 PromQL 即时查询与范围查询
- 可将查询结果展示成图表
- Prometheus 查询失败时有清晰错误提示

---

## 八、结论

BigOps 监控中心下一步最值得做的，不是继续堆大盘，而是把它推进到：

```text
指标
-> 告警
-> 事件
-> 通知
-> 工单 / 自动修复
```

Agent 是主机监控底座，Prometheus 是应用与云原生监控入口。  
两条线并存，才能让监控中心真正像一个运维平台，而不是单一页面。
