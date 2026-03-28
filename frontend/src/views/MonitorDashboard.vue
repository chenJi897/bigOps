<script setup lang="ts">
defineOptions({ name: 'MonitorDashboard' })

import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { alertRuleApi, monitorApi } from '../api'

type MetricSample = {
  id: number
  metric_value: number
  collected_at: string
}

const loading = ref(false)
const eventLoading = ref(false)
const trendLoading = ref(false)
const autoRefresh = ref(true)
const agents = ref<any[]>([])
const alertEvents = ref<any[]>([])
const serviceTreeAggregates = ref<any[]>([])
const ownerAggregates = ref<any[]>([])
const detailVisible = ref(false)
const currentAgent = ref<any | null>(null)
const trends = ref<Record<string, MetricSample[]>>({
  cpu_usage: [],
  memory_usage: [],
  disk_usage: [],
})

const summary = ref<any>({
  agent_total: 0,
  agent_online: 0,
  agent_offline: 0,
  rule_enabled_total: 0,
  alert_firing_total: 0,
  last_collected_at: '',
  cpu_high_agents: [],
  memory_high_agents: [],
  disk_high_agents: [],
  recent_alerts: [],
  alert_status_counts: [],
  alert_severity_counts: [],
})

const filters = ref({
  keyword: '',
  status: '',
})

const pager = ref({
  page: 1,
  size: 20,
  total: 0,
})

let refreshTimer: number | null = null
const router = useRouter()

const statusOptions = [
  { label: '全部状态', value: '' },
  { label: '在线', value: 'online' },
  { label: '离线', value: 'offline' },
]

const metricLabels: Record<string, string> = {
  cpu_usage: 'CPU 使用率',
  memory_usage: '内存使用率',
  disk_usage: '磁盘使用率',
  agent_offline: 'Agent 离线',
}

const activeEventCount = computed(() => {
  return (summary.value.alert_status_counts || []).reduce((acc: number, item: any) => {
    if (item.status === 'firing' || item.status === 'acknowledged') {
      return acc + Number(item.total || 0)
    }
    return acc
  }, 0)
})

async function fetchSummaryAndAgents(showLoading = true) {
  if (showLoading) {
    loading.value = true
  }
  try {
    const [treeRes, ownerRes, summaryRes, agentRes, eventRes] = await Promise.all([
      monitorApi.aggregateServiceTrees(),
      monitorApi.aggregateOwners(),
      monitorApi.summary(),
      monitorApi.agents({
        page: pager.value.page,
        size: pager.value.size,
        status: filters.value.status,
        keyword: filters.value.keyword.trim(),
      }),
      alertRuleApi.events({
        page: 1,
        size: 8,
        status: '',
      }),
    ])

    serviceTreeAggregates.value = (treeRes as any).data || []
    ownerAggregates.value = (ownerRes as any).data || []
    summary.value = (summaryRes as any).data || summary.value
    agents.value = (agentRes as any).data?.list || []
    pager.value.total = Number((agentRes as any).data?.total || 0)
    alertEvents.value = (eventRes as any).data?.list || []
  } finally {
    loading.value = false
    eventLoading.value = false
  }
}

async function fetchTrends() {
  if (!currentAgent.value?.agent_id) {
    return
  }
  trendLoading.value = true
  try {
    const [cpuRes, memRes, diskRes] = await Promise.all([
      monitorApi.trends(currentAgent.value.agent_id, 'cpu_usage', 180, 60),
      monitorApi.trends(currentAgent.value.agent_id, 'memory_usage', 180, 60),
      monitorApi.trends(currentAgent.value.agent_id, 'disk_usage', 180, 60),
    ])
    trends.value = {
      cpu_usage: (cpuRes as any).data || [],
      memory_usage: (memRes as any).data || [],
      disk_usage: (diskRes as any).data || [],
    }
  } finally {
    trendLoading.value = false
  }
}

async function refreshAll(showMessage = false) {
  eventLoading.value = true
  await fetchSummaryAndAgents(true)
  if (detailVisible.value && currentAgent.value?.agent_id) {
    await fetchTrends()
  }
  if (showMessage) {
    ElMessage.success('监控数据已刷新')
  }
}

function applyFilters() {
  pager.value.page = 1
  fetchSummaryAndAgents(true)
}

