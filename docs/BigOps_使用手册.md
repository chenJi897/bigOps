# BigOps 运维平台 — 完整使用手册

> 版本: v1.0 | 更新时间: 2026-04-10

---

## 一、平台概览

BigOps 是一个全栈运维管理平台，采用 **Go (Gin) + Vue 3 (Element Plus)** 技术栈，覆盖以下核心领域：

| 模块 | 功能范围 | 后端文件数 | 前端页面数 |
|------|----------|-----------|-----------|
| 平台基础 | 用户/角色/菜单/部门/审计日志 | 15 | 7 |
| 资产管理 | CMDB/服务树/云账号/同步 | 12 | 4 |
| 监控中心 | Agent/Golden Signals/SLO/异常检测/容量预测 | 8 | 6 |
| 告警中心 | 规则/事件/静默/OnCall/拓扑/根因/SLA | 10 | 5 |
| 任务中心 | 模板/执行/取消/重试/审批门禁/WebSocket日志 | 10 | 7 |
| 巡检中心 | 模板/计划/记录/报告/diff/健康评分/自动重试 | 4 | 1 |
| 工单中心 | 类型/模板/审批策略/流转/工单全生命周期 | 10 | 8 |
| CI/CD | 项目/流水线/执行/回滚/Webhook | 4 | 3 |
| 通知中心 | 渠道/模板/发送组/站内信/Webhook | 6 | 3 |

**技术架构**: 197+ 个 API 端点 | 39 个数据模型 | 51 个前端页面 | gRPC Agent 通信

---

## 二、环境要求与部署

### 2.1 本地开发

**依赖服务**：
- MySQL 5.7+
- Redis 6+
- Go 1.22+
- Node.js 18+

**后端启动**：
```bash
cd backend
cp config/config.yaml.example config/config.yaml  # 修改数据库/Redis 配置
go mod download
go run ./cmd/core/main.go
```
后端默认监听 `:8080`（HTTP）和 `:9090`（gRPC）。

**前端启动**：
```bash
cd frontend
npm install
npm run dev
```
前端默认访问 `http://localhost:5173`，API 代理到后端 `/api/v1`。

### 2.2 Docker 一键部署

```bash
docker-compose up -d
```
自动拉起 MySQL、Redis、Backend、Frontend(Nginx)，访问 `http://localhost` 即可使用。

### 2.3 配置文件说明

配置文件位于 `backend/config/config.yaml`，关键配置项：

| 配置路径 | 说明 | 默认值 |
|----------|------|--------|
| `server.port` | HTTP 端口 | 8080 |
| `server.mode` | Gin 运行模式 | debug |
| `database.host` | MySQL 地址 | 127.0.0.1 |
| `database.auto_migrate` | 自动建表 | true |
| `redis.host` | Redis 地址 | 127.0.0.1 |
| `grpc.port` | gRPC 端口 | 9090 |
| `log.filename` | 日志文件路径 | logs/bigops.log |

---

## 三、用户认证与权限

### 3.1 登录

- 访问 `/login`，输入用户名和密码
- 默认管理员账号：首次注册的用户自动为 admin 角色
- 登录成功后获取 JWT Token，自动存入 localStorage
- 登录有限流保护：连续失败过多会锁定 15 分钟

### 3.2 密码策略

- 密码必须包含大小写字母、数字、特殊字符
- 最少 8 位
- 修改密码：点击右上角用户头像 → 个人设置 → 修改密码

### 3.3 角色与权限 (RBAC)

| 操作 | 路径 |
|------|------|
| 角色管理 | 系统管理 → 角色管理 |
| 用户角色分配 | 系统管理 → 用户管理 → 设置角色 |
| 菜单权限绑定 | 系统管理 → 角色管理 → 设置菜单 |

- 每个菜单可绑定 API 路径和方法
- 通过 Casbin 实现 API 级别的访问控制
- `admin` 角色默认绕过所有权限检查

### 3.4 部门管理

系统管理 → 部门管理：支持部门树结构、用户归属部门设置。

---

## 四、仪表盘

### 4.1 工作台 (`/dashboard/workbench`)

个人维度的工作概览：
- 我的待办工单
- 我的待审批
- 最近操作记录

### 4.2 概览 (`/dashboard/overview`)

