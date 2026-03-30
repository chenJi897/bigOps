# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

优先使用augment-context-engine命令进行检索代码。如果augment-context-engine中没有找到相关信息，再尝试使用本地文件系统的工具作为兜底。   

## 大文件写入规则

当需要写入超过 500 行的文件时，禁止使用 Write 工具，改用 Bash 的 cat heredoc 分段写入。

## Repository overview

BigOps is a full-stack operations platform with a Go backend (`backend/`) and a Vue 3 + Vite frontend (`frontend/`).

The current implemented foundation is concentrated in the backend core module:
- authentication with JWT
- RBAC with Casbin
- user / role / menu management APIs
- Zap-based application and HTTP request logging
- Swagger route integration at `/swagger/index.html`

## Common commands

### Backend

All backend commands assume you are in `/root/bigOps/backend`.

```bash
# install deps
go mod download

# run core service
go run ./cmd/core/main.go

# build core binary
go build -o bin/bigops-core ./cmd/core

# use the helper script
./scripts/dev.sh build
./scripts/dev.sh run
./scripts/dev.sh test
./scripts/dev.sh clean

# run all backend tests
go test ./...

# run a single package test
go test ./internal/pkg/logger -v
go test ./internal/pkg/jwt -v
go test ./internal/pkg/config -v

# run a single test function
go test ./internal/pkg/crypto -run TestHashAndCompare -v

# regenerate module metadata
go mod tidy

# generate Swagger docs (requires swag installed)
swag init -g cmd/core/main.go -o docs/swagger
```

### Frontend

All frontend commands assume you are in `/root/bigOps/frontend`.

```bash
# install deps
npm install

# start dev server
npm run dev

# build production bundle
npm run build

# preview production build
npm run preview
```

### Local development dependencies

The backend expects:
- MySQL
- Redis

The active backend config is in `backend/config/config.yaml`.
The frontend uses Vite and talks to the backend through `/api/v1`.

## High-level architecture

### Backend startup flow

The main entrypoint is `backend/cmd/core/main.go`.

Startup order:
1. load config via `internal/pkg/config`
2. initialize Zap logger via `internal/pkg/logger`
3. initialize MySQL via `internal/pkg/database`
4. auto-migrate current GORM models (`User`, `Role`, `Menu`, `UserRole`)
5. initialize Casbin using the same GORM connection
6. initialize Redis
7. build Gin router via `api/http/router`
8. start HTTP server and wait for termination signals

This means most backend work plugs into the existing boot sequence by adding models, repositories, services, handlers, and then registering routes.

### Backend layering

The backend follows a conventional layered structure under `backend/internal/`:

- `model/`: GORM models and shared data types
- `repository/`: database access and persistence operations
- `service/`: business rules and orchestration
- `handler/`: Gin HTTP handlers and request/response boundary
- `middleware/`: auth, Casbin authorization, request logging
- `pkg/`: infrastructure and reusable internals (config, database, logger, jwt, crypto, casbin, response)

The intended request flow is:

`router -> middleware -> handler -> service -> repository -> database`

### Router and middleware model

`backend/api/http/router/router.go` is the central route registry.

Important router behaviors:
- uses `gin.New()` rather than `gin.Default()`
- request logging is handled by custom `middleware.GinLogger()` instead of Gin’s default logger
- `NoRoute` returns the project’s standard JSON 404 response
- `NoMethod` returns the project’s standard JSON 405 response
- authenticated APIs are grouped under `authGroup.Use(middleware.AuthMiddleware())`

Current route groups are organized functionally rather than by versioned subfiles, so changes usually happen in this one router file.

### Response contract

All handlers should use `backend/internal/pkg/response/response.go`.

The API contract is business-code based:
- success => `code: 0`
- error => non-zero `code`
- payload is wrapped under `data`

Even 400/401/403/404/500-style helpers ultimately return the project’s standard JSON shape. Handlers should stay consistent with that rather than returning ad hoc JSON.

### Auth and RBAC

Authentication is implemented in `internal/handler/auth_handler.go` + `internal/service/auth_service.go`.

Key points:
- JWT auth uses `internal/pkg/jwt`
- passwords are hashed with bcrypt in `internal/pkg/crypto`
- token blacklist is stored in Redis for logout invalidation
- password complexity is enforced in the auth service
- current auth endpoints use only GET/POST routes

Authorization is implemented with Casbin:
- Casbin initialization is in `internal/pkg/casbin/casbin.go`
- role/menu assignment lives in `role_service.go`
- menu visibility for users is derived from role-menu associations
- `middleware.CasbinMiddleware()` exists for API-level authorization

Important detail: user-to-role mapping in Casbin uses the real username, not the numeric user ID.

### Data model shape

The currently active platform foundation models are:
- `User`
- `Role`
- `Menu`
- `UserRole`

`Role` and `Menu` support RBAC and dynamic menu tree rendering.
`Menu` is also used as a permission carrier for API access.

Time fields in models use the custom `LocalTime` type in `backend/internal/model/local_time.go`, so JSON timestamps serialize as:

```text
2006-01-02 15:04:05
```

If you add new API-facing models with timestamps, use `LocalTime` instead of raw `time.Time` when you want consistency with existing responses.

### Logging model

There are two distinct logging paths:
- application / business logs: `internal/pkg/logger`
- HTTP access logs: `internal/middleware/logger.go`

Both end up in Zap, and file output is controlled by `config.yaml` (`log.filename`, rotation settings, etc.).

Important existing behavior:
- request logs are written with status / method / path / latency / IP
- write operations in handlers already emit audit-style logs (login, role changes, menu changes, status changes, delete actions, etc.)
- log timestamps were intentionally formatted as `YYYY-MM-DD HH:MM:SS`

### Swagger state

Swagger is wired into the router and main entrypoint, and handlers already contain Swag annotations for the implemented APIs.

To view full docs, docs must be generated first:

```bash
cd /root/bigOps/backend
swag init -g cmd/core/main.go -o docs/swagger
```

Then start the backend and visit:

```text
http://localhost:8080/swagger/index.html
```

### Frontend shape

The frontend is a simple Vue 3 + TypeScript admin shell using:
- Element Plus
- Axios
- Vue Router
- Pinia

The API wrapper is `frontend/src/api/index.ts`.
It centralizes:
- token injection from `localStorage`
- handling of the project’s `code !== 0` response convention
- redirect to `/login` on 401-style business responses

The current page shell is:
- `views/Layout.vue`
- `views/Login.vue`
- `views/Users.vue`
- `views/Roles.vue`
- `views/Menus.vue`

The frontend assumes the backend API prefix is `/api/v1`.

## Current implementation boundaries

A lot of repository docs describe future modules (task center, monitor, CMDB, etc.), but the codebase currently has the strongest implementation in the platform foundation module only.

When working in this repo, prefer the actual code under `backend/internal`, `backend/api/http/router`, and `frontend/src` over roadmap-style descriptions in docs.

## File patterns to follow

When adding backend capability, the existing pattern is:
1. add/extend `model`
2. implement `repository`
3. implement `service`
4. add `handler`
5. register routes in `api/http/router/router.go`
6. if needed, wire infra in `cmd/core/main.go`

When returning API data, reuse `response.Success`, `response.Error`, and `response.Page` rather than returning raw `gin.H` for business endpoints.