function resetFilters() {
  filters.value.keyword = ''
  filters.value.status = ''
  pager.value.page = 1
  fetchSummaryAndAgents(true)
}

function handlePageChange(page: number) {
  pager.value.page = page
  fetchSummaryAndAgents(true)
}

function handleSizeChange(size: number) {
  pager.value.size = size
  pager.value.page = 1
  fetchSummaryAndAgents(true)
}

async function openAgentDetail(row: any) {
  currentAgent.value = row
  detailVisible.value = true
  await fetchTrends()
}

function goAgentDetail(row: any) {
  if (!row?.agent_id) return
  router.push(`/monitor/agents/${encodeURIComponent(row.agent_id)}`)
}

function goAlertCenter(agentID = '') {
  const query = agentID ? `?agent_id=${encodeURIComponent(agentID)}` : ''
  router.push(`/monitor/alerts${query}`)
}

function goDatasourcePage() {
  router.push('/monitor/datasources')
}

function goQueryPage() {
  router.push('/monitor/query')
}

function goSilencePage() {
  router.push('/monitor/silences')
}

function goOnCallPage() {
  router.push('/monitor/oncall')
}

function setupRefreshTimer() {
  if (refreshTimer) {
    window.clearInterval(refreshTimer)
    refreshTimer = null
  }
  if (!autoRefresh.value) {
    return
  }
  refreshTimer = window.setInterval(() => {
    fetchSummaryAndAgents(false)
    if (detailVisible.value && currentAgent.value?.agent_id) {
      fetchTrends()
    }
  }, 30000)
}

function formatPercent(value: number) {
  return `${Number(value || 0).toFixed(1)}%`
}

function formatBytes(value: number) {
  const size = Number(value || 0)
  if (!size) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let current = size
  let index = 0
  while (current >= 1024 && index < units.length - 1) {
    current /= 1024
    index++
  }
  return `${current.toFixed(current >= 10 ? 0 : 1)} ${units[index]}`
}

function statusTagType(status: string) {
  if (status === 'online') return 'success'
  if (status === 'acknowledged') return 'warning'
  if (status === 'resolved') return 'info'
  return 'danger'
}

function statusLabel(status: string) {
  const map: Record<string, string> = {
    online: '在线',
    offline: '离线',
    firing: '触发中',
    acknowledged: '已确认',
    resolved: '已恢复',
  }
  return map[status] || status
}

function severityTagType(severity: string) {
  const map: Record<string, string> = {
    critical: 'danger',
    warning: 'warning',
    info: 'info',
  }
  return map[severity] || 'info'
}

function severityLabel(severity: string) {
  const map: Record<string, string> = {
    critical: '严重',
    warning: '警告',
    info: '提示',
  }
  return map[severity] || severity
}

function metricLabel(metricType: string) {
  return metricLabels[metricType] || metricType
}

function aggregateStatus(item: any) {
  if (Number(item.offline_total || 0) > 0) return 'warning'
  if (Number(item.online_total || 0) > 0) return 'success'
  return 'info'
}

function metricPath(metricType: string) {
  const items = trends.value[metricType] || []
  if (!items.length) {
    return ''
  }
  const values = items.map(item => Number(item.metric_value || 0))
  const maxValue = Math.max(...values, 1)
  return values.map((value, index) => {
    const x = values.length === 1 ? 50 : (index / (values.length - 1)) * 100
    const y = 92 - (value / maxValue) * 84
    return `${index === 0 ? 'M' : 'L'} ${x.toFixed(2)} ${y.toFixed(2)}`
  }).join(' ')
}

function metricLatest(metricType: string) {
  const items = trends.value[metricType] || []
  if (!items.length) {
    return '-'
  }
  return formatPercent(Number(items[items.length - 1].metric_value || 0))
}

function metricMin(metricType: string) {
  const items = trends.value[metricType] || []
  if (!items.length) {
    return '-'
  }
  return formatPercent(Math.min(...items.map(item => Number(item.metric_value || 0))))
}

function metricMax(metricType: string) {
  const items = trends.value[metricType] || []
  if (!items.length) {
    return '-'
  }
  return formatPercent(Math.max(...items.map(item => Number(item.metric_value || 0))))
}

function latestPointTime(metricType: string) {
  const items = trends.value[metricType] || []
  return items.length ? items[items.length - 1].collected_at : '-'
}

