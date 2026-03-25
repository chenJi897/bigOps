# Progress: BigOps 开发进度日志

## Session: 2026-03-25 (Module 04 + 安全修复)

### 完成事项

**Module 04 任务执行中心 — 全栈实现**
- [x] gRPC Proto 定义 + protoc 代码生成 (agent.proto → agent.pb.go + agent_grpc.pb.go)
- [x] 4 个 GORM 模型: Task, TaskExecution, TaskHostResult, AgentInfo
- [x] 3 个 Repository: TaskRepository, TaskExecutionRepository, AgentRepository
- [x] TaskService: CRUD + ExecuteTask 下发 + GetExecution
- [x] TaskHandler: 10 个 API + WebSocket 实时日志
- [x] gRPC Server: AgentManager 连接管理 + Heartbeat + ReportOutput
- [x] Agent 二进制: gRPC 客户端 + 命令执行器 + 超时控制
- [x] 前端 4 页面: TaskList, TaskCreate, TaskExecution, AgentList
- [x] 菜单自动初始化 (seedTaskMenus, 幂等)
- [x] Swagger 文档更新 (+10 API)

**代码审查修复**
- [x] WebSocket 认证: AuthMiddleware 支持 query param token
- [x] WebSocket 断线重连 (3s 间隔 + destroyed flag)
- [x] executor env 继承: `cmd.Env = os.Environ()` + extras
- [x] ReportOutput 原子追加: GORM CONCAT 表达式
- [x] ListExecutions taskID=0 列出全部
- [x] agent_manager 清理未使用 json.Marshal

**安全加固**
- [x] 启用 Casbin API 权限控制 (白名单 + syncCasbinPolicies)
- [x] task_service.go 5 处 `_ =` → logger.Warn
- [x] UserHandler → UserService 重构 (消除 N+1 部门查询)
- [x] 注册限流: 每 IP 每分钟 3 次
- [x] 登录限流: 每 IP 每分钟 10 次 + 5 次失败锁定 15 分钟
- [x] 分页 size 上限: parsePageSize() 全局 max=100

### Commits
```
2e9ba9c fix: 注册/登录限流 + 登录锁定 + 分页 size 上限
a031299 feat: 启用 Casbin API 权限控制 + 错误处理 + UserService 重构
3ca9694 docs: Swagger 文档更新 — 新增任务中心 10 个 API
57dda89 fix: 代码审查修复 4 项问题
033f7e1 feat: 任务中心菜单自动初始化（幂等 seed）
de7aa92 fix: WebSocket 认证 + 断线重连
a2f73ad feat: Module 04 任务执行中心 — 全栈实现
```

### 文件变更统计
- 新增文件: 20+
- 修改文件: 15+
- 代码增量: ~4,000 行 (Go + Vue)

---

## Next Steps (Phase 5+)
1. Module 05: 监控系统 (Prometheus 集成 + 告警)
2. CMDB 补充: Excel 导入导出 + 批量操作
3. 前端优化: 暗色主题、国际化