平台级统计：
- 资产总量
- Agent 在线/离线数
- 告警分布
- 资产来源分布饼图

---

## 五、资产管理 (CMDB)

### 5.1 服务树

**路径**: CMDB → 服务树

树形结构管理服务拓扑，支持：
- 创建/编辑/删除节点
- 拖拽移动节点层级
- 查看每个节点关联的资产数量

### 5.2 资产管理

**路径**: CMDB → 资产管理

| 功能 | 说明 |
|------|------|
| 资产列表 | 分页+搜索+状态筛选+服务树筛选 |
| 创建资产 | 手动录入 IP、主机名、配置等 |
| 资产变更历史 | 查看每条资产的变更记录（自动记录每次修改） |
| 关联服务树 | 将资产挂到服务树节点 |

### 5.3 云账号管理

**路径**: CMDB → 云账号

- 支持添加多个云账号（阿里云/AWS/腾讯云等）
- 配置自动同步间隔
- 一键手动同步
- 查看同步任务日志

---

## 六、监控中心

### 6.1 监控大盘 (`/monitor/dashboard`)

核心看板，分为多个区域：

**摘要卡片**：
- Agent 在线/离线数
- 启用规则数
- 活跃告警数

**Golden Signals**：
- 可用性（%）
- 错误率（%）
- 平均延迟（ms）
- 吞吐量（次/分钟）
- 可信度标签（high/medium/low）
- 覆盖率显示（在线Agent/总Agent）

**Golden Signals 维度拆分**：
- 实例维度：按主机名分组
- 服务维度：按服务IP分组
- 接口维度：按指标类型分组

**SLO 配置**：
- 设置目标可用性（默认 99.9%）
- 设置目标延迟上限（默认 200ms）
- 达标/未达标状态实时展示

**异常检测**：
- 基于 Z-Score 的自动异常发现
- 异常归类标签：spike / drop / sustained_high / capacity_risk
- 可调节灵敏度（标准差倍数）

**容量预测**：
- 基于线性回归预测 CPU/内存/磁盘何时达到阈值
- 展示预计到达时间

### 6.2 Agent 列表与详情

**路径**: 监控 → Agent 列表

- 查看所有注册 Agent 的状态（online/offline）
- 点击 Agent 查看详情：CPU/内存/磁盘/网络趋势图
- 支持按状态和关键字筛选

### 6.3 监控数据源

**路径**: 监控 → 数据源

- 添加 Prometheus 数据源
- 健康检查
- 用于 PromQL 查询台

### 6.4 PromQL 查询台

**路径**: 监控 → 查询台

- 直接输入 PromQL 表达式
- 支持 instant query 和 range query
- 结果以表格展示

---

## 七、告警中心

### 7.1 告警规则

**路径**: 监控 → 告警规则

| 字段 | 说明 |
|------|------|
| 名称 | 规则名称（唯一） |
| 监控项 | cpu_usage / memory_usage / disk_usage / agent_offline |
| 运算符 | gt / gte / lt / lte / eq / neq |
| 阈值 | 0-100 |
| 告警级别 | info / warning / critical |
| 动作 | notify_only / create_ticket / execute_task |
| 通知渠道 | in_app / email / webhook |
| 修复任务 | 关联任务模板（动作为 execute_task 时必填） |
| OnCall 排班 | 关联值班表，告警自动通知当前值班人 |

**自动评估**：系统每 30 秒自动检查所有 Agent 指标与规则匹配，触发/恢复告警。

### 7.2 告警事件

**路径**: 监控 → 告警事件

| 功能 | 说明 |
|------|------|
| 事件列表 | 分页+状态筛选+级别筛选+关键字搜索 |
| 事件分组 | 按指纹聚合，展示首次/最后触发时间和持续时长 |
| 确认(ACK) | 人工确认收到告警 |
| 解决(Resolve) | 人工关闭告警 |
| 评论 | 添加处置备注 |
| 指派 | 指派给特定用户，自动设置 SLA 截止时间 |
| 时间线 | 查看告警完整生命周期（触发→确认→升级→解决） |
| 根因分析 | 自动分析告警原因（复发频率、变更关联、部署关联） |
| 上下文 | 查看关联工单、修复任务、处置建议 |
| 拓扑视图 | 查看同服务树下所有主机的健康状态 |
| 变更风险评估 | 评估对目标主机执行变更的风险分数 |

