# BigOps 下一阶段战略方案

> 作者：AI Architect | 日期：2026-03-31

---

## 一、当前平台现状评估

### 已落地能力（传统运维闭环）

| 模块 | 能力 | 成熟度 |
|------|------|--------|
| 底座 | 用户/角色/RBAC/菜单/审计 | ★★★★★ |
| CMDB | 服务树/资产/云同步 | ★★★★☆ |
| 工单 | 模板/多级审批/通知 | ★★★★☆ |
| 任务 | 远程执行/Agent/WebSocket日志 | ★★★★☆ |
| 监控 | Agent指标/告警规则/事件/Prometheus | ★★★★☆ |
| CI/CD | 项目/流水线/构建→审批→部署 | ★★★★☆ |
| 通知 | 站内信/全部已读/偏好/多渠道 | ★★★★☆ |

**代码量**：后端 ~15K 行 / 前端 ~8K 行 / ~104 API / 24 页面

### 现存短板（用户说的"很传统"）

1. **纯人驱动** — 所有操作都需要人发起、人判断、人执行
2. **告警只报不治** — 告警触发后只发通知，没有自动修复闭环
3. **知识不沉淀** — 故障处理经验散落在工单/聊天记录中，下次遇到还要重来
4. **监控看图猜因** — 看到 CPU 高了，但为什么高、该怎么处理全靠经验
5. **变更靠胆量** — 没有变更风险预判，上线全靠人肉把关
6. **没有度量** — 不知道团队效率如何、哪个服务最不稳定、MTTR 多少

---

## 二、战略方向：AI-Native 运维平台

### 核心理念

> **从"人操作平台"进化到"AI 操作平台、人监督 AI"**

不是在现有平台上"加个 ChatBot"，而是让 AI 成为运维流程的第一公民：

```
传统：  人 → 看告警 → 人判断 → 人操作 → 人确认
AI版：  告警 → AI 诊断 → AI 生成修复方案 → 人审批 → AI 执行 → AI 验证
```

### 分层架构

```
┌─────────────────────────────────────────────────┐
│                  AI 交互层                        │
│  自然语言运维台 / Copilot / 语音指令               │
├─────────────────────────────────────────────────┤
│                  AI 引擎层                        │
│  LLM Gateway / RAG / Agent Framework / MCP       │
├─────────────────────────────────────────────────┤
│                  知识层                           │
│  运维知识库 / 故障案例库 / Runbook / Embedding     │
├─────────────────────────────────────────────────┤
│                  数据层                           │
│  指标 / 日志 / Trace / 变更 / 拓扑 / CMDB         │
├─────────────────────────────────────────────────┤
│                  执行层（已有）                     │
│  Agent / 任务中心 / CI/CD / 工单 / 通知            │
└─────────────────────────────────────────────────┘
```

---

## 三、AI 能力落地路线图

### Phase 1：AI Copilot（1-2 个月）— 让 AI 能看懂、能说话

#### 1.1 运维 ChatBot（自然语言查询）

**目标**：用户用自然语言和平台交互，替代 80% 的点击操作。

```
用户："线上 web-01 服务器 CPU 多少？"
AI：  "web-01 当前 CPU 78.3%，过去1小时平均 65.2%，
       触发了告警规则「CPU > 80%」3 次，
       最近一次变更是 2 小时前的部署任务 #42。
       需要我查看详细趋势吗？"
```

**技术方案**：
- 后端新增 `/api/v1/ai/chat` 接口
- 接入 LLM（Claude API / 本地 Ollama）
- 实现 Function Calling / Tool Use：
  - `query_asset(hostname)` → 调 CMDB
  - `query_metrics(agent_id, metric, range)` → 调监控
  - `list_alerts(status, severity)` → 调告警
  - `list_recent_changes(host, hours)` → 调 CI/CD + 工单
  - `execute_command(host, command)` → 调任务中心（需审批）
