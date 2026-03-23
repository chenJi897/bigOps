<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { statsApi } from '../api'

const summary = ref({
  asset_total: 0, asset_online: 0, asset_offline: 0,
  cloud_account_total: 0, cloud_account_failed: 0,
  service_tree_total: 0, user_total: 0,
})
const distribution = ref<{ status_dist: any[]; source_dist: any[]; top_services: any[] }>({
  status_dist: [], source_dist: [], top_services: [],
})
const loading = ref(true)

async function fetchData() {
  loading.value = true
  try {
    const [summaryRes, distRes] = await Promise.allSettled([
      statsApi.summary(),
      statsApi.assetDistribution(),
    ])
    if (summaryRes.status === 'fulfilled') summary.value = (summaryRes.value as any).data
    if (distRes.status === 'fulfilled') distribution.value = (distRes.value as any).data || { status_dist: [], source_dist: [], top_services: [] }
  } finally {
    loading.value = false
  }
}

function sourceLabel(s: string) {
  const map: Record<string, string> = { manual: '手工录入', aliyun: '阿里云', tencent: '腾讯云', aws: 'AWS' }
  return map[s] || s
}

function maxServiceCount() {
  if (!distribution.value.top_services?.length) return 1
  return Math.max(...distribution.value.top_services.map((t: any) => t.count), 1)
}

onMounted(fetchData)
</script>

<template>
  <div class="dashboard" v-loading="loading">
    <!-- 欢迎 -->
    <div class="welcome">
      <h2>BigOps 平台总览</h2>
      <p>实时掌握平台资源与运行状态</p>
    </div>

    <!-- 摘要卡片 -->
    <el-row :gutter="16" class="cards">
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #e8f4fd;"><el-icon size="28" color="#409EFF"><Monitor /></el-icon></div>
          <div class="stat-body">
            <div class="stat-value">{{ summary.asset_total }}</div>
            <div class="stat-label">主机资产</div>
            <div class="stat-sub">
              <span style="color: #67c23a;">在线 {{ summary.asset_online }}</span>
              <span style="color: #909399; margin: 0 6px;">/</span>
              <span style="color: #f56c6c;">离线 {{ summary.asset_offline }}</span>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #e8f8e8;"><el-icon size="28" color="#67C23A"><Connection /></el-icon></div>
          <div class="stat-body">
            <div class="stat-value">{{ summary.cloud_account_total }}</div>
            <div class="stat-label">云账号</div>
            <div class="stat-sub">
              <span v-if="summary.cloud_account_failed > 0" style="color: #f56c6c;">{{ summary.cloud_account_failed }} 个同步异常</span>
              <span v-else style="color: #67c23a;">全部正常</span>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #fdf2e8;"><el-icon size="28" color="#E6A23C"><Share /></el-icon></div>
          <div class="stat-body">
            <div class="stat-value">{{ summary.service_tree_total }}</div>
            <div class="stat-label">服务树节点</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="cards" style="margin-top: 16px;">
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #f0e8fd;"><el-icon size="28" color="#9B59B6"><User /></el-icon></div>
          <div class="stat-body">
            <div class="stat-value">{{ summary.user_total }}</div>
            <div class="stat-label">平台用户</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card alert-card" :class="{ 'has-alert': summary.asset_offline > 0 }">
          <div class="stat-icon" style="background: #fde8e8;"><el-icon size="28" color="#F56C6C"><WarningFilled /></el-icon></div>
          <div class="stat-body">
            <div class="stat-value" :style="{ color: summary.asset_offline > 0 ? '#f56c6c' : '#909399' }">{{ summary.asset_offline }}</div>
            <div class="stat-label">离线资产</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card alert-card" :class="{ 'has-alert': summary.cloud_account_failed > 0 }">
          <div class="stat-icon" style="background: #fde8e8;"><el-icon size="28" color="#F56C6C"><CircleCloseFilled /></el-icon></div>
          <div class="stat-body">
            <div class="stat-value" :style="{ color: summary.cloud_account_failed > 0 ? '#f56c6c' : '#909399' }">{{ summary.cloud_account_failed }}</div>
            <div class="stat-label">同步异常账号</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 下半区 -->
    <el-row :gutter="16" style="margin-top: 20px;">
      <!-- 来源分布 -->
      <el-col :span="10">
        <el-card shadow="never">
          <template #header><span style="font-weight: 600;">资产来源分布</span></template>
          <div v-if="distribution.source_dist?.length">
            <div v-for="item in distribution.source_dist" :key="item.label" class="dist-row">
              <span class="dist-label">{{ sourceLabel(item.label) }}</span>
              <el-progress
                :percentage="summary.asset_total ? Math.round(item.count / summary.asset_total * 100) : 0"
                :stroke-width="18" :show-text="false"
                style="flex: 1; margin: 0 12px;"
              />
              <span class="dist-count">{{ item.count }}</span>
            </div>
          </div>
          <el-empty v-else description="暂无数据" :image-size="60" />
        </el-card>
      </el-col>
      <!-- 服务树 Top 10 -->
      <el-col :span="14">
        <el-card shadow="never">
          <template #header><span style="font-weight: 600;">服务树资产 Top 10</span></template>
          <div v-if="distribution.top_services?.length">
            <div v-for="item in distribution.top_services" :key="item.id" class="dist-row">
              <span class="dist-label" style="min-width: 140px;" :title="item.name">{{ item.name }}</span>
              <el-progress
                :percentage="Math.round(item.count / maxServiceCount() * 100)"
                :stroke-width="18" :show-text="false" color="#67c23a"
                style="flex: 1; margin: 0 12px;"
              />
              <span class="dist-count">{{ item.count }}</span>
            </div>
          </div>
          <el-empty v-else description="暂无数据" :image-size="60" />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.dashboard { padding: 20px; }
.welcome { margin-bottom: 20px; }
.welcome h2 { margin: 0 0 4px 0; font-size: 22px; }
.welcome p { margin: 0; color: #909399; font-size: 14px; }

.stat-card {
  display: flex;
  align-items: center;
  padding: 8px 0;
}
.stat-card :deep(.el-card__body) {
  display: flex;
  align-items: center;
  gap: 16px;
  width: 100%;
}
.stat-icon {
  width: 56px; height: 56px;
  border-radius: 12px;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.stat-body { flex: 1; }
.stat-value { font-size: 28px; font-weight: 700; line-height: 1.2; }
.stat-label { font-size: 14px; color: #909399; margin-top: 2px; }
.stat-sub { font-size: 12px; margin-top: 4px; }

.alert-card.has-alert { border-color: #fde2e2; }

.dist-row {
  display: flex; align-items: center;
  margin-bottom: 12px;
}
.dist-row:last-child { margin-bottom: 0; }
.dist-label { min-width: 80px; font-size: 13px; color: #606266; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.dist-count { min-width: 40px; text-align: right; font-size: 14px; font-weight: 600; color: #303133; }
</style>