**SLA 机制**：
- 指派后自动计算 SLA 截止时间（critical=15分钟, warning=30分钟, info=60分钟）
- 超时自动生成催办通知和 activity 记录
- 前端展示 SLA 剩余时间和超时状态

**自动升级**：
- 5 分钟未确认的 firing 告警自动升级通知
- 升级记录写入 activity 时间线

### 7.3 告警静默

**路径**: 监控 → 告警静默

- 按规则/Agent/服务树/负责人创建静默规则
- 设置生效/过期时间
- 过期自动失效（后台定期清理）

### 7.4 OnCall 值班

**路径**: 监控 → OnCall 排班

- 创建值班表（多用户轮换）
- 设置升级超时时间
- 告警规则绑定值班表后，自动通知当前值班人

---

## 八、任务中心

### 8.1 任务模板

**路径**: 任务中心 → 任务模板

| 字段 | 说明 |
|------|------|
| 名称 | 任务名称 |
| 任务大类 | script / file_transfer / api_call |
| 脚本语言 | bash / python / sh / powershell（仅 script 类型） |
| 脚本内容 | CodeMirror 编辑器，支持语法高亮和静态检查 |
| 超时时间 | 秒（默认 60） |
| 执行用户 | 目标机器上运行脚本的用户（如 root） |
| 风险等级 | low / medium / high / critical（自动推断或手动设置） |
| 需要审批 | 高危任务自动设为需审批 |

**风险自动推断规则**：
- 包含 `rm -rf /`、`mkfs`、`dd if=`、`shutdown`、`reboot` 等命令 → critical（必须审批）
- 执行用户为 root → 至少 medium
- 包含 `systemctl restart`、`service` → high（需审批）

**脚本安全检查**：
- 前端 ScriptEditor 组件实时检测危险命令
- 后端 scriptguard 包二次校验
- 未通过检查的脚本不允许执行

### 8.2 任务审批（审批门禁）

**触发条件**: 任务的 `require_approval = 1`

**流程**：
1. 用户点击「申请审批」→ 创建 pending 状态的审批记录
2. 管理员在「待审批」中查看 → 通过/拒绝
3. 通过后用户才可执行任务
4. 未审批直接执行 → 返回 409 错误

**相关 API**：
- `POST /tasks/:id/request-approval` — 申请审批
- `GET /task-approvals/pending` — 待审批列表
- `POST /task-approvals/:id/approve` — 通过
- `POST /task-approvals/:id/reject` — 拒绝
- `GET /tasks/:id/approvals` — 审批记录

### 8.3 任务执行

**操作步骤**：
1. 在任务模板列表点击「执行」
2. 输入目标主机 IP（每行一个）
3. 点击「立即执行」
4. 自动跳转到执行详情页

**执行过程**：
- 系统通过 gRPC 将脚本下发到目标 Agent
- Agent 执行后实时回传日志（WebSocket）
- 每台主机独立记录执行结果（成功/失败/超时/取消）

### 8.4 执行详情

**路径**: 任务中心 → 执行记录 → 点击查看

| 功能 | 说明 |
|------|------|
| 实时日志 | WebSocket 推送，自动滚动 |
| 主机结果 | 每台主机的退出码、stdout、stderr |
| 取消执行 | 终止进行中的任务 |
| 重试 | 重试失败的主机（支持单主机重试） |
| 导出报告 | Markdown 格式的执行报告下载 |

### 8.5 Agent 管理

**路径**: 任务中心 → Agent 列表

- 查看所有注册 Agent 的在线状态
- Agent 通过 gRPC 双向流与后端保持连接
- 离线 Agent 在 45 秒后自动标记为 offline

---

## 九、巡检中心

**路径**: 巡检中心 → 巡检中心

### 9.1 巡检模板

| 字段 | 说明 |
|------|------|
| 名称 | 模板名称 |
| 关联任务 | 选择一个任务模板作为巡检脚本 |
| 默认主机 | 巡检目标主机列表 |
| 修复任务 | 可选，巡检失败时自动触发的修复脚本 |
| 最大重试 | 失败后自动重试次数上限（0=不重试） |

### 9.2 巡检计划