- 前端新增全局 AI 对话面板（类 GitHub Copilot Chat）

#### 1.2 告警智能摘要

**目标**：告警触发时，AI 自动分析上下文，生成"这个告警是什么、为什么、建议怎么做"。

```
传统告警：web-01 CPU > 90% 持续 5 分钟
AI 增强：
  ■ 现象：web-01 CPU 92.3%，内存 78%
  ■ 可能原因：30 分钟前执行了部署任务 #42（新版本 v2.3.1）
  ■ 关联：同集群 web-02 正常（CPU 35%），排除流量问题
  ■ 建议：检查 v2.3.1 变更内容，考虑回滚到 v2.3.0
  ■ 快速操作：[查看部署详情] [一键回滚] [静默1小时]
```

**技术方案**：
- 告警触发时收集上下文（指标趋势、最近变更、关联资产）
- 调 LLM 生成结构化分析
- 结果写入 `alert_events.ai_analysis` 字段
- 通知中附带 AI 分析摘要

#### 1.3 工单智能填写

**目标**：用户描述需求，AI 自动选模板、填字段、推荐审批人。

```
用户："帮我申请扩容 web 集群，加 2 台 4C8G 的机器"
AI：  自动选择「资源申请」模板
      填入：类型=ECS扩容, 规格=4C8G, 数量=2, 集群=web
      推荐审批人：张三（web 集群负责人）
      [确认提交]
```

---

### Phase 2：AI 诊断（2-3 个月）— 让 AI 能分析、能定位

#### 2.1 根因分析引擎（RCA）

**目标**：告警触发后，AI 自动执行诊断流程，输出根因。

```
触发：web-01 HTTP 5xx 激增
AI 自动执行：
  1. 查看 web-01 资源指标 → CPU 正常, 内存 95%
  2. 查看最近变更 → 2h 前部署了 order-service v3.1
  3. 查看日志 → OOM Kill 出现 12 次
  4. 查看 order-service 内存趋势 → v3.1 内存泄漏
  5. 结论：order-service v3.1 内存泄漏导致 OOM
  6. 建议：回滚到 v3.0，通知开发修复
```

**技术方案**：
- 实现 AI Agent 框架（ReAct 模式）
- 预定义诊断 Tool 集：
  - `check_resource` — 资源指标
  - `check_logs` — 日志分析（需接入日志系统）
  - `check_changes` — 变更历史
  - `check_topology` — 服务拓扑
  - `check_similar_incidents` — 历史相似故障
- Agent 自主决定调用顺序，逐步缩小范围

#### 2.2 运维知识库（RAG）

**目标**：将团队的运维经验变成 AI 可检索的知识。

**知识来源**：
- 历史工单（问题 + 解决方案）
- 告警处理记录
- Runbook / SOP 文档
- 故障复盘报告
- 内部 Wiki

**技术方案**：
- 新增 `知识库管理` 模块
- 文档上传 → 切片 → Embedding → 向量数据库（pgvector / Milvus）
- AI 对话和诊断时自动 RAG 检索相关知识
- 每次故障处理后提示"是否沉淀为知识"

#### 2.3 变更风险评估

**目标**：CI/CD 发布前，AI 评估变更风险等级。

```
流水线触发前：
  ■ 风险等级：中等 ⚠️
  ■ 原因：
    - 变更涉及数据库 migration（含 ALTER TABLE）
    - 目标环境有 3 条活跃告警
    - 上次发布 2 小时前刚回滚过
  ■ 建议：错峰发布，优先处理现有告警
  ■ [继续发布] [延后发布] [查看详情]
```

---

### Phase 3：AI 自愈（3-6 个月）— 让 AI 能执行、能修复

#### 3.1 自动修复（Auto-Remediation）

**目标**：常见故障自动修复，无需人工介入。

**修复策略库**：

