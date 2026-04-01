<script setup lang="ts">
defineOptions({ name: 'DashboardWorkbench' })

import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { dashboardApi } from '../../api'
import { usePermissionStore } from '../../stores/permission'
import { useUserStore } from '../../stores/user'

const router = useRouter()
const permissionStore = usePermissionStore()
const userStore = useUserStore()

const loading = ref(true)

const personal = ref({
  my_pending_tickets: 0,
  my_created_tickets: 0,
  my_assets: 0,
  my_alerts: 0,
  my_task_executions: 0,
  my_pipeline_runs: 0,
})

function flattenMenus(items: any[]): any[] {
  const result: any[] = []
  for (const item of items || []) {
    result.push(item)
    if (Array.isArray(item.children) && item.children.length > 0) {
      result.push(...flattenMenus(item.children))
    }
  }
  return result
}

const flatMenus = computed(() => flattenMenus(permissionStore.menus))
const visiblePaths = computed(() => new Set(flatMenus.value.map(item => item.path).filter(Boolean)))

function hasPath(path: string) {
  return visiblePaths.value.has(path)
}

const canViewAssets = computed(() => hasPath('/cmdb/assets'))
const canLaunchTicket = computed(() => hasPath('/ticket/create'))
const canViewTask = computed(() => hasPath('/task/list'))
const canViewCICD = computed(() => hasPath('/cicd/projects') || hasPath('/cicd/pipelines') || hasPath('/cicd/runs'))
const canViewMonitor = computed(() => hasPath('/monitor/dashboard') || hasPath('/monitor/alert-rules') || hasPath('/monitor/alerts'))

const displayName = computed(() => userStore.userInfo?.real_name || userStore.userInfo?.username || '用户')

const quickActions = computed(() => {
  const items = [
    { key: 'launch-ticket', title: '发起工单', subtitle: '提交新的请求或变更', path: '/ticket/create', visible: canLaunchTicket.value },
    { key: 'todo-ticket', title: '我的待办', subtitle: '进入待处理工单列表', path: '/ticket/todo', visible: hasPath('/ticket/todo') },
    { key: 'my-apply', title: '我的申请', subtitle: '查看自己发起的工单', path: '/ticket/applied', visible: hasPath('/ticket/applied') },
    { key: 'alert-events', title: '告警事件', subtitle: '处理触发中的告警', path: '/monitor/alerts', visible: hasPath('/monitor/alerts') },
    { key: 'task-list', title: '任务中心', subtitle: '查看执行与任务状态', path: '/task/list', visible: hasPath('/task/list') },
    { key: 'pipeline-runs', title: '运行记录', subtitle: '跟踪最近发布与回滚', path: '/cicd/runs', visible: hasPath('/cicd/runs') },
  ]
  return items.filter(item => item.visible)
})

const workbenchCards = computed(() => {
  const items = [
    { key: 'pending', title: '我的待办', value: personal.value.my_pending_tickets, hint: '等待我处理的工单', tone: 'bg-yellow-50 text-yellow-800 border-yellow-200', path: '/ticket/todo', visible: hasPath('/ticket/todo') },
    { key: 'created', title: '我的申请', value: personal.value.my_created_tickets, hint: '我发起的工单', tone: 'bg-blue-50 text-blue-800 border-blue-200', path: '/ticket/applied', visible: hasPath('/ticket/applied') },
    { key: 'assets', title: '我负责的资产', value: personal.value.my_assets, hint: '归属到我名下的主机', tone: 'bg-green-50 text-green-800 border-green-200', path: '/cmdb/assets', visible: canViewAssets.value },
    { key: 'alerts', title: '我的相关告警', value: personal.value.my_alerts, hint: '归属到我的告警事件', tone: 'bg-red-50 text-red-800 border-red-200', path: '/monitor/alerts', visible: canViewMonitor.value },
    { key: 'tasks', title: '我的任务执行', value: personal.value.my_task_executions, hint: '我发起的任务执行记录', tone: 'bg-cyan-50 text-cyan-800 border-cyan-200', path: '/task/list', visible: canViewTask.value },
    { key: 'runs', title: '我的流水线', value: personal.value.my_pipeline_runs, hint: '我触发过的流水线运行', tone: 'bg-purple-50 text-purple-800 border-purple-200', path: '/cicd/runs', visible: canViewCICD.value },
  ]
  return items.filter(item => item.visible)
})

async function fetchData() {
  loading.value = true
  try {
    const res = await dashboardApi.personal()
    personal.value = (res as any).data || personal.value
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)
</script>

<template>
  <div class="h-full flex flex-col gap-6" v-loading="loading">
    <!-- Welcome / Hero Section -->
    <div class="flex flex-col lg:flex-row gap-6 bg-gradient-to-br from-white to-blue-50 p-6 rounded-2xl border border-blue-100 shadow-sm">
      <div class="flex-1 flex flex-col justify-center">
        <span class="text-xs font-bold tracking-widest uppercase text-blue-500 mb-2">Workbench</span>
        <h2 class="text-2xl lg:text-3xl font-bold text-gray-900 mb-2">{{ displayName }}，这里是你的工作台</h2>
        <p class="text-sm text-gray-500">首页优先展示你能操作、也最需要处理的内容。</p>
      </div>
      <div class="flex-1 grid grid-cols-1 sm:grid-cols-2 gap-3">
        <button
          v-for="item in quickActions"
          :key="item.key"
          class="text-left bg-gray-900 text-white p-4 rounded-xl hover:-translate-y-0.5 hover:shadow-lg transition-all duration-200 border border-transparent hover:border-gray-700"
          type="button"
          @click="router.push(item.path)"
        >
          <span class="block text-sm font-bold">{{ item.title }}</span>
          <span class="block mt-1 text-xs text-gray-400">{{ item.subtitle }}</span>
        </button>
      </div>
    </div>

    <!-- Personal Tasks & Stats -->
    <div class="flex flex-col gap-4">
      <div class="flex items-end justify-between">
        <div>
          <h3 class="text-lg font-bold text-gray-900">个人待办与数据</h3>
          <p class="text-sm text-gray-500 mt-1">这里的数字都与你自己有关，先把手边的事情处理掉。</p>
        </div>
      </div>
      
      <div v-if="workbenchCards.length" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <button
          v-for="card in workbenchCards"
          :key="card.key"
          class="text-left p-5 rounded-xl border transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md"
          :class="card.tone"
          type="button"
          @click="router.push(card.path)"
        >
          <div class="text-xs font-bold opacity-80 mb-3">{{ card.title }}</div>
          <div class="text-3xl font-black mb-2">{{ card.value }}</div>
          <div class="text-xs opacity-70">{{ card.hint }}</div>
        </button>
      </div>
      
      <el-empty v-else description="当前没有可展示的个人工作台内容" :image-size="56" class="bg-gray-50 rounded-xl" />
    </div>
  </div>
</template>
