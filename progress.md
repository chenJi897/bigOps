# BigOps - 开发进度日志

---

## Session: 2026-03-21 (Module 01 前端完善 + Module 02 全栈完成)

### 完成事项

**Module 01 前端闭环：**
- 动态路由: 从后端菜单树自动生成 Vue Router 路由 `7134056`
- 动态菜单: Layout 从 permissionStore 渲染侧边栏
- v-permission 指令: 按钮级权限控制 (`directives/permission.ts`)
- 面包屑导航
- Pinia stores 已就绪 (userStore + permissionStore)
- 审计日志前端页面 (`views/AuditLogs.vue`)
- Dashboard + 404 页面
- 菜单数据初始化 (9 个菜单项, admin 角色全部分配)

**Module 02 前端页面：**
- 服务树页面: 左树右资产联动 + 节点 badge + 增删改 `68352ec`
- 云账号页面: 列表 + 同步 + 密钥更新 + 增删改 `7134056`
- 资产列表页: 多条件筛选 + 详情抽屉(含变更历史 tab) + 增删改 `7134056`

**服务树与资产关联闭环（两批）：**
- 第一批: 删除校验 + 表单选择器 + 列表展示 `6804c23`
- 第二批: 递归查子节点 + 资产数量统计 API + 联动资产列表 + 跳转筛选 `68352ec`

**Bug 修复：**
- AES key 非法 hex 字符 → 替换为合法 key `6deee1f`
- ServiceTree code uniqueIndex 空串冲突 → 改为普通 index `6deee1f`
- Dashboard 路由 warning → 子路由去掉前导 / `6deee1f`
- 前端 sync_status 字段名与后端不匹配 → last_sync_status `6deee1f`
- GORM SQL 日志刷屏 → 降为 Error 级别 `6deee1f`
- 云同步缺日志 → AliyunProvider 全链路日志 `6deee1f`
- 阿里云 endpoint 硬编码 → 写入 config.yaml 可配置 `6deee1f`
- last_sync_message varchar(500) 放不下 → TEXT `65629cb`
- sync upsert 覆盖 CreatedAt/Tags → 在 existing 上更新 `60ff9d1`
- Asset IDC 列名 / tags JSON 空值 → gorm tag 修复 `60ff9d1`

### 当前 API 总数: 42 个端点
### 当前前端页面: 12 个

### 下一步工作
- P1: 标签页多页签、主题切换
- P2: 资产导入/导出 Excel、批量操作
- P3: Module 03 工单系统

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
Phase 2: Module 01 底座    [██████████████████░░]  90%
  - 认证                   [████████████████████] 100%
  - RBAC + 权限指令         [████████████████████] 100%
  - 菜单管理 + 动态菜单     [████████████████████] 100%
  - 用户管理               [████████████████████] 100%
  - 布局                   [████████████████░░░░]  80%  (缺标签页/主题切换)
  - 审计日志               [████████████████████] 100%
  - Swagger 文档           [████████████████████] 100%
  - Pinia 状态管理          [████████████████████] 100%
Phase 3: Module 02 CMDB    [██████████████████░░]  90%
  - 服务树                 [████████████████████] 100%
  - 云账号                 [████████████████████] 100%
  - 资产管理               [████████████████░░░░]  80%  (缺 Excel 导入导出/批量)
  - 阿里云同步             [████████████████████] 100%
  - 变更历史               [████████████████████] 100%
Phase 4: Module 03 工单    [░░░░░░░░░░░░░░░░░░░░]   0%
Phase 5: Module 04 任务    [░░░░░░░░░░░░░░░░░░░░]   0%
Phase 6: Module 05 监控    [░░░░░░░░░░░░░░░░░░░░]   0%
Phase 7: Module 08 数据库  [░░░░░░░░░░░░░░░░░░░░]   0%
```
