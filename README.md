# BigOps 大运维平台

<div align="center">

**面向中小团队的综合运维管理平台**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org/)
[![Vue Version](https://img.shields.io/badge/vue-3.4+-green.svg)](https://vuejs.org/)

</div>

## 📖 项目简介

BigOps 是一个面向中小团队（50人以下）的综合运维管理平台，提供资产管理、任务执行、监控告警、工单管理等核心功能，旨在提升运维效率，降低运维成本。

### 核心特性

- 🔐 **统一认证授权** - 基于JWT的认证系统，Casbin RBAC权限控制
- 📦 **服务树和CMDB** - 灵活的服务树管理，完善的资产管理
- 📝 **自助工单系统** - 可配置的工单类型，灵活的审批流程
- 🚀 **任务执行中心** - 远程任务执行，实时日志输出
- 📊 **监控告警平台** - Prometheus集成，可视化监控大盘
- 💾 **数据库管理** - SQL执行，慢查询分析，数据库管理

## 🏗️ 技术架构

### 后端技术栈

- **框架**: Gin v1.9+ (Web框架)
- **数据库**: MySQL 8.0+ (关系型数据库)
- **缓存**: Redis 7.0+ (缓存和任务队列)
- **ORM**: GORM v2 (对象关系映射)
- **认证**: JWT (golang-jwt/jwt v5)
- **权限**: Casbin v2 (RBAC权限控制)
- **任务队列**: Asynq v0.24+ (异步任务)
- **RPC**: gRPC v1.60+ (服务通信)
- **配置**: Viper v1.18+ (配置管理)
- **日志**: Zap v1.26+ (结构化日志)
- **文档**: Swag (Swagger文档生成)

### 前端技术栈

- **框架**: Vue 3.4+ (渐进式框架)
- **构建工具**: Vite 5.0+ (下一代前端工具)
- **UI组件**: Element Plus v2.5+ (企业级UI库)
- **状态管理**: Pinia v2.1+ (状态管理)
- **路由**: Vue Router v4.2+ (路由管理)
- **HTTP**: Axios v1.6+ (HTTP客户端)
- **图表**: ECharts v5.4+ (数据可视化)
- **编辑器**: Monaco Editor (代码编辑器)
- **语言**: TypeScript (类型系统)

### 部署架构

- **容器化**: Docker + Docker Compose
- **反向代理**: Nginx
- **混合架构**: 核心模块单体 + 独立模块容器化

## 📦 功能模块

### 模块01：前后端底座
- 用户管理（增删改查、状态管理）
- 角色管理（角色权限分配）
- 权限管理（RBAC权限控制）
- 菜单管理（动态菜单加载）
- 操作日志（审计追溯）

### 模块02：服务树和CMDB
- 服务树管理（树形结构、节点管理）
- 资产管理（服务器、网络设备）
- 资产发现（自动扫描、信息采集）
- 变更历史（变更记录、审计追溯）

### 模块03：自助工单
- 工单管理（创建、流转、关闭）
- 工单类型（自定义字段、工单模板）
- 审批流程（多级审批、审批意见）
- 通知功能（邮件、站内信）

### 模块04：任务执行中心
- 任务管理（脚本执行、命令执行、文件分发）
- Agent管理（心跳检测、状态监控）
- 实时日志（WebSocket实时输出）
- 执行控制（并发控制、超时控制）

### 模块05：Prometheus监控平台
- 数据源管理（多数据源支持）
- 监控查询（PromQL查询、图表展示）
- 告警管理（告警规则、告警通知）
- 监控大盘（可配置面板）

### 模块08：数据库和SQL管理
- 数据源管理（连接信息加密）
- SQL执行（查询、执行计划）
- SQL审核（规则检查、危险操作拦截）
- 数据库管理（库表结构、数据浏览）

## 🚀 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+
- Docker 20.10+
- Docker Compose 2.0+
- MySQL 8.0+
- Redis 7.0+

### 本地开发

#### 1. 克隆项目

```bash
git clone https://github.com/your-org/bigops.git
cd bigops
```

#### 2. 启动基础服务

```bash
cd deploy
docker-compose up -d mysql redis
```

#### 3. 后端开发

```bash
cd backend

# 安装依赖
go mod download

# 复制配置文件
cp config/config.yaml.example config/config.yaml

# 修改配置文件（数据库、Redis等）
vim config/config.yaml

# 运行数据库迁移
go run cmd/core/main.go migrate

# 创建管理员账号
go run cmd/core/main.go admin create --username admin --password admin123

# 启动核心模块
go run cmd/core/main.go

# 启动任务中心（另开终端）
go run cmd/task/main.go

# 启动监控平台（另开终端）
go run cmd/monitor/main.go

# 启动数据库管理（另开终端）
go run cmd/dbmgr/main.go
```

#### 4. 前端开发

```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 访问 http://localhost:5173
```

### Docker部署

#### 1. 构建前端

```bash
cd frontend
npm install
npm run build
```

#### 2. 启动所有服务

```bash
cd deploy

# 复制环境变量
cp .env.example .env

# 修改环境变量
vim .env

# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 查看服务状态
docker-compose ps
```

#### 3. 初始化数据

```bash
# 进入核心模块容器
docker exec -it bigops-core sh

# 执行数据库迁移
./bigops-core migrate

# 创建管理员账号
./bigops-core admin create --username admin --password admin123
```

#### 4. 访问系统

- 前端地址: http://localhost
- API文档: http://localhost/swagger/index.html
- 默认账号: admin / admin123

## 📚 文档

详细文档请查看 [docs](./docs) 目录：

- [技术选型方案](./docs/01-技术选型方案.md)
- [模块设计方案](./docs/02-模块设计方案.md)
- [数据库设计方案](./docs/03-数据库设计方案.md)
- [部署架构方案](./docs/04-部署架构方案.md)
- [实施计划](./docs/05-实施计划.md)
- [项目目录结构](./docs/06-项目目录结构.md)

## 🗂️ 项目结构

```
bigOps/
├── backend/                # 后端代码
│   ├── cmd/               # 应用程序入口
│   ├── internal/          # 内部代码
│   ├── api/               # API定义
│   ├── config/            # 配置文件
│   └── migrations/        # 数据库迁移
├── frontend/              # 前端代码
│   ├── src/               # 源代码
│   └── public/            # 静态资源
├── deploy/                # 部署文件
│   ├── docker-compose.yml # Docker Compose配置
│   └── nginx/             # Nginx配置
└── docs/                  # 文档
```

详细目录结构请查看 [项目目录结构](./docs/06-项目目录结构.md)

## 🔧 开发指南

### 后端开发

#### 添加新接口

1. 在 `internal/model/` 定义数据模型
2. 在 `internal/repository/` 实现数据访问
3. 在 `internal/service/` 实现业务逻辑
4. 在 `internal/handler/` 实现HTTP处理器
5. 在 `api/http/router/` 注册路由

#### 代码规范

```bash
# 格式化代码
go fmt ./...

# 代码检查
golangci-lint run

# 运行测试
go test ./...
```

### 前端开发

#### 添加新页面

1. 在 `src/views/` 创建页面组件
2. 在 `src/api/` 添加API请求
3. 在 `src/router/` 注册路由
4. 在 `src/stores/` 添加状态管理（如需要）

#### 代码规范

```bash
# 代码检查
npm run lint

# 代码格式化
npm run format

# 类型检查
npm run type-check
```

## 🧪 测试

### 后端测试

```bash
cd backend

# 运行所有测试
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 前端测试

```bash
cd frontend

# 运行单元测试
npm run test:unit

# 运行E2E测试
npm run test:e2e
```

## 📊 性能指标

### 后端性能

- API响应时间: < 100ms (P95)
- 并发支持: 1000+ QPS
- 内存占用: < 500MB (单模块)

### 前端性能

- 首屏加载: < 2s
- 页面切换: < 500ms
- 构建大小: < 2MB (gzip)

## 🔒 安全

### 认证安全

- 密码使用Bcrypt加密（cost=10）
- JWT Token签名验证
- Token黑名单机制
- 登录失败次数限制

### 权限安全

- Casbin RBAC权限控制
- API级别权限验证
- 数据级别权限过滤
- 操作审计日志

### 通信安全

- 生产环境强制HTTPS
- 敏感数据AES加密
- SQL注入防护
- XSS防护

## 🤝 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

### 提交规范

遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建/工具相关

## 📝 更新日志

查看 [CHANGELOG.md](./CHANGELOG.md) 了解版本更新历史。

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](./LICENSE) 文件了解详情。

## 👥 团队

- 项目负责人: [@your-name](https://github.com/your-name)
- 后端开发: [@backend-dev](https://github.com/backend-dev)
- 前端开发: [@frontend-dev](https://github.com/frontend-dev)

## 📮 联系我们

- 问题反馈: [GitHub Issues](https://github.com/your-org/bigops/issues)
- 邮件: bigops@example.com
- 文档: [在线文档](https://docs.bigops.example.com)

## 🙏 致谢

感谢以下开源项目：

- [Gin](https://github.com/gin-gonic/gin) - Go Web框架
- [Vue](https://github.com/vuejs/vue) - 渐进式JavaScript框架
- [Element Plus](https://github.com/element-plus/element-plus) - Vue 3 UI组件库
- [GORM](https://github.com/go-gorm/gorm) - Go ORM框架
- [Casbin](https://github.com/casbin/casbin) - 权限管理框架

## ⭐ Star History

如果这个项目对你有帮助，请给我们一个 Star ⭐

---

<div align="center">

**Built with ❤️ by BigOps Team**

</div>
