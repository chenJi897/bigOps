<script setup lang="ts">
defineOptions({ name: 'TicketMine' })
import { ref, onMounted, onActivated } from 'vue'
import { Search } from '@element-plus/icons-vue'
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
  <div class="p-4 sm:p-6 lg:p-8 space-y-4 sm:space-y-6">
    <el-card shadow="never" class="border-0 shadow-sm ring-1 ring-gray-200/50 rounded-xl">
      <template #header>
        <div class="flex items-center gap-3">
          <h2 class="text-xl font-semibold text-gray-800 tracking-tight">我的申请</h2>
        </div>
      </template>

      <div class="flex flex-col space-y-5">
        <div class="flex flex-col xl:flex-row justify-end items-start xl:items-center gap-4 bg-gray-50/50 p-4 rounded-lg border border-gray-100">
          <el-form :inline="true" @submit.prevent="handleSearch" class="flex flex-wrap gap-3 w-full xl:w-auto xl:justify-end" style="margin-bottom: 0;">
            <el-input v-model="query.keyword" placeholder="搜索标题或编号" clearable class="w-48 sm:w-64" @keyup.enter="handleSearch">
              <template #prefix>
                <el-icon class="text-gray-400"><Search /></el-icon>
              </template>
            </el-input>
            
            <el-select v-model="query.status" placeholder="所有状态" clearable class="w-28 sm:w-32">
              <el-option v-for="(v, k) in statusMap" :key="k" :label="v.label" :value="k" />
            </el-select>
            
            <div class="flex items-center gap-2">
              <el-button type="primary" @click="handleSearch">查询</el-button>
              <el-button @click="handleReset" plain>重置</el-button>
            </div>
          </el-form>
        </div>

        <el-table :data="tableData" v-loading="loading" stripe :border="false" class="w-full shadow-sm rounded-lg overflow-hidden border border-gray-100" @row-click="openDetail" style="cursor: pointer;">
          <el-table-column prop="ticket_no" label="工单号" width="160" />
          <el-table-column prop="title" label="标题" min-width="240" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="font-medium text-gray-800 group-hover:text-indigo-600 transition-colors">{{ row.title }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="type_name" label="工单类型" width="120">
            <template #default="{ row }">
              <span class="text-gray-600">{{ row.type_name || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="优先级" width="90">
            <template #default="{ row }">
              <el-tag :type="(priorityMap[row.priority]?.type as any) || 'info'" size="small" effect="light" class="rounded-md">
                {{ priorityMap[row.priority]?.label || row.priority }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="(statusMap[row.status]?.type as any) || 'info'" size="small" effect="light" class="rounded-md">
                {{ statusMap[row.status]?.label || row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="assignee_name" label="处理人" width="100">
            <template #default="{ row }">
              <span class="text-gray-700">{{ row.assignee_name || '-' }}</span>
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
:deep(.el-table__row) {
  @apply hover:bg-indigo-50/50 transition-colors duration-200;
}
:deep(.el-table th.el-table__cell) {
  @apply bg-gray-50/80 text-gray-600 font-medium;
}
</style>
