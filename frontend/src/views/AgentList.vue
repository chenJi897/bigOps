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
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>Agent 管理</span>
          <el-button @click="fetchData" :loading="loading">刷新</el-button>
        </div>
      </template>

      <el-form :inline="true" @submit.prevent="handleSearch" style="margin-bottom: 12px;">
        <el-form-item>
          <el-select v-model="query.status" placeholder="状态" clearable style="width: 120px;">
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">筛选</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="agent_id" label="Agent ID" min-width="180" show-overflow-tooltip />
        <el-table-column prop="hostname" label="主机名" min-width="150" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP 地址" width="140" />
        <el-table-column prop="version" label="版本" width="90" />
        <el-table-column prop="os" label="系统" width="120" show-overflow-tooltip />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 'online' ? 'success' : 'danger'" size="small">
              {{ row.status === 'online' ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="CPU" width="70" prop="cpu_count">
          <template #default="{ row }">{{ row.cpu_count || '-' }}</template>
        </el-table-column>
        <el-table-column label="内存" width="90">
          <template #default="{ row }">{{ formatMemory(row.memory_total) }}</template>
        </el-table-column>
        <el-table-column prop="last_heartbeat" label="最后心跳" width="170" />
        <el-table-column prop="created_at" label="注册时间" width="170" />
      </el-table>

      <el-pagination v-if="total > 0" style="margin-top: 16px; justify-content: flex-end;" background layout="total, prev, pager, next" :total="total" :page-size="query.size" :current-page="query.page" @current-change="(p: number) => { query.page = p; fetchData() }" />
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
</style>
