<script setup lang="ts">
defineOptions({ name: 'TicketDetail' })
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ticketApi, userApi } from '../api'

const route = useRoute()
const router = useRouter()
const ticket = ref<any>(null)
const activities = ref<any[]>([])
const loading = ref(true)
const allUsers = ref<any[]>([])

// 操作弹窗
const actionDialog = ref(false)
const actionType = ref('') // assign/process/close/reopen/comment/transfer
const actionForm = ref<any>({ assignee_id: 0, action: '', content: '', resolution: '', note: '' })

const statusMap: Record<string, { label: string; type: string }> = {
  open: { label: '待处理', type: 'info' },
  processing: { label: '处理中', type: 'warning' },
  resolved: { label: '已解决', type: 'success' },
  closed: { label: '已关闭', type: '' },
  rejected: { label: '已驳回', type: 'danger' },
}

const priorityMap: Record<string, { label: string; color: string }> = {
  low: { label: '低', color: '#909399' },
  medium: { label: '中', color: '#409eff' },
  high: { label: '高', color: '#e6a23c' },
  urgent: { label: '紧急', color: '#f56c6c' },
}

const resolutionOptions = [
  { label: '已修复', value: 'fixed' },
  { label: '临时方案', value: 'workaround' },
  { label: '不予处理', value: 'wontfix' },
  { label: '重复工单', value: 'duplicate' },
  { label: '无效工单', value: 'invalid' },
]

const sourceMap: Record<string, string> = { manual: '手动', monitor: '监控', sync: '同步', system: '系统', cicd: 'CICD' }
const activityTypeMap: Record<string, string> = {
  create: '创建工单', assign: '分配处理人', comment: '评论', resolve: '解决',
  close: '关闭', reject: '驳回', reopen: '重新打开', transfer: '转交', auto_create: '自动创建',
}

async function fetchData() {
  loading.value = true
  const id = Number(route.params.id)
  try {
    const [ticketRes, activitiesRes] = await Promise.all([
      ticketApi.getById(id),
      ticketApi.activities(id),
    ])
    ticket.value = (ticketRes as any).data
    activities.value = (activitiesRes as any).data?.list || []
  } finally { loading.value = false }
}

function openAction(type: string) {
  actionType.value = type
  actionForm.value = { assignee_id: 0, action: '', content: '', resolution: '', note: '' }
  if (type === 'process') actionForm.value.action = 'resolve'
  actionDialog.value = true
}

async function submitAction() {
  const id = ticket.value.id
  try {
    switch (actionType.value) {
      case 'assign':
        if (!actionForm.value.assignee_id) { ElMessage.warning('请选择处理人'); return }
        await ticketApi.assign(id, actionForm.value.assignee_id)
        break
      case 'process':
        await ticketApi.process(id, actionForm.value.action, actionForm.value.content)
        break
      case 'close':
        if (!actionForm.value.resolution) { ElMessage.warning('请选择处理结果'); return }
        await ticketApi.close(id, actionForm.value.resolution, actionForm.value.note)
        break
      case 'reopen':
        await ticketApi.reopen(id, actionForm.value.content)
        break
      case 'comment':
        if (!actionForm.value.content) { ElMessage.warning('请输入评论'); return }
        await ticketApi.comment(id, actionForm.value.content)
        break
      case 'transfer':
        if (!actionForm.value.assignee_id) { ElMessage.warning('请选择转交人'); return }
        await ticketApi.transfer(id, actionForm.value.assignee_id, actionForm.value.content)
        break
    }
    ElMessage.success('操作成功')
    actionDialog.value = false
    fetchData()
  } catch {}
}

onMounted(() => {
  fetchData()
  userApi.list(1, 200).then((res: any) => { allUsers.value = res.data?.list || [] }).catch(() => {})
})
</script>

