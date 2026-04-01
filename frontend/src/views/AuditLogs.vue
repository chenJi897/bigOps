<script setup lang="ts">
defineOptions({ name: 'AuditLogs' })
import { ref, onMounted } from 'vue'
import { auditLogApi } from '../api'

const loading = ref(false)
const tableData = ref<any[]>([])
const total = ref(0)
const query = ref({ page: 1, size: 20, username: '', action: '', resource: '' })

const actionOptions = [
  { label: '全部', value: '' },
  { label: '创建', value: 'create' },
  { label: '更新', value: 'update' },
  { label: '删除', value: 'delete' },
  { label: '登录', value: 'login' },
  { label: '登出', value: 'logout' },
]

const resourceOptions = [
  { label: '全部', value: '' },
  { label: '用户', value: 'user' },
  { label: '角色', value: 'role' },
  { label: '菜单', value: 'menu' },
  { label: '云账号', value: 'cloud_account' },
  { label: '资产', value: 'asset' },
  { label: '服务树', value: 'service_tree' },
]

async function fetchData() {
  loading.value = true
  try {
    const res: any = await auditLogApi.list(query.value)
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

function handlePageChange(page: number) {
  query.value.page = page
  fetchData()
}

onMounted(fetchData)
</script>

<template>
  <div class="p-4 md:p-6 min-h-full flex flex-col">
    <el-card shadow="never" class="border-0 shadow-sm flex-1 flex flex-col">
      <template #header>
        <div class="flex items-center">
          <span class="text-base font-medium text-gray-800">操作审计日志</span>
        </div>
      </template>

      <div class="flex flex-wrap items-center gap-4 mb-4">
        <div class="flex items-center gap-2">
          <span class="text-sm text-gray-600">用户名</span>
          <el-input v-model="query.username" placeholder="请输入" clearable class="w-40" @keyup.enter="handleSearch" />
        </div>
        <div class="flex items-center gap-2">
          <span class="text-sm text-gray-600">操作类型</span>
          <el-select v-model="query.action" class="w-32" clearable placeholder="全部">
            <el-option v-for="o in actionOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-sm text-gray-600">资源类型</span>
          <el-select v-model="query.resource" class="w-32" clearable placeholder="全部">
            <el-option v-for="o in resourceOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </div>
        <el-button type="primary" @click="handleSearch">
          <el-icon class="mr-1"><Search /></el-icon> 搜索
        </el-button>
      </div>

      <el-table :data="tableData" v-loading="loading" stripe border class="w-full">
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column prop="username" label="操作人" width="140" align="center">
          <template #default="{ row }">
            <span class="font-medium text-gray-700">{{ row.username }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="action" label="操作类型" width="120" align="center">
          <template #default="{ row }">
            <el-tag :type="row.action === 'delete' ? 'danger' : row.action === 'create' ? 'success' : row.action === 'update' ? 'warning' : 'info'" size="small" effect="plain" round>
              {{ actionOptions.find(o => o.value === row.action)?.label || row.action }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="resource" label="资源类型" width="140" align="center">
          <template #default="{ row }">
            <span class="text-gray-600">{{ resourceOptions.find(o => o.value === row.resource)?.label || row.resource }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="detail" label="操作详情" min-width="250" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="font-mono text-xs text-gray-500 bg-gray-50 px-2 py-1 rounded">{{ row.detail || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="IP" width="140" align="center">
          <template #default="{ row }">
            <span class="text-gray-500">{{ row.ip || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="status_code" label="状态码" width="100" align="center">
          <template #default="{ row }">
            <span :class="row.status_code >= 400 ? 'text-red-500 font-medium' : 'text-emerald-500 font-medium'">
              {{ row.status_code }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="操作时间" width="180" align="center" />
      </el-table>

      <div v-if="total > 0" class="mt-4 flex justify-end">
        <el-pagination
          background 
          layout="total, prev, pager, next"
          :total="total" 
          :page-size="query.size" 
          :current-page="query.page"
          @current-change="handlePageChange"
        />
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
:deep(.el-table) {
  flex: 1;
}
</style>