| 故障类型 | 自动修复动作 | 人工审批 |
|---------|-------------|---------|
| 磁盘满 | 清理日志/临时文件 | 否 |
| OOM | 重启服务 | 否 |
| 证书过期 | 自动续签 | 否 |
| 部署导致 5xx | 自动回滚 | 是（紧急时跳过） |
| 流量突增 | 自动扩容 | 是 |
| 数据库慢查询 | Kill 慢查询 + 通知 | 否 |

**技术方案**：
- 告警规则扩展：`action = auto_remediate`
- 修复策略配置：触发条件 → 修复动作 → 验证条件 → 回滚策略
- 复用现有任务中心执行，修复脚本以 Runbook 形式管理
- 所有自动修复操作记录到审计日志，支持事后复盘

#### 3.2 智能编排（AI-Driven Workflow）

**目标**：AI 根据场景自动编排多步骤操作。

```
场景：需要对 web 集群做灰度升级
AI 自动编排：
  Step 1: 对 web-01 执行部署（灰度 25%）
  Step 2: 等待 10 分钟，检查错误率
  Step 3: 如果错误率 < 1%，继续 web-02
  Step 4: 全量完成后通知负责人
  Step 5: 如果任何步骤错误率 > 5%，自动回滚全部
```

#### 3.3 预测性运维

**目标**：在故障发生前预警。

- **磁盘满预测**：基于增长趋势预测何时满，提前告警
- **证书过期预测**：扫描证书有效期，提前 30 天告警
- **容量预测**：基于业务增长趋势预测资源不足时间点
- **异常检测**：基于历史基线的无阈值告警（用 AI 替代手工设阈值）

---

### Phase 4：AI 平台化（6-12 个月）— 让 AI 能力可扩展

#### 4.1 MCP Server 化

**目标**：BigOps 自身作为 MCP Server，让外部 AI 工具能调用运维能力。

```
BigOps MCP Server 暴露的 Tools：
  - bigops_query_asset      查询资产信息
  - bigops_execute_task     执行远程命令
  - bigops_create_ticket    创建工单
  - bigops_deploy           触发部署
  - bigops_query_metrics    查询监控指标
  - bigops_manage_alert     管理告警
```

这样 Claude Code、Cursor、VS Code 中的 AI 助手可以直接操作运维平台。

#### 4.2 Agentic Workflow 引擎

**目标**：可视化编排 AI Agent 工作流。

```
┌──────────┐    ┌──────────┐    ┌──────────┐
│  触发器   │ →  │  AI 诊断  │ →  │ 人工审批  │
│ (告警/定时)│    │ (RCA Agent)│    │ (工单系统) │
└──────────┘    └──────────┘    └──────────┘
                                       │
                     ┌─────────────────┘
                     ↓
              ┌──────────┐    ┌──────────┐
              │  AI 执行  │ →  │  AI 验证  │
              │ (任务中心) │    │ (指标检查) │
              └──────────┘    └──────────┘
```

#### 4.3 多模型适配

| 场景 | 推荐模型 | 原因 |
|------|---------|------|
| 运维对话 | Claude Sonnet | 性价比高，响应快 |
| 根因分析 | Claude Opus | 复杂推理能力强 |
| 告警摘要 | Claude Haiku | 低延迟，批量处理 |
| 日志分析 | 本地 Ollama | 敏感数据不出网 |
| Embedding | BGE / Nomic | 知识库向量化 |

---

## 四、非 AI 方向的必要补充

### 4.1 可观测性补齐

| 能力 | 当前状态 | 目标 |
|------|---------|------|
| Metrics | Agent 基础指标 | 接入 Prometheus + 自定义指标 |
| Logs | 无 | 接入 Loki / ES，日志查询页 |
| Traces | 无 | 接入 Jaeger / Tempo |
| 拓扑 | 无 | 服务拓扑自动发现 + 可视化 |

> AI 诊断的前提是数据丰富。没有日志和 Trace，AI 也只能看指标猜原因。

### 4.2 数据库管理

