# 同步收尾 + 资产生命周期 + 首页改版 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:executing-plans to implement this plan.

**Goal:** 同步流程收尾（补齐资产时间字段+恢复逻辑）、服务树防环校验、首页改版为平台总览、新增统计接口。

**前置分析 — 已完成项（无需重复）：**

| 计划项 | 状态 | 说明 |
|--------|------|------|
| 1. CloudSyncTask 结构化字段 | ✅ 已完成 | created/updated/unchanged/offline/total_count 均已存在 |
| 2. offline 变更历史 | ✅ 已完成 | markOfflineAssets 已写 AssetChange (field=status, change_type=sync) |
| 3. 统一同步结果文案 | ✅ 已完成 | "同步完成: 新增 X, 更新 X, 无变化 X, 离线 X, 总计 X" |
| 4. 同云账号并发控制 | ✅ 已完成 | per-account sync.Mutex TryLock |
| 9. 删除服务树校验 | ✅ 已完成 | Delete 已校验子节点 + 关联资产 |

**实际需要开发的 7 个任务：**

---

## Task 1: Asset 模型补时间字段 + 同步更新

**Files:**
- Modify: `backend/internal/model/asset.go`
- Modify: `backend/internal/service/cloud_sync/runner.go`

- [ ] **Step 1: Asset 模型新增三个时间字段**

在 `asset.go` 的 `Remark` 字段后、`CreatedAt` 之前添加：

```go
LastSyncAt  *LocalTime `json:"last_sync_at" swaggertype:"string" example:"2024-01-01 00:00:00"`   // 最后同步时间
LastSeenAt  *LocalTime `json:"last_seen_at" swaggertype:"string" example:"2024-01-01 00:00:00"`   // 最后在云端被发现的时间
OfflineAt   *LocalTime `json:"offline_at" swaggertype:"string" example:"2024-01-01 00:00:00"`     // 标记离线的时间
```

- [ ] **Step 2: upsertAssets 中更新 last_seen_at / last_sync_at**

在 `runner.go` 的 `upsertAssets` 函数中：
- 新增资产时设置 `ca.LastSyncAt` 和 `ca.LastSeenAt` 为当前时间
- 已存在资产（无论是否有 diff）都更新 `existing.LastSyncAt` 和 `existing.LastSeenAt`
- **注意**：unchanged 的资产也要更新这两个字段（表示"这次同步看到了它"）

这意味着 unchanged 分支不能直接 `continue`，需要更新时间后再跳过 diff 记录。

- [ ] **Step 3: markOfflineAssets 中设置 offline_at**

在 `markOfflineAssets` 中标记 offline 时，同时设置 `local.OfflineAt` 为当前时间。

- [ ] **Step 4: 验证编译**

```bash
cd /root/bigOps/backend && go build ./cmd/core/
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/model/asset.go backend/internal/service/cloud_sync/runner.go
git commit -m "feat: Asset 补 last_sync_at/last_seen_at/offline_at 时间字段"
```

---

## Task 2: 离线资产恢复为 online 的处理

**Files:**
- Modify: `backend/internal/service/cloud_sync/runner.go`

当一台之前被标记 offline 的资产重新出现在云端（例如机器重启、API 延迟恢复），需要自动恢复为 online。

- [ ] **Step 1: upsertAssets 处理恢复逻辑**

在 upsert 的"已存在"分支中，如果 `existing.Status == "offline"` 但本次云端返回了它：
- 将 `existing.Status` 设为 `"online"`
- 清空 `existing.OfflineAt`（设为 nil）
- 写入变更历史：field=status, old=offline, new=online, change_type=sync

注意：这个状态变更应该被算入 `result.Updated` 而非 `result.Unchanged`。

- [ ] **Step 2: 验证编译**

```bash
cd /root/bigOps/backend && go build ./cmd/core/
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/cloud_sync/runner.go
git commit -m "feat: 离线资产恢复为 online 自动处理 + 变更历史"
```

---

## Task 3: 服务树移动防环校验

**Files:**
- Modify: `backend/internal/service/service_tree_service.go`

当前 Move 只校验 `newParentID == id`，没有检查"移动到自己的子孙节点"的情况。

- [ ] **Step 1: Move 方法增加防环校验**

在 `service_tree_service.go` 的 `Move` 方法中，`newParentID > 0` 之后、更新之前：

```go
// 防环校验：目标父节点不能是当前节点的后代
descendants, err := s.repo.GetAllDescendantIDs(id)
if err == nil {
    for _, did := range descendants {
        if did == newParentID {
            return errors.New("不能移动到自身的子节点下，会形成环")
        }
    }
}
```

`GetAllDescendantIDs` 已存在于 `service_tree_repository.go`。

- [ ] **Step 2: 验证编译**

```bash
cd /root/bigOps/backend && go build ./cmd/core/
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/service_tree_service.go
git commit -m "fix: 服务树移动增加防环校验"
```

---

## Task 4: 抽通用递归查询能力

**Files:**
- Modify: `backend/internal/repository/service_tree_repository.go`
- Check: 确认 `GetAllDescendantIDs` 已被资产查询、统计、删除校验复用

- [ ] **Step 1: 确认 GetAllDescendantIDs 已被复用**