| 字段 | 说明 |
|------|------|
| 名称 | 计划名称 |
| 关联模板 | 选择巡检模板 |
| Cron 表达式 | 定时执行周期（如 `0 2 * * *` 每天凌晨2点） |
| 启用 | 开关 |

**冲突检测**: 同一模板+同一 Cron 表达式的计划不允许同时存在（服务端强制阻断）。

**批量操作**: 支持批量启用/禁用/删除计划。

### 9.3 巡检记录与报告

- 每次执行生成独立记录
- 报告包含每台主机的执行结果、stdout、stderr
- 支持 JSON / CSV 格式导出
- 支持两次记录间的 diff 对比（新增失败/已恢复/持续失败）

### 9.4 健康评分

- 按模板维度展示历史成功率
- 趋势图展示最近 30 次执行的成功/失败分布

### 9.5 失败自动重试

- 当巡检执行失败且模板配置了 `max_retries > 0` 时
- 系统自动创建重试记录并重新执行
- `retry_count` 递增直到达到上限
- 重试报告中记录来源记录 ID

---

## 十、工单中心

### 10.1 工单类型

**路径**: 工单 → 工单类型

- 创建不同类型的工单（如故障、变更、需求）
- 每种类型可绑定审批策略

### 10.2 请求模板

**路径**: 工单 → 请求模板

- 预定义工单创建表单
- 支持字段模板化

### 10.3 审批策略

**路径**: 工单 → 审批策略

- 多级审批流（支持多个审批阶段）
- 每阶段可配置审批人
- 工单创建时自动触发审批流

### 10.4 发起工单

**路径**: 工单 → 发起工单

1. 选择请求模板（或空白工单）
2. 填写标题、描述
3. 提交后自动进入审批流（如有）
4. 审批通过后进入处理阶段

### 10.5 工单处理

| 操作 | 说明 |
|------|------|
| 指派 | 分配处理人 |
| 处理 | 记录处理过程 |
| 转交 | 转给其他人处理 |
| 评论 | 添加协作备注 |
| 关闭 | 填写解决方案后关闭 |
| 重新打开 | 已关闭的工单可重新打开 |

### 10.6 审批待办

**路径**: 工单 → 审批待办

- 查看需要我审批的工单
- 通过 / 拒绝

---

## 十一、CI/CD

### 11.1 项目管理

**路径**: CI/CD → 项目

- 创建项目（Git 仓库地址、描述）
- 启用/禁用项目

### 11.2 流水线

**路径**: CI/CD → 流水线

- 创建流水线（YAML 配置）
- 手动触发 / Webhook 触发
- 支持外部系统通过 Webhook URL 触发

**Webhook 地址**: `POST /api/v1/cicd/webhook/{pipeline_code}`

### 11.3 执行记录

**路径**: CI/CD → 执行记录

- 查看每次执行的状态和日志
- 支持重试失败的执行
- 支持回滚到上一次成功

---

## 十二、通知中心

### 12.1 通知配置

**路径**: 系统 → 通知配置中心

| 功能 | 说明 |
|------|------|
| 渠道配置 | 启用/配置 站内信、Email、Webhook |
| Webhook 测试 | 测试 Webhook URL 连通性 |
| 通知模板 | 自定义告警/工单/系统通知模板（Go template 语法） |
| 模板预览 | 实时预览模板渲染效果 |

### 12.2 发送组

**路径**: 系统 → 发送组

- 创建一组通知接收人 + 渠道配置
- 告警规则可直接绑定发送组
- 支持配置是否发送恢复通知

### 12.3 站内信

- 右上角铃铛图标显示未读数
- 站内信列表：标记已读、全部已读、清除已读
- 系统所有通知自动推送站内信

### 12.4 个人通知偏好

**路径**: 个人设置 → 通知偏好

- 按事件类型（告警/工单/系统）配置接收渠道
- 可关闭不关心的通知类型

---

## 十三、系统管理

### 13.1 用户管理

- 用户列表（搜索/分页）
- 编辑用户信息
- 启用/禁用用户
- 设置用户角色
- 设置用户部门

### 13.2 角色管理

- 角色 CRUD
- 设置角色可访问的菜单（即 API 权限）

### 13.3 菜单管理

- 菜单树管理
- 每个菜单节点可绑定 API 路径和方法
- 菜单类型：目录 / 菜单 / 按钮

