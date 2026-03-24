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
  if (!execId.value) return
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
    // 连接关闭时刷新一次状态
    setTimeout(() => fetchExecution(), 1000)
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
  if (ws) { ws.close(); ws = null }
  if (timer) { clearInterval(timer); timer = null }
})
</script>

<template>
  <div class="page">
    <el-card shadow="never" v-loading="loading">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <div style="display: flex; align-items: center; gap: 12px;">
            <span>执行详情 #{{ execId }}</span>
            <el-tag v-if="execution" :type="(execStatusMap[execution.status]?.type as any) || ''" size="default">
              {{ execStatusMap[execution.status]?.label || execution.status }}
            </el-tag>
            <span v-if="elapsed" style="color: #909399; font-size: 13px;">耗时: {{ elapsed }}</span>
          </div>
          <div style="display: flex; gap: 8px;">
            <el-button v-if="!isRunning" @click="fetchExecution" :loading="loading">刷新</el-button>
            <el-button @click="goBack">返回</el-button>
          </div>
        </div>
      </template>

      <!-- 概览 -->
      <div v-if="execution" style="margin-bottom: 16px; display: flex; gap: 24px; color: #606266; font-size: 13px;">
        <span>任务: {{ execution.task_name || '-' }}</span>
        <span>执行人: {{ execution.operator_name || '-' }}</span>
        <span>总计: {{ execution.total_count }} 台</span>
        <span style="color: #67c23a;">成功: {{ execution.success_count }}</span>
        <span style="color: #f56c6c;">失败: {{ execution.fail_count }}</span>
      </div>

      <!-- 主体：左侧主机列表 + 右侧日志 -->
      <div class="exec-body">
        <div class="host-list">
          <div v-for="hr in hostResults" :key="hr.host_ip" class="host-item" :class="{ active: selectedHostIP === hr.host_ip }" @click="selectHost(hr.host_ip)">
            <div class="host-ip">{{ hr.host_ip }}</div>
            <div class="host-name">{{ hr.hostname || '-' }}</div>
            <el-tag :type="(execStatusMap[hr.status]?.type as any) || ''" size="small">
              {{ execStatusMap[hr.status]?.label || hr.status }}
            </el-tag>
          </div>
          <div v-if="hostResults.length === 0" style="padding: 20px; color: #909399; text-align: center;">暂无主机</div>
        </div>
        <div class="log-panel" ref="logContainerRef">
          <div v-if="selectedHostLogs.length === 0" class="log-empty">暂无日志输出</div>
          <div v-for="(line, idx) in selectedHostLogs" :key="idx" class="log-line" :class="{ stderr: line.is_stderr }">{{ line.line }}</div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
.exec-body {
  display: flex;
  gap: 12px;
  height: 500px;
}
.host-list {
  width: 220px;
  flex-shrink: 0;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  overflow-y: auto;
}
.host-item {
  padding: 10px 12px;
  cursor: pointer;
  border-bottom: 1px solid #f0f0f0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  transition: background 0.15s;
}
.host-item:hover { background: #f5f7fa; }
.host-item.active { background: #ecf5ff; border-left: 3px solid #409eff; }
.host-ip { font-weight: 500; font-size: 13px; width: 100%; }
.host-name { font-size: 12px; color: #909399; flex: 1; }
.log-panel {
  flex: 1;
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Courier New', Consolas, monospace;
  font-size: 13px;
  line-height: 1.6;
  padding: 12px;
  border-radius: 4px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
.log-empty {
  color: #666;
  text-align: center;
  padding: 40px;
}
.log-line { padding: 1px 0; }
.log-line.stderr { color: #f56c6c; }
</style>
