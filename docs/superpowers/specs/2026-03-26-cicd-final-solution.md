# CI/CD 最终版方案

**日期**: 2026-03-26  
**状态**: 可执行设计  
**目标模块**: Module 06 CI/CD

---

## 一、目标

把当前仅有“项目/流水线骨架”的 CI/CD 模块推进到可用闭环，做到：

1. 能管理代码项目与发布流水线
2. 能手动触发一次流水线运行
3. 能记录运行历史和运行详情
4. 能在需要审批时先走审批，再自动触发部署任务
5. 能复用现有任务中心能力执行发布动作
6. 能复用现有工单模板 / 审批流做发布审批

最终形成这条链：

```text
项目
  -> 流水线
    -> 手动触发
      -> 运行记录
        -> 如配置审批模板则生成发布审批工单
          -> 审批通过
            -> 自动调用任务中心执行部署任务
              -> 回写执行状态、摘要、运行详情
```

---

## 二、设计原则

### 2.1 不重复造轮子

CI/CD 不自己实现脚本执行器，也不自己重新造审批系统：

- 执行复用 `Task / TaskExecution / Agent`
- 审批复用 `Ticket / RequestTemplate / ApprovalInstance`
- 通知复用现有 `NotificationService`

### 2.2 先做“可交付”，不是“完整 Jenkins 替代品”

第一阶段不做：

- Git webhook 全自动触发
- 多阶段编排 DSL
- 产物仓库
- 蓝绿/灰度/金丝雀复杂发布
- 完整构建节点池

第一阶段只做：

- 项目
- 流水线
- 运行记录
- 运行详情
- 审批联动
- 部署任务联动

### 2.3 运行记录是业务真相

所有发布行为以 `cicd_pipeline_runs` 为准。

- 工单是审批真相
- 任务执行记录是部署真相
- CI/CD 运行记录是发布过程真相

这三者需要互相引用，但不能混为一张表。

---

## 三、领域模型

### 3.1 `CICDProject`

表示一个代码项目或交付单元。

核心字段：

- `name`
- `code`
- `repo_provider`
- `repo_url`
- `default_branch`
- `description`
- `owner_id`
- `status`

职责：

- 作为流水线归属对象
- 作为运行记录的聚合维度
- 作为后续 webhook、仓库凭据、环境变量的挂载点

### 3.2 `CICDPipeline`

表示一条流水线定义。

核心字段：

- `project_id`
- `name`
- `code`
- `environment`
- `trigger_type`
- `trigger_ref`
- `branch`
- `schedule`
- `build_task_id`
- `deploy_task_id`
- `request_template_id`
- `target_hosts`
- `status`

职责：

- 决定如何触发
- 决定用哪个部署任务
- 决定是否需要审批
- 决定目标主机

### 3.3 `CICDPipelineRun`

表示一次流水线运行实例。

核心字段：

- `pipeline_id`
- `project_id`
- `run_number`
- `trigger_type`
- `trigger_ref`
- `branch`
- `commit_id`
- `status`
- `task_execution_id`
- `approval_ticket_id`
- `triggered_by`
- `target_hosts`
- `summary`
- `error_message`

状态建议：

- `created`
- `waiting_approval`
- `running`
- `success`
- `failed`
- `canceled`

职责：

- 串起流水线、审批单、任务执行记录
- 给前端展示运行历史与当前状态

---

## 四、核心业务流

### 4.1 无审批流水线

```text
管理员手动触发流水线
-> 创建 PipelineRun(status=created)
-> 若配置 deploy_task_id + target_hosts
-> 调用 TaskService.ExecuteTask()
-> 写入 task_execution_id
-> PipelineRun.status = running
-> 后续根据任务执行结果回写 success/failed
```

### 4.2 需要审批的流水线

```text
管理员手动触发流水线
-> 创建 PipelineRun(status=waiting_approval)
-> 创建发布审批工单 Ticket(source=manual, ticket_kind=change)
-> 工单审批流启动
-> 审批通过
-> 根据 approval_ticket_id 找到 PipelineRun
-> 自动触发 deploy_task_id
-> 写入 task_execution_id
-> PipelineRun.status = running
-> 后续根据任务执行结果回写 success/failed
```

### 4.3 审批拒绝

```text
审批拒绝
-> Ticket.approval_status = rejected
-> 回查关联 PipelineRun
-> PipelineRun.status = failed
-> summary = 审批拒绝
-> error_message = 拒绝原因
```

---

## 五、最终版功能清单

### 5.1 项目管理

- 项目列表
- 新增/编辑/删除
- 启用/禁用
- 基础仓库信息维护

### 5.2 流水线管理

- 流水线列表
- 新增/编辑/删除
- 启用/停用
- 手动触发
- 展示最近一次运行状态

### 5.3 运行记录

- 按项目筛选
- 按流水线筛选
- 按状态筛选
- 查看运行号、触发方式、分支、触发人、状态、摘要

### 5.4 运行详情

运行详情页或详情抽屉至少展示：

- 流水线信息
- 项目信息
- 分支 / commit
- 状态流转
- 审批单链接
- 任务执行记录链接
- 执行摘要
- 失败信息

