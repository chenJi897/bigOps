<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { userApi, cloudAccountApi, assetApi, auditLogApi } from '../api'

const stats = ref({ assets: 0, cloudAccounts: 0, users: 0, auditLogs: 0 })
const recentLogs = ref<any[]>([])
const loading = ref(false)
const onlineAssets = ref(0)
const offlineAssets = ref(0)

async function fetchStats() {
  loading.value = true
  try {
    const [userRes, cloudRes, assetRes, assetOnlineRes, assetOfflineRes, logRes]: any[] = await Promise.allSettled([
      userApi.list(1, 1),
      cloudAccountApi.list(1, 1),
      assetApi.list({ page: 1, size: 1 }),
      assetApi.list({ page: 1, size: 1, status: 'online' }),
      assetApi.list({ page: 1, size: 1, status: 'offline' }),
      auditLogApi.list({ page: 1, size: 5 }),
    ])
    if (userRes.status === 'fulfilled') stats.value.users = userRes.value?.data?.total ?? 0
    if (cloudRes.status === 'fulfilled') stats.value.cloudAccounts = cloudRes.value?.data?.total ?? 0
    if (assetRes.status === 'fulfilled') stats.value.assets = assetRes.value?.data?.total ?? 0
    if (assetOnlineRes.status === 'fulfilled') onlineAssets.value = assetOnlineRes.value?.data?.total ?? 0
    if (assetOfflineRes.status === 'fulfilled') offlineAssets.value = assetOfflineRes.value?.data?.total ?? 0
    if (logRes.status === 'fulfilled') {
      recentLogs.value = logRes.value?.data?.list ?? []
      stats.value.auditLogs = logRes.value?.data?.total ?? 0
    }
  } finally {
    loading.value = false
  }
}

const actionTagMap: Record<string, '' | 'success' | 'warning' | 'danger' | 'info'> = {
  create: 'success',
  update: '',
  delete: 'danger',
  login: 'success',
  logout: 'info',
}

onMounted(fetchStats)
</script>

<template>
  <div class="dashboard" v-loading="loading">
    <!-- 欢迎横幅 -->
    <div class="welcome-banner">
      <div class="welcome-text">
        <h2>欢迎使用 BigOps 运维平台</h2>
        <p>统一管理云账号、主机资产与服务树，让运维更高效</p>
      </div>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stat-row">
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover" class="stat-card stat-blue">
          <div class="stat-icon"><el-icon><Monitor /></el-icon></div>
          <div class="stat-info">
            <div class="stat-num">{{ stats.assets }}</div>
            <div class="stat-label">主机资产</div>
          </div>
          <div class="stat-sub">
            <el-tag type="success" size="small">在线 {{ onlineAssets }}</el-tag>
            <el-tag type="danger" size="small" style="margin-left:6px">离线 {{ offlineAssets }}</el-tag>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover" class="stat-card stat-green">
          <div class="stat-icon"><el-icon><Cloud /></el-icon></div>
          <div class="stat-info">
            <div class="stat-num">{{ stats.cloudAccounts }}</div>
            <div class="stat-label">云账号</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover" class="stat-card stat-orange">
          <div class="stat-icon"><el-icon><User /></el-icon></div>
          <div class="stat-info">
            <div class="stat-num">{{ stats.users }}</div>
            <div class="stat-label">用户数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover" class="stat-card stat-purple">
          <div class="stat-icon"><el-icon><Document /></el-icon></div>
          <div class="stat-info">
            <div class="stat-num">{{ stats.auditLogs }}</div>
            <div class="stat-label">操作记录</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 最近操作日志 -->
    <el-card shadow="never" class="log-card">
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center">
          <span>最近操作记录</span>
          <el-button link type="primary" @click="$router.push('/audit-logs')">查看全部</el-button>
        </div>
      </template>
      <el-table :data="recentLogs" stripe size="small">
        <el-table-column prop="username" label="操作人" width="120" />
        <el-table-column prop="action" label="操作" width="90">
          <template #default="{ row }">
            <el-tag :type="actionTagMap[row.action] ?? 'info'" size="small">{{ row.action }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="resource" label="资源类型" width="120" />
        <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP" width="130" />
        <el-table-column prop="created_at" label="时间" width="170" />
      </el-table>
      <el-empty v-if="!loading && recentLogs.length === 0" description="暂无操作记录" />
    </el-card>
  </div>
</template>

<style scoped>
.dashboard { padding: 20px; }

.welcome-banner {
  background: linear-gradient(135deg, #304156 0%, #409eff 100%);
  border-radius: 8px;
  padding: 24px 32px;
  margin-bottom: 20px;
  display: flex;
  align-items: center;
}
.welcome-text h2 { margin: 0 0 6px; color: #fff; font-size: 20px; }
.welcome-text p { margin: 0; color: rgba(255,255,255,0.8); font-size: 14px; }

.stat-row { margin-bottom: 20px; }

.stat-card {
  position: relative;
  overflow: hidden;
  cursor: default;
}
.stat-card :deep(.el-card__body) {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.stat-icon {
  font-size: 36px;
  position: absolute;
  right: 20px;
  top: 50%;
  transform: translateY(-50%);
  opacity: 0.15;
}
.stat-blue .stat-icon { color: #409eff; }
.stat-green .stat-icon { color: #67c23a; }
.stat-orange .stat-icon { color: #e6a23c; }
.stat-purple .stat-icon { color: #9b59b6; }

.stat-info { z-index: 1; }
.stat-num { font-size: 36px; font-weight: 700; line-height: 1; }
.stat-blue .stat-num { color: #409eff; }
.stat-green .stat-num { color: #67c23a; }
.stat-orange .stat-num { color: #e6a23c; }
.stat-purple .stat-num { color: #9b59b6; }
.stat-label { font-size: 13px; color: #909399; margin-top: 4px; }
.stat-sub { z-index: 1; }

.log-card { }
</style>