- SQL 工单（审核 → 执行 → 回滚）
- 慢查询分析
- 数据脱敏查询
- AI 辅助：自然语言转 SQL、SQL 优化建议

### 4.3 效能度量

- DORA 指标（部署频率、变更前置时间、MTTR、变更失败率）
- 服务 SLI/SLO 管理
- 团队效能看板
- AI 辅助：自动识别效能瓶颈，生成改进建议

### 4.4 多租户 / SaaS 化

- 租户隔离
- 计量计费
- 自助开通
- API 开放平台

---

## 五、技术选型建议

### AI 基础设施

| 组件 | 推荐方案 | 备选 |
|------|---------|------|
| LLM Gateway | LiteLLM / OneAPI | 自研网关 |
| LLM 调用 | Claude API (Anthropic) | OpenAI / 国产大模型 |
| 本地推理 | Ollama + Qwen2.5 | vLLM |
| 向量数据库 | pgvector (复用 PG) | Milvus / Qdrant |
| RAG 框架 | LangChain Go | 自研 |
| Agent 框架 | Claude Agent SDK | AutoGen |
| MCP | Model Context Protocol | 自定义 |

### 后端扩展

| 组件 | 推荐方案 | 用途 |
|------|---------|------|
| 消息队列 | NATS / Redis Stream | 事件驱动 |
| 日志存储 | Loki | 日志查询 |
| 时序数据库 | VictoriaMetrics | 大规模指标 |
| 工作流引擎 | Temporal | 复杂编排 |

---

## 六、优先级排序（建议执行顺序）

### 第一优先级 — 立刻能做、价值最高

| 序号 | 功能 | 工作量 | 价值 |
|------|------|--------|------|
| 1 | AI 对话台（Copilot） | 2 周 | 替代大量查询点击 |
| 2 | 告警 AI 摘要 | 1 周 | 提升告警处理效率 |
| 3 | 补完 Swagger 文档 | 1 天 | AI Tool 调用的基础 |
| 4 | 工单智能填写 | 1 周 | 降低发起工单门槛 |

### 第二优先级 — 形成 AI 闭环

| 序号 | 功能 | 工作量 | 价值 |
|------|------|--------|------|
| 5 | 运维知识库 (RAG) | 3 周 | 知识沉淀，AI 回答更准 |
| 6 | 根因分析 Agent | 3 周 | 自动诊断，降低 MTTR |
| 7 | 变更风险评估 | 2 周 | 减少变更故障 |
| 8 | 日志接入 | 2 周 | AI 诊断数据源 |

### 第三优先级 — 平台升级

| 序号 | 功能 | 工作量 | 价值 |
|------|------|--------|------|
| 9 | 自动修复 | 4 周 | 无人值守 |
| 10 | MCP Server | 2 周 | 生态打通 |
| 11 | 效能度量 | 3 周 | 管理决策支撑 |
| 12 | 数据库管理 | 4 周 | DBA 自助化 |

---

## 七、AI Copilot 最小闭环设计（推荐首先落地）

### 交互形态

```
┌──────────────────────────────────────────────┐
│  BigOps                              [AI ✨] │
├──────────────────────────────────────────────┤
│                                              │
│  ┌─ AI 助手 ────────────────────────────┐    │
│  │                                      │    │
│  │  🤖 你好，我是 BigOps AI 助手。       │    │
│  │     可以用自然语言操作运维平台。       │    │
│  │                                      │    │
│  │  👤 帮我看看 web-01 的情况            │    │
│  │                                      │    │
│  │  🤖 web-01 当前状态：                 │    │
│  │     CPU: 45.2% | 内存: 72.8%         │    │
│  │     磁盘: 85.3% ⚠️                   │    │
│  │     最近告警: 磁盘使用率 > 80%        │    │
│  │     最近变更: 3h前部署 user-svc v2.1  │    │
│  │                                      │    │
│  │     磁盘使用率较高，建议清理日志。     │    │
│  │     [清理日志] [查看趋势] [创建工单]   │    │
│  │                                      │    │
│  │  ┌──────────────────────────┐ [发送]  │    │
│  │  │ 输入你的问题...           │        │    │
│  │  └──────────────────────────┘        │    │
│  └──────────────────────────────────────┘    │
│                                              │
└──────────────────────────────────────────────┘
```

