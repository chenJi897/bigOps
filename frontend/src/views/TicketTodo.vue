<script setup lang="ts">
defineOptions({ name: 'TicketTodo' })
import { ref, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { ticketApi, approvalApi } from '../api'
import { useViewStateStore } from '../stores/viewState'

const router = useRouter()
const viewStateStore = useViewStateStore()
const activeTab = ref('assigned')

// 待处理工单
const assignedLoading = ref(false)
const assignedData = ref<any[]>([])
const assignedTotal = ref(0)
const assignedQuery = ref({ page: 1, size: 20, scope: 'my_assigned' })

// 待审批
const approvalLoading = ref(false)
const approvalData = ref<any[]>([])

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

async function fetchAssigned() {
  assignedLoading.value = true
  try {
    const res: any = await ticketApi.list(assignedQuery.value)
    assignedData.value = res.data?.list || []
    assignedTotal.value = res.data?.total || 0
  } finally { assignedLoading.value = false }
}

async function fetchApprovals() {
  approvalLoading.value = true
  try {
    const res: any = await approvalApi.pending()
    approvalData.value = res.data || []
  } finally { approvalLoading.value = false }
}

function openDetail(row: any) {
  router.push('/ticket/detail/' + (row.ticket_id || row.id))
}

function handleTabChange() {
  if (activeTab.value === 'assigned') fetchAssigned()
  else fetchApprovals()
}

onMounted(() => {
  fetchAssigned()
  fetchApprovals()
})

onActivated(() => {
  if (viewStateStore.consumeTicketListDirty()) {
    fetchAssigned()
    fetchApprovals()
  }
})
</script>

<template>
  <div class="p-4 sm:p-6 lg:p-8 space-y-4 sm:space-y-6">
    <el-card shadow="never" class="border-0 shadow-sm ring-1 ring-gray-200/50 rounded-xl">
      <template #header>
        <div class="flex items-center gap-3">
          <h2 class="text-xl font-semibold text-gray-800 tracking-tight">我的待办</h2>
          <el-badge v-if="assignedTotal > 0 || approvalData.length > 0" :value="assignedTotal + approvalData.length" class="ml-2" type="danger" />
        </div>
      </template>

      <div class="flex flex-col space-y-5">
        <el-tabs v-model="activeTab" @tab-change="handleTabChange" class="w-full">
          <el-tab-pane label="待处理工单" name="assigned">
            <el-table :data="assignedData" v-loading="assignedLoading" stripe :border="false" class="w-full shadow-sm rounded-lg overflow-hidden border border-gray-100" @row-click="openDetail" style="cursor: pointer;">
              <el-table-column prop="ticket_no" label="工单号" width="160" />
              <el-table-column prop="title" label="标题" min-width="240" show-overflow-tooltip>
                <template #default="{ row }">
                  <span class="font-medium text-gray-800 group-hover:text-indigo-600 transition-colors">{{ row.title }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="type_name" label="类型" width="120">
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
              <el-table-column prop="creator_name" label="创建人" width="100">
                <template #default="{ row }">
                  <span class="text-gray-700">{{ row.creator_name || '-' }}</span>
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
                v-if="assignedTotal > 0" 
                background 
                layout="total, sizes, prev, pager, next, jumper" 
                :page-sizes="[10, 20, 50, 100]"
                :total="assignedTotal" 
                v-model:page-size="assignedQuery.size"
                :current-page="assignedQuery.page" 
                @size-change="(s: number) => { assignedQuery.size = s; fetchAssigned() }"
                @current-change="(p: number) => { assignedQuery.page = p; fetchAssigned() }" 
              />
            </div>
          </el-tab-pane>

          <el-tab-pane label="待审批工单" name="approval">
            <el-table :data="approvalData" v-loading="approvalLoading" stripe :border="false" class="w-full shadow-sm rounded-lg overflow-hidden border border-gray-100" @row-click="openDetail" style="cursor: pointer;">
              <el-table-column prop="ticket_no" label="工单号" width="160" />
              <el-table-column prop="ticket_title" label="标题" min-width="240" show-overflow-tooltip>
                <template #default="{ row }">
                  <span class="font-medium text-gray-800 group-hover:text-indigo-600 transition-colors">{{ row.ticket_title }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="policy_name" label="审批策略" width="140">
                <template #default="{ row }">
                  <span class="text-gray-600">{{ row.policy_name || '-' }}</span>
                </template>
              </el-table-column>
              <el-table-column label="当前阶段" width="100">
                <template #default="{ row }">
                  <el-tag type="warning" size="small" effect="light" class="rounded-md">第 {{ row.current_stage }} 级</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="applicant_name" label="申请人" width="100">
                <template #default="{ row }">
                  <span class="text-gray-700">{{ row.applicant_name || '-' }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="created_at" label="到达时间" width="170">
                <template #default="{ row }">
                  <span class="text-gray-500 text-sm">{{ row.created_at }}</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="100" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click.stop="openDetail(row)">查看处理</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
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
:deep(.el-tabs__item) {
  @apply text-base;
}
</style>
