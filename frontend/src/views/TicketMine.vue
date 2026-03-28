<script setup lang="ts">
defineOptions({ name: 'TicketMine' })
import { ref, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { ticketApi } from '../api'
import { useViewStateStore } from '../stores/viewState'

const router = useRouter()
const viewStateStore = useViewStateStore()
const loading = ref(false)
const tableData = ref<any[]>([])
const total = ref(0)
const query = ref({ page: 1, size: 20, scope: 'my_created', status: '', keyword: '' })

const statusMap: Record<string, { label: string; type: string }> = {
  open: { label: '待处理', type: 'info' },
  processing: { label: '处理中', type: 'warning' },
  resolved: { label: '已解决', type: 'success' },
  closed: { label: '已关闭', type: 'info' },
  rejected: { label: '已驳回', type: 'danger' },
}

const priorityMap: Record<string, { label: string; type: string }> = {
  low: { label: '低', type: 'info' },
  medium: { label: '中', type: 'info' },
  high: { label: '高', type: 'warning' },
  urgent: { label: '紧急', type: 'danger' },
}

async function fetchData() {
  loading.value = true
  try {
    const params: any = { ...query.value }
    Object.keys(params).forEach(k => { if (params[k] === '' || params[k] === null) delete params[k] })
    const res: any = await ticketApi.list(params)
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally { loading.value = false }
}

function handleSearch() { query.value.page = 1; fetchData() }
function handleReset() {
  query.value = { page: 1, size: 20, scope: 'my_created', status: '', keyword: '' }
  fetchData()
}

function openDetail(row: any) {
  router.push('/ticket/detail/' + row.id)
}

onMounted(() => { fetchData() })

onActivated(() => {
  if (viewStateStore.consumeTicketListDirty()) {
    fetchData()
  }
})
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <span>我的申请</span>
      </template>

      <el-form :inline="true" @submit.prevent="handleSearch" style="margin-bottom: 12px;">
        <el-form-item>
          <el-input v-model="query.keyword" placeholder="搜索标题/编号" clearable style="width: 200px;" @keyup.enter="handleSearch" />
        </el-form-item>
        <el-form-item>
          <el-select v-model="query.status" placeholder="状态" clearable style="width: 110px;">
            <el-option v-for="(v, k) in statusMap" :key="k" :label="v.label" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">查询</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="tableData" v-loading="loading" stripe border @row-click="openDetail" style="cursor: pointer;">
        <el-table-column prop="ticket_no" label="工单号" width="160" />
        <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip />
        <el-table-column prop="type_name" label="工单类型" width="100" />
        <el-table-column label="优先级" width="80">
          <template #default="{ row }">
            <el-tag :type="(priorityMap[row.priority]?.type as any) || 'info'" size="small">{{ priorityMap[row.priority]?.label || row.priority }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="(statusMap[row.status]?.type as any) || 'info'" size="small">{{ statusMap[row.status]?.label || row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="assignee_name" label="处理人" width="90">
          <template #default="{ row }">{{ row.assignee_name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170" />
      </el-table>

      <el-pagination v-if="total > 0" style="margin-top: 16px; justify-content: flex-end;" background layout="total, prev, pager, next" :total="total" :page-size="query.size" :current-page="query.page" @current-change="(p: number) => { query.page = p; fetchData() }" />
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
</style>