### 后端 API 设计

```go
// POST /api/v1/ai/chat
type AIChatRequest struct {
    Message       string `json:"message"`       // 用户输入
    ConversationID string `json:"conversation_id"` // 会话ID
    Context       struct {
        CurrentPage string `json:"current_page"` // 当前所在页面
        SelectedIDs []int64 `json:"selected_ids"` // 当前选中的资源
    } `json:"context"`
}

type AIChatResponse struct {
    Reply       string            `json:"reply"`        // AI 回复
    Actions     []AIAction        `json:"actions"`      // 可执行操作
    References  []AIReference     `json:"references"`   // 引用的数据源
    ToolCalls   []AIToolCallLog   `json:"tool_calls"`   // AI 调用了哪些工具
}

type AIAction struct {
    Label  string      `json:"label"`   // "清理日志"
    Type   string      `json:"type"`    // "execute_task" / "create_ticket" / "navigate"
    Params interface{} `json:"params"`  // 执行参数
    NeedConfirm bool   `json:"need_confirm"` // 是否需要用户确认
}
```

### Tool 定义（对接现有模块）

```go
var BigOpsTools = []Tool{
    {Name: "query_asset",       Description: "查询资产信息",     Handler: queryAsset},
    {Name: "query_metrics",     Description: "查询监控指标",     Handler: queryMetrics},
    {Name: "query_alerts",      Description: "查询告警事件",     Handler: queryAlerts},
    {Name: "query_recent_changes", Description: "查询最近变更",  Handler: queryChanges},
    {Name: "query_tickets",     Description: "查询工单",        Handler: queryTickets},
    {Name: "execute_command",   Description: "执行远程命令",     Handler: executeCommand},
    {Name: "create_ticket",     Description: "创建工单",        Handler: createTicket},
    {Name: "trigger_pipeline",  Description: "触发流水线",      Handler: triggerPipeline},
    {Name: "silence_alert",     Description: "静默告警",        Handler: silenceAlert},
    {Name: "search_knowledge",  Description: "搜索运维知识库",  Handler: searchKnowledge},
}
```

---

## 八、竞品对标

| 平台 | AI 能力 | BigOps 差距 |
|------|---------|------------|
| PagerDuty AIOps | 告警降噪、智能分组、根因建议 | 无 AI 能力 |
| Datadog Bits AI | 自然语言查询、自动诊断 | 无 AI 能力 |
| 飞书 Oncall | AI 值班助手、告警摘要 | 无 AI 能力 |
| Shoreline.io | 自动修复、Runbook 自动化 | 已有任务执行基础 |
| Kubiya | AI 运维 Agent、自然语言操作 | 无 AI 能力 |

**BigOps 的差异化优势**：
1. 全栈自研，AI 可以打通全链路（其他产品多是单点 AI）
2. 已有完整执行层（Agent + 任务 + CI/CD），AI 能直接"动手"
3. 面向中小团队，不需要复杂的 MLOps 基础设施

---

## 九、总结

BigOps 当前已具备完整的**运维执行底座**，这是大多数 AI 运维产品最缺的部分。下一步的核心不是"加更多 CRUD 页面"，而是：

1. **让 AI 成为运维入口** — 自然语言替代点击
2. **让 AI 成为诊断引擎** — 自动分析替代人肉排查
3. **让 AI 成为执行者** — 自动修复替代人工操作
4. **让数据驱动决策** — 度量替代感觉

**一句话方向：从 "Tools for Humans" 到 "AI Agents with Human Oversight"。**
