<script setup lang="ts">
defineOptions({ name: 'AgentDetail' })

import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { alertRuleApi, assetApi, monitorApi, serviceTreeApi, taskApi, userApi } from '../api'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const trendLoading = ref(false)
const executionLoading = ref(false)
const agent = ref<any>(null)
const assetContext = ref<any | null>(null)
const userMap = ref<Record<number, string>>({})
const serviceTreeMap = ref<Record<number, string>>({})
const alertEvents = ref<any[]>([])
const taskExecutions = ref<any[]>([])
const trends = ref<Record<string, any[]>>({
  cpu_usage: [],
  memory_usage: [],
  disk_usage: [],
})

const agentID = computed(() => String(route.params.agentId || ''))

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

function metricLabel(metricType: string) {
  const map: Record<string, string> = {
    cpu_usage: 'CPU 使用率',
    memory_usage: '内存使用率',
    disk_usage: '磁盘使用率',
  }
  return map[metricType] || metricType
}

function severityTagType(severity: string) {
  const map: Record<string, string> = { info: 'info', warning: 'warning', critical: 'danger' }
  return map[severity] || 'info'
}

function statusTagType(status: string) {
  const map: Record<string, string> = {
    online: 'success',
    offline: 'danger',
    firing: 'danger',
    acknowledged: 'warning',
    resolved: 'info',
    success: 'success',
    failed: 'danger',
    running: 'primary',
  }
  return map[status] || 'info'
}

function latestMetric(metricType: string) {
  const items = trends.value[metricType] || []
  if (!items.length) return '-'
  return formatPercent(items[items.length - 1].metric_value)
}

async function loadAgent() {
  loading.value = true
  try {
    const res: any = await monitorApi.agents({ page: 1, size: 200, keyword: agentID.value })
    agent.value = (res.data?.list || []).find((item: any) => item.agent_id === agentID.value) || null
  } finally {
    loading.value = false
  }
}

async function loadContext() {
  if (!agent.value?.ip) {
    assetContext.value = null
    return
  }
  const [assetRes, userRes, treeRes] = await Promise.all([
    assetApi.list({ page: 1, size: 50, keyword: agent.value.ip }),
    userApi.list(1, 500),
    serviceTreeApi.tree(),
  ])
  const assets = (assetRes as any).data?.list || []
  assetContext.value = assets.find((item: any) => item.ip === agent.value.ip) || null

  const users = (userRes as any).data?.list || []
  userMap.value = users.reduce((acc: Record<number, string>, item: any) => {
    acc[item.id] = item.real_name || item.username
    return acc
  }, {})

  const trees = flattenTree((treeRes as any).data || [])
  serviceTreeMap.value = trees.reduce((acc: Record<number, string>, item: any) => {
    acc[item.id] = item.name
    return acc
  }, {})
}

