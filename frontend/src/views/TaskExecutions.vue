<template>
  <div class="p-6">
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-800">执行记录</h1>
        <p class="text-gray-500 mt-1">查看所有任务执行历史，支持实时日志和结果分析</p>
      </div>
      <div class="flex gap-3">
        <div class="flex items-center text-xs text-gray-500 px-2">
          <span v-if="isRefreshing" class="text-blue-500">刷新中...</span>
          <span v-else>上次刷新：{{ lastRefreshAtText }}</span>
        </div>
        <el-input v-model="searchQuery" placeholder="搜索执行ID或模板名..." class="w-80" clearable>
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <div class="flex items-center gap-2 rounded-xl border px-3">
          <span class="text-xs text-gray-500">自动刷新</span>
          <el-switch v-model="autoRefreshEnabled" />
          <el-select v-model="refreshIntervalSec" size="small" class="w-24" :disabled="!autoRefreshEnabled">
            <el-option :value="5" label="5s" />
            <el-option :value="10" label="10s" />
            <el-option :value="30" label="30s" />
          </el-select>
        </div>
        <el-button type="primary" @click="refreshList">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 状态统计卡片 -->
    <div class="grid grid-cols-4 gap-4 mb-8">
      <div class="bg-white rounded-3xl p-6 shadow-sm border border-green-100">
        <div class="text-green-500 text-sm font-medium mb-1">成功</div>
        <div class="text-4xl font-bold text-green-600">{{ stats.success }}</div>
        <div class="text-xs text-gray-400 mt-2">本周执行</div>
      </div>
      <div class="bg-white rounded-3xl p-6 shadow-sm border border-amber-100">
        <div class="text-amber-500 text-sm font-medium mb-1">运行中</div>
        <div class="text-4xl font-bold text-amber-600">{{ stats.running }}</div>
        <div class="text-xs text-gray-400 mt-2">实时监控</div>
      </div>
      <div class="bg-white rounded-3xl p-6 shadow-sm border border-red-100">
        <div class="text-red-500 text-sm font-medium mb-1">失败</div>
        <div class="text-4xl font-bold text-red-600">{{ stats.failed }}</div>
        <div class="text-xs text-gray-400 mt-2">需关注</div>
      </div>
      <div class="bg-white rounded-3xl p-6 shadow-sm border border-purple-100">
        <div class="text-purple-500 text-sm font-medium mb-1">总执行</div>
        <div class="text-4xl font-bold text-purple-600">{{ stats.total }}</div>
        <div class="text-xs text-gray-400 mt-2">历史累计</div>
      </div>
    </div>

    <!-- 执行记录表格 -->
    <el-table :data="filteredExecutions" stripe style="width: 100%" class="rounded-3xl overflow-hidden" 
              @row-click="showDetail">
      <el-table-column prop="execution_id" label="执行ID" width="160" />
      <el-table-column prop="task_name" label="任务名称" width="220" />
      <el-table-column prop="status" label="状态" width="110">
        <template #default="{ row }">
          <el-tag 
            :type="getStatusType(row.status)" 
            size="small"
            effect="light"
            class="font-medium">
            {{ getStatusText(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="进度" width="140">
        <template #default="{ row }">
          <div class="flex items-center gap-2">
            <el-progress 
              :percentage="row.progress" 
              :color="getProgressColor(row.progress)"
              :stroke-width="6" 
              style="width: 80px" />
            <span class="text-xs text-gray-500">{{ row.success_count }}/{{ row.total_count }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="operator_name" label="执行人" width="110" />
      <el-table-column prop="started_at" label="开始时间" width="160" />
      <el-table-column prop="duration" label="耗时" width="100">
        <template #default="{ row }">
          <span class="font-mono text-sm">{{ row.duration }}ms</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link size="small" @click.stop="showDetail(row)">
            详情
          </el-button>
          <el-button
            v-if="row.status === 'pending' || row.status === 'running'"
            type="danger"
            link
            size="small"
            @click.stop="cancelExecution(row)"
          >
            取消
          </el-button>
          <el-button
            v-if="row.status === 'failed' || row.status === 'partial_fail' || row.status === 'canceled'"
            type="warning"
            link
            size="small"
            @click.stop="retryExecution(row)"
          >
            重试失败
          </el-button>
          <el-button
            v-if="row.status === 'failed' || row.status === 'partial_fail' || row.status === 'canceled'"
            type="success"
            link
            size="small"
            @click.stop="retryExecutionAll(row)"
          >
            重试全部
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { taskApi } from '@/api'

const AUTO_REFRESH_ENABLED_KEY = 'task-executions:auto-refresh-enabled'
const AUTO_REFRESH_INTERVAL_KEY = 'task-executions:auto-refresh-interval'

const router = useRouter()
const route = useRoute()
const searchQuery = ref('')
const executions = ref<any[]>([])
const autoRefreshEnabled = ref(true)
const refreshIntervalSec = ref(5)
const lastRefreshAt = ref<Date | null>(null)
const isRefreshing = ref(false)
const stats = ref({
  total: 0,
  success: 0,
  running: 0,
  failed: 0,
})
let refreshTimer: number | null = null
let fullRefreshInFlight = false
let activeRefreshInFlight = false
let pendingActiveRefresh = false

const lastRefreshAtText = computed(() => {
  if (!lastRefreshAt.value) return '未刷新'
  return lastRefreshAt.value.toLocaleTimeString()
})

const getStatusType = (status: string) => {
  const map: any = {
    'success': 'success',
    'running': 'warning',
    'failed': 'danger',
    'partial_fail': 'warning',
    'canceled': 'info',
    'pending': 'info',
  }
  return map[status] || 'info'
}

const getStatusText = (status: string) => {
  const map: any = {
    'success': '成功',
    'running': '执行中',
    'failed': '失败',
    'partial_fail': '部分失败',
    'canceled': '已取消',
    'pending': '等待中',
  }
  return map[status] || status
}

const getProgressColor = (progress: number) => {
  if (progress > 90) return '#10b981'
  if (progress > 60) return '#3b82f6'
  return '#f59e0b'
}

function calcProgress(successCount: number, failCount: number, totalCount: number) {
  if (!totalCount) return 0
  const done = successCount + failCount
  return Math.max(0, Math.min(100, Math.round((done / totalCount) * 100)))
}

function calcDurationMs(startedAt?: string, finishedAt?: string) {
  if (!startedAt) return 0
  const start = new Date(startedAt).getTime()
  if (Number.isNaN(start)) return 0
  const end = finishedAt ? new Date(finishedAt).getTime() : Date.now()
  if (Number.isNaN(end) || end < start) return 0
  return end - start
}

const filteredExecutions = computed(() => {
  const kw = searchQuery.value.trim().toLowerCase()
  if (!kw) return executions.value
  return executions.value.filter(item => {
    const id = String(item.execution_id || item.id || '').toLowerCase()
    const name = String(item.task_name || '').toLowerCase()
    return id.includes(kw) || name.includes(kw)
  })
})

const recalculateStats = () => {
  const list = executions.value
  stats.value.total = list.length
  stats.value.success = list.filter(item => item.status === 'success').length
  stats.value.running = list.filter(item => item.status === 'running' || item.status === 'pending').length
  stats.value.failed = list.filter(item => item.status === 'failed' || item.status === 'partial_fail').length
}

const loadExecutions = async (silent = false) => {
  if (fullRefreshInFlight) return
  fullRefreshInFlight = true
  isRefreshing.value = true
  try {
    const resp = await taskApi.executions({ page: 1, size: 100 })
    const list = (resp as any)?.data?.list || []
    executions.value = list.map((item: any) => {
      const success = Number(item.success_count || 0)
      const fail = Number(item.fail_count || 0)
      const total = Number(item.total_count || 0)
      return {
        ...item,
        execution_id: String(item.id || ''),
        progress: calcProgress(success, fail, total),
        duration: calcDurationMs(item.started_at, item.finished_at),
      }
    })
    recalculateStats()
    lastRefreshAt.value = new Date()
    if (!silent) ElMessage.success('执行记录已刷新')
  } catch (err: any) {
    if (!silent) ElMessage.error(err?.message || '执行记录加载失败')
  } finally {
    fullRefreshInFlight = false
    isRefreshing.value = activeRefreshInFlight
  }
}

const refreshActiveExecutions = async () => {
  if (activeRefreshInFlight || fullRefreshInFlight) {
    pendingActiveRefresh = true
    return
  }
  const active = executions.value.filter(item => item.status === 'pending' || item.status === 'running')
  if (!active.length) return
  activeRefreshInFlight = true
  isRefreshing.value = true
  await Promise.all(
    active.map(async (item) => {
      try {
        const resp = await taskApi.getExecution(item.id)
        const data = (resp as any)?.data || {}
        const success = Number(data.success_count || 0)
        const fail = Number(data.fail_count || 0)
        const total = Number(data.total_count || 0)
        const next = {
          ...item,
          ...data,
          execution_id: String(data.id || item.id || ''),
          progress: calcProgress(success, fail, total),
          duration: calcDurationMs(data.started_at, data.finished_at),
        }
        const idx = executions.value.findIndex(row => row.id === item.id)
        if (idx >= 0) executions.value[idx] = next
      } catch {
        // ignore single-row refresh failure to avoid breaking timer loop
      }
    }),
  )
  recalculateStats()
  lastRefreshAt.value = new Date()
  activeRefreshInFlight = false
  isRefreshing.value = false
  if (pendingActiveRefresh) {
    pendingActiveRefresh = false
    refreshActiveExecutions()
  }
}

const refreshList = () => {
  loadExecutions()
}

const stopRefreshTimer = () => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer)
    refreshTimer = null
  }
}

const startRefreshTimer = () => {
  stopRefreshTimer()
  if (!autoRefreshEnabled.value) return
  refreshTimer = window.setInterval(() => {
    refreshActiveExecutions()
  }, refreshIntervalSec.value * 1000)
}

const persistRefreshSettings = () => {
  localStorage.setItem(AUTO_REFRESH_ENABLED_KEY, String(autoRefreshEnabled.value))
  localStorage.setItem(AUTO_REFRESH_INTERVAL_KEY, String(refreshIntervalSec.value))
}

const handleVisibilityChange = () => {
  if (document.visibilityState === 'visible') {
    if (executions.value.length === 0) {
      loadExecutions(true)
      return
    }
    refreshActiveExecutions()
  }
}

const showDetail = (row: any) => {
  router.push(`/task/executions/${row.id}`)
}

const cancelExecution = async (row: any) => {
  try {
    await taskApi.cancelExecution(row.id)
    ElMessage.success('已取消执行')
    await loadExecutions(true)
  } catch (err: any) {
    ElMessage.error(err?.message || '取消失败')
  }
}

const retryExecution = async (row: any) => {
  try {
    await taskApi.retryExecution(row.id, 'failed')
    ElMessage.success('已创建失败主机重试执行')
    await loadExecutions(true)
  } catch (err: any) {
    ElMessage.error(err?.message || '重试失败')
  }
}

const retryExecutionAll = async (row: any) => {
  try {
    await taskApi.retryExecution(row.id, 'all')
    ElMessage.success('已创建全量重试执行')
    await loadExecutions(true)
  } catch (err: any) {
    ElMessage.error(err?.message || '重试全部失败')
  }
}

onMounted(() => {
  const cachedEnabled = localStorage.getItem(AUTO_REFRESH_ENABLED_KEY)
  if (cachedEnabled === 'false') {
    autoRefreshEnabled.value = false
  }
  const cachedInterval = Number(localStorage.getItem(AUTO_REFRESH_INTERVAL_KEY) || 5)
  if ([5, 10, 30].includes(cachedInterval)) {
    refreshIntervalSec.value = cachedInterval
  }

  loadExecutions(true).then(() => {
    if (route.query.new_exec) {
      refreshActiveExecutions()
    }
  })
  startRefreshTimer()
  document.addEventListener('visibilitychange', handleVisibilityChange)
})

watch([autoRefreshEnabled, refreshIntervalSec], () => {
  persistRefreshSettings()
  startRefreshTimer()
})

onBeforeUnmount(() => {
  stopRefreshTimer()
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>