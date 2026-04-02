<script setup lang="ts">
defineOptions({ name: 'TicketDetail' })
import { computed, ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Check, Switch, Select, RefreshRight, ChatDotSquare } from '@element-plus/icons-vue'
import { ticketApi, userApi } from '../api'
import { useViewStateStore } from '../stores/viewState'

const route = useRoute()
const router = useRouter()
const viewStateStore = useViewStateStore()
const ticket = ref<any>(null)
const approvalInstance = ref<any>(null)
const activities = ref<any[]>([])
const loading = ref(true)
const allUsers = ref<any[]>([])
const emptyDescription = ref('工单不存在或已被删除')

// 操作弹窗
const actionDialog = ref(false)
const actionType = ref('') // assign/process/close/reopen/comment/transfer
const actionForm = ref<any>({ assignee_id: undefined, action: '', content: '', resolution: '', note: '' })

const statusMap: Record<string, { label: string; type: string }> = {
  open: { label: '待处理', type: 'info' },
  processing: { label: '处理中', type: 'warning' },
  resolved: { label: '已解决', type: 'success' },
  closed: { label: '已关闭', type: 'info' },
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
  approval_pending: '发起审批',
  approval_rejected: '审批拒绝',
  approval_approved: '审批完成',
  approval_recorded: '审批记录',
}

const approvalStatusMap: Record<string, string> = {
  not_required: '无需审批',
  pending: '待审批',
  in_progress: '审批中',
  approved: '审批通过',
  rejected: '审批拒绝',
  canceled: '审批取消',
}

const ticketKindMap: Record<string, string> = {
  incident: '事件工单',
  request: '请求工单',
  change: '变更工单',
}

const approvalRecordStatusMap: Record<string, string> = {
  pending: '待处理',
  approve: '已通过',
  reject: '已拒绝',
  return: '已退回',
  transfer: '已转审',
  add_sign: '已加签',
  timeout: '已超时',
}

const approvalRecordTagTypeMap: Record<string, string> = {
  pending: 'info',
  approve: 'success',
  reject: 'danger',
  return: 'warning',
  transfer: '',
  add_sign: '',
  timeout: 'danger',
}

const extraFields = computed<Record<string, any>>(() => {
  if (!ticket.value?.extra_fields) return {}
  try {
    return typeof ticket.value.extra_fields === 'string'
      ? JSON.parse(ticket.value.extra_fields)
      : ticket.value.extra_fields
  } catch {
    return {}
  }
})

const relatedAssets = computed(() => {
  return extraFields.value?.resource_items || []
})

const resourceSummary = computed(() => {
  if (!ticket.value) return ''
  if (ticket.value.resource_type === 'asset' && relatedAssets.value.length > 0) {
    return `共 ${relatedAssets.value.length} 台主机资产`
  }
  return ticket.value.resource_name || '-'
})

const assigneeOptions = computed(() => {
  if (actionType.value === 'transfer' && ticket.value?.assignee_id) {
    return allUsers.value.filter((user: any) => user.id !== ticket.value.assignee_id)
  }
  return allUsers.value
})

const requestFormEntries = computed(() => {
  const raw = extraFields.value?.request_form
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return []
  }
  return Object.entries(raw)
    .map(([key, value]) => ({
      key,
      label: formatRequestFieldLabel(key),
      value: formatRequestFieldValue(value),
    }))
    .filter((item) => item.value !== '')
})

function formatRequestFieldLabel(key: string) {
  if (!key) return '-'
  return key
    .replace(/[_-]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function formatRequestFieldValue(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'boolean') return value ? '是' : '否'
  if (typeof value === 'number') return String(value)
  if (typeof value === 'string') return value.trim()
  if (Array.isArray(value)) {
    return value.map((item) => formatRequestFieldValue(item)).filter(Boolean).join('，')
  }
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value, null, 2)
    } catch {
      return String(value)
    }
  }
  return String(value)
}

