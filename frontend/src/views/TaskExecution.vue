<script setup lang="ts">
defineOptions({ name: 'TaskExecution' })
import { ref, onMounted, onUnmounted, computed, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { taskApi } from '../api'

const route = useRoute()
const router = useRouter()
const execId = computed(() => Number(route.params.id) || 0)
const loading = ref(false)
const execution = ref<any>(null)
const hostResults = ref<any[]>([])
const selectedHostIP = ref('')
const logLines = ref<any[]>([])
const logContainerRef = ref<HTMLElement | null>(null)
let ws: WebSocket | null = null
let wsReconnectTimer: ReturnType<typeof setTimeout> | null = null
let destroyed = false

const execStatusMap: Record<string, { label: string; type: string }> = {
  pending: { label: '等待中', type: 'info' },
  running: { label: '执行中', type: 'warning' },
  success: { label: '成功', type: 'success' },
  partial_fail: { label: '部分失败', type: 'warning' },
  failed: { label: '失败', type: 'danger' },
  timeout: { label: '超时', type: 'danger' },
  canceled: { label: '已取消', type: 'info' },
}

const isRunning = computed(() => {
  const s = execution.value?.status
  return s === 'pending' || s === 'running'
})

const selectedHostLogs = computed(() => {
  if (!selectedHostIP.value) return logLines.value
  return logLines.value.filter(l => l.host_ip === selectedHostIP.value)
})

async function fetchExecution() {
  if (!execId.value) return
  loading.value = true
  try {
    const res: any = await taskApi.getExecution(execId.value)
    execution.value = res.data
    hostResults.value = res.data?.host_results || []
    if (!selectedHostIP.value && hostResults.value.length > 0) {
      selectedHostIP.value = hostResults.value[0].host_ip
    }
    // 已完成时，将 stdout/stderr 作为初始日志填充
    if (!isRunning.value) {
      logLines.value = []
      for (const hr of hostResults.value) {
        if (hr.stdout) {
          hr.stdout.split('\n').forEach((line: string) => {
            if (line) logLines.value.push({ host_ip: hr.host_ip, line, is_stderr: false })
          })
        }
        if (hr.stderr) {
          hr.stderr.split('\n').forEach((line: string) => {
            if (line) logLines.value.push({ host_ip: hr.host_ip, line, is_stderr: true })
          })
        }
      }
    }
  } finally { loading.value = false }
}

function connectWebSocket() {
  if (!execId.value || destroyed) return
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  const token = localStorage.getItem('token') || ''
  const url = `${proto}://${location.host}/api/v1/ws/task-executions/${execId.value}/logs?token=${token}`
  ws = new WebSocket(url)
  ws.onmessage = (evt) => {
    try {
      const data = JSON.parse(evt.data)
      logLines.value.push(data)
      nextTick(() => scrollToBottom())
      // 更新 host result status locally
      if (data.phase === 'finished' || data.phase === 'error') {
        const hr = hostResults.value.find(h => h.host_ip === data.host_ip)
        if (hr) {
          hr.status = data.phase === 'finished' ? 'success' : 'failed'
          hr.exit_code = data.exit_code
        }
      }
    } catch {}
  }
  ws.onclose = () => {
    ws = null
    if (destroyed) return
    // 刷新状态，如果仍在运行则重连
    fetchExecution().then(() => {
      if (isRunning.value && !destroyed) {
        wsReconnectTimer = setTimeout(() => connectWebSocket(), 3000)
      }
    })
  }
  ws.onerror = () => {
    // onclose will fire after onerror
  }
}

function scrollToBottom() {
  if (logContainerRef.value) {
    logContainerRef.value.scrollTop = logContainerRef.value.scrollHeight
  }
}

function selectHost(ip: string) {
  selectedHostIP.value = ip
}

function goBack() { router.push('/task/list') }

// 计时器
const elapsed = ref('')
let timer: ReturnType<typeof setInterval> | null = null
function startTimer() {
  timer = setInterval(() => {
    if (!execution.value?.started_at) return
    const start = new Date(execution.value.started_at).getTime()
    const now = execution.value?.finished_at ? new Date(execution.value.finished_at).getTime() : Date.now()
    const diff = Math.floor((now - start) / 1000)
    const m = Math.floor(diff / 60)
    const s = diff % 60
    elapsed.value = `${m}分${s}秒`
  }, 1000)
}

onMounted(async () => {
  await fetchExecution()
  if (isRunning.value) {
    connectWebSocket()
  }
  startTimer()
})

onUnmounted(() => {
  destroyed = true
  if (ws) { ws.close(); ws = null }
  if (wsReconnectTimer) { clearTimeout(wsReconnectTimer); wsReconnectTimer = null }
  if (timer) { clearInterval(timer); timer = null }
})
</script>

<template>
  <div class="p-4 md:p-6 min-h-full flex flex-col">
    <el-card shadow="never" class="border-0 shadow-sm flex-1 flex flex-col" v-loading="loading">
      <template #header>
        <div class="flex justify-between items-center">
          <div class="flex items-center gap-3">
            <el-button link @click="goBack" class="text-gray-500 hover:text-gray-700 -ml-2">
              <el-icon class="text-lg"><Back /></el-icon>
            </el-button>
            <span class="text-base font-medium text-gray-800">执行详情 #{{ execId }}</span>
            <el-tag v-if="execution" :type="(execStatusMap[execution.status]?.type as any) || ''" size="default" effect="light">
              {{ execStatusMap[execution.status]?.label || execution.status }}
            </el-tag>
            <span v-if="elapsed" class="text-gray-500 text-sm ml-2">耗时: {{ elapsed }}</span>
          </div>
          <div class="flex items-center gap-2">
            <el-button v-if="!isRunning" @click="fetchExecution" :loading="loading">
              <el-icon class="mr-1"><Refresh /></el-icon> 刷新
            </el-button>
          </div>
        </div>
      </template>

      <!-- 概览 -->
      <div v-if="execution" class="mb-4 flex flex-wrap items-center gap-6 text-sm text-gray-600 bg-gray-50 p-3 rounded-lg border border-gray-100">
        <div class="flex items-center gap-2">
          <span class="text-gray-500">任务:</span>
          <span class="font-medium text-gray-800">{{ execution.task_name || '-' }}</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-gray-500">执行人:</span>
          <span class="font-medium text-gray-800">{{ execution.operator_name || '-' }}</span>
        </div>
        <el-divider direction="vertical" class="hidden md:block" />
        <div class="flex items-center gap-2">
          <span class="text-gray-500">总计:</span>
          <span class="font-medium text-gray-800">{{ execution.total_count }} 台</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-gray-500">成功:</span>
          <span class="font-medium text-emerald-600">{{ execution.success_count }}</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-gray-500">失败:</span>
          <span class="font-medium text-red-500">{{ execution.fail_count }}</span>
        </div>
      </div>

      <!-- 主体：左侧主机列表 + 右侧日志 -->
      <div class="flex flex-col lg:flex-row gap-4 flex-1 h-[600px] xl:h-[700px]">
        <div class="w-full lg:w-64 shrink-0 border border-gray-200 rounded-lg overflow-y-auto bg-white flex flex-col h-64 lg:h-full">
          <div 
            v-for="hr in hostResults" 
            :key="hr.host_ip" 
            class="p-3 cursor-pointer border-b border-gray-100 flex flex-col gap-2 transition-colors relative"
            :class="selectedHostIP === hr.host_ip ? 'bg-indigo-50/50' : 'hover:bg-gray-50'"
            @click="selectHost(hr.host_ip)"
          >
            <div v-if="selectedHostIP === hr.host_ip" class="absolute left-0 top-0 bottom-0 w-1 bg-indigo-500"></div>
            <div class="font-medium text-sm text-gray-800">{{ hr.host_ip }}</div>
            <div class="flex items-center justify-between gap-2">
              <div class="text-xs text-gray-500 truncate flex-1">{{ hr.hostname || '-' }}</div>
              <el-tag :type="(execStatusMap[hr.status]?.type as any) || ''" size="small" effect="plain" round>
                {{ execStatusMap[hr.status]?.label || hr.status }}
              </el-tag>
            </div>
          </div>
          <div v-if="hostResults.length === 0" class="p-8 text-gray-400 text-center text-sm my-auto">
            暂无主机
          </div>
        </div>
        
        <div 
          class="flex-1 bg-gray-900 text-gray-300 font-mono text-sm leading-relaxed p-4 rounded-lg overflow-y-auto whitespace-pre-wrap break-all shadow-inner border border-gray-800" 
          ref="logContainerRef"
        >
          <div v-if="selectedHostLogs.length === 0" class="text-gray-600 text-center py-12 flex flex-col items-center gap-3">
            <el-icon class="text-4xl opacity-50"><Document /></el-icon>
            暂无日志输出
          </div>
          <div 
            v-for="(line, idx) in selectedHostLogs" 
            :key="idx" 
            class="py-px hover:bg-white/5 transition-colors" 
            :class="{ 'text-red-400': line.is_stderr }"
          >
            {{ line.line }}
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
:deep(.el-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
}
</style>
