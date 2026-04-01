<script setup lang="ts">
defineOptions({ name: 'DashboardOverview' })

import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { monitorApi, statsApi } from '../../api'
import { usePermissionStore } from '../../stores/permission'

const router = useRouter()
const permissionStore = usePermissionStore()

const loading = ref(true)

const platform = ref({
  asset_total: 0,
  asset_online: 0,
  asset_offline: 0,
  cloud_account_total: 0,
  cloud_account_failed: 0,
  service_tree_total: 0,
  user_total: 0,
  department_total: 0,
  ticket_open: 0,
  ticket_total: 0,
})

const distribution = ref<{ status_dist: any[]; source_dist: any[]; top_services: any[] }>({
  status_dist: [],
  source_dist: [],
  top_services: [],
})

const monitorSummary = ref({
  agent_total: 0,
  agent_online: 0,
  agent_offline: 0,
  alert_firing_total: 0,
  last_collected_at: '',
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

const canViewCMDB = computed(() => hasPath('/cmdb/assets') || hasPath('/cmdb/cloud-accounts') || hasPath('/cmdb/service-tree'))
const canViewAssets = computed(() => hasPath('/cmdb/assets'))
const canViewCloudAccounts = computed(() => hasPath('/cmdb/cloud-accounts'))
const canViewServiceTree = computed(() => hasPath('/cmdb/service-tree'))
const canViewTicket = computed(() => hasPath('/ticket/todo') || hasPath('/ticket/applied') || hasPath('/ticket/create'))
const canViewTask = computed(() => hasPath('/task/list'))
const canViewCICD = computed(() => hasPath('/cicd/projects') || hasPath('/cicd/pipelines') || hasPath('/cicd/runs'))
const canViewMonitor = computed(() => hasPath('/monitor/dashboard') || hasPath('/monitor/alert-rules') || hasPath('/monitor/alerts'))
const canViewSystem = computed(() => hasPath('/system/users') || hasPath('/system/roles') || hasPath('/system/menus'))

const overviewCards = computed(() => {
  const items = [
    { key: 'asset', title: '主机资产', value: platform.value.asset_total, meta: `在线 ${platform.value.asset_online} / 离线 ${platform.value.asset_offline}`, tone: 'bg-white border-blue-100', path: '/cmdb/assets', visible: canViewAssets.value },
    { key: 'cloud', title: '云账号', value: platform.value.cloud_account_total, meta: platform.value.cloud_account_failed > 0 ? `${platform.value.cloud_account_failed} 个同步异常` : '同步状态正常', tone: 'bg-white border-green-100', path: '/cmdb/cloud-accounts', visible: canViewCloudAccounts.value },
    { key: 'tree', title: '服务树', value: platform.value.service_tree_total, meta: '服务与资源归属结构', tone: 'bg-white border-yellow-100', path: '/cmdb/service-tree', visible: canViewServiceTree.value },
    { key: 'alert', title: '触发中告警', value: monitorSummary.value.alert_firing_total, meta: `在线 ${monitorSummary.value.agent_online} / 离线 ${monitorSummary.value.agent_offline}`, tone: 'bg-white border-red-100', path: '/monitor/alerts', visible: canViewMonitor.value },
    { key: 'ticket', title: '打开中的工单', value: platform.value.ticket_open, meta: `工单总数 ${platform.value.ticket_total}`, tone: 'bg-white border-orange-100', path: '/ticket/todo', visible: canViewTicket.value },
    { key: 'task', title: '在线 Agent', value: monitorSummary.value.agent_online, meta: monitorSummary.value.last_collected_at ? `最近采样 ${monitorSummary.value.last_collected_at}` : '等待采样', tone: 'bg-white border-cyan-100', path: '/monitor/dashboard', visible: canViewTask.value || canViewMonitor.value },
    { key: 'user', title: '平台用户', value: platform.value.user_total, meta: `部门 ${platform.value.department_total}`, tone: 'bg-white border-indigo-100', path: '/system/users', visible: canViewSystem.value },
    { key: 'run', title: '流水线运行', value: 0, meta: '暂无全局运行统计', tone: 'bg-white border-purple-100', path: '/cicd/runs', visible: canViewCICD.value },
  ]
  return items.filter(item => item.visible)
})

function maxServiceCount() {
  if (!distribution.value.top_services?.length) return 1
  return Math.max(...distribution.value.top_services.map((t: any) => t.count), 1)
}

function sourceLabel(s: string) {
  const map: Record<string, string> = { manual: '手工录入', aliyun: '阿里云', tencent: '腾讯云', aws: 'AWS' }
  return map[s] || s
}

async function fetchData() {
  loading.value = true
  try {
    const jobs: Promise<any>[] = []
    const keys: string[] = []
    if (canViewCMDB.value || canViewTicket.value || canViewSystem.value) {
      jobs.push(statsApi.summary())
      keys.push('summary')
    }
    if (canViewCMDB.value) {
      jobs.push(statsApi.assetDistribution())
      keys.push('distribution')
    }
    if (canViewMonitor.value || canViewTask.value) {
      jobs.push(monitorApi.summary())
      keys.push('monitor')
    }

    if (jobs.length) {
      const results = await Promise.allSettled(jobs)
      results.forEach((item, index) => {
        if (item.status !== 'fulfilled') return
        const key = keys[index]
        const data = (item.value as any).data
        if (key === 'summary') platform.value = data || platform.value
        if (key === 'distribution') distribution.value = data || distribution.value
        if (key === 'monitor') {
          monitorSummary.value = {
            agent_total: data?.agent_total || 0,
            agent_online: data?.agent_online || 0,
            agent_offline: data?.agent_offline || 0,
            alert_firing_total: data?.alert_firing_total || 0,
            last_collected_at: data?.last_collected_at || '',
          }
        }
      })
    }
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)
</script>

<template>
  <div class="h-full flex flex-col gap-6" v-loading="loading">
    <!-- Platform Overview -->
    <div class="flex flex-col gap-4">
      <div class="flex items-end justify-between">
        <div>
          <h3 class="text-lg font-bold text-gray-900">平台总览</h3>
          <p class="text-sm text-gray-500 mt-1">全局资源与状态汇总，只显示你当前有权限访问的模块概况。</p>
        </div>
      </div>
      
      <div v-if="overviewCards.length" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <button
          v-for="card in overviewCards"
          :key="card.key"
          class="text-left p-5 rounded-xl border shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md hover:border-gray-300"
          :class="card.tone"
          type="button"
          @click="router.push(card.path)"
        >
          <div class="text-xs font-bold text-gray-500 mb-2">{{ card.title }}</div>
          <div class="text-2xl font-black text-gray-900 mb-2">{{ card.value }}</div>
          <div class="text-xs text-gray-400">{{ card.meta }}</div>
        </button>
      </div>
      <el-empty v-else description="当前角色没有可展示的平台概况模块" :image-size="56" class="bg-gray-50 rounded-xl" />
    </div>

    <!-- Charts / Distributions -->
    <div v-if="canViewCMDB" class="flex flex-col gap-4">
      <div class="flex items-end justify-between">
        <div>
          <h3 class="text-lg font-bold text-gray-900">资源分布</h3>
          <p class="text-sm text-gray-500 mt-1">保留资产来源和服务树排行，帮助快速定位资源聚集点。</p>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-12 gap-4">
        <!-- Source Distribution -->
        <div class="lg:col-span-5 bg-white border border-gray-100 shadow-sm rounded-xl p-5">
          <div class="text-sm font-bold text-gray-700 mb-4">资产来源分布</div>
          <div v-if="distribution.source_dist?.length" class="flex flex-col gap-3">
            <div
              v-for="item in distribution.source_dist"
              :key="item.label"
              class="flex items-center cursor-pointer group"
              @click="router.push('/cmdb/assets?source=' + item.label)"
            >
              <span class="w-20 text-xs text-gray-600 group-hover:text-blue-600 truncate">{{ sourceLabel(item.label) }}</span>
              <el-progress
                :percentage="platform.asset_total ? Math.round((item.count / platform.asset_total) * 100) : 0"
                :stroke-width="12"
                :show-text="false"
                class="flex-1 mx-3"
              />
              <span class="w-10 text-right text-sm font-bold text-gray-800">{{ item.count }}</span>
            </div>
          </div>
          <el-empty v-else description="暂无数据" :image-size="40" />
        </div>

        <!-- Top Services -->
        <div class="lg:col-span-7 bg-white border border-gray-100 shadow-sm rounded-xl p-5">
          <div class="text-sm font-bold text-gray-700 mb-4">服务树资产 Top 10</div>
          <div v-if="distribution.top_services?.length" class="flex flex-col gap-3">
            <div
              v-for="item in distribution.top_services"
              :key="item.id"
              class="flex items-center cursor-pointer group"
              @click="router.push('/cmdb/assets?service_tree_id=' + item.id)"
            >
              <span class="w-32 lg:w-40 text-xs text-gray-600 group-hover:text-green-600 truncate" :title="item.name">{{ item.name }}</span>
              <el-progress
                :percentage="Math.round((item.count / maxServiceCount()) * 100)"
                :stroke-width="12"
                :show-text="false"
                color="#10b981"
                class="flex-1 mx-3"
              />
              <span class="w-10 text-right text-sm font-bold text-gray-800">{{ item.count }}</span>
            </div>
          </div>
          <el-empty v-else description="暂无数据" :image-size="40" />
        </div>
      </div>
    </div>
  </div>
</template>