检查以下调用点是否都已使用：
- `asset_repository.go` List() 中 Recursive 分支 → 已使用
- `service_tree_service.go` Delete() → 当前只检查直接子节点（HasChildren），需要改为检查所有后代节点关联的资产
- `service_tree_service.go` Move() → Task 3 新增的防环校验

- [ ] **Step 2: 改进 Delete 校验为递归检查**

当前 Delete 先检查 `HasChildren`（有子节点不允许删），再检查 `CountByServiceTreeID`（单节点资产数）。
如果要删除的是叶子节点，当前逻辑已够用。
如果未来需要支持"连带删除空子树"，再扩展。当前保持不变即可。

- [ ] **Step 3: Commit（如有改动）**

---

## Task 5: 后端统计接口

**Files:**
- Create: `backend/internal/handler/stats_handler.go`
- Modify: `backend/api/http/router/router.go`

当前首页做了 6 次 API 调用取统计数据，改为一个 `/stats/summary` 接口一次返回。

- [ ] **Step 1: 创建 StatsHandler**

```go
// backend/internal/handler/stats_handler.go
package handler

// GET /stats/summary 返回：
type SummaryResponse struct {
    AssetTotal       int64 `json:"asset_total"`
    AssetOnline      int64 `json:"asset_online"`
    AssetOffline     int64 `json:"asset_offline"`
    CloudAccountTotal    int64 `json:"cloud_account_total"`
    CloudAccountFailed   int64 `json:"cloud_account_failed"`  // last_sync_status=failed
    ServiceTreeTotal int64 `json:"service_tree_total"`
    UserTotal        int64 `json:"user_total"`
}

// GET /stats/asset-distribution 返回：
type AssetDistribution struct {
    StatusDist  []DistItem `json:"status_dist"`   // online/offline 分布
    SourceDist  []DistItem `json:"source_dist"`   // manual/aliyun/tencent 分布
    TopServices []TopItem  `json:"top_services"`  // 服务树资产 Top 10
}

type DistItem struct {
    Label string `json:"label"`
    Count int64  `json:"count"`
}

type TopItem struct {
    ID    int64  `json:"id"`
    Name  string `json:"name"`
    Count int64  `json:"count"`
}
```

Summary 接口通过直接 COUNT 查询实现，不经过分页 List。

- [ ] **Step 2: 注册路由**

```go
// --- 统计 ---
statsHandler := handler.NewStatsHandler()
authGroup.GET("/stats/summary", statsHandler.Summary)
authGroup.GET("/stats/asset-distribution", statsHandler.AssetDistribution)
```

- [ ] **Step 3: 验证编译**

```bash
cd /root/bigOps/backend && go build ./cmd/core/
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/stats_handler.go backend/api/http/router/router.go
git commit -m "feat: 统计接口 /stats/summary + /stats/asset-distribution"
```

---

## Task 6: 前端首页改版

**Files:**
- Modify: `frontend/src/views/Dashboard.vue`
- Modify: `frontend/src/api/index.ts`

- [ ] **Step 1: 新增前端 statsApi**

```ts
export const statsApi = {
  summary: () => api.get('/stats/summary'),
  assetDistribution: () => api.get('/stats/asset-distribution'),
}
```

- [ ] **Step 2: Dashboard 改版**

改为平台总览布局：
- 上半区：6 个摘要卡片（2 行 3 列）
  - 主机资产总数（含 online/offline 小字）
  - 云账号数量（含同步异常数标红）
  - 服务树节点数
  - 用户总数
  - 离线资产数（独立卡片，数字标红醒目）
  - 同步异常账号数（独立卡片，数字标红）
- 下半区：两栏
  - 左栏：资产来源分布（简单数据列表或进度条，不强求图表）
  - 右栏：服务树资产 Top 10（横向条形，用 Element Plus 的 el-progress 实现）

移除原来的"最近操作记录"表格主区域。

- [ ] **Step 3: 验证编译**

```bash
cd /root/bigOps/frontend && npm run build
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/index.ts frontend/src/views/Dashboard.vue
git commit -m "feat: 首页改版为平台总览 — 摘要卡片 + 资产分布"
```

---

## Task 7: 全栈验证 + 文档更新

- [ ] **Step 1: 全栈编译验证**

```bash
cd /root/bigOps/backend && go build ./cmd/core/ && go test ./...
cd /root/bigOps/frontend && npm run build
```

- [ ] **Step 2: 更新 progress.md**

记录今日完成内容、新增 API 数、代码行数。

- [ ] **Step 3: 最终 Commit**

---

## 架构决策备忘

| 决策 | 选择 | 理由 |
|------|------|------|
| last_seen_at 更新时机 | 每次同步命中（含 unchanged） | 区分"同步过"和"最后看到" |
| offline 资产处理 | 标记不删除 | 保留历史记录，可恢复 |
| 恢复逻辑 | 云端再次返回自动 online | 无需人工干预 |
| 统计接口 | 独立 COUNT 查询 | 不复用 List 分页，避免无效数据传输 |
| 首页图表 | el-progress 条形 | 不引入 ECharts，保持轻量 |
| 递归查询 | 复用 GetAllDescendantIDs | 已有 BFS 实现，无需重写 |
