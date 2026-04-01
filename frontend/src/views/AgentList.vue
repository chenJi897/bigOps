<script setup lang="ts">
defineOptions({ name: 'AgentList' })
import { ref, onMounted, onUnmounted } from 'vue'
import { agentApi } from '../api'

const loading = ref(false)
const tableData = ref<any[]>([])
const total = ref(0)
const query = ref({ page: 1, size: 20, status: '' })
let refreshTimer: ReturnType<typeof setInterval> | null = null

async function fetchData() {
  loading.value = true
  try {
    const params: any = { ...query.value }
    Object.keys(params).forEach(k => { if (params[k] === '' || params[k] === null) delete params[k] })
    const res: any = await agentApi.list(params)
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally { loading.value = false }
}

function handleSearch() { query.value.page = 1; fetchData() }
function handleReset() { query.value = { page: 1, size: 20, status: '' }; fetchData() }

function formatMemory(bytes: number) {
  if (!bytes) return '-'
  const gb = bytes / (1024 * 1024 * 1024)
  return gb >= 1 ? `${gb.toFixed(1)} GB` : `${(bytes / (1024 * 1024)).toFixed(0)} MB`
}

onMounted(() => {
  fetchData()
  refreshTimer = setInterval(() => fetchData(), 30000)
})

onUnmounted(() => {
  if (refreshTimer) { clearInterval(refreshTimer); refreshTimer = null }
})
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50">
    <div class="bg-white border-b border-gray-200 px-6 py-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold text-gray-900">Agent 管理</h1>
        <p class="text-sm text-gray-500 mt-1">查看和管理所有已注册的监控 Agent 节点状态。</p>
      </div>
      <div class="flex items-center gap-3">
        <el-button type="primary" plain @click="fetchData" :loading="loading">刷新</el-button>
      </div>
    </div>

    <div class="flex-1 overflow-auto p-6">
      <el-card shadow="never" class="border-gray-200">
        <el-form :inline="true" @submit.prevent="handleSearch" class="mb-4 flex flex-wrap gap-2">
          <el-form-item class="mb-0">
            <el-select v-model="query.status" placeholder="状态" clearable class="w-32">
              <el-option label="在线" value="online" />
              <el-option label="离线" value="offline" />
            </el-select>
          </el-form-item>
          <el-form-item class="mb-0">
            <el-button type="primary" @click="handleSearch">筛选</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>

        <el-table :data="tableData" v-loading="loading" stripe border class="w-full">
          <el-table-column prop="agent_id" label="Agent ID" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">
              <router-link :to="`/monitor/agents/${row.agent_id}`" class="text-indigo-600 hover:text-indigo-800 hover:underline">
                {{ row.agent_id }}
              </router-link>
            </template>
          </el-table-column>
          <el-table-column prop="hostname" label="主机名" min-width="150" show-overflow-tooltip />
          <el-table-column prop="ip" label="IP 地址" width="140" />
          <el-table-column prop="version" label="版本" width="90" align="center" />
          <el-table-column prop="os" label="系统" width="120" show-overflow-tooltip align="center" />
          <el-table-column label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 'online' ? 'success' : 'danger'" size="small">
                {{ row.status === 'online' ? '在线' : '离线' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="CPU" width="70" prop="cpu_count" align="center">
            <template #default="{ row }">{{ row.cpu_count || '-' }}</template>
          </el-table-column>
          <el-table-column label="内存" width="90" align="right">
            <template #default="{ row }">{{ formatMemory(row.memory_total) }}</template>
          </el-table-column>
          <el-table-column prop="last_heartbeat" label="最后心跳" width="170" align="center" />
          <el-table-column prop="created_at" label="注册时间" width="170" align="center" />
        </el-table>

        <div v-if="total > 0" class="mt-6 flex justify-end">
          <el-pagination 
            background 
            layout="total, sizes, prev, pager, next" 
            :total="total" 
            :page-sizes="[10, 20, 50, 100]"
            :page-size="query.size" 
            :current-page="query.page" 
            @size-change="(s: number) => { query.size = s; query.page = 1; fetchData() }"
            @current-change="(p: number) => { query.page = p; fetchData() }" 
          />
        </div>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
/* Scoped styles replaced with Tailwind utility classes */
</style>