async function fetchData() {
  loading.value = true
  const rawID = route.params.id
  const id = Number(rawID)

  if (!rawID || Number.isNaN(id) || id <= 0) {
    ticket.value = null
    activities.value = []
    emptyDescription.value = '请从工单列表选择具体工单'
    loading.value = false
    return
  }

  try {
    const [ticketRes, activitiesRes, approvalRes] = await Promise.all([
      ticketApi.getById(id),
      ticketApi.activities(id),
      ticketApi.approvalInstance(id).catch(() => ({ data: null })),
    ])
    ticket.value = (ticketRes as any).data
    activities.value = (activitiesRes as any).data?.list || []
    approvalInstance.value = (approvalRes as any).data || null
    emptyDescription.value = '工单不存在或已被删除'
  } catch {
    ticket.value = null
    approvalInstance.value = null
    activities.value = []
  } finally { loading.value = false }
}

function openAction(type: string) {
  actionType.value = type
  actionForm.value = { assignee_id: undefined, action: '', content: '', resolution: '', note: '' }
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
    if (actionType.value !== 'comment') {
      viewStateStore.markTicketListDirty()
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

watch(() => route.params.id, (newId, oldId) => {
  if (newId && newId !== oldId) {
    fetchData()
  }
})
</script>

<template>
  <div class="p-4 md:p-6 min-h-full">
    <el-button link @click="router.back()" class="mb-4 !text-slate-500 hover:!text-indigo-600 transition-colors">
      <el-icon class="mr-1"><ArrowLeft /></el-icon> 返回
    </el-button>

    <template v-if="ticket">
      <div class="flex flex-col lg:flex-row gap-4 xl:gap-6">
        <!-- 左侧主区域 -->
        <div class="flex-1 min-w-0">
          <el-card shadow="never" class="border-0 ring-1 ring-slate-100 rounded-xl bg-white mb-6">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between pb-4 border-b border-slate-100 mb-6 gap-4">
              <div class="flex items-center gap-2 flex-wrap">
                <el-tag size="large" effect="dark" class="!rounded-md font-bold tracking-wide !bg-slate-700 !border-slate-700">{{ ticket.ticket_no }}</el-tag>
                <el-tag type="info" size="large" v-if="ticket.type_name" class="!rounded-md">{{ ticket.type_name }}</el-tag>
                <el-tag :type="(statusMap[ticket.status]?.type as any)" size="large" effect="light" class="!rounded-md">
                  <span class="flex items-center gap-1.5">
                    <span class="w-1.5 h-1.5 rounded-full" :class="{
                      'bg-red-500': ticket.status === 'open',
                      'bg-orange-400': ticket.status === 'processing',
                      'bg-blue-500': ticket.status === 'resolved',
                      'bg-green-500': ticket.status === 'closed',
                      'bg-slate-500': ticket.status === 'rejected'
                    }"></span>
                    {{ statusMap[ticket.status]?.label }}
                  </span>
                </el-tag>
                <el-tag :color="priorityMap[ticket.priority]?.color" class="!text-white !border-none !rounded-md" size="large">{{ priorityMap[ticket.priority]?.label }}</el-tag>
              </div>
              
              <div class="flex items-center gap-2 flex-wrap">
                <el-button v-if="ticket.status === 'open' || (ticket.status === 'processing' && !ticket.assignee_id)" type="primary" @click="openAction('assign')" :icon="User" class="!rounded-lg shadow-sm">分配处理人</el-button>
                <el-button v-if="ticket.status === 'processing' && ticket.assignee_id" type="success" @click="openAction('process')" :icon="Check" class="!rounded-lg shadow-sm">处理工单</el-button>
                <el-button v-if="ticket.status === 'processing' && ticket.assignee_id" @click="openAction('transfer')" :icon="Switch" class="!rounded-lg">转交</el-button>
                <el-button v-if="ticket.status === 'resolved' || ticket.status === 'rejected'" type="primary" @click="openAction('close')" :icon="Select" class="!rounded-lg shadow-sm">确认关闭</el-button>
                <el-button v-if="ticket.status === 'closed' || ticket.status === 'rejected'" @click="openAction('reopen')" :icon="RefreshRight" class="!rounded-lg">重新打开</el-button>
                <el-button @click="openAction('comment')" :icon="ChatDotSquare" class="!rounded-lg hover:!text-indigo-600 hover:!border-indigo-300">评论</el-button>
              </div>
            </div>
            
            <h2 class="text-xl font-semibold text-slate-800 mb-4">{{ ticket.title }}</h2>
            <div class="bg-slate-50 p-4 rounded-lg text-slate-700 leading-relaxed min-h-[80px] text-sm whitespace-pre-wrap border border-slate-100/60">{{ ticket.description || '无详细描述' }}</div>

            <!-- 请求表单 -->
            <div v-if="requestFormEntries.length" class="mt-6 p-4 border border-slate-100 rounded-xl bg-gradient-to-b from-white to-slate-50/50 shadow-sm">
              <div class="flex items-center justify-between mb-4">
                <span class="font-semibold text-slate-700">请求表单</span>
                <el-tag size="small" type="info" class="!rounded">{{ ticket.request_template_name || ticketKindMap[ticket.ticket_kind] || '流程单' }}</el-tag>
              </div>
              <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div v-for="item in requestFormEntries" :key="item.key" class="p-3 rounded-lg bg-white border border-slate-100 shadow-sm transition-shadow hover:shadow-md">
                  <div class="text-xs text-slate-500 mb-1.5">{{ item.label }}</div>
                  <div class="text-slate-800 whitespace-pre-wrap break-words leading-relaxed text-sm font-medium">{{ item.value }}</div>
                </div>
              </div>
            </div>

            <!-- 关联资源 -->
            <div v-if="ticket.resource_type" class="mt-6 p-4 bg-slate-50/80 rounded-lg border border-slate-100 flex items-center gap-3">
              <span class="text-slate-500 text-sm">关联资源：</span>
              <el-tag size="small" class="!rounded-md">{{ ticket.resource_type }}</el-tag>
              <span class="font-medium text-slate-700">{{ resourceSummary }}</span>
            </div>
            <div v-if="ticket.resource_type === 'asset' && relatedAssets.length" class="mt-4 p-4 border border-slate-100 rounded-xl bg-gradient-to-b from-white to-slate-50/50 shadow-sm">
              <div class="font-semibold text-slate-700 mb-3 text-sm">关联主机</div>
              <div class="flex flex-col gap-3">
                <div v-for="asset in relatedAssets" :key="asset.id" class="p-3 rounded-lg bg-white border border-slate-100 shadow-sm hover:border-indigo-100 transition-colors">
                  <div class="flex items-center gap-2 flex-wrap mb-2">
                    <span class="font-semibold text-slate-800">{{ asset.hostname }}</span>
                    <el-tag size="small" type="info" class="!font-mono">{{ asset.ip }}</el-tag>
                    <el-tag size="small" :type="asset.status === 'online' ? 'success' : 'danger'" class="!rounded-md">
                      {{ asset.status === 'online' ? '在线' : '离线' }}
                    </el-tag>
                  </div>
                  <div class="flex items-center gap-3 flex-wrap text-xs text-slate-500">
                    <span class="flex items-center gap-1"><el-icon><Folder /></el-icon>{{ asset.service_tree_path || asset.service_tree_name || '未归属服务树' }}</span>
                    <span class="flex items-center gap-1"><el-icon><Monitor /></el-icon>{{ asset.os || '未知系统' }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 处理结果 -->
            <div v-if="ticket.resolution" class="mt-6 p-4 bg-green-50 border border-green-100 rounded-lg flex flex-col sm:flex-row sm:items-center gap-2">
              <span class="text-green-700 font-semibold flex items-center gap-1.5"><el-icon><SuccessFilled /></el-icon> 处理结果：{{ ticket.resolution }}</span>
              <span v-if="ticket.resolution_note" class="text-green-600/80 text-sm mt-1 sm:mt-0 sm:ml-2">{{ ticket.resolution_note }}</span>
            </div>

            <!-- 审批信息 -->
            <div v-if="ticket.approval_status && ticket.approval_status !== 'not_required'" class="mt-6 p-4 border border-slate-100 rounded-xl bg-gradient-to-b from-white to-slate-50/50 shadow-sm">
              <div class="flex items-center justify-between mb-4">
                <span class="font-semibold text-slate-700">审批信息</span>
                <el-tag size="small" :type="ticket.approval_status === 'approved' ? 'success' : ticket.approval_status === 'rejected' ? 'danger' : 'warning'" class="!rounded">
                  {{ approvalStatusMap[ticket.approval_status] || ticket.approval_status || '未开始' }}
                </el-tag>
              </div>
              <div v-if="approvalInstance">
                <div class="flex flex-wrap gap-4 text-sm text-slate-500 mb-4 bg-white p-3 rounded-lg border border-slate-100">
                  <span class="flex items-center gap-1.5"><el-icon><Guide /></el-icon> 审批策略：<span class="font-medium text-slate-700">{{ approvalInstance.policy_name || '-' }}</span></span>
                  <span class="flex items-center gap-1.5"><el-icon><Target /></el-icon> 当前阶段：<span class="font-medium text-slate-700">{{ approvalInstance.current_stage_name || approvalInstance.current_stage_no || '-' }}</span></span>
                </div>
                <div class="flex flex-col gap-3">
                  <div v-for="record in approvalInstance.records || []" :key="record.id" class="p-3 rounded-lg bg-white border border-slate-100 shadow-sm">
                    <div class="flex justify-between items-center mb-2">
                      <span class="font-semibold text-slate-800 flex items-center gap-2">
                        <div class="w-6 h-6 rounded-full bg-indigo-50 text-indigo-600 flex items-center justify-center text-xs">
                          {{ (record.approver_name || 'U')[0].toUpperCase() }}
                        </div>
                        {{ record.approver_name || `用户#${record.approver_id}` }}
                      </span>
                      <el-tag size="small" :type="(approvalRecordTagTypeMap[record.status] as any)" class="!rounded-md">
                        {{ approvalRecordStatusMap[record.status] || record.status }}
                      </el-tag>
                    </div>
                    <div class="flex justify-between text-xs text-slate-400 mt-2">
                      <span>阶段 {{ record.stage_no }}</span>
                      <span>{{ record.acted_at || record.created_at }}</span>
                    </div>
                    <div v-if="record.comment" class="mt-3 text-sm text-slate-600 bg-slate-50 p-2.5 rounded border border-slate-100 whitespace-pre-wrap">{{ record.comment }}</div>
                  </div>
                </div>
              </div>
              <div v-else class="text-sm text-slate-400 py-4 text-center bg-white rounded-lg border border-slate-100 border-dashed">该工单暂未生成审批实例。</div>
            </div>
          </el-card>
        </div>

        <!-- 右侧信息栏 -->
        <div class="w-full lg:w-80 xl:w-96 flex-shrink-0 flex flex-col gap-6">
          <el-card shadow="never" class="border-0 ring-1 ring-slate-100 rounded-xl bg-white">
            <template #header>
              <div class="font-semibold text-slate-800 text-sm">工单属性</div>
            </template>
            <el-descriptions :column="1" size="small" class="!mt-2" label-class-name="!text-slate-500 !w-20" content-class-name="!text-slate-800 !font-medium">
              <el-descriptions-item label="单据类型">{{ ticketKindMap[ticket.ticket_kind] || ticket.ticket_kind }}</el-descriptions-item>
              <el-descriptions-item v-if="ticket.request_template_name" label="请求模板">{{ ticket.request_template_name }}</el-descriptions-item>
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

          <!-- 活动时间线 -->
          <el-card shadow="never" class="border-0 ring-1 ring-slate-100 rounded-xl bg-white flex-1">
            <template #header>
              <div class="font-semibold text-slate-800 text-sm flex items-center gap-2">
                <el-icon class="text-indigo-500"><List /></el-icon> 活动记录
              </div>
            </template>
            <el-scrollbar max-height="600px" class="-mx-2 px-2">
              <div v-if="!activities.length" class="text-center text-sm text-slate-400 py-8">暂无活动记录</div>
              <el-timeline v-else class="pt-2 pl-1">
                <el-timeline-item 
                  v-for="a in activities" 
                  :key="a.id" 
                  :timestamp="a.created_at" 
                  placement="top" 
                  :type="a.type === 'create' ? 'success' : a.type.includes('approval') ? 'warning' : a.type === 'resolve' || a.type === 'close' ? 'danger' : 'primary'"
                  size="large"
                  :hollow="false"
                >
                  <div class="bg-white p-3 rounded-lg border border-slate-100 shadow-sm mb-2 group hover:border-indigo-100 transition-colors">
                    <div class="flex items-center gap-2 flex-wrap mb-1">
                      <span class="font-semibold text-sm text-slate-800">{{ a.is_system ? '系统' : a.user_name }}</span>
                      <el-tag size="small" type="info" class="!rounded-md !border-none !bg-slate-100 !text-slate-600">{{ activityTypeMap[a.type] || a.type }}</el-tag>
                    </div>
                    <div v-if="a.old_value || a.new_value" class="mt-2 text-xs bg-slate-50 p-2 rounded-md border border-slate-100/80 flex items-center flex-wrap gap-2">
                      <span class="line-through text-slate-400">{{ a.old_value || '无' }}</span>
                      <el-icon class="text-slate-300"><Right /></el-icon>
                      <span class="font-medium text-slate-700">{{ a.new_value || '空' }}</span>
                    </div>
                    <div v-if="a.content" class="mt-2 text-sm text-slate-600 bg-indigo-50/50 p-2.5 rounded-md border-l-2 border-indigo-300 whitespace-pre-wrap leading-relaxed">{{ a.content }}</div>
                  </div>
                </el-timeline-item>
              </el-timeline>
              <el-empty v-if="activities.length === 0" description="暂无活动记录" :image-size="60" />
            </el-scrollbar>
          </el-card>
        </div>
      </div>
    </template>
    <el-empty v-else :description="emptyDescription" :image-size="80" class="bg-white rounded-xl shadow-sm border border-slate-100 p-12">
      <el-button @click="router.back()">返回上一页</el-button>
    </el-empty>

    <!-- 操作弹窗 -->
    <el-dialog v-model="actionDialog" :title="{ assign: '分配处理人', process: '处理工单', close: '关闭工单', reopen: '重新打开', comment: '添加评论', transfer: '转交工单' }[actionType]" width="480px" class="!rounded-xl">
      <el-form :model="actionForm" label-width="80px" class="mt-2">
        <el-form-item v-if="actionType === 'assign' || actionType === 'transfer'" label="处理人">
          <el-select v-model="actionForm.assignee_id" placeholder="选择处理人" clearable class="w-full">
            <el-option v-for="u in assigneeOptions" :key="u.id" :label="u.real_name || u.username" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="actionType === 'process'" label="操作">
          <el-radio-group v-model="actionForm.action">
            <el-radio value="resolve">解决</el-radio>
            <el-radio value="reject">驳回</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="actionType === 'close'" label="处理结果">
          <el-select v-model="actionForm.resolution" placeholder="选择处理结果" class="w-full">
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
        <div class="flex items-center justify-end gap-2">
          <el-button @click="actionDialog = false" class="!rounded-lg">取消</el-button>
          <el-button type="primary" @click="submitAction" class="!rounded-lg">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
/* 覆盖 Element Plus 样式 */
:deep(.el-card__header) {
  padding: 12px 16px;
  border-bottom: 1px solid #f1f5f9;
  background-color: #f8fafc;
}
</style>