### 13.4 审计日志

**路径**: 系统管理 → 审计日志

- 记录所有写操作（创建/更新/删除/执行/审批）
- 按用户名、操作类型、资源类型筛选
- 自动记录操作人、时间、详情

---

## 十四、API 规范

### 14.1 请求格式

- Base URL: `/api/v1`
- 认证: `Authorization: Bearer {token}`
- Content-Type: `application/json`

### 14.2 响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

- `code = 0` 表示成功
- `code != 0` 表示业务错误

### 14.3 分页响应

```json
{
  "code": 0,
  "data": {
    "list": [...],
    "total": 100,
    "page": 1,
    "size": 20
  }
}
```

### 14.4 Swagger 文档

```bash
cd backend
swag init -g cmd/core/main.go -o docs/swagger
```

启动后访问: `http://localhost:8080/swagger/index.html`

---

## 十五、回归测试

### 15.1 后端测试

```bash
cd backend
go test ./... -v
```

### 15.2 冒烟测试脚本

```bash
cd backend
BASE_URL="http://127.0.0.1:8080" \
USERNAME="admin" \
PASSWORD="your_password" \
bash scripts/smoke_test.sh
```

覆盖范围：登录 → 任务CRUD → 告警列表 → 监控摘要 → Golden Signals → 巡检模板 → 待审批 → 登出

### 15.3 前端构建验证

```bash
cd frontend
npx vue-tsc -b --noEmit  # TypeScript 类型检查
npm run build            # 生产构建
```

---

## 十六、快速上手指南

### 第一步：登录系统
访问平台地址，使用管理员账号登录。

### 第二步：创建第一个任务
1. 进入「任务中心 → 任务模板」
2. 点击「新建任务」
3. 填写名称，选择脚本类型，输入脚本内容
4. 保存

### 第三步：执行任务
1. 在任务列表点击「执行」
2. 输入目标主机 IP
3. 查看实时日志和执行结果

### 第四步：配置告警
1. 进入「监控 → 告警规则」
2. 创建规则（如 CPU > 80%）
3. 配置通知渠道和接收人
4. 规则自动每 30 秒评估

### 第五步：创建巡检计划
1. 进入「巡检中心」
2. 创建巡检模板（关联任务模板）
3. 创建计划（设置 Cron 表达式）
4. 启用计划

---

## 附录A：完整 API 端点清单

### 认证
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/auth/login` | 登录 |
| POST | `/auth/register` | 注册 |
| POST | `/auth/logout` | 登出 |
| GET | `/auth/info` | 获取当前用户信息 |
| POST | `/auth/password` | 修改密码 |

### 用户管理
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/users` | 用户列表 |
| POST | `/users/:id` | 更新用户 |
| POST | `/users/:id/status` | 启用/禁用 |
| POST | `/users/:id/delete` | 删除用户 |
| GET | `/users/:id/roles` | 获取用户角色 |
| POST | `/users/:id/roles` | 设置用户角色 |
| POST | `/users/:id/department` | 设置部门 |

### 角色管理
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/roles` | 角色列表 |
| GET | `/roles/:id` | 角色详情 |
| POST | `/roles` | 创建角色 |
| POST | `/roles/:id` | 更新角色 |
| POST | `/roles/:id/delete` | 删除角色 |
| POST | `/roles/:id/status` | 启用/禁用 |
| POST | `/roles/:id/menus` | 设置菜单权限 |

### 任务管理
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/tasks` | 任务列表 |
| GET | `/tasks/:id` | 任务详情 |
| POST | `/tasks` | 创建任务 |
| POST | `/tasks/:id` | 更新任务 |
| POST | `/tasks/:id/delete` | 删除任务 |
| POST | `/tasks/:id/execute` | 执行任务 |
| POST | `/tasks/:id/request-approval` | 申请审批 |
| GET | `/tasks/:id/approvals` | 审批记录 |
| GET | `/task-approvals/pending` | 待审批列表 |
| POST | `/task-approvals/:id/approve` | 通过审批 |
| POST | `/task-approvals/:id/reject` | 拒绝审批 |
| GET | `/task-executions` | 执行记录列表 |
| GET | `/task-executions/:id` | 执行详情 |
| POST | `/task-executions/:id/cancel` | 取消执行 |
| POST | `/task-executions/:id/retry` | 重试执行 |
| GET | `/task-executions/:id/report` | 导出报告 |

