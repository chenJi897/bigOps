<script setup lang="ts">
defineOptions({ name: 'ApprovalInbox' })
import { ref, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { approvalApi } from '../api'

const router = useRouter()
const loading = ref(false)
const tableData = ref<any[]>([])
const actionDialog = ref(false)
const actionType = ref<'approve' | 'reject'>('approve')
const currentRow = ref<any>(null)
const form = ref({ comment: '' })
const submitting = ref(false)

async function fetchData() {
  loading.value = true
  try {
    const res: any = await approvalApi.pending()
    tableData.value = res.data || []
  } finally {
    loading.value = false
  }
}

function openAction(row: any, type: 'approve' | 'reject') {
  currentRow.value = row
  actionType.value = type
  form.value = { comment: '' }
  actionDialog.value = true
}

async function submitAction() {
  if (!currentRow.value) return
  if (actionType.value === 'reject' && !form.value.comment) {
    ElMessage.warning('请填写拒绝原因')
    return
  }
  submitting.value = true
  try {
    if (actionType.value === 'approve') {
      await approvalApi.approve(currentRow.value.instance_id, form.value.comment)
      ElMessage.success('审批通过')
    } else {
      await approvalApi.reject(currentRow.value.instance_id, form.value.comment)
      ElMessage.success('审批已拒绝')
    }
    actionDialog.value = false
    fetchData()
  } finally {
    submitting.value = false
  }
}

async function quickApprove(row: any) {
  try {
    await ElMessageBox.confirm(`确认通过 ${row.ticket_no} 吗？`, '提示', { type: 'warning' })
    await approvalApi.approve(row.instance_id)
    ElMessage.success('审批通过')
    fetchData()
  } catch {}
}

function openTicket(row: any) {
  router.push('/ticket/detail/' + row.ticket_id)
}

onMounted(fetchData)
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- Header -->
    <div class="flex justify-between items-center mb-4">
      <div class="flex items-center gap-3">
        <div>
          <h2 class="text-lg font-bold text-gray-900">审批待办</h2>
          <p class="text-xs text-gray-500 mt-1">需要您处理的审批事项</p>
        </div>
        <el-badge v-if="tableData.length > 0" :value="tableData.length" type="danger" />
      </div>
      <el-button @click="fetchData">
        <template #icon><el-icon><Refresh /></el-icon></template>
        刷新
      </el-button>
    </div>

    <!-- Table -->
    <div class="flex-1 bg-white border border-gray-200 rounded-lg shadow-sm flex flex-col overflow-hidden">
      <el-table :data="tableData" v-loading="loading" class="flex-1 w-full" stripe @row-click="openTicket" style="cursor: pointer;">
        <el-table-column prop="ticket_no" label="工单号" width="180">
          <template #default="{ row }">
            <span class="font-medium text-gray-800">{{ row.ticket_no }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="ticket_title" label="标题" min-width="240" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="font-medium text-gray-800 group-hover:text-indigo-600 transition-colors">{{ row.ticket_title }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="policy_name" label="审批策略" width="160">
          <template #default="{ row }">
            <span class="text-gray-600">{{ row.policy_name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="stage_name" label="当前阶段" width="160">
          <template #default="{ row }">
            <el-tag type="warning" size="small">{{ row.stage_name || '审批中' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="到达时间" width="170" align="center">
          <template #default="{ row }">
            <span class="text-gray-500 text-sm">{{ row.created_at }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="180" align="center">
          <template #default="{ row }">
            <div class="flex items-center justify-center gap-2">
              <el-button link size="small" @click.stop="openTicket(row)">查看</el-button>
              <el-button link type="primary" size="small" @click.stop="quickApprove(row)">通过</el-button>
              <el-button link type="danger" size="small" @click.stop="openAction(row, 'reject')">拒绝</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!loading && tableData.length === 0" description="暂无待审批事项" :image-size="80" class="py-12 bg-gray-50 flex-1" />
    </div>

    <!-- Dialog -->
    <el-dialog v-model="actionDialog" :title="actionType === 'approve' ? '审批通过' : '审批拒绝'" width="460px" destroy-on-close align-center>
      <el-form label-width="80px" label-position="top" @submit.prevent>
        <el-form-item :label="actionType === 'approve' ? '审批意见 (选填)' : '拒绝原因 (必填)'">
          <el-input v-model="form.comment" type="textarea" :rows="4" :placeholder="actionType === 'approve' ? '同意' : '请填写拒绝原因'" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="actionDialog = false">取消</el-button>
          <el-button :type="actionType === 'approve' ? 'primary' : 'danger'" :loading="submitting" @click="submitAction">
            {{ actionType === 'approve' ? '确认通过' : '确认拒绝' }}
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
</style>