async function loadTrends() {
  if (!agentID.value) return
  trendLoading.value = true
  try {
    const [cpuRes, memRes, diskRes] = await Promise.all([
      monitorApi.trends(agentID.value, 'cpu_usage', 180, 60),
      monitorApi.trends(agentID.value, 'memory_usage', 180, 60),
      monitorApi.trends(agentID.value, 'disk_usage', 180, 60),
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

async function loadAlerts() {
  const res = await alertRuleApi.events({ page: 1, size: 20, agent_id: agentID.value, status: '', severity: '', keyword: '' })
  alertEvents.value = (res as any).data?.list || []
}

async function loadExecutions() {
  executionLoading.value = true
  try {
    const res = await taskApi.executions({ page: 1, size: 100 })
    const items = (res as any).data?.list || []
    taskExecutions.value = items.filter((item: any) => {
      const targetHosts = String(item.target_hosts || '')
      return agent.value?.ip && targetHosts.includes(agent.value.ip)
    }).slice(0, 10)
  } finally {
    executionLoading.value = false
  }
}

async function loadAll() {
  await loadAgent()
  await Promise.all([loadContext(), loadTrends(), loadAlerts(), loadExecutions()])
}

function goTaskExecution(id: number) {
  router.push(`/task/execution/${id}`)
}

function goAlertCenter() {
  router.push(`/monitor/alerts?agent_id=${encodeURIComponent(agentID.value)}`)
}

function ownerNames() {
  if (!assetContext.value?.owner_ids) return '—'
  try {
    const ids = JSON.parse(assetContext.value.owner_ids)
    if (!Array.isArray(ids) || !ids.length) return '—'
    return ids.map((id: number) => userMap.value[id] || `#${id}`).join('、')
  } catch {
    return '—'
  }
}

function serviceTreeName() {
  const id = Number(assetContext.value?.service_tree_id || 0)
  if (!id) return '—'
  return serviceTreeMap.value[id] || `#${id}`
}

function flattenTree(nodes: any[]): any[] {
  const result: any[] = []
  nodes.forEach((node) => {
    result.push({ id: node.id, name: node.name })
    if (Array.isArray(node.children) && node.children.length > 0) {
      result.push(...flattenTree(node.children))
    }
  })
  return result
}

onMounted(loadAll)
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50" v-loading="loading">
    <div class="bg-white border-b border-gray-200 px-6 py-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div class="flex items-center gap-3">
        <el-button @click="router.back()" circle icon="Back" class="mr-2" />
        <div>
          <h1 class="text-xl font-bold text-gray-900">Agent 详情</h1>
          <p class="text-sm text-gray-500 mt-1">{{ agent?.hostname || agentID }} · {{ agent?.ip || '-' }}</p>
        </div>
      </div>
      <div class="flex items-center gap-3">
        <el-button plain @click="goAlertCenter">查看告警</el-button>
        <el-button type="primary" plain @click="loadAll">刷新</el-button>
      </div>
    </div>

    <div class="flex-1 overflow-auto p-6 space-y-6">
      <el-card shadow="never" class="border-gray-200">
        <el-descriptions v-if="agent" :column="3" border class="w-full">
          <el-descriptions-item label="Agent ID">{{ agent.agent_id }}</el-descriptions-item>
          <el-descriptions-item label="系统">{{ agent.os || '-' }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="statusTagType(agent.status)">{{ agent.status }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="服务树">{{ serviceTreeName() }}</el-descriptions-item>
          <el-descriptions-item label="负责人">{{ ownerNames() }}</el-descriptions-item>
          <el-descriptions-item label="资产来源">{{ assetContext?.source || '-' }}</el-descriptions-item>
          <el-descriptions-item label="CPU">{{ formatPercent(agent.cpu_usage_pct) }}</el-descriptions-item>
          <el-descriptions-item label="内存">{{ formatPercent(agent.memory_usage_pct) }}</el-descriptions-item>
          <el-descriptions-item label="磁盘">{{ formatPercent(agent.disk_usage_pct) }}</el-descriptions-item>
          <el-descriptions-item label="内存占用">{{ formatBytes(agent.memory_used) }} / {{ formatBytes(agent.memory_total) }}</el-descriptions-item>
          <el-descriptions-item label="磁盘占用">{{ formatBytes(agent.disk_used) }} / {{ formatBytes(agent.disk_total) }}</el-descriptions-item>
          <el-descriptions-item label="最后心跳">{{ agent.last_heartbeat || '-' }}</el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-row :gutter="16" v-loading="trendLoading">
        <el-col v-for="metricType in ['cpu_usage', 'memory_usage', 'disk_usage']" :key="metricType" :xs="24" :md="8" class="mb-4 md:mb-0">
          <el-card shadow="never" class="border-gray-200 h-32 flex flex-col justify-center">
            <div class="text-sm text-gray-500">{{ metricLabel(metricType) }}</div>
            <div class="mt-2 text-3xl font-bold text-gray-900">{{ latestMetric(metricType) }}</div>
            <div class="mt-2 text-xs text-gray-400">最近 3 小时共 {{ trends[metricType]?.length || 0 }} 个采样点</div>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="16">
        <el-col :xs="24" :lg="12" class="mb-4 lg:mb-0">
          <el-card shadow="never" class="border-gray-200 h-full">
            <template #header>
              <div class="flex justify-between items-center">
                <span class="font-medium text-gray-900">最近告警</span>
                <el-button link type="primary" @click="goAlertCenter">更多</el-button>
              </div>
            </template>
            <el-table :data="alertEvents" size="small" stripe border class="w-full">
              <el-table-column prop="rule_name" label="规则" min-width="160" show-overflow-tooltip />
              <el-table-column prop="metric_type" label="监控项" width="120" />
              <el-table-column label="级别" width="90" align="center">
                <template #default="{ row }">
                  <el-tag size="small" :type="severityTagType(row.severity)">{{ row.severity }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="100" align="center">
                <template #default="{ row }">
                  <el-tag size="small" :type="statusTagType(row.status)">{{ row.status }}</el-tag>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-col>

        <el-col :xs="24" :lg="12">
          <el-card shadow="never" class="border-gray-200 h-full" v-loading="executionLoading">
            <template #header>
              <span class="font-medium text-gray-900">最近任务执行</span>
            </template>
            <el-table :data="taskExecutions" size="small" stripe border class="w-full">
              <el-table-column prop="id" label="执行ID" width="90" align="center" />
              <el-table-column prop="task_name" label="任务" min-width="160" show-overflow-tooltip />
              <el-table-column prop="status" label="状态" width="100" align="center">
                <template #default="{ row }">
                  <el-tag size="small" :type="statusTagType(row.status)">{{ row.status }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="created_at" label="开始时间" width="170" align="center" />
              <el-table-column label="操作" width="80" align="center">
                <template #default="{ row }">
                  <el-button link type="primary" @click="goTaskExecution(row.id)">查看</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<style scoped>
/* Scoped styles replaced with Tailwind utility classes */
</style>