### 监控
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/monitor/summary` | 监控摘要 |
| GET | `/monitor/agents` | Agent 列表 |
| GET | `/monitor/golden-signals` | Golden Signals |
| GET | `/monitor/golden-signals/dimensions` | 维度拆分 |
| GET | `/monitor/slo-config` | SLO 配置 |
| POST | `/monitor/slo-config` | 更新 SLO |
| GET | `/monitor/anomalies` | 异常检测 |
| GET | `/monitor/capacity-prediction` | 容量预测 |

### 告警
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/alert-rules` | 规则列表 |
| POST | `/alert-rules` | 创建规则 |
| POST | `/alert-rules/evaluate` | 手动触发评估 |
| GET | `/alert-events` | 事件列表 |
| GET | `/alert-events/groups` | 事件分组 |
| GET | `/alert-events/:id` | 事件详情 |
| GET | `/alert-events/:id/timeline` | 时间线 |
| GET | `/alert-events/:id/root-cause` | 根因分析 |
| POST | `/alert-events/:id/ack` | 确认 |
| POST | `/alert-events/:id/resolve` | 解决 |
| POST | `/alert-events/:id/assign` | 指派 |

### 巡检
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/inspection/templates` | 模板列表 |
| POST | `/inspection/templates` | 创建模板 |
| GET | `/inspection/plans` | 计划列表 |
| POST | `/inspection/plans` | 创建计划 |
| POST | `/inspection/plans/:id/run` | 手动执行 |
| GET | `/inspection/records` | 记录列表 |
| GET | `/inspection/records/:id/report` | 查看报告 |
| GET | `/inspection/records/diff` | 记录对比 |
| GET | `/inspection/health-scores` | 健康评分 |

---

## 附录B：数据模型清单

| 模型 | 表名 | 说明 |
|------|------|------|
| User | users | 用户 |
| Role | roles | 角色 |
| Menu | menus | 菜单 |
| UserRole | user_roles | 用户-角色关联 |
| Department | departments | 部门 |
| AuditLog | audit_logs | 审计日志 |
| ServiceTree | service_trees | 服务树 |
| CloudAccount | cloud_accounts | 云账号 |
| CloudSyncTask | cloud_sync_tasks | 云同步任务 |
| Asset | assets | 资产 |
| AssetChange | asset_changes | 资产变更 |
| Task | tasks | 任务模板 |
| TaskApproval | task_approvals | 任务审批 |
| TaskExecution | task_executions | 任务执行 |
| TaskHostResult | task_host_results | 主机执行结果 |
| AgentInfo | agent_infos | Agent 信息 |
| AgentMetricSample | agent_metric_samples | Agent 指标采样 |
| AlertRule | alert_rules | 告警规则 |
| AlertEvent | alert_events | 告警事件 |
| AlertEventActivity | alert_event_activities | 告警事件活动 |
| AlertSilence | alert_silences | 告警静默 |
| OnCallSchedule | oncall_schedules | 值班排班 |
| MonitorDatasource | monitor_datasources | 监控数据源 |
| Ticket | tickets | 工单 |
| TicketType | ticket_types | 工单类型 |
| TicketActivity | ticket_activities | 工单活动 |
| RequestTemplate | request_templates | 请求模板 |
| ApprovalPolicy | approval_policies | 审批策略 |
| ApprovalInstance | approval_instances | 审批实例 |
| ApprovalRecord | approval_records | 审批记录 |
| NotificationEvent | notification_events | 通知事件 |
| NotificationDelivery | notification_deliveries | 通知投递 |
| InAppNotification | in_app_notifications | 站内信 |
| NotificationTemplate | notification_templates | 通知模板 |
| NotifyGroup | notify_groups | 发送组 |
| InspectionTemplate | inspection_templates | 巡检模板 |
| InspectionPlan | inspection_plans | 巡检计划 |
| InspectionRecord | inspection_records | 巡检记录 |
| CICDProject | cicd_projects | CI/CD 项目 |
| CICDPipeline | cicd_pipelines | CI/CD 流水线 |
| CICDPipelineRun | cicd_pipeline_runs | 流水线执行 |
