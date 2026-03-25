# Findings: BigOps 技术发现与经验

## 已解决的技术问题

### GORM 相关
| 问题 | 原因 | 解决方案 |
|------|------|----------|
| 缩写字段列名不匹配 | GORM 默认 snake_case 转换对缩写处理不一致 | 添加 `gorm:"column:xxx"` tag（如 IDC → idc） |
| JSON 字段空值报错 | MySQL JSON 列不接受空字符串 | BeforeSave hook 将 `""` 转为 `"[]"` 或 `"{}"` |
| LocalTime 零值更新丢失 | 构造新对象 Save() 会覆盖零值时间 | 先查 existing 再改字段 |
| 软删除 + uniqueIndex 冲突 | 同步已删除记录时违反唯一索引 | Unscoped 查找已删除记录并恢复 |
| stdout/stderr 并发追加覆盖 | 多 goroutine 同时 Save() 互相覆盖 | 使用 `gorm.Expr("CONCAT(COALESCE(col,''), ?)")` 原子追加 |

### 前端相关
| 问题 | 原因 | 解决方案 |
|------|------|----------|
| keep-alive 缓存失效 | include 匹配的是组件名，不是路由名 | `defineOptions({ name: 'XxxComponent' })` |
| 菜单路径跳转 404 | 前端路径和数据库菜单路径不一致 | 数据库路径含模块前缀（/cmdb/assets），前端跳转需匹配 |
| WebSocket 认证失败 | AuthMiddleware 只读 Header，WS 无法设 Header | 添加 `c.Query("token")` 降级读取 |
| el-tag type 类型报错 | Element Plus TS 类型严格 | 使用 `(xxx as any)` 类型断言 |

### 安全相关
| 问题 | 风险 | 解决方案 |
|------|------|----------|
| 注册/登录无限流 | 暴力注册/破解密码 | Redis IP 限流 + 账号失败锁定 |
| 分页 size 无上限 | 传 size=999999 造成 OOM | `parsePageSize()` 全局限制 max=100 |
| executor cmd.Env 丢失 PATH | 设置 Env 后丢失系统 PATH | `cmd.Env = os.Environ()` 先继承 |
| Casbin 未启用 | 所有认证用户可访问任何 API | 启用中间件 + 白名单 + 启动同步 |
| 关键错误 `_ =` 忽略 | 数据不一致无法排查 | 替换为 `logger.Warn()` |

### 架构模式
| 发现 | 说明 |
|------|------|
| 云同步架构 | SyncRunner 统一入口 + Scheduler 60s 巡检 + per-account mutex |
| Agent 通信 | gRPC 双向流心跳 → Server 通过 HeartbeatResponse.Task 下发任务 |
| WebSocket 日志 | AgentManager pub/sub 模式，channel 满时非阻塞跳过 |
| Casbin 同步时机 | SetMenus() 和 SetUserRoles() 时自动同步 + 启动时全量同步 |
| 菜单驱动路由 | 后端菜单树 → 前端 generateRoutes → 动态路由 + companion 隐藏路由 |

## 代码模式参考

### 后端分层标准流程
```
1. model/xxx.go        — GORM 模型 + TableName() + BeforeSave()
2. repository/xxx.go   — DB 操作 + 分页 + 批量查询
3. service/xxx.go      — 业务逻辑 + 批量填充关联名称
4. handler/xxx.go      — 参数绑定 + Swagger + 审计日志 + response
5. router.go           — 注册路由 (GET 读 / POST 写)
6. main.go             — AutoMigrate + seed + 启动
```

### 前端页面标准模式
```vue
<script setup lang="ts">
defineOptions({ name: 'XxxPage' })  // keep-alive 必须
// imports, refs, fetchData, handlers
onMounted(() => { fetchData() })
</script>
<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 筛选 → 表格 → 分页 -->
    </el-card>
  </div>
</template>
<style scoped>
.page { padding: 20px; }
</style>
```
