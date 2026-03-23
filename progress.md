# BigOps - 开发进度日志

---

## Session: 2026-03-23 (同步收尾 + 部门 + 首页改版 + 标签页 + 表格优化)

### 完成事项

**同步流程增强：**
- 云同步离线收敛 — 本地有但云端未返回的资产标记 offline `c5a1954`
- 资产生命周期时间字段 last_sync_at/last_seen_at/offline_at `31b575b`
- 离线资产重新出现自动恢复 online + 变更历史
- 手动编辑资产记录变更历史（diffAsset + change_type=manual）`6c5b79e`
- 阿里云同步补磁盘信息（DescribeDisks API）+ 详细实例日志 `19cbf0a`
- 同步不更新磁盘 fix（diffAsset 补 disk_gb 对比）`a0e6507`
- 删除后重新同步资产丢失 fix（软删除恢复 Unscoped + RestoreSoftDeleted）`a0e6507`

**首页改版 + 统计接口：**
- 统计接口 /stats/summary + /stats/asset-distribution `f1b1144`
- 首页改版为平台总览（6 摘要卡片 + 来源分布 + 服务树 Top 10）`3b027c4`
- 首页卡片/分布行点击跳转 `0d5c605`
- 首页跳转路径修正（/assets → /cmdb/assets 等）`2af5941`

**轻量版部门功能：**
- Department 模型/Repository/Service/Handler 全栈 `1d0006a`
- 部门 CRUD 6 个 API + 删除校验关联用户
- User 模型新增 department_id + 用户编辑功能
- 新建用户时选部门+角色 `64ffdbc`
- Departments.vue 部门管理页面
- 部门菜单修复（parent name system_dir → system）

**标签页 + 表格体验：**
- 标签页多页签 + 自定义标签样式 `e2b28bc`
- keep-alive 状态保留（切 tab 保留搜索/分页）`88606eb`
- 标签右键菜单（关闭当前/其他/左侧/右侧/全部）`88606eb`
- keep-alive 缓存失效修复（componentName 匹配 + defineOptions）`d11b25a`
- 表格 border 拖拽列宽 + 操作列 min-width + 全局 padding

**规则补严：**
- 服务树移动防环校验 `06cd666`

### 新增文件
| 文件 | 说明 |
|------|------|
| `backend/internal/model/department.go` | 部门模型 |
| `backend/internal/repository/department_repository.go` | 部门 CRUD |
| `backend/internal/service/department_service.go` | 部门业务逻辑 |
| `backend/internal/handler/department_handler.go` | 部门 HTTP handler |
| `backend/internal/handler/stats_handler.go` | 统计接口 |
| `frontend/src/views/Departments.vue` | 部门管理页面 |
| `frontend/src/stores/tagsView.ts` | 标签页状态管理 |
| `frontend/src/composables/useTableResize.ts` | 表格列宽持久化 |

### 当前统计
- 后端代码: ~6868 行 (+865)
- 前端代码: ~3196 行 (+510)
- API 端点: 60 个 (+12)
- 前端页面: 12 个 (+1)
- Commits: 14 个

---

## Session: 2026-03-22 (定时同步系统 + Bug 修复 + 云账号关联服务树)

### 完成事项

**云账号定时同步系统（9 个子任务全部完成）：** `a1ac2ab`
- CloudSyncTask 模型 — 记录每次同步完整生命周期
- CloudSyncTask Repository — CRUD + 分页 + 按账号/状态筛选
- CloudAccount 新增 sync_enabled / sync_interval 字段
- SyncRunner 统一同步执行器 — 从 handler 抽取 200 行逻辑，per-account mutex 防重复
- 重构 handler Sync 方法 → 调用 RunSync()
- Scheduler 定时调度器 — 60s 巡检 + Go ticker + 优雅停止
- 同步日志查询 API (GET /sync-tasks + GET /cloud-accounts/:id/sync-tasks)
- 同步配置接口 (POST /cloud-accounts/:id/sync-config)
- 前端: 编辑表单同步开关 + 周期选择 + 同步记录抽屉 + 定时同步列
- main.go 集成: AutoMigrate CloudSyncTask + 启动 Scheduler

**云账号关联服务树：** `7b94db2`
- CloudAccount 新增 service_tree_id 字段
- 新增/编辑表单加 el-tree-select 服务树选择器
- SyncRunner 同步新资产时自动设置 service_tree_id
- 列表新增"所属服务"列，显示完整路径

**Bug 修复：**
- 菜单更新排序 Error 1292 `045d3ef` — handler 构造新对象零值 CreatedAt → service 层先查 existing
- LocalTime 零值写入 0000-00-00 `76b766a` — Value() 零值返回 nil + 修复数据库 NULL 数据
- 系统管理菜单权限失效 `50afceb` — Layout.vue 硬编码 + 静态路由 → 全部走动态路由
- 系统管理菜单被软删除 — UPDATE menus SET deleted_at = NULL WHERE id = 1
- 资产所属服务显示完整路径 `bba7186`
- ServiceTree badge 递归计数 + 防止点击节点收起 `e7a426e`

