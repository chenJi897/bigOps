<script setup lang="ts">
defineOptions({ name: 'MonitorDashboard' })

import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { alertRuleApi, monitorApi } from '../api'

const sloDialogVisible = ref(false)
const sloForm = ref({ target_availability: 99.9, target_latency_ms: 3000 })
const anomalies = ref<any[]>([])
const anomalyLoading = ref(false)
const predictions = ref<any[]>([])
const predictionLoading = ref(false)

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
const goldenWindow = ref(60)
const goldenSignals = ref<any>({
  window_minutes: 60,
  availability_pct: 100,
  error_rate_pct: 0,
  avg_latency_ms: 0,
  throughput_per_minute: 0,
  total_requests: 0,
  total_errors: 0,
  slo_breached: false,
  slo_target_availability: 99.9,
  slo_target_latency_ms: 3000,
})
const goldenDimension = ref<'service' | 'interface' | 'instance'>('service')
const goldenDimensionRows = ref<any[]>([])
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

const goldenDimensionTypeLabel: Record<string, string> = {
  service: '服务',
  interface: '接口',
  instance: '实例',
  operator: '执行人',
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
    const [treeRes, ownerRes, summaryRes, agentRes, eventRes, goldenRes, goldenDimRes] = await Promise.all([
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
      monitorApi.goldenSignals(goldenWindow.value),
      monitorApi.goldenSignalsDimensions(goldenWindow.value, goldenDimension.value),
    ])

    serviceTreeAggregates.value = (treeRes as any).data || []
    ownerAggregates.value = (ownerRes as any).data || []
    summary.value = (summaryRes as any).data || summary.value
    agents.value = (agentRes as any).data?.list || []
    pager.value.total = Number((agentRes as any).data?.total || 0)
    alertEvents.value = (eventRes as any).data?.list || []
    goldenSignals.value = (goldenRes as any).data || goldenSignals.value
    goldenDimensionRows.value = (goldenDimRes as any).data || []
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

async function changeGoldenWindow() {
  await fetchSummaryAndAgents(true)
}

async function changeGoldenDimension() {
  await fetchSummaryAndAgents(true)
}

async function openSLODialog() {
  try {
    const res = await monitorApi.sloConfig()
    const data = (res as any).data
    sloForm.value = {
      target_availability: data?.target_availability ?? 99.9,
      target_latency_ms: data?.target_latency_ms ?? 3000,
    }
  } catch {}
  sloDialogVisible.value = true
}

async function saveSLOConfig() {
  await monitorApi.updateSloConfig(sloForm.value)
  ElMessage.success('SLO 配置已更新')
  sloDialogVisible.value = false
  await fetchSummaryAndAgents(true)
}

async function loadAnomalies() {
  anomalyLoading.value = true
  try {
    const res = await monitorApi.anomalies({ stddev_multiplier: 2.0 })
    anomalies.value = (res as any).data || []
  } finally {
    anomalyLoading.value = false
  }
}

async function loadPredictions() {
  predictionLoading.value = true
  try {
    const res = await monitorApi.capacityPrediction({ metric_type: 'disk_usage', threshold: 90 })
    predictions.value = (res as any).data || []
  } finally {
    predictionLoading.value = false
  }
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

function sloTagType() {
  return goldenSignals.value?.slo_breached ? 'danger' : 'success'
}

function metricLabel(metricType: string) {
  return metricLabels[metricType] || metricType
}

function goldenDimensionTypeText(value: string) {
  return goldenDimensionTypeLabel[value] || value
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
  loadAnomalies()
  loadPredictions()
})

onUnmounted(() => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer)
    refreshTimer = null
  }
})
</script>

