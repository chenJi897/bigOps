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
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <span>我的待办</span>
      </template>

      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <el-tab-pane label="待处理工单" name="assigned">
          <el-table :data="assignedData" v-loading="assignedLoading" stripe border @row-click="openDetail" style="cursor: pointer;">
            <el-table-column prop="ticket_no" label="工单号" width="160" />
            <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip />
            <el-table-column prop="type_name" label="类型" width="100" />
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
            <el-table-column prop="creator_name" label="创建人" width="90" />
            <el-table-column prop="created_at" label="创建时间" width="170" />
          </el-table>
          <el-pagination v-if="assignedTotal > 0" style="margin-top: 16px; justify-content: flex-end;" background layout="total, prev, pager, next" :total="assignedTotal" :page-size="assignedQuery.size" :current-page="assignedQuery.page" @current-change="(p: number) => { assignedQuery.page = p; fetchAssigned() }" />
        </el-tab-pane>

        <el-tab-pane label="待审批工单" name="approval">
          <el-table :data="approvalData" v-loading="approvalLoading" stripe border @row-click="openDetail" style="cursor: pointer;">
            <el-table-column prop="ticket_no" label="工单号" width="160" />
            <el-table-column prop="ticket_title" label="标题" min-width="200" show-overflow-tooltip />
            <el-table-column prop="policy_name" label="审批策略" width="120" />
            <el-table-column label="当前阶段" width="100">
              <template #default="{ row }">第 {{ row.current_stage }} 级</template>
            </el-table-column>
            <el-table-column prop="applicant_name" label="申请人" width="90" />
            <el-table-column prop="created_at" label="到达时间" width="170" />
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click.stop="openDetail(row)">查看</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
</style>
