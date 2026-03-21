<script setup lang="ts">
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
  <div class="page">
    <el-card shadow="never">
      <template #header><span>操作审计日志</span></template>

      <el-form :inline="true" @submit.prevent="handleSearch" style="margin-bottom: 16px;">
        <el-form-item label="用户名">
          <el-input v-model="query.username" placeholder="用户名" clearable style="width: 150px;" />
        </el-form-item>
        <el-form-item label="操作类型">
          <el-select v-model="query.action" style="width: 120px;">
            <el-option v-for="o in actionOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="资源类型">
          <el-select v-model="query.resource" style="width: 120px;">
            <el-option v-for="o in resourceOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="tableData" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="操作人" width="120" />
        <el-table-column prop="action" label="操作类型" width="100">
          <template #default="{ row }">
            <el-tag :type="row.action === 'delete' ? 'danger' : row.action === 'create' ? 'success' : 'info'" size="small">
              {{ row.action }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="resource" label="资源类型" width="120" />
        <el-table-column prop="detail" label="操作详情" min-width="200" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP" width="140" />
        <el-table-column prop="status_code" label="状态码" width="80" />
        <el-table-column prop="created_at" label="操作时间" width="180" />
      </el-table>

      <el-pagination
        v-if="total > 0"
        style="margin-top: 16px; justify-content: flex-end;"
        background layout="total, prev, pager, next"
        :total="total" :page-size="query.size" :current-page="query.page"
        @current-change="handlePageChange"
      />
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
</style>