onMounted(async () => {
  await fetchSummaryAndAgents(true)
  setupRefreshTimer()
})

onUnmounted(() => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer)
    refreshTimer = null
  }
})
</script>

<template>
  <div class="monitor-page" v-loading="loading">
    <div class="hero-card">
      <div>
        <div class="hero-title">监控中心</div>
        <div class="hero-subtitle">聚合 Agent 在线状态、资源水位与最近告警，适合值守时快速扫面全局。</div>
      </div>
      <div class="hero-actions">
        <el-button plain @click="goSilencePage">告警静默</el-button>
        <el-button plain @click="goOnCallPage">OnCall</el-button>
        <el-button plain @click="goDatasourcePage">数据源</el-button>
        <el-button plain @click="goQueryPage">PromQL 查询</el-button>
        <el-switch
          v-model="autoRefresh"
          inline-prompt
          active-text="自动刷新"
          inactive-text="手动"
          @change="setupRefreshTimer"
        />
        <el-button type="primary" plain @click="refreshAll(true)">
          <el-icon><RefreshRight /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <el-row :gutter="16" class="stat-grid">
      <el-col :xs="24" :sm="12" :lg="8" :xl="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-kicker">Agent</div>
          <div class="stat-value">{{ summary.agent_total }}</div>
          <div class="stat-meta">已接入主机</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="8" :xl="4">
        <el-card shadow="never" class="stat-card success">
          <div class="stat-kicker">在线</div>
          <div class="stat-value">{{ summary.agent_online }}</div>
          <div class="stat-meta">心跳正常</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="8" :xl="4">
        <el-card shadow="never" class="stat-card danger">
          <div class="stat-kicker">离线</div>
          <div class="stat-value">{{ summary.agent_offline }}</div>
          <div class="stat-meta">超过阈值未上报</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="8" :xl="4">
        <el-card shadow="never" class="stat-card warning">
          <div class="stat-kicker">启用规则</div>
          <div class="stat-value">{{ summary.rule_enabled_total }}</div>
          <div class="stat-meta">当前生效</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="8" :xl="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-kicker">触发中</div>
          <div class="stat-value">{{ activeEventCount }}</div>
          <div class="stat-meta">告警事件</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="8" :xl="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-kicker">最近采样</div>
          <div class="stat-value stat-time">{{ summary.last_collected_at || '-' }}</div>
          <div class="stat-meta">监控数据新鲜度</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-grid">
      <el-col :xs="24" :xl="24">
        <el-row :gutter="16">
          <el-col :xs="24" :lg="12">
            <el-card shadow="never" class="panel-card aggregate-card">
              <template #header>
                <div class="panel-header">
                  <span>按服务树聚合</span>
                  <span class="panel-hint">业务视角查看 Agent 水位</span>
                </div>
              </template>
              <el-table :data="serviceTreeAggregates.slice(0, 8)" size="small" stripe border>
                <el-table-column prop="name" label="服务树" min-width="180" show-overflow-tooltip />
                <el-table-column label="状态" width="90">
                  <template #default="{ row }">
                    <el-tag size="small" :type="aggregateStatus(row)">
                      {{ Number(row.offline_total || 0) > 0 ? '关注' : '健康' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="agent_total" label="主机数" width="90" />
                <el-table-column label="在线 / 离线" width="120">
                  <template #default="{ row }">{{ row.online_total }} / {{ row.offline_total }}</template>
                </el-table-column>
                <el-table-column label="CPU" width="90">
                  <template #default="{ row }">{{ formatPercent(row.avg_cpu_usage_pct) }}</template>
                </el-table-column>
                <el-table-column label="内存" width="90">
                  <template #default="{ row }">{{ formatPercent(row.avg_memory_usage_pct) }}</template>
                </el-table-column>
                <el-table-column label="磁盘" width="90">
                  <template #default="{ row }">{{ formatPercent(row.avg_disk_usage_pct) }}</template>
                </el-table-column>
              </el-table>
              <el-empty v-if="!serviceTreeAggregates.length" description="暂无服务树聚合数据" :image-size="54" />
            </el-card>
          </el-col>

          <el-col :xs="24" :lg="12">
            <el-card shadow="never" class="panel-card aggregate-card">
              <template #header>
                <div class="panel-header">
                  <span>按负责人聚合</span>
                  <span class="panel-hint">谁在负责异常对象，一眼看到</span>
                </div>
              </template>
              <el-table :data="ownerAggregates.slice(0, 8)" size="small" stripe border>
                <el-table-column prop="name" label="负责人" min-width="160" show-overflow-tooltip />
                <el-table-column label="状态" width="90">
                  <template #default="{ row }">
                    <el-tag size="small" :type="aggregateStatus(row)">
                      {{ Number(row.offline_total || 0) > 0 ? '关注' : '健康' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="agent_total" label="主机数" width="90" />
                <el-table-column label="在线 / 离线" width="120">
                  <template #default="{ row }">{{ row.online_total }} / {{ row.offline_total }}</template>
                </el-table-column>
                <el-table-column label="CPU" width="90">
                  <template #default="{ row }">{{ formatPercent(row.avg_cpu_usage_pct) }}</template>
                </el-table-column>
                <el-table-column label="内存" width="90">
                  <template #default="{ row }">{{ formatPercent(row.avg_memory_usage_pct) }}</template>
                </el-table-column>
                <el-table-column label="磁盘" width="90">
                  <template #default="{ row }">{{ formatPercent(row.avg_disk_usage_pct) }}</template>
                </el-table-column>
              </el-table>
              <el-empty v-if="!ownerAggregates.length" description="暂无负责人聚合数据" :image-size="54" />
            </el-card>
          </el-col>
        </el-row>
      </el-col>

      <el-col :xs="24" :xl="14">
        <el-card shadow="never" class="panel-card">
          <template #header>
            <div class="panel-header">
              <span>Agent 实时列表</span>
              <el-form inline>
                <el-form-item>
                  <el-input v-model="filters.keyword" clearable placeholder="搜索主机名 / IP / Agent ID" style="width: 220px" @keyup.enter="applyFilters" />
                </el-form-item>
                <el-form-item>
                  <el-select v-model="filters.status" style="width: 120px">
                    <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
                  </el-select>
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" @click="applyFilters">筛选</el-button>
                  <el-button @click="resetFilters">重置</el-button>
                </el-form-item>
              </el-form>
            </div>
          </template>

          <el-table :data="agents" stripe border>
            <el-table-column prop="hostname" label="主机名" min-width="180" show-overflow-tooltip />
            <el-table-column prop="ip" label="IP" width="150" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="CPU" width="110">
              <template #default="{ row }">{{ formatPercent(row.cpu_usage_pct) }}</template>
            </el-table-column>
            <el-table-column label="内存" width="110">
              <template #default="{ row }">{{ formatPercent(row.memory_usage_pct) }}</template>
            </el-table-column>
            <el-table-column label="磁盘" width="110">
              <template #default="{ row }">{{ formatPercent(row.disk_usage_pct) }}</template>
            </el-table-column>
            <el-table-column prop="last_heartbeat" label="最后心跳" width="180" />
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openAgentDetail(row)">趋势</el-button>
                <el-button link type="warning" @click="goAgentDetail(row)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="table-footer">
            <el-pagination
              background
              layout="total, sizes, prev, pager, next"
              :total="pager.total"
              :current-page="pager.page"
              :page-size="pager.size"
              :page-sizes="[10, 20, 50]"
              @current-change="handlePageChange"
              @size-change="handleSizeChange"
            />
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :xl="10">
        <el-row :gutter="16">
          <el-col :span="24">
            <el-card shadow="never" class="panel-card rank-panel">
              <template #header><span>热点 Top</span></template>
              <div class="rank-block">
                <div class="rank-title">CPU Top</div>
                <div v-if="summary.cpu_high_agents?.length" class="rank-list">
                  <div v-for="agent in summary.cpu_high_agents" :key="`cpu-${agent.agent_id}`" class="rank-row">
                    <span>{{ agent.hostname || agent.agent_id }}</span>
                    <strong>{{ formatPercent(agent.cpu_usage_pct) }}</strong>
                  </div>
                </div>
                <el-empty v-else description="暂无高 CPU Agent" :image-size="46" />
              </div>
              <div class="rank-block">
                <div class="rank-title">内存 Top</div>
                <div v-if="summary.memory_high_agents?.length" class="rank-list">
                  <div v-for="agent in summary.memory_high_agents" :key="`mem-${agent.agent_id}`" class="rank-row">
                    <span>{{ agent.hostname || agent.agent_id }}</span>
                    <strong>{{ formatPercent(agent.memory_usage_pct) }}</strong>
                  </div>
                </div>
                <el-empty v-else description="暂无高内存 Agent" :image-size="46" />
              </div>
              <div class="rank-block">
                <div class="rank-title">磁盘 Top</div>
                <div v-if="summary.disk_high_agents?.length" class="rank-list">
                  <div v-for="agent in summary.disk_high_agents" :key="`disk-${agent.agent_id}`" class="rank-row">
                    <span>{{ agent.hostname || agent.agent_id }}</span>
                    <strong>{{ formatPercent(agent.disk_usage_pct) }}</strong>
                  </div>
                </div>
                <el-empty v-else description="暂无高磁盘 Agent" :image-size="46" />
              </div>
            </el-card>
          </el-col>

          <el-col :span="24">
            <el-card shadow="never" class="panel-card">
              <template #header><span>最近告警</span></template>
              <div v-if="alertEvents.length" class="event-list">
                <div v-for="item in alertEvents" :key="item.id" class="event-item">
                  <div class="event-main">
                    <div class="event-title">{{ item.rule_name }}</div>
                    <div class="event-sub">{{ item.hostname || item.agent_id }} · {{ metricLabel(item.metric_type) }}</div>
                  </div>
                  <div class="event-side">
                    <el-tag size="small" :type="severityTagType(item.severity)">{{ severityLabel(item.severity) }}</el-tag>
                    <el-tag size="small" :type="statusTagType(item.status)">{{ statusLabel(item.status) }}</el-tag>
                  </div>
                </div>
                <div class="event-footer">
                  <el-button link type="primary" @click="goAlertCenter()">进入告警事件中心</el-button>
                </div>
              </div>
              <el-empty v-else description="暂无告警事件" :image-size="60" />
            </el-card>
          </el-col>
        </el-row>
      </el-col>
    </el-row>

    <el-drawer v-model="detailVisible" size="720px" :title="currentAgent ? `${currentAgent.hostname || currentAgent.agent_id} 指标趋势` : '指标趋势'">
      <div v-loading="trendLoading" class="detail-wrap">
        <div v-if="currentAgent" class="detail-header">
          <div>
            <div class="detail-host">{{ currentAgent.hostname || currentAgent.agent_id }}</div>
            <div class="detail-sub">{{ currentAgent.ip }} · {{ statusLabel(currentAgent.status) }}</div>
          </div>
          <el-button plain @click="fetchTrends">刷新趋势</el-button>
        </div>

        <el-row :gutter="16">
          <el-col :span="8">
            <el-card shadow="never" class="mini-stat">
              <div class="mini-kicker">内存占用</div>
              <div class="mini-value">{{ formatBytes(currentAgent?.memory_used) }}</div>
              <div class="mini-sub">总计 {{ formatBytes(currentAgent?.memory_total) }}</div>
            </el-card>
          </el-col>
          <el-col :span="8">
            <el-card shadow="never" class="mini-stat">
              <div class="mini-kicker">磁盘占用</div>
              <div class="mini-value">{{ formatBytes(currentAgent?.disk_used) }}</div>
              <div class="mini-sub">总计 {{ formatBytes(currentAgent?.disk_total) }}</div>
            </el-card>
          </el-col>
          <el-col :span="8">
            <el-card shadow="never" class="mini-stat">
              <div class="mini-kicker">最后心跳</div>
              <div class="mini-value mini-time">{{ currentAgent?.last_heartbeat || '-' }}</div>
              <div class="mini-sub">Agent 在线性</div>
            </el-card>
          </el-col>
        </el-row>

        <div class="trend-grid">
          <el-card v-for="metricType in ['cpu_usage', 'memory_usage', 'disk_usage']" :key="metricType" shadow="never" class="trend-card">
            <template #header>
              <div class="trend-head">
                <span>{{ metricLabel(metricType) }}</span>
                <span>{{ metricLatest(metricType) }}</span>
              </div>
            </template>
            <div v-if="(trends[metricType] || []).length" class="trend-body">
              <svg viewBox="0 0 100 100" preserveAspectRatio="none" class="trend-chart">
                <path d="M 0 92 L 100 92" class="trend-axis" />
                <path :d="metricPath(metricType)" class="trend-line" />
              </svg>
              <div class="trend-meta">
                <span>最小 {{ metricMin(metricType) }}</span>
                <span>最大 {{ metricMax(metricType) }}</span>
                <span>最近 {{ latestPointTime(metricType) }}</span>
              </div>
            </div>
            <el-empty v-else description="暂无趋势采样" :image-size="54" />
          </el-card>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
.monitor-page {
  padding: 20px;
  background:
    radial-gradient(circle at top left, rgba(59, 130, 246, 0.12), transparent 30%),
    radial-gradient(circle at top right, rgba(14, 165, 233, 0.12), transparent 26%),
    #f5f7fb;
  min-height: 100%;
}

.hero-card,
.panel-card,
.stat-card,
.mini-stat,
.trend-card {
  border: 1px solid #e7ecf3;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.04);
}

.hero-card {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
  padding: 20px 24px;
  border-radius: 18px;
  background: linear-gradient(135deg, #0f172a 0%, #102c57 55%, #0ea5e9 100%);
  color: #fff;
}

.hero-title {
  font-size: 24px;
  font-weight: 700;
}

.hero-subtitle {
  margin-top: 8px;
  max-width: 720px;
  color: rgba(255, 255, 255, 0.8);
  line-height: 1.6;
}

.hero-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.stat-grid,
.panel-grid {
  margin-top: 16px;
}

.stat-card {
  border-radius: 16px;
}

.stat-card :deep(.el-card__body) {
  padding: 18px 20px;
}

.stat-kicker {
  font-size: 12px;
  letter-spacing: 0.08em;
  color: #64748b;
  text-transform: uppercase;
}

.stat-value {
  margin-top: 12px;
  font-size: 30px;
  font-weight: 700;
  color: #0f172a;
}

.stat-time {
  font-size: 15px;
  line-height: 1.5;
}

.stat-meta {
  margin-top: 10px;
  font-size: 13px;
  color: #64748b;
}

.stat-card.success .stat-value { color: #16a34a; }
.stat-card.danger .stat-value { color: #dc2626; }
.stat-card.warning .stat-value { color: #d97706; }

.panel-card {
  border-radius: 18px;
}

.aggregate-card {
  height: 100%;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.panel-hint {
  font-size: 12px;
  color: #64748b;
}

.table-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.rank-panel .rank-block + .rank-block {
  margin-top: 18px;
}

.rank-title {
  margin-bottom: 10px;
  font-size: 13px;
  font-weight: 600;
  color: #475569;
}

.rank-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rank-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  border-radius: 12px;
  background: #f8fafc;
  color: #334155;
}

.event-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.event-item {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 14px;
  background: #f8fafc;
}

.event-title {
  font-size: 14px;
  font-weight: 600;
  color: #0f172a;
}

.event-sub {
  margin-top: 6px;
  font-size: 12px;
  color: #64748b;
}

.event-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
}

.detail-wrap {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.detail-host {
  font-size: 20px;
  font-weight: 700;
  color: #0f172a;
}

.detail-sub {
  margin-top: 6px;
  color: #64748b;
}

.mini-stat {
  border-radius: 14px;
}

.mini-kicker {
  font-size: 12px;
  color: #64748b;
}

.mini-value {
  margin-top: 10px;
  font-size: 22px;
  font-weight: 700;
  color: #0f172a;
}

.mini-time {
  font-size: 14px;
  line-height: 1.5;
}

.mini-sub {
  margin-top: 8px;
  font-size: 12px;
  color: #64748b;
}

.trend-grid {
  display: grid;
  grid-template-columns: repeat(1, minmax(0, 1fr));
  gap: 16px;
}

.trend-card {
  border-radius: 16px;
}

.trend-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
}

.trend-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.trend-chart {
  width: 100%;
  height: 150px;
  background: linear-gradient(180deg, rgba(37, 99, 235, 0.06), rgba(37, 99, 235, 0));
  border-radius: 12px;
}

.trend-axis {
  stroke: #cbd5e1;
  stroke-width: 1;
  fill: none;
}

.trend-line {
  stroke: #2563eb;
  stroke-width: 2.4;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.trend-meta {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
  font-size: 12px;
  color: #64748b;
}

@media (max-width: 960px) {
  .hero-card,
  .panel-header,
  .detail-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .event-item {
    flex-direction: column;
  }

  .event-side {
    align-items: flex-start;
  }
}
</style>
