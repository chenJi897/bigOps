<template>
  <div class="p-6 max-w-7xl mx-auto">
    <div class="flex items-center justify-between mb-8">
      <div class="flex items-center gap-4">
        <el-button @click="goBack" class="rounded-full">
          <el-icon><ArrowLeft /></el-icon>
        </el-button>
        <div>
          <div class="flex items-center gap-3">
            <h1 class="text-3xl font-bold text-gray-800">{{ execution.task_name }}</h1>
            <el-tag :type="getStatusType(execution.status)" size="large" effect="light" class="text-base px-4">
              {{ getStatusText(execution.status) }}
            </el-tag>
          </div>
          <div class="text-gray-500 font-mono text-sm mt-1">{{ execution.execution_id }}</div>
        </div>
      </div>
      
      <div class="flex items-center gap-6 text-sm">
        <div>
          <div class="text-gray-400">执行人</div>
          <div class="font-medium">{{ execution.operator_name }}</div>
        </div>
        <div>
          <div class="text-gray-400">开始时间</div>
          <div class="font-medium">{{ execution.started_at }}</div>
        </div>
        <div>
          <div class="text-gray-400">耗时</div>
          <div class="font-medium font-mono">{{ execution.duration }}ms</div>
        </div>
        <div class="flex items-center gap-2">
          <el-button v-if="canCancel" type="danger" size="small" @click="cancelExecution">取消执行</el-button>
          <el-button v-if="canRetry" type="warning" size="small" @click="retryExecution">重试失败主机</el-button>
          <el-button v-if="canRetry" type="success" size="small" @click="retryExecutionAll">重试全部主机</el-button>
          <el-button size="small" @click="exportMarkdownReport">导出报告</el-button>
        </div>
      </div>
    </div>

    <!-- 进度概览 -->
    <div class="bg-white rounded-3xl p-8 shadow-sm mb-8">
      <div class="flex justify-between items-end mb-6">
        <div class="text-lg font-semibold">执行进度</div>
        <div class="text-right">
          <div class="text-5xl font-bold text-emerald-600">{{ execution.progress }}<span class="text-2xl">%</span></div>
          <div class="text-sm text-gray-500">{{ execution.success_count }}/{{ execution.total_count }} 主机成功</div>
        </div>
      </div>
      <el-progress :percentage="execution.progress" :stroke-width="12" status="success" />
      <div class="grid grid-cols-3 gap-3 mt-4">
        <div class="rounded-xl border border-emerald-100 bg-emerald-50 px-4 py-3">
          <div class="text-xs text-emerald-700">成功率</div>
          <div class="text-xl font-semibold text-emerald-700">{{ successRate }}%</div>
        </div>
        <div class="rounded-xl border border-rose-100 bg-rose-50 px-4 py-3">
          <div class="text-xs text-rose-700">失败率</div>
          <div class="text-xl font-semibold text-rose-700">{{ failRate }}%</div>
        </div>
        <div class="rounded-xl border border-blue-100 bg-blue-50 px-4 py-3">
          <div class="text-xs text-blue-700">平均耗时</div>
          <div class="text-xl font-semibold text-blue-700">{{ avgHostDuration }}ms</div>
        </div>
      </div>
    </div>

    <!-- 主机执行结果 -->
    <div class="bg-white rounded-3xl shadow-sm overflow-hidden">
      <div class="px-8 py-5 border-b flex items-center justify-between bg-gray-50">
        <div class="font-semibold text-lg">主机执行结果 ({{ execution.total_count }})</div>
        <el-input v-model="hostFilter" placeholder="过滤主机IP..." class="w-72" size="small" clearable />
      </div>
      
      <el-table :data="filteredHosts" :show-header="true" style="width: 100%">
        <el-table-column prop="host_ip" label="主机IP" width="140" />
        <el-table-column prop="hostname" label="主机名" width="180" />
        <el-table-column prop="status" label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="getHostStatusType(row.status)" size="small">
              {{ getHostStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="exit_code" label="退出码" width="90" />
        <el-table-column prop="duration" label="耗时" width="100">
          <template #default="{ row }">
            <span class="font-mono">{{ row.duration }}ms</span>
          </template>
        </el-table-column>
        <el-table-column prop="error_summary" label="错误摘要" min-width="160" show-overflow-tooltip />
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button type="primary" size="small" link @click="viewHostLog(row)">
              查看日志
            </el-button>
            <el-button
              v-if="row.status === 'failed' || row.status === 'timeout' || row.status === 'canceled'"
              type="warning"
              size="small"
              link
              @click="retryOneHost(row.host_ip)"
            >
              重试本机
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 实时日志抽屉 -->
    <el-drawer
      v-model="logDrawerVisible"
      :title="currentHost ? `实时日志 - ${currentHost.host_ip}` : '日志'"
      direction="rtl"
      size="65%"
    >
      <div class="h-full flex flex-col">
        <div class="px-4 py-3 border-b bg-gray-50 text-xs text-gray-600 flex items-center justify-between gap-4">
          <div class="flex items-center gap-4">
            <span>连接状态：{{ wsStateText }}</span>
            <span>日志条数：{{ logs.length }}</span>
          </div>
          <div class="flex items-center gap-2">
            <el-input
              v-model="logContentFilter"
              size="small"
              clearable
              placeholder="按内容过滤"
              class="w-44"
            />
            <el-input
              v-model="logHostFilter"
              size="small"
              clearable
              placeholder="按 Host 过滤"
              class="w-40"
            />
            <el-select v-model="logPhaseFilter" size="small" class="w-36">
              <el-option label="全部阶段" value="" />
              <el-option label="running" value="running" />
              <el-option label="finished" value="finished" />
              <el-option label="error" value="error" />
              <el-option label="回放(replay)" value="replay" />
              <el-option label="落库快照(captured)" value="captured" />
            </el-select>
            <el-checkbox v-model="stderrOnly" size="small">仅错误</el-checkbox>
          </div>
        </div>
        <div ref="logViewportRef" class="flex-1 bg-[#1e2937] text-[#e2e8f0] p-6 font-mono text-sm overflow-auto" style="font-size: 13px; line-height: 1.6;">
          <div v-for="(line, i) in filteredLogs" :key="i" 
               :class="line.isStderr ? 'text-red-400' : 'text-emerald-300'">
            <span class="text-gray-500 mr-4">{{ line.timestamp }}</span>
            <span v-if="line.hostIP" class="text-cyan-300 mr-3">[{{ line.hostIP }}]</span>
            <span v-if="line.phase" class="text-amber-300 mr-3">[{{ line.phase }}]</span>
            {{ line.content }}
          </div>
          <div v-if="filteredLogs.length === 0" class="text-gray-500 italic py-12 text-center">
            暂无日志输出...
          </div>
        </div>
        
        <div class="p-4 border-t bg-white flex gap-3">
          <el-button @click="reconnectLogs">重连</el-button>
          <el-button @click="togglePause">{{ isPaused ? '继续' : '暂停' }}</el-button>
          <el-button @click="clearLogs">清空</el-button>
          <el-button @click="logDrawerVisible = false">关闭</el-button>
          <el-button type="primary" @click="exportLogs">导出日志</el-button>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { taskApi } from '@/api'

type LogLine = {
  timestamp: string
  content: string
  isStderr: boolean
  hostIP?: string
  phase?: string
}

const router = useRouter()
const route = useRoute()
const executionID = Number(route.params.id || 0)

const execution = ref({
  execution_id: String(route.params.id || "1"),
  task_name: "重启Nginx服务",
  status: "pending",
  progress: 0,
  success_count: 0,
  total_count: 0,
  operator_name: "-",
  started_at: "-",
  duration: 0
})

const hosts = ref<any[]>([])

const logs = ref<LogLine[]>([])
const logDrawerVisible = ref(false)
const currentHost = ref<any>(null)
const hostFilter = ref('')
const logViewportRef = ref<HTMLElement | null>(null)
const wsState = ref<'idle' | 'connecting' | 'connected' | 'closed' | 'error'>('idle')
const isPaused = ref(false)
const logContentFilter = ref('')
const logHostFilter = ref('')
const logPhaseFilter = ref('')
const stderrOnly = ref(false)
let ws: WebSocket | null = null
let wsReconnectTimer: ReturnType<typeof setTimeout> | null = null
let wsReconnectAttempt = 0

const getStatusType = (status: string) => {
  if (status === 'success') return 'success'
  if (status === 'failed' || status === 'error' || status === 'partial_fail') return 'danger'
  if (status === 'running') return 'warning'
  return 'info'
}
const getStatusText = (status: string) => {
  const map: Record<string, string> = {
    pending: '待执行',
    running: '执行中',
    success: '执行成功',
    failed: '执行失败',
    error: '执行异常',
    partial_fail: '部分失败',
    canceled: '已取消',
  }
  return map[status] || status
}

const getHostStatusType = (status: string) =>
  status === 'success' ? 'success' : status === 'failed' ? 'danger' : status === 'canceled' ? 'info' : 'warning'
const getHostStatusText = (status: string) => {
  const map: any = { success: '成功', failed: '失败', running: '运行中', timeout: '超时', pending: '等待', canceled: '已取消' }
  return map[status] || status
}

const filteredHosts = computed(() => {
  const kw = hostFilter.value.trim()
  if (!kw) return hosts.value
  return hosts.value.filter(h => h.host_ip.includes(kw) || h.hostname.includes(kw))
})

const wsStateText = computed(() => {
  const map = {
    idle: '未连接',
    connecting: '连接中',
    connected: '已连接',
    closed: '已断开',
    error: '连接异常',
  }
  return map[wsState.value]
})

const canCancel = computed(() => execution.value.status === 'pending' || execution.value.status === 'running')
const canRetry = computed(() =>
  execution.value.status === 'failed' || execution.value.status === 'partial_fail' || execution.value.status === 'canceled',
)
const successRate = computed(() => {
  if (!execution.value.total_count) return 0
  return Math.round((execution.value.success_count / execution.value.total_count) * 100)
})
const failRate = computed(() => {
  if (!execution.value.total_count) return 0
  const realFailCount = hosts.value.filter(
    (h: any) => h.status === 'failed' || h.status === 'timeout' || h.status === 'canceled',
  ).length
  return Math.round((realFailCount / execution.value.total_count) * 100)
})
const avgHostDuration = computed(() => {
  if (!hosts.value.length) return 0
  const durations = hosts.value
    .map((item: any) => Number(item.duration || 0))
    .filter((ms: number) => Number.isFinite(ms) && ms > 0)
  if (!durations.length) return 0
  const total = durations.reduce((sum: number, ms: number) => sum + ms, 0)
  return Math.round(total / durations.length)
})

const filteredLogs = computed(() => {
  const contentKw = logContentFilter.value.trim().toLowerCase()
  const kw = logHostFilter.value.trim()
  return logs.value.filter(line => {
    if (contentKw && !(line.content || '').toLowerCase().includes(contentKw)) return false
    if (kw && !(line.hostIP || '').includes(kw)) return false
    if (logPhaseFilter.value && (line.phase || '') !== logPhaseFilter.value) return false
    if (stderrOnly.value && !line.isStderr) return false
    return true
  })
})

function formatLogTimestamp(ts: unknown): string {
  if (typeof ts === 'number' && ts > 0) {
    const d = new Date(ts * 1000)
    if (!Number.isNaN(d.getTime())) return d.toLocaleString()
  }
  if (typeof ts === 'string' && ts.trim()) return ts
  return new Date().toLocaleTimeString()
}

function resolveWsURL(
  executionID: string,
  q: { token?: string | null; replay?: string; hostIp?: string },
) {
  const apiBase = (import.meta as any).env?.VITE_API_BASE_URL as string | undefined
  let base: string
  if (apiBase) {
    try {
      const parsed = new URL(apiBase)
      const wsProtocol = parsed.protocol === 'https:' ? 'wss:' : 'ws:'
      base = `${wsProtocol}//${parsed.host}/api/v1/ws/task-executions/${executionID}/logs`
    } catch {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      base = `${protocol}//${window.location.host}/api/v1/ws/task-executions/${executionID}/logs`
    }
  } else {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    base = `${protocol}//${window.location.host}/api/v1/ws/task-executions/${executionID}/logs`
  }
  const params = new URLSearchParams()
  if (q.token) params.set('token', q.token)
  if (q.replay != null && q.replay !== '') params.set('replay', q.replay)
  if (q.hostIp) params.set('host_ip', q.hostIp)
  const qs = params.toString()
  return qs ? `${base}?${qs}` : base
}

function appendLog(line: LogLine) {
  logs.value.push(line)
  if (logs.value.length > 5000) {
    logs.value.splice(0, logs.value.length - 5000)
  }
}

/** 将已落库的主机 stdout/stderr 填入抽屉（执行结束后 WebSocket 往往无回放） */
function hydrateLogsFromHostCapture(host: any) {
  const ip = host?.host_ip || host?.hostIP || ''
  const stamp = new Date().toLocaleTimeString()
  const splitLines = (s: unknown) => {
    if (s == null || String(s).trim() === '') return [] as string[]
    return String(s).split(/\r?\n/)
  }
  const out = splitLines(host?.stdout ?? host?.Stdout)
  const err = splitLines(host?.stderr ?? host?.Stderr)
  for (const line of out) {
    if (line.length) appendLog({ timestamp: stamp, content: line, isStderr: false, hostIP: ip, phase: 'captured' })
  }
  for (const line of err) {
    if (line.length) appendLog({ timestamp: stamp, content: line, isStderr: true, hostIP: ip, phase: 'captured' })
  }
}

function applyRealtimeHostUpdate(payload: any) {
  const hostIP = payload.host_ip
  if (!hostIP) return
  const idx = hosts.value.findIndex((h: any) => h.host_ip === hostIP)
  if (idx < 0) return
  const current = hosts.value[idx]
  let nextStatus = current.status
  if (payload.phase === 'running') nextStatus = 'running'
  if (payload.phase === 'finished' || payload.phase === 'success') nextStatus = 'success'
  if (payload.phase === 'error') nextStatus = 'failed'
  hosts.value[idx] = {
    ...current,
    status: nextStatus,
    exit_code: payload.exit_code ?? current.exit_code,
  }
}

async function scrollToBottom() {
  await nextTick()
  if (logViewportRef.value) {
    logViewportRef.value.scrollTop = logViewportRef.value.scrollHeight
  }
}

function closeWS() {
  if (ws) {
    ws.close()
    ws = null
  }
}

function calculateDurationMs(startedAt?: string, finishedAt?: string) {
  if (!startedAt) return 0
  const start = new Date(startedAt).getTime()
  if (Number.isNaN(start)) return 0
  const end = finishedAt ? new Date(finishedAt).getTime() : Date.now()
  if (Number.isNaN(end) || end < start) return 0
  return end - start
}

function calcProgress(successCount: number, failCount: number, totalCount: number) {
  if (!totalCount) return 0
  const done = successCount + failCount
  return Math.max(0, Math.min(100, Math.round((done / totalCount) * 100)))
}

async function refreshExecutionDetail() {
  if (!executionID) return
  const resp = await taskApi.getExecution(executionID)
  const data = (resp as any)?.data || {}

  const total = Number(data.total_count || 0)
  const success = Number(data.success_count || 0)
  const fail = Number(data.fail_count || 0)

  execution.value = {
    execution_id: String(data.id || execution.value.execution_id),
    task_name: data.task_name || execution.value.task_name,
    status: data.status || execution.value.status,
    progress: calcProgress(success, fail, total),
    success_count: success,
    total_count: total,
    operator_name: data.operator_name || '-',
    started_at: data.started_at || '-',
    duration: calculateDurationMs(data.started_at, data.finished_at),
  }

  hosts.value = Array.isArray(data.host_results)
    ? data.host_results.map((h: any) => ({
        ...h,
        duration:
          Number(h.duration_ms || 0) ||
          calculateDurationMs(h.started_at, h.finished_at),
      }))
    : []
}

function scheduleWsReconnect() {
  if (!logDrawerVisible.value) return
  const st = execution.value.status
  if (st !== 'pending' && st !== 'running') return
  if (wsReconnectAttempt >= 12) {
    ElMessage.warning('日志连接已多次重试失败，请点击「重连」')
    return
  }
  const delay = Math.min(30000, 1000 * Math.pow(2, wsReconnectAttempt))
  wsReconnectAttempt++
  wsReconnectTimer = setTimeout(() => {
    wsReconnectTimer = null
    if (!logDrawerVisible.value) return
    openWS(true)
  }, delay)
}

/** @param isReconnect 为 true 时从 DB 回放（replay=1），用于断线恢复 */
function openWS(isReconnect = false) {
  closeWS()
  if (isReconnect) {
    logs.value = []
  }
  wsState.value = 'connecting'
  const token = localStorage.getItem('token')
  const replay = isReconnect ? '1' : '0'
  const hostIp = currentHost.value?.host_ip || ''
  const url = resolveWsURL(execution.value.execution_id, { token, replay, hostIp })
  ws = new WebSocket(url)

  ws.onopen = () => {
    wsState.value = 'connected'
    wsReconnectAttempt = 0
  }
  ws.onerror = () => {
    wsState.value = 'error'
  }
  ws.onclose = () => {
    wsState.value = 'closed'
    if (logDrawerVisible.value && (execution.value.status === 'pending' || execution.value.status === 'running')) {
      scheduleWsReconnect()
    }
  }
  ws.onmessage = (event: MessageEvent<string>) => {
    if (isPaused.value) return
    try {
      const payload = JSON.parse(event.data || '{}')
      const hip = payload.host_ip || ''
      if (
        currentHost.value?.host_ip &&
        hip &&
        hip !== currentHost.value.host_ip
      ) {
        return
      }
      appendLog({
        timestamp: formatLogTimestamp(payload.timestamp),
        content: payload.content ?? payload.output_line ?? payload.line ?? JSON.stringify(payload),
        isStderr: Boolean(payload.is_stderr),
        hostIP: hip,
        phase: payload.phase,
      })
      const totalFromEvent = Number(payload.total_count || 0)
      const doneFromEvent = Number(payload.done_count || 0)
      const successFromEvent = Number(payload.success_count || execution.value.success_count || 0)
      const failFromEvent = Number(payload.fail_count || 0)
      if (totalFromEvent > 0) {
        execution.value.total_count = totalFromEvent
        execution.value.success_count = successFromEvent
        execution.value.progress = calcProgress(successFromEvent, failFromEvent, totalFromEvent)
      } else if (doneFromEvent > 0 && execution.value.total_count > 0) {
        const inferredFail = Math.max(0, doneFromEvent - successFromEvent)
        execution.value.success_count = successFromEvent
        execution.value.progress = calcProgress(successFromEvent, inferredFail, execution.value.total_count)
      }
      applyRealtimeHostUpdate(payload)
      if (payload.phase === 'error') {
        refreshExecutionDetail().catch(() => {})
      }
      if (payload.phase === 'finished' || payload.phase === 'success') {
        refreshExecutionDetail().catch(() => {})
      }
      scrollToBottom()
    } catch {
      appendLog({
        timestamp: new Date().toLocaleTimeString(),
        content: event.data || '',
        isStderr: false,
      })
      scrollToBottom()
    }
  }
}

const viewHostLog = (host: any) => {
  currentHost.value = host
  logs.value = []
  hydrateLogsFromHostCapture(host)
  logDrawerVisible.value = true
  wsReconnectAttempt = 0
  const st = execution.value.status
  if (st === 'pending' || st === 'running') {
    openWS(false)
  } else {
    wsState.value = 'idle'
  }
}

const reconnectLogs = () => {
  const st = execution.value.status
  if (st === 'pending' || st === 'running') {
    wsReconnectAttempt = 0
    if (wsReconnectTimer) {
      clearTimeout(wsReconnectTimer)
      wsReconnectTimer = null
    }
    openWS(true)
    return
  }
  if (currentHost.value) {
    logs.value = []
    hydrateLogsFromHostCapture(currentHost.value)
    ElMessage.success('已从落库结果刷新日志')
  }
}

const clearLogs = () => {
  logs.value = []
}

const togglePause = () => {
  isPaused.value = !isPaused.value
  ElMessage.info(isPaused.value ? '已暂停日志滚动' : '已恢复日志滚动')
}

const exportLogs = () => {
  const content = logs.value.map(l => `[${l.timestamp}] ${l.isStderr ? 'ERR' : 'OUT'} ${l.content}`).join('\n')
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `execution-${execution.value.execution_id}-logs.txt`
  a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('日志已导出')
}

const exportMarkdownReport = () => {
  const url = taskApi.executionReportUrl(executionID, 'markdown')
  const token = localStorage.getItem('token')
  const a = document.createElement('a')
  a.href = `${url}&token=${token}`
  a.download = `execution-${executionID}-report.md`
  a.target = '_blank'
  a.click()
  ElMessage.success('报告下载中...')
}

const goBack = () => {
  router.push('/task/executions')
}

const cancelExecution = async () => {
  if (!executionID) return
  try {
    await taskApi.cancelExecution(executionID)
    ElMessage.success('执行已取消')
    await refreshExecutionDetail()
  } catch (err: any) {
    ElMessage.error(err?.message || '取消失败')
  }
}

const retryExecution = async () => {
  if (!executionID) return
  try {
    const resp = await taskApi.retryExecution(executionID, 'failed')
    const newID = (resp as any)?.data?.id
    ElMessage.success('已创建失败主机重试执行')
    if (newID) {
      router.push(`/task/executions/${newID}`)
      return
    }
    await refreshExecutionDetail()
  } catch (err: any) {
    ElMessage.error(err?.message || '重试失败')
  }
}

const retryOneHost = async (hostIp: string) => {
  if (!executionID || !hostIp) return
  try {
    const resp = await taskApi.retryExecution(executionID, 'failed', [hostIp])
    const newID = (resp as any)?.data?.id
    ElMessage.success('已为本机创建重试执行')
    if (newID) {
      router.push(`/task/executions/${newID}`)
      return
    }
    await refreshExecutionDetail()
  } catch (err: any) {
    ElMessage.error(err?.message || '重试失败')
  }
}

const retryExecutionAll = async () => {
  if (!executionID) return
  try {
    const resp = await taskApi.retryExecution(executionID, 'all')
    const newID = (resp as any)?.data?.id
    ElMessage.success('已创建全量重试执行')
    if (newID) {
      router.push(`/task/executions/${newID}`)
      return
    }
    await refreshExecutionDetail()
  } catch (err: any) {
    ElMessage.error(err?.message || '重试失败')
  }
}

onMounted(() => {
  refreshExecutionDetail().catch(() => {
    ElMessage.warning('执行详情加载失败，已保留当前显示内容')
  })
})

watch(logDrawerVisible, (visible) => {
  if (!visible) {
    if (wsReconnectTimer) {
      clearTimeout(wsReconnectTimer)
      wsReconnectTimer = null
    }
    wsReconnectAttempt = 0
    closeWS()
    wsState.value = 'closed'
  }
})

onBeforeUnmount(() => {
  if (wsReconnectTimer) clearTimeout(wsReconnectTimer)
  closeWS()
})
</script>