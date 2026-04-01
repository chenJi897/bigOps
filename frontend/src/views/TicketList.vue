<script setup lang="ts">
defineOptions({ name: 'TicketList' })
import { computed, ref, onActivated, onMounted } from 'vue'
import { Plus, Search } from '@element-plus/icons-vue'
import { useRoute, useRouter } from 'vue-router'
import { ticketApi, ticketTypeApi, userApi } from '../api'
import { useViewStateStore } from '../stores/viewState'
import { useUserStore } from '../stores/user'

const router = useRouter()
const route = useRoute()
const viewStateStore = useViewStateStore()
const userStore = useUserStore()
const loading = ref(false)
const tableData = ref<any[]>([])
const total = ref(0)
const query = ref<any>({ page: 1, size: 20, status: '', priority: '', type_id: '', source: '', keyword: '', scope: 'all' })

const allTypes = ref<any[]>([])
const isAdmin = ref(false)
const seenTicketTypeVersion = ref(0)

const statusMap: Record<string, { label: string; type: 'primary' | 'success' | 'info' | 'warning' | 'danger' }> = {
  open: { label: '待处理', type: 'info' },
  processing: { label: '处理中', type: 'warning' },
  resolved: { label: '已解决', type: 'success' },
  closed: { label: '已关闭', type: 'info' },
  rejected: { label: '已驳回', type: 'danger' },
}

const priorityMap: Record<string, { label: string; type: 'primary' | 'success' | 'info' | 'warning' | 'danger' }> = {
  low: { label: '低', type: 'info' },
  medium: { label: '中', type: 'info' },
  high: { label: '高', type: 'warning' },
  urgent: { label: '紧急', type: 'danger' },
}

const sourceMap: Record<string, string> = {
  manual: '手动', monitor: '监控', sync: '同步', system: '系统', cicd: 'CICD',
}

const modeScopeMap = {
  todo: 'my_assigned',
  applied: 'my_created',
} as const

type TicketMode = keyof typeof modeScopeMap
const modeTitleMap: Record<TicketMode, string> = {
  todo: '我的待办',
  applied: '我的申请',
}

const routeMode = computed<TicketMode | ''>(() => {
  const metaMode = route.meta?.ticketMode ?? route.meta?.mode
  if (typeof metaMode === 'string') {
    if (metaMode === 'todo' || metaMode === 'applied') {
      return metaMode
    }
  }
  const path = route.fullPath || route.path || ''
  if (path.includes('ticket/applied')) {
    return 'applied'
  }
  if (path.includes('ticket/todo')) {
    return 'todo'
  }
  return ''
})

const fixedScope = computed(() => (routeMode.value ? modeScopeMap[routeMode.value] : ''))
const showScopeTabs = computed(() => !Boolean(fixedScope.value))
const showLaunchButton = computed(() => Boolean(routeMode.value))
const pageTitle = computed(() => {
  const currentMode = routeMode.value
  if (currentMode) return modeTitleMap[currentMode]
  return '工单中心'
})

const scopeTabs = computed(() => {
  if (isAdmin.value) {
    return [
      { label: '全部工单', value: 'all' },
      { label: '我创建的', value: 'my_created' },
      { label: '我处理的', value: 'my_assigned' },
      { label: '本部门', value: 'my_dept' },
    ]
  }
  return [
    { label: '我创建的', value: 'my_created' },
    { label: '我处理的', value: 'my_assigned' },
  ]
})

const personalScopeValues = ['my_created', 'my_assigned']

async function fetchData() {
  loading.value = true
  try {
    const params = { ...query.value }
    Object.keys(params).forEach(k => { if (params[k] === '' || params[k] === null) delete params[k] })
    const res: any = await ticketApi.list(params)
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  query.value.page = 1
  fetchData()
}

function handleReset() {
  const currentScope = query.value.scope
  query.value = { page: 1, size: 20, status: '', priority: '', type_id: '', source: '', keyword: '', scope: currentScope }
  fetchData()
}

function handleScopeChange(scope: string) {
  query.value.scope = scope
  query.value.page = 1
  fetchData()
}

function openDetail(row: any) {
  const currentMode = routeMode.value
  const queryFrom = currentMode || undefined
  router.push({ path: '/ticket/detail/' + row.id, query: queryFrom ? { from: queryFrom } : undefined })
}

function openCreate() {
  router.push('/ticket/create')
}

function ensureScopeFromMode() {
  if (fixedScope.value) {
    query.value.scope = fixedScope.value
  }
}

function updateScopeForRoles(admin: boolean) {
  isAdmin.value = admin
  if (!fixedScope.value && !admin && !personalScopeValues.includes(query.value.scope)) {
    query.value.scope = 'my_created'
  }
}

onMounted(() => {
  ensureScopeFromMode()
  const currentUserID = userStore.userInfo?.id
  if (currentUserID) {
    userApi.getRoles(currentUserID).then((res: any) => {
      const roles = res.data || []
      const admin = roles.some((role: any) => role.name === 'admin')
      updateScopeForRoles(admin)
      fetchData()
    }).catch(() => {
      updateScopeForRoles(false)
      fetchData()
    })
  } else {
    updateScopeForRoles(false)
    fetchData()
  }
  ticketTypeApi.all().then((res: any) => { allTypes.value = res.data || [] }).catch(() => {})
  seenTicketTypeVersion.value = viewStateStore.ticketTypeVersion
})

onActivated(() => {
  if (viewStateStore.consumeTicketListDirty()) {
    fetchData()
  }
  if (seenTicketTypeVersion.value !== viewStateStore.ticketTypeVersion) {
    seenTicketTypeVersion.value = viewStateStore.ticketTypeVersion
    ticketTypeApi.all().then((res: any) => { allTypes.value = res.data || [] }).catch(() => {})
  }
})
</script>

<template>
  <div class="p-4 sm:p-6 lg:p-8 space-y-4 sm:space-y-6">
    <el-card shadow="never" class="border-0 shadow-sm ring-1 ring-gray-200/50 rounded-xl">
      <template #header>
        <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
          <div class="flex items-center gap-3">
            <h2 class="text-xl font-semibold text-gray-800 tracking-tight">{{ pageTitle }}</h2>
            <el-badge v-if="total > 0" :value="total" :max="99" class="ml-2" type="primary" />
          </div>
          <el-button v-if="showLaunchButton" type="primary" @click="openCreate" class="shadow-sm">
            <template #icon><el-icon><Plus /></el-icon></template>
            发起工单
          </el-button>
        </div>
      </template>

      <div class="flex flex-col space-y-5">
        <!-- Filters Area -->
        <div class="flex flex-col xl:flex-row justify-between items-start xl:items-center gap-4 bg-gray-50/50 p-4 rounded-lg border border-gray-100">
          <div v-if="showScopeTabs" class="flex-shrink-0">
            <el-radio-group v-model="query.scope" @change="handleScopeChange" size="default">
              <el-radio-button v-for="t in scopeTabs" :key="t.value" :value="t.value">{{ t.label }}</el-radio-button>
            </el-radio-group>
          </div>
          
          <el-form :inline="true" @submit.prevent="handleSearch" class="flex flex-wrap gap-3 w-full xl:w-auto xl:justify-end" style="margin-bottom: 0;">
            <el-select v-model="query.status" placeholder="所有状态" clearable class="w-28 sm:w-32">
              <el-option v-for="(v, k) in statusMap" :key="k" :label="v.label" :value="k" />
            </el-select>
            
            <el-select v-model="query.priority" placeholder="所有优先级" clearable class="w-28 sm:w-32">
              <el-option v-for="(v, k) in priorityMap" :key="k" :label="v.label" :value="k" />
            </el-select>
            
            <el-select v-model="query.type_id" placeholder="所有工单模板" clearable class="w-36 sm:w-44">
              <el-option v-for="t in allTypes" :key="t.id" :label="t.name" :value="t.id" />
            </el-select>
            
            <el-input v-model="query.keyword" placeholder="搜索标题或编号" clearable class="w-48 sm:w-64" @keyup.enter="handleSearch">
              <template #prefix>
                <el-icon class="text-gray-400"><Search /></el-icon>
              </template>
            </el-input>
            
            <div class="flex items-center gap-2">
              <el-button type="primary" @click="handleSearch">搜索</el-button>
              <el-button @click="handleReset" plain>重置</el-button>
            </div>
          </el-form>
        </div>

        <!-- Table Area -->
        <el-table :data="tableData" v-loading="loading" stripe :border="false" class="w-full shadow-sm rounded-lg overflow-hidden border border-gray-100" @row-click="openDetail" style="cursor: pointer;">
          <el-table-column prop="ticket_no" label="编号" width="160" />
          <el-table-column prop="title" label="标题" min-width="240" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="font-medium text-gray-800 group-hover:text-indigo-600 transition-colors">{{ row.title }}</span>
            </template>
          </el-table-column>
          <el-table-column label="模板" width="140">
            <template #default="{ row }">
              <span class="text-gray-600">{{ row.request_template_name || row.type_name || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="优先级" width="90">
            <template #default="{ row }">
              <el-tag :type="priorityMap[row.priority]?.type || 'info'" size="small" effect="light" class="rounded-md">
                {{ priorityMap[row.priority]?.label || row.priority }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="statusMap[row.status]?.type || 'info'" size="small" effect="light" class="rounded-md">
                {{ statusMap[row.status]?.label || row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="creator_name" label="创建人" width="100">
            <template #default="{ row }">
              <span class="text-gray-700">{{ row.creator_name || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="assignee_name" label="处理人" width="100">
            <template #default="{ row }">
              <span class="text-gray-700">{{ row.assignee_name || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="来源" width="80">
            <template #default="{ row }">
              <span class="text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded">{{ sourceMap[row.source] || row.source }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="170">
            <template #default="{ row }">
              <span class="text-gray-500 text-sm">{{ row.created_at }}</span>
            </template>
          </el-table-column>
        </el-table>

        <div class="flex justify-end pt-4 pb-2">
          <el-pagination 
            v-if="total > 0" 
            background 
            layout="total, sizes, prev, pager, next, jumper" 
            :page-sizes="[10, 20, 50, 100]"
            :total="total" 
            v-model:page-size="query.size"
            :current-page="query.page" 
            @size-change="(s: number) => { query.size = s; handleSearch() }"
            @current-change="(p: number) => { query.page = p; fetchData() }" 
          />
        </div>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
/* Scoped styles can be removed as we are using Tailwind utilities now */
:deep(.el-table__row) {
  @apply hover:bg-indigo-50/50 transition-colors duration-200;
}
:deep(.el-table th.el-table__cell) {
  @apply bg-gray-50/80 text-gray-600 font-medium;
}
</style>