### 5.5 审批联动

- 配置审批模板的流水线，触发后自动生成发布审批工单
- 审批通过后自动触发部署任务
- 审批拒绝后回写失败状态

### 5.6 任务执行联动

- 运行详情可跳转到 `TaskExecution`
- 若部署任务已执行，显示：
  - `task_execution_id`
  - 执行状态
  - 执行摘要

---

## 六、页面结构

### 6.1 一级入口

`CI/CD`

二级菜单：

- 项目管理
- 流水线管理
- 运行记录

### 6.2 页面职责

#### `CicdProjects.vue`

- 维护项目
- 只做项目级配置

#### `CicdPipelines.vue`

- 维护流水线
- 手动触发
- 展示最近一次运行
- 跳转运行记录

#### `CicdRuns.vue`

- 展示运行历史
- 筛选 / 分页
- 进入运行详情

#### `CicdRunDetail`

建议新增。

职责：

- 统一展示审批、运行、任务执行三条信息

---

## 七、接口设计

### 7.1 项目

- `GET /api/v1/cicd/projects`
- `POST /api/v1/cicd/projects`
- `POST /api/v1/cicd/projects/:id`
- `POST /api/v1/cicd/projects/:id/status`
- `POST /api/v1/cicd/projects/:id/delete`

### 7.2 流水线

- `GET /api/v1/cicd/pipelines`
- `POST /api/v1/cicd/pipelines`
- `POST /api/v1/cicd/pipelines/:id`
- `POST /api/v1/cicd/pipelines/:id/trigger`
- `POST /api/v1/cicd/pipelines/:id/delete`

### 7.3 运行记录

- `GET /api/v1/cicd/runs`
- `GET /api/v1/cicd/runs/:id`

### 7.4 第二批建议补充

- `POST /api/v1/cicd/runs/:id/retry`
- `POST /api/v1/cicd/runs/:id/cancel`

---

## 八、关键实现点

### 8.1 流水线列表要带 `latest_run`

前端最关心的一定不是流水线静态字段，而是：

- 最近一次有没有跑
- 最近一次状态是什么
- 最近一次摘要是什么

所以 `GET /cicd/pipelines` 应直接返回：

- `latest_run.id`
- `latest_run.status`
- `latest_run.run_number`
- `latest_run.summary`
- `latest_run.created_at`

### 8.2 审批通过后的自动部署必须在后端完成

不能靠前端轮询后再手工点。

正确逻辑：

- 审批服务在 `approval_approved` 的最终节点完成后
- 调用 `CICDService.StartApprovedRunsByTicketID(ticketID)`
- 由后端直接触发部署任务

### 8.3 运行状态要和任务执行状态解耦

`PipelineRun.status` 不应该简单复制 `TaskExecution.status`，但需要映射：

- `TaskExecution.running` -> `PipelineRun.running`
- `TaskExecution.success` -> `PipelineRun.success`
- `TaskExecution.failed/partial_fail` -> `PipelineRun.failed`

---

## 九、测试方案

### 9.1 接口层

必须覆盖：

- 创建项目
- 创建流水线
- 触发无审批流水线
- 触发有审批流水线
- 查询运行记录
- 查询运行详情
- 审批通过后自动部署
- 审批拒绝后运行失败

### 9.2 浏览器层

必须同时使用：

- Playwright
- Chrome DevTools / CDP

#### Playwright 覆盖

- 登录 admin
- 项目管理页打开
- 创建项目
- 流水线页打开
- 创建流水线
- 触发流水线
- 查看运行记录
- 查看运行详情

#### Chrome DevTools 覆盖

- 页面 console 无错误
- 关键请求无 4xx/5xx
- 路由存在
- 关键页面文本命中

### 9.3 角色覆盖

至少覆盖：

- `admin`
- `ops` 普通用户

#### admin

- 能看全部 CI/CD 页面
- 能创建/编辑/触发

#### ops

- 至少能查看页面和运行记录
- 如果后续要控制更细权限，再补写权限矩阵

### 9.4 审批链覆盖

至少跑一条真实链路：

1. 创建带审批模板的流水线
2. 手动触发
3. 生成发布审批工单
4. 一级审批人通过
5. 二级审批人通过
6. 自动触发部署任务
7. 回写 `task_execution_id`
8. 运行记录状态变为 `running/success/failed`

---

## 十、验收标准

达到“最终版可用闭环”的最低标准：

- 项目、流水线、运行记录三个页面都可正常使用
- 可手动触发流水线
- 可看到运行记录
- 可进入运行详情
- 配置审批模板后，审批通过能自动触发部署
- 任务执行记录能回链到流水线运行记录
- Playwright + Chrome DevTools 均通过关键路径验收

---

## 十一、当前缺口与下一步

当前已经有第一批骨架，但距离最终版仍差：

1. `运行详情页`
2. `审批通过后自动部署` 的完整浏览器验收
3. `任务执行结果 -> 运行状态` 的自动回写
4. `失败重试 / 取消运行`

下一步开发顺序建议：

1. 运行详情
2. 审批通过自动部署
3. 任务执行状态回写
4. 浏览器回归全链路
