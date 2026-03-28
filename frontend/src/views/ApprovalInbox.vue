<script setup lang="ts">
defineOptions({ name: 'ApprovalInbox' })
import { ref, onMounted } from 'vue'
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
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div class="page-head">
          <span>审批待办</span>
          <el-button @click="fetchData">刷新</el-button>
        </div>
      </template>

      <el-table :data="tableData" v-loading="loading" stripe border @row-click="openTicket" style="cursor: pointer;">
        <el-table-column prop="ticket_no" label="工单号" width="180" />
        <el-table-column prop="ticket_title" label="标题" min-width="220" show-overflow-tooltip />
        <el-table-column prop="policy_name" label="审批策略" width="160" />
        <el-table-column prop="stage_name" label="当前阶段" width="160" />
        <el-table-column prop="created_at" label="到达时间" width="170" />
        <el-table-column label="操作" fixed="right" min-width="180">
          <template #default="{ row }">
            <el-button link size="small" @click.stop="openTicket(row)">查看</el-button>
            <el-button link type="primary" size="small" @click.stop="quickApprove(row)">通过</el-button>
            <el-button link type="danger" size="small" @click.stop="openAction(row, 'reject')">拒绝</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!loading && tableData.length === 0" description="暂无待审批事项" />
    </el-card>

    <el-dialog v-model="actionDialog" :title="actionType === 'approve' ? '审批通过' : '审批拒绝'" width="460px">
      <el-form label-width="80px">
        <el-form-item label="备注">
          <el-input
            v-model="form.comment"
            type="textarea"
            :rows="4"
            :placeholder="actionType === 'approve' ? '可填写审批说明（选填）' : '请填写拒绝原因'"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="actionDialog = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitAction">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
.page-head { display: flex; justify-content: space-between; align-items: center; }
</style>