<template>
  <div class="page" v-loading="loading">
    <el-button link @click="router.back()" style="margin-bottom: 12px;"><el-icon><ArrowLeft /></el-icon> 返回</el-button>

    <template v-if="ticket">
      <el-row :gutter="16">
        <!-- 左侧主区域 -->
        <el-col :span="17">
          <el-card shadow="never">
            <div style="display: flex; align-items: center; gap: 12px; margin-bottom: 16px;">
              <el-tag>{{ ticket.ticket_no }}</el-tag>
              <el-tag type="info" v-if="ticket.type_name">{{ ticket.type_name }}</el-tag>
              <el-tag :type="(statusMap[ticket.status]?.type as any)" size="default">{{ statusMap[ticket.status]?.label }}</el-tag>
              <el-tag :color="priorityMap[ticket.priority]?.color" style="color: #fff; border: none;" size="small">{{ priorityMap[ticket.priority]?.label }}</el-tag>
            </div>
            <h3 style="margin: 0 0 16px 0;">{{ ticket.title }}</h3>
            <div style="white-space: pre-wrap; color: #606266; line-height: 1.8; min-height: 60px;">{{ ticket.description || '无描述' }}</div>

            <!-- 关联资源 -->
            <div v-if="ticket.resource_type" style="margin-top: 20px; padding: 12px; background: #f5f7fa; border-radius: 4px;">
              <span style="color: #909399; font-size: 13px;">关联资源：</span>
              <el-tag size="small" style="margin-left: 4px;">{{ ticket.resource_type }}</el-tag>
              <span style="margin-left: 8px; font-weight: 500;">{{ ticket.resource_name }}</span>
            </div>

            <!-- 处理结果 -->
            <div v-if="ticket.resolution" style="margin-top: 12px; padding: 12px; background: #f0f9eb; border-radius: 4px;">
              <span style="color: #67c23a; font-weight: 600;">处理结果：{{ ticket.resolution }}</span>
              <span v-if="ticket.resolution_note" style="margin-left: 12px; color: #606266;">{{ ticket.resolution_note }}</span>
            </div>
          </el-card>

          <!-- 操作按钮 -->
          <div style="margin: 16px 0; display: flex; gap: 8px;">
            <el-button v-if="ticket.status === 'open'" type="primary" @click="openAction('assign')">分配处理人</el-button>
            <el-button v-if="ticket.status === 'processing'" type="success" @click="openAction('process')">处理</el-button>
            <el-button v-if="ticket.status === 'processing'" @click="openAction('transfer')">转交</el-button>
            <el-button v-if="ticket.status === 'resolved' || ticket.status === 'rejected'" type="primary" @click="openAction('close')">确认关闭</el-button>
            <el-button v-if="ticket.status === 'closed' || ticket.status === 'rejected'" @click="openAction('reopen')">重新打开</el-button>
            <el-button @click="openAction('comment')">评论</el-button>
          </div>

          <!-- 活动时间线 -->
          <el-card shadow="never">
            <template #header><span style="font-weight: 600;">活动记录</span></template>
            <el-timeline>
              <el-timeline-item v-for="a in activities" :key="a.id" :timestamp="a.created_at" placement="top" :type="a.is_system ? 'primary' : ''">
                <div>
                  <span style="font-weight: 600; margin-right: 8px;">{{ a.is_system ? '系统' : a.user_name }}</span>
                  <el-tag size="small" type="info">{{ activityTypeMap[a.type] || a.type }}</el-tag>
                  <span v-if="a.old_value || a.new_value" style="margin-left: 8px; color: #909399; font-size: 12px;">{{ a.old_value }} → {{ a.new_value }}</span>
                </div>
                <p v-if="a.content" style="margin: 8px 0 0; color: #606266; white-space: pre-wrap;">{{ a.content }}</p>
              </el-timeline-item>
            </el-timeline>
            <el-empty v-if="activities.length === 0" description="暂无活动记录" :image-size="60" />
          </el-card>
        </el-col>

        <!-- 右侧信息栏 -->
        <el-col :span="7">
          <el-card shadow="never">
            <el-descriptions :column="1" size="small">
              <el-descriptions-item label="创建人">{{ ticket.creator_name }}</el-descriptions-item>
              <el-descriptions-item label="处理人">{{ ticket.assignee_name || '-' }}</el-descriptions-item>
              <el-descriptions-item label="提交部门">{{ ticket.submit_dept_name || '-' }}</el-descriptions-item>
              <el-descriptions-item label="处理部门">{{ ticket.handle_dept_name || '-' }}</el-descriptions-item>
              <el-descriptions-item label="来源">{{ sourceMap[ticket.source] || ticket.source }}</el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ ticket.created_at }}</el-descriptions-item>
              <el-descriptions-item label="更新时间">{{ ticket.updated_at }}</el-descriptions-item>
              <el-descriptions-item v-if="ticket.resolved_at" label="解决时间">{{ ticket.resolved_at }}</el-descriptions-item>
              <el-descriptions-item v-if="ticket.closed_at" label="关闭时间">{{ ticket.closed_at }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
        </el-col>
      </el-row>
    </template>

    <!-- 操作弹窗 -->
    <el-dialog v-model="actionDialog" :title="{ assign: '分配处理人', process: '处理工单', close: '关闭工单', reopen: '重新打开', comment: '添加评论', transfer: '转交工单' }[actionType]" width="480px">
      <el-form :model="actionForm" label-width="80px">
        <el-form-item v-if="actionType === 'assign' || actionType === 'transfer'" label="处理人">
          <el-select v-model="actionForm.assignee_id" placeholder="选择处理人" style="width: 100%;">
            <el-option v-for="u in allUsers" :key="u.id" :label="u.real_name || u.username" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="actionType === 'process'" label="操作">
          <el-radio-group v-model="actionForm.action">
            <el-radio value="resolve">解决</el-radio>
            <el-radio value="reject">驳回</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="actionType === 'close'" label="处理结果">
          <el-select v-model="actionForm.resolution" placeholder="选择处理结果" style="width: 100%;">
            <el-option v-for="o in resolutionOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="actionType === 'close'" label="说明">
          <el-input v-model="actionForm.note" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
        <el-form-item v-if="['process', 'reopen', 'comment', 'transfer'].includes(actionType)" label="内容">
          <el-input v-model="actionForm.content" type="textarea" :rows="3" :placeholder="actionType === 'comment' ? '输入评论...' : '说明原因...'" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="actionDialog = false">取消</el-button>
        <el-button type="primary" @click="submitAction">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
</style>