### 新增文件
| 文件 | 说明 |
|------|------|
| `backend/internal/model/cloud_sync_task.go` | 同步任务模型 |
| `backend/internal/repository/cloud_sync_task_repository.go` | 同步任务 CRUD |
| `backend/internal/service/cloud_sync/runner.go` | 统一同步执行器 |
| `backend/internal/service/cloud_sync/scheduler.go` | 定时调度器 |
| `backend/internal/handler/cloud_sync_task_handler.go` | 同步日志查询 API |

### 当前统计
- 后端代码: ~6003 行
- 前端代码: ~2686 行
- API 端点: 48 个
- 前端页面: 11 个
- 新增 API: 4 个 (sync-config, sync-tasks x2, cloud-accounts 关联服务树)

### 下一步工作
- 见文末 Next Steps

---

## Session: 2026-03-21 (项目状态审计 + Swagger 完善)

### 完成事项

**Swagger API 文档完善**
- 为全部 21 个 API 端点添加了 Swagger 注解
- 导出所有请求类型 (RegisterRequest, LoginRequest 等) 并添加 example 标签
- 添加 BearerAuth 安全定义到 main.go
- 解决 swag CLI v1.16.6 与 go.mod v1.8.12 不兼容问题 (升级 swaggo/swag)
- 解决 swag 跨包类型引用问题 (model.User → 通过 `var _ model.User` 导入)
- Model LocalTime 字段添加 `swaggertype:"string"` 标签
- 验证 `go build ./cmd/core/` 编译通过
- 生成 docs/ 目录: docs.go, swagger.json, swagger.yaml

**项目状态审计**
- 扫描全部后端代码: 2785 行 (13 个包, 21 个文件)
- 扫描全部前端代码: 729 行 (5 个页面, 1 个 API 模块)
- 运行后端测试: 4 个包 PASS (config/crypto/jwt/logger)
- 核对实施计划文档与实际代码完成度
- 创建 planning-with-files 计划文件体系

### 修改的文件
| 文件 | 操作 |
|------|------|
| `backend/internal/handler/auth_handler.go` | 添加 Swagger 注解 + 导出请求类型 + model 导入 |
| `backend/internal/handler/user_handler.go` | 添加 Swagger 注解 + 导出请求类型 + model 导入 |
| `backend/internal/handler/rbac_handler.go` | 添加 Swagger 注解 + 导出请求类型 |
| `backend/cmd/core/main.go` | 添加 securityDefinitions + docs 导入 |
| `backend/internal/model/menu.go` | LocalTime 字段添加 swaggertype 标签 |
| `backend/go.mod` / `backend/go.sum` | 升级 swaggo/swag v1.8.12 → v1.16.6 |
| `backend/docs/*` | 新生成 swagger 文档 |

### 当前阻塞项
- 无

### 下一步工作
- P0: 操作审计日志 (model + repository + handler + 前端页面)
- P0: Pinia stores 初始化 (userStore + permissionStore)
- P1: 前端动态菜单 + 动态路由改造
- P1: v-permission 权限指令

---

## 项目总体进度

```
Phase 1: 基础设施搭建     [████████████████████] 100%
Phase 2: Module 01 底座    [███████████████████░]  97%
  - 认证                   [████████████████████] 100%
  - RBAC + 权限指令         [████████████████████] 100%
  - 菜单管理 + 动态菜单     [████████████████████] 100%
  - 用户管理 + 编辑         [████████████████████] 100%  ← 新增编辑/部门
  - 部门管理               [████████████████████] 100%  ← NEW
  - 布局 + 标签页           [███████████████████░]  95%  ← 多页签+keep-alive
  - 审计日志               [████████████████████] 100%
  - Swagger 文档           [████████████████████] 100%
  - Pinia 状态管理          [████████████████████] 100%
Phase 3: Module 02 CMDB    [███████████████████░]  97%
  - 服务树                 [████████████████████] 100%  ← 防环校验
  - 云账号                 [████████████████████] 100%
  - 定时同步系统            [████████████████████] 100%
  - 离线收敛 + 恢复         [████████████████████] 100%  ← NEW
  - 资产管理               [████████████████░░░░]  85%  (缺 Excel 导入导出/批量)
  - 阿里云同步             [████████████████████] 100%  ← 补磁盘
  - 变更历史               [████████████████████] 100%  ← 手动编辑也记录
  - 统计接口               [████████████████████] 100%  ← NEW
  - 首页总览               [████████████████████] 100%  ← NEW
Phase 4: Module 03 工单    [░░░░░░░░░░░░░░░░░░░░]   0%
Phase 5: Module 04 任务    [░░░░░░░░░░░░░░░░░░░░]   0%
Phase 6: Module 05 监控    [░░░░░░░░░░░░░░░░░░░░]   0%
Phase 7: Module 08 数据库  [░░░░░░░░░░░░░░░░░░░░]   0%
```

---

## Next Steps

| 优先级 | 任务 | 说明 |
|--------|------|------|
| P1 | Module 03 工单系统 | 工单类型/创建/流转/审批 — 平台最缺的运维流程能力 |
| P2 | 资产导入/导出 Excel | excelize 后端 + 前端上传/下载，CMDB 数据迁移必备 |
| P2 | 资产批量操作 | 批量修改服务树/批量删除 |
| P2 | 腾讯云/AWS 同步 | 扩展 CloudProvider 接口 |
| P3 | Module 04 任务执行中心 | Agent + gRPC + 远程执行 |