<template>
  <div class="min-h-full bg-slate-50 p-4 md:p-6" style="background: radial-gradient(circle at top left, rgba(59, 130, 246, 0.12), transparent 30%), radial-gradient(circle at top right, rgba(14, 165, 233, 0.12), transparent 26%), #f8fafc;" v-loading="loading">
    <div class="flex flex-col lg:flex-row justify-between gap-4 items-start lg:items-center p-6 md:px-8 md:py-6 rounded-2xl bg-gradient-to-br from-slate-900 via-slate-800 to-sky-700 text-white shadow-lg mb-6">
      <div>
        <div class="text-2xl font-bold">监控中心</div>
        <div class="mt-2 max-w-2xl text-white/80 leading-relaxed text-sm md:text-base">
          聚合 Agent 在线状态、资源水位与最近告警，适合值守时快速扫面全局。
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-3">
        <el-button plain @click="goSilencePage" class="!bg-white/10 !text-white !border-white/20 hover:!bg-white/20">告警静默</el-button>
        <el-button plain @click="goOnCallPage" class="!bg-white/10 !text-white !border-white/20 hover:!bg-white/20">OnCall</el-button>
        <el-button plain @click="goDatasourcePage" class="!bg-white/10 !text-white !border-white/20 hover:!bg-white/20">数据源</el-button>
        <el-button plain @click="goQueryPage" class="!bg-white/10 !text-white !border-white/20 hover:!bg-white/20">PromQL 查询</el-button>
        <div class="bg-white/10 px-3 py-1.5 rounded border border-white/20 flex items-center gap-2">
          <span class="text-sm">自动刷新</span>
          <el-switch
            v-model="autoRefresh"
            size="small"
            @change="setupRefreshTimer"
          />
        </div>
        <el-select v-model="goldenWindow" class="!w-36" size="small" @change="changeGoldenWindow">
          <el-option :value="30" label="30分钟窗口" />
          <el-option :value="60" label="60分钟窗口" />
          <el-option :value="180" label="180分钟窗口" />
        </el-select>
        <el-select v-model="goldenDimension" class="!w-36" size="small" @change="changeGoldenDimension">
          <el-option value="service" label="服务维度" />
          <el-option value="interface" label="接口维度" />
          <el-option value="instance" label="实例维度" />
        </el-select>
        <el-button type="primary" @click="refreshAll(true)" class="!bg-sky-500 hover:!bg-sky-400 !border-none">
          <el-icon class="mr-1"><RefreshRight /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <el-row :gutter="16" class="mb-6">
      <el-col :xs="12" :sm="8" :lg="4" class="mb-4 lg:mb-0">
        <el-card shadow="hover" class="border-0 shadow-sm rounded-2xl h-full">
          <div class="text-xs tracking-wider text-slate-500 uppercase">Agent</div>
          <div class="mt-2 text-3xl font-bold text-slate-900">{{ summary.agent_total }}</div>
          <div class="mt-2 text-xs text-slate-500">已接入主机</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :lg="4" class="mb-4 lg:mb-0">
        <el-card shadow="hover" class="border-0 shadow-sm rounded-2xl h-full">
          <div class="text-xs tracking-wider text-slate-500 uppercase">在线</div>
          <div class="mt-2 text-3xl font-bold text-emerald-600">{{ summary.agent_online }}</div>
          <div class="mt-2 text-xs text-slate-500">心跳正常</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :lg="4" class="mb-4 lg:mb-0">
        <el-card shadow="hover" class="border-0 shadow-sm rounded-2xl h-full">
          <div class="text-xs tracking-wider text-slate-500 uppercase">离线</div>
          <div class="mt-2 text-3xl font-bold text-red-600">{{ summary.agent_offline }}</div>
          <div class="mt-2 text-xs text-slate-500">超过阈值未上报</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :lg="4" class="mb-4 lg:mb-0">
        <el-card shadow="hover" class="border-0 shadow-sm rounded-2xl h-full">
          <div class="text-xs tracking-wider text-slate-500 uppercase">启用规则</div>
          <div class="mt-2 text-3xl font-bold text-amber-600">{{ summary.rule_enabled_total }}</div>
          <div class="mt-2 text-xs text-slate-500">当前生效</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :lg="4" class="mb-4 lg:mb-0">
        <el-card shadow="hover" class="border-0 shadow-sm rounded-2xl h-full">
          <div class="text-xs tracking-wider text-slate-500 uppercase">触发中</div>
          <div class="mt-2 text-3xl font-bold text-slate-900">{{ activeEventCount }}</div>
          <div class="mt-2 text-xs text-slate-500">告警事件</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :lg="4">
        <el-card shadow="hover" class="border-0 shadow-sm rounded-2xl h-full">
          <div class="text-xs tracking-wider text-slate-500 uppercase">最近采样</div>
          <div class="mt-2 text-lg lg:text-base xl:text-lg font-bold text-slate-900 leading-tight truncate">{{ summary.last_collected_at || '-' }}</div>
          <div class="mt-2 text-xs text-slate-500">监控数据新鲜度</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="mb-6">
      <el-col :xs="12" :sm="12" :lg="6">
        <el-card shadow="hover" class="border-0 shadow-sm rounded-2xl h-full">
          <div class="text-xs tracking-wider text-slate-500 uppercase">可用性</div>
          <div class="mt-2 text-3xl font-bold text-emerald-600">{{ Number(goldenSignals.availability_pct || 0).toFixed(2) }}%</div>
          <div class="mt-2 text-xs text-slate-500">{{ goldenSignals.window_minutes }} 分钟窗口</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="12" :lg="6">
        <el-card shadow="hover" class="border-0 shadow-sm rounded-2xl h-full">
          <div class="text-xs tracking-wider text-slate-500 uppercase">错误率</div>
          <div class="mt-2 text-3xl font-bold text-red-600">{{ Number(goldenSignals.error_rate_pct || 0).toFixed(2) }}%</div>
          <div class="mt-2 text-xs text-slate-500">错误 {{ goldenSignals.total_errors }} / 总量 {{ goldenSignals.total_requests }}</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="12" :lg="6">
        <el-card shadow="hover" class="border-0 shadow-sm rounded-2xl h-full">
          <div class="text-xs tracking-wider text-slate-500 uppercase">平均延迟</div>
          <div class="mt-2 text-3xl font-bold text-amber-600">{{ Number(goldenSignals.avg_latency_ms || 0).toFixed(2) }}ms</div>
          <div class="mt-2 text-xs text-slate-500">心跳 RTT 均值</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="12" :lg="6">
        <el-card shadow="hover" class="border-0 shadow-sm rounded-2xl h-full">
          <div class="text-xs tracking-wider text-slate-500 uppercase">吞吐</div>
          <div class="mt-2 text-3xl font-bold text-sky-600">{{ Number(goldenSignals.throughput_per_minute || 0).toFixed(2) }}/min</div>
          <div class="mt-2 text-xs text-slate-500">近窗口请求速率</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="border-0 shadow-sm rounded-2xl mb-6">
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium text-slate-800">Golden Signals — {{ goldenDimensionTypeLabel[goldenDimension] || goldenDimension }}维度</span>
          <div class="flex items-center gap-2">
            <el-tag :type="sloTagType()">
              SLO {{ goldenSignals.slo_breached ? '未达标' : '达标' }}
              (A>= {{ goldenSignals.slo_target_availability }}%, L<= {{ goldenSignals.slo_target_latency_ms }}ms)
            </el-tag>
            <el-button size="small" plain @click="openSLODialog">设置 SLO</el-button>
          </div>
        </div>
      </template>
      <el-table :data="goldenDimensionRows.slice(0, 15)" size="small" stripe empty-text="暂无维度数据">
        <el-table-column prop="dimension_name" :label="goldenDimension === 'instance' ? '实例(主机)' : goldenDimension === 'interface' ? '指标类型' : '服务/IP'" min-width="200" show-overflow-tooltip />
        <el-table-column prop="total_requests" label="采样总数" width="100" align="right" />
        <el-table-column prop="total_errors" label="超阈值" width="80" align="right">
          <template #default="{ row }">
            <span :class="row.total_errors > 0 ? 'text-red-500 font-medium' : ''">{{ row.total_errors }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="error_rate_pct" label="错误率" width="90" align="right">
          <template #default="{ row }">
            <span :class="row.error_rate_pct > 5 ? 'text-red-500 font-medium' : ''">{{ row.error_rate_pct }}%</span>
          </template>
        </el-table-column>
        <el-table-column prop="avg_latency_ms" label="均值" width="90" align="right">
          <template #default="{ row }">{{ Number(row.avg_latency_ms || 0).toFixed(1) }}</template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-row :gutter="16">
      <el-col :xs="24" :xl="24" class="mb-6">
        <el-row :gutter="16">
          <el-col :xs="24" :lg="12" class="mb-6 lg:mb-0">
            <el-card shadow="never" class="border-0 shadow-sm rounded-2xl h-full flex flex-col">
              <template #header>
                <div class="flex justify-between items-center flex-wrap gap-2">
                  <span class="font-medium text-slate-800">按服务树聚合</span>
                  <span class="text-xs text-slate-500">业务视角查看 Agent 水位</span>
                </div>
              </template>
              <el-table :data="serviceTreeAggregates.slice(0, 8)" size="small" stripe border class="w-full">
                <el-table-column prop="name" label="服务树" min-width="180" show-overflow-tooltip />
                <el-table-column label="状态" width="80" align="center">
                  <template #default="{ row }">
                    <el-tag size="small" :type="aggregateStatus(row)" effect="plain" round>
                      {{ Number(row.offline_total || 0) > 0 ? '关注' : '健康' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="agent_total" label="主机数" width="70" align="center" />
                <el-table-column label="在线 / 离线" width="100" align="center">
                  <template #default="{ row }">
                    <span class="text-emerald-600 font-medium">{{ row.online_total }}</span>
                    <span class="text-slate-300 mx-1">/</span>
                    <span class="text-red-500 font-medium">{{ row.offline_total }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="CPU" width="80" align="center">
                  <template #default="{ row }">{{ formatPercent(row.avg_cpu_usage_pct) }}</template>
                </el-table-column>
                <el-table-column label="内存" width="80" align="center">
                  <template #default="{ row }">{{ formatPercent(row.avg_memory_usage_pct) }}</template>
                </el-table-column>
                <el-table-column label="磁盘" width="80" align="center">
                  <template #default="{ row }">{{ formatPercent(row.avg_disk_usage_pct) }}</template>
                </el-table-column>
              </el-table>
              <el-empty v-if="!serviceTreeAggregates.length" description="暂无服务树聚合数据" :image-size="54" />
            </el-card>
          </el-col>

          <el-col :xs="24" :lg="12">
            <el-card shadow="never" class="border-0 shadow-sm rounded-2xl h-full flex flex-col">
              <template #header>
                <div class="flex justify-between items-center flex-wrap gap-2">
                  <span class="font-medium text-slate-800">按负责人聚合</span>
                  <span class="text-xs text-slate-500">谁在负责异常对象，一眼看到</span>
                </div>
              </template>
              <el-table :data="ownerAggregates.slice(0, 8)" size="small" stripe border class="w-full">
                <el-table-column prop="name" label="负责人" min-width="160" show-overflow-tooltip>
                  <template #default="{ row }">
                    <span>{{ row.name || '-' }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="状态" width="80" align="center">
                  <template #default="{ row }">
                    <el-tag size="small" :type="aggregateStatus(row)" effect="plain" round>
                      {{ Number(row.offline_total || 0) > 0 ? '关注' : '健康' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="agent_total" label="主机数" width="70" align="center" />
                <el-table-column label="在线 / 离线" width="100" align="center">
                  <template #default="{ row }">
                    <span class="text-emerald-600 font-medium">{{ row.online_total }}</span>
                    <span class="text-slate-300 mx-1">/</span>
                    <span class="text-red-500 font-medium">{{ row.offline_total }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="CPU" width="80" align="center">
                  <template #default="{ row }">{{ formatPercent(row.avg_cpu_usage_pct) }}</template>
                </el-table-column>
                <el-table-column label="内存" width="80" align="center">
                  <template #default="{ row }">{{ formatPercent(row.avg_memory_usage_pct) }}</template>
                </el-table-column>
                <el-table-column label="磁盘" width="80" align="center">
                  <template #default="{ row }">{{ formatPercent(row.avg_disk_usage_pct) }}</template>
                </el-table-column>
              </el-table>
              <el-empty v-if="!ownerAggregates.length" description="暂无负责人聚合数据" :image-size="54" />
            </el-card>
          </el-col>
        </el-row>
      </el-col>

      <el-col :xs="24" :xl="14" class="mb-6 xl:mb-0">
        <el-card shadow="never" class="border-0 shadow-sm rounded-2xl h-full flex flex-col">
          <template #header>
            <div class="flex flex-col lg:flex-row justify-between lg:items-center gap-4">
              <span class="font-medium text-slate-800">Agent 实时列表</span>
              <div class="flex flex-wrap items-center gap-2">
                <el-input v-model="filters.keyword" clearable placeholder="搜索主机名 / IP / Agent ID" class="w-56" @keyup.enter="applyFilters">
                  <template #prefix><el-icon><Search /></el-icon></template>
                </el-input>
                <el-select v-model="filters.status" class="w-32" clearable placeholder="全部状态">
                  <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
                <el-button type="primary" @click="applyFilters">筛选</el-button>
                <el-button @click="resetFilters">重置</el-button>
              </div>
            </div>
          </template>

          <el-table :data="agents" stripe border class="w-full">
            <el-table-column prop="hostname" label="主机名" min-width="160" show-overflow-tooltip>
              <template #default="{ row }">
                <span class="font-medium text-slate-800">{{ row.hostname || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="ip" label="IP" width="130" align="center">
              <template #default="{ row }">
                <span class="text-slate-600">{{ row.ip }}</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="statusTagType(row.status)" effect="light" round size="small">{{ statusLabel(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="CPU" width="90" align="center">
              <template #default="{ row }">
                <span :class="row.cpu_usage_pct > 80 ? 'text-red-500 font-medium' : ''">{{ formatPercent(row.cpu_usage_pct) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="内存" width="90" align="center">
              <template #default="{ row }">
                <span :class="row.memory_usage_pct > 80 ? 'text-red-500 font-medium' : ''">{{ formatPercent(row.memory_usage_pct) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="磁盘" width="90" align="center">
              <template #default="{ row }">
                <span :class="row.disk_usage_pct > 80 ? 'text-red-500 font-medium' : ''">{{ formatPercent(row.disk_usage_pct) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="last_heartbeat" label="最后心跳" width="160" align="center" />
            <el-table-column label="操作" width="120" fixed="right" align="center">
              <template #default="{ row }">
                <div class="flex items-center justify-center gap-1">
                  <el-button link type="primary" @click="openAgentDetail(row)">趋势</el-button>
                  <el-divider direction="vertical" />
                  <el-button link type="warning" @click="goAgentDetail(row)">详情</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>

          <div class="mt-4 flex justify-end">
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
        <el-row :gutter="16" class="h-full">
          <el-col :span="24" class="mb-6">
            <el-card shadow="never" class="border-0 shadow-sm rounded-2xl h-full">
              <template #header><span class="font-medium text-slate-800">热点 Top</span></template>
              <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div class="flex flex-col gap-3">
                  <div class="text-sm font-semibold text-slate-600 mb-1 border-b border-slate-100 pb-2">CPU Top</div>
                  <div v-if="summary.cpu_high_agents?.length" class="flex flex-col gap-2">
                    <div v-for="agent in summary.cpu_high_agents" :key="`cpu-${agent.agent_id}`" class="flex justify-between items-center p-2.5 rounded-lg bg-slate-50 hover:bg-slate-100 transition-colors text-sm">
                      <span class="truncate pr-2 text-slate-700">{{ agent.hostname || agent.agent_id }}</span>
                      <strong class="text-red-500">{{ formatPercent(agent.cpu_usage_pct) }}</strong>
                    </div>
                  </div>
                  <el-empty v-else description="暂无高 CPU Agent" :image-size="46" />
                </div>
                <div class="flex flex-col gap-3">
                  <div class="text-sm font-semibold text-slate-600 mb-1 border-b border-slate-100 pb-2">内存 Top</div>
                  <div v-if="summary.memory_high_agents?.length" class="flex flex-col gap-2">
                    <div v-for="agent in summary.memory_high_agents" :key="`mem-${agent.agent_id}`" class="flex justify-between items-center p-2.5 rounded-lg bg-slate-50 hover:bg-slate-100 transition-colors text-sm">
                      <span class="truncate pr-2 text-slate-700">{{ agent.hostname || agent.agent_id }}</span>
                      <strong class="text-red-500">{{ formatPercent(agent.memory_usage_pct) }}</strong>
                    </div>
                  </div>
                  <el-empty v-else description="暂无高内存 Agent" :image-size="46" />
                </div>
                <div class="flex flex-col gap-3">
                  <div class="text-sm font-semibold text-slate-600 mb-1 border-b border-slate-100 pb-2">磁盘 Top</div>
                  <div v-if="summary.disk_high_agents?.length" class="flex flex-col gap-2">
                    <div v-for="agent in summary.disk_high_agents" :key="`disk-${agent.agent_id}`" class="flex justify-between items-center p-2.5 rounded-lg bg-slate-50 hover:bg-slate-100 transition-colors text-sm">
                      <span class="truncate pr-2 text-slate-700">{{ agent.hostname || agent.agent_id }}</span>
                      <strong class="text-red-500">{{ formatPercent(agent.disk_usage_pct) }}</strong>
                    </div>
                  </div>
                  <el-empty v-else description="暂无高磁盘 Agent" :image-size="46" />
                </div>
              </div>
            </el-card>
          </el-col>

          <el-col :span="24">
            <el-card shadow="never" class="border-0 shadow-sm rounded-2xl h-full flex flex-col">
              <template #header><span class="font-medium text-slate-800">最近告警</span></template>
              <div v-if="alertEvents.length" class="flex flex-col gap-3">
                <div v-for="item in alertEvents" :key="item.id" class="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3 p-3 rounded-xl bg-slate-50 hover:bg-slate-100 border border-slate-100 transition-colors">
                  <div class="flex-1 min-w-0">
                    <div class="text-sm font-semibold text-slate-800 truncate">{{ item.rule_name }}</div>
                    <div class="mt-1 text-xs text-slate-500 truncate">{{ item.hostname || item.agent_id }} · {{ metricLabel(item.metric_type) }}</div>
                  </div>
                  <div class="flex items-center gap-2 shrink-0">
                    <el-tag size="small" :type="severityTagType(item.severity)" effect="dark">{{ severityLabel(item.severity) }}</el-tag>
                    <el-tag size="small" :type="statusTagType(item.status)" effect="plain">{{ statusLabel(item.status) }}</el-tag>
                  </div>
                </div>
                <div class="mt-2 text-right">
                  <el-button link type="primary" @click="goAlertCenter()">
                    进入告警事件中心 <el-icon class="ml-1"><ArrowRight /></el-icon>
                  </el-button>
                </div>
              </div>
              <el-empty v-else description="暂无告警事件" :image-size="60" />
            </el-card>
          </el-col>
        </el-row>
      </el-col>
    </el-row>

    <!-- 异常检测 + 容量预测 -->
    <el-row :gutter="16" class="mt-6">
      <el-col :xs="24" :lg="12" class="mb-6 lg:mb-0">
        <el-card shadow="never" class="border-0 shadow-sm rounded-2xl h-full" v-loading="anomalyLoading">
          <template #header>
            <div class="flex justify-between items-center">
              <span class="font-medium text-slate-800">指标异常检测</span>
              <el-button size="small" plain @click="loadAnomalies">刷新</el-button>
            </div>
          </template>
          <el-table v-if="anomalies.length" :data="anomalies.slice(0, 10)" size="small" stripe>
            <el-table-column prop="agent_id" label="Agent" min-width="120" show-overflow-tooltip />
            <el-table-column prop="metric_type" label="指标" width="100">
              <template #default="{ row }">{{ metricLabel(row.metric_type) }}</template>
            </el-table-column>
            <el-table-column prop="current_value" label="当前值" width="80" align="right">
              <template #default="{ row }">{{ Number(row.current_value || 0).toFixed(1) }}</template>
            </el-table-column>
            <el-table-column prop="baseline" label="基线" width="70" align="right">
              <template #default="{ row }">{{ Number(row.baseline || 0).toFixed(1) }}</template>
            </el-table-column>
            <el-table-column prop="deviation" label="偏差" width="80" align="right">
              <template #default="{ row }">
                <span class="text-red-500 font-medium">{{ Number(row.deviation || 0).toFixed(1) }}σ</span>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-else description="未发现异常指标" :image-size="54" />
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="12">
        <el-card shadow="never" class="border-0 shadow-sm rounded-2xl h-full" v-loading="predictionLoading">
          <template #header>
            <div class="flex justify-between items-center">
              <span class="font-medium text-slate-800">容量预测（磁盘）</span>
              <el-button size="small" plain @click="loadPredictions">刷新</el-button>
            </div>
          </template>
          <el-table v-if="predictions.length" :data="predictions.slice(0, 10)" size="small" stripe>
            <el-table-column prop="agent_id" label="Agent" min-width="120" show-overflow-tooltip />
            <el-table-column prop="hostname" label="主机" min-width="100" show-overflow-tooltip />
            <el-table-column prop="current_value" label="当前" width="70" align="right">
              <template #default="{ row }">{{ Number(row.current_value || 0).toFixed(1) }}%</template>
            </el-table-column>
            <el-table-column prop="predicted_days" label="预计耗尽" width="90" align="right">
              <template #default="{ row }">
                <span :class="(row.predicted_days || 999) < 30 ? 'text-red-500 font-medium' : 'text-slate-600'">
                  {{ row.predicted_days > 0 ? row.predicted_days + '天' : '安全' }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="trend_per_day" label="日增长" width="80" align="right">
              <template #default="{ row }">{{ Number(row.trend_per_day || 0).toFixed(2) }}%</template>
            </el-table-column>
          </el-table>
          <el-empty v-else description="暂无预测数据" :image-size="54" />
        </el-card>
      </el-col>
    </el-row>

    <!-- SLO 设置弹窗 -->
    <el-dialog v-model="sloDialogVisible" title="SLO 目标设置" width="400px" append-to-body>
      <el-form label-position="top">
        <el-form-item label="可用性目标 (%)">
          <el-input-number v-model="sloForm.target_availability" :min="90" :max="100" :step="0.1" :precision="2" class="w-full" />
        </el-form-item>
        <el-form-item label="延迟目标 (ms)">
          <el-input-number v-model="sloForm.target_latency_ms" :min="100" :max="60000" :step="100" class="w-full" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="sloDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveSLOConfig">保存</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" size="720px" :title="currentAgent ? `${currentAgent.hostname || currentAgent.agent_id} 指标趋势` : '指标趋势'">
      <div v-loading="trendLoading" class="flex flex-col gap-5 p-2">
        <div v-if="currentAgent" class="flex justify-between items-start gap-3">
          <div>
            <div class="text-xl font-bold text-slate-900">{{ currentAgent.hostname || currentAgent.agent_id }}</div>
            <div class="mt-1 text-sm text-slate-500 flex items-center gap-2">
              <span>{{ currentAgent.ip }}</span>
              <el-divider direction="vertical" />
              <el-tag :type="statusTagType(currentAgent.status)" size="small" effect="plain" round>{{ statusLabel(currentAgent.status) }}</el-tag>
            </div>
          </div>
          <el-button plain @click="fetchTrends" size="small">刷新趋势</el-button>
        </div>

        <el-row :gutter="16">
          <el-col :span="8">
            <el-card shadow="never" class="border-0 bg-slate-50 rounded-xl">
              <div class="text-xs text-slate-500">内存占用</div>
              <div class="mt-2 text-xl font-bold text-slate-800">{{ formatBytes(currentAgent?.memory_used) }}</div>
              <div class="mt-1 text-xs text-slate-400">总计 {{ formatBytes(currentAgent?.memory_total) }}</div>
            </el-card>
          </el-col>
          <el-col :span="8">
            <el-card shadow="never" class="border-0 bg-slate-50 rounded-xl">
              <div class="text-xs text-slate-500">磁盘占用</div>
              <div class="mt-2 text-xl font-bold text-slate-800">{{ formatBytes(currentAgent?.disk_used) }}</div>
              <div class="mt-1 text-xs text-slate-400">总计 {{ formatBytes(currentAgent?.disk_total) }}</div>
            </el-card>
          </el-col>
          <el-col :span="8">
            <el-card shadow="never" class="border-0 bg-slate-50 rounded-xl">
              <div class="text-xs text-slate-500">最后心跳</div>
              <div class="mt-2 text-sm font-bold text-slate-800 h-[28px] flex items-center">{{ currentAgent?.last_heartbeat || '-' }}</div>
              <div class="mt-1 text-xs text-slate-400">Agent 在线性</div>
            </el-card>
          </el-col>
        </el-row>

        <div class="flex flex-col gap-4">
          <el-card v-for="metricType in ['cpu_usage', 'memory_usage', 'disk_usage']" :key="metricType" shadow="never" class="border border-slate-100 rounded-xl">
            <template #header>
              <div class="flex justify-between items-center font-medium text-sm">
                <span class="text-slate-800">{{ metricLabel(metricType) }}</span>
                <span class="text-indigo-600">{{ metricLatest(metricType) }}</span>
              </div>
            </template>
            <div v-if="(trends[metricType] || []).length" class="flex flex-col gap-3">
              <svg viewBox="0 0 100 100" preserveAspectRatio="none" class="w-full h-32 rounded-lg bg-gradient-to-b from-indigo-50/50 to-transparent">
                <path d="M 0 92 L 100 92" stroke="#e2e8f0" stroke-width="1" fill="none" />
                <path :d="metricPath(metricType)" stroke="#4f46e5" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
              <div class="flex flex-wrap justify-between gap-3 text-xs text-slate-500">
                <span>最小: <strong class="text-slate-700">{{ metricMin(metricType) }}</strong></span>
                <span>最大: <strong class="text-slate-700">{{ metricMax(metricType) }}</strong></span>
                <span>最近: {{ latestPointTime(metricType) }}</span>
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
:deep(.el-card__header) {
  border-bottom: 1px solid #f1f5f9;
}
:deep(.el-table) {
  --el-table-border-color: #f1f5f9;
}
</style>
