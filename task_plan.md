# Task Plan: BigOps 运维平台全模块实现

## Goal
构建完整的 BigOps 运维平台，覆盖 7 大模块：底座、CMDB、工单、任务执行中心、监控、CI/CD、知识库。

## Current Phase
Phase 5（待启动）

## Phases

### Phase 1: 基础设施 + 底座 (Module 01)
- [x] Go + Gin + GORM 后端骨架
- [x] Vue 3 + Element Plus 前端骨架
- [x] JWT 认证 + Redis token 黑名单
- [x] RBAC: 用户/角色/菜单管理
- [x] Casbin API 级别权限控制（已启用白名单 + 角色菜单同步）
- [x] Zap 日志 + 请求日志中间件
- [x] 审计日志
- [x] 部门管理
- [x] 登录限流 + 失败锁定 + 分页 size 上限
- [x] Swagger 文档
- **Status:** complete

### Phase 2: CMDB (Module 02)
- [x] 服务树（树形 CRUD + 拖拽排序）
- [x] 云账号管理（AES 加密 AK/SK）
- [x] 资产管理（主机 CRUD + 变更历史）
- [x] 阿里云 ECS 自动同步（SyncRunner + Scheduler + 软删除恢复）
- [x] 资产负责人 + 云账号负责人
- [x] Dashboard 统计
- **Status:** complete (缺 Excel 导入导出、批量操作)

### Phase 3: 工单系统 (Module 03)
- [x] 工单 CRUD + 完整状态机（open→processing→resolved→closed）
- [x] 工单类型 + 请求模板（动态表单 schema）
- [x] 通用资源关联（asset/cloud_account）
- [x] 多级审批（ApprovalPolicy → ApprovalInstance → ApprovalRecord）
- [x] 通知中心（Webhook/Email/MessagePusher + 重试机制）
- [x] 审批待办 + 站内通知
- **Status:** complete

### Phase 4: 任务执行中心 (Module 04)
- [x] gRPC Proto 定义 + 代码生成
- [x] Server 端: gRPC Server + Agent 连接管理 + WebSocket 日志推送
- [x] Agent 端: gRPC 客户端 + 命令执行器 + 超时控制
- [x] 任务 CRUD + 执行下发 + 实时日志
- [x] 前端: TaskList / TaskCreate / TaskExecution / AgentList
- [x] 安全审查修复（WebSocket 认证、env 继承、原子 DB 追加、断线重连）
- **Status:** complete

### Phase 5: 监控系统 (Module 05)
- [ ] Prometheus 指标采集集成
- [ ] 告警规则管理
- [ ] 告警通知（复用通知中心）
- [ ] 监控大盘 + 图表
- [ ] Agent 指标上报扩展
- **Status:** pending

### Phase 6: CI/CD (Module 06)
- [ ] 项目管理（Git 仓库关联）
- [ ] 构建流水线定义
- [ ] 部署任务（复用任务执行中心）
- [ ] 发布审批（复用工单审批）
- [ ] 构建日志 + 部署日志
- **Status:** pending

### Phase 7: 知识库 (Module 07)
- [ ] Markdown 文档 CRUD
- [ ] 知识分类 + 标签
- [ ] 全文搜索
- [ ] 运维手册模板
- **Status:** pending

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| Go + Gin + GORM | 高性能、强类型、运维工具标准栈 |
| Vue 3 + Element Plus | 企业级 Admin UI，组件丰富 |
| gRPC 双向流心跳 | Agent 长连接 + 任务下发 + 实时日志 |
| Casbin RBAC | 模型灵活，支持 keyMatch2 路径匹配，admin bypass |
| HTTP: 只用 GET/POST | 简化前端调用，DELETE 用 POST /:id/delete |
| LocalTime 自定义类型 | JSON 时间格式统一 2006-01-02 15:04:05 |
| Redis 限流 + 锁定 | IP 限流 + 账号锁定，Redis 故障时降级放行 |

## Architecture

```
前端 (Vue 3 + Element Plus)
  ↓ HTTP /api/v1
后端 (Gin)
  ├── middleware: Auth → AuditLog → Casbin → RateLimit
  ├── handler → service → repository → MySQL (GORM)
  ├── gRPC Server :9090 ←→ Agent (Heartbeat + Task)
  └── Redis (token 黑名单 + 限流 + 锁定)
```

## Metrics

| 指标 | 数值 |
|------|------|
| 后端代码 | ~18,500 行 Go |
| 前端代码 | ~7,400 行 Vue/TS |
| API 端点 | 82 个 (Swagger) |
| 前端页面 | 24 个 Vue 组件 |
| 数据库模型 | 22 个 GORM 模型 |
| 模块完成 | 4/7 |
