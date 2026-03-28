<script setup lang="ts">
defineOptions({ name: 'TicketDetail' })
import { computed, ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
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

            <div v-if="requestFormEntries.length" class="request-form-panel">
              <div class="request-form-panel-head">
                <span>请求表单</span>
                <el-tag size="small" type="info">
                  {{ ticket.request_template_name || ticketKindMap[ticket.ticket_kind] || '流程单' }}
                </el-tag>
              </div>
              <div class="request-form-grid">
                <div v-for="item in requestFormEntries" :key="item.key" class="request-form-item">
                  <div class="request-form-label">{{ item.label }}</div>
                  <div class="request-form-value">{{ item.value }}</div>
                </div>
              </div>
            </div>

            <!-- 关联资源 -->
            <div v-if="ticket.resource_type" style="margin-top: 20px; padding: 12px; background: #f5f7fa; border-radius: 4px;">
              <span style="color: #909399; font-size: 13px;">关联资源：</span>
              <el-tag size="small" style="margin-left: 4px;">{{ ticket.resource_type }}</el-tag>
              <span style="margin-left: 8px; font-weight: 500;">{{ resourceSummary }}</span>
            </div>
            <div v-if="ticket.resource_type === 'asset' && relatedAssets.length" class="related-assets">
              <div class="related-assets-title">关联主机</div>
              <div class="related-assets-list">
                <div v-for="asset in relatedAssets" :key="asset.id" class="related-asset-item">
                  <div class="related-asset-top">
                    <span class="related-asset-name">{{ asset.hostname }}</span>
                    <el-tag size="small" type="info">{{ asset.ip }}</el-tag>
                    <el-tag size="small" :type="asset.status === 'online' ? 'success' : 'danger'">
                      {{ asset.status === 'online' ? '在线' : '离线' }}
                    </el-tag>
                  </div>
                  <div class="related-asset-meta">
                    <span>{{ asset.service_tree_path || asset.service_tree_name || '未归属服务树' }}</span>
                    <span>{{ asset.os || '未知系统' }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 处理结果 -->
            <div v-if="ticket.resolution" style="margin-top: 12px; padding: 12px; background: #f0f9eb; border-radius: 4px;">
              <span style="color: #67c23a; font-weight: 600;">处理结果：{{ ticket.resolution }}</span>
              <span v-if="ticket.resolution_note" style="margin-left: 12px; color: #606266;">{{ ticket.resolution_note }}</span>
            </div>

            <div v-if="ticket.ticket_kind !== 'incident'" class="approval-panel">
              <div class="approval-panel-head">
                <span>审批信息</span>
                <el-tag size="small" :type="ticket.approval_status === 'approved' ? 'success' : ticket.approval_status === 'rejected' ? 'danger' : 'warning'">
                  {{ approvalStatusMap[ticket.approval_status] || ticket.approval_status || '未开始' }}
                </el-tag>
              </div>
              <div v-if="approvalInstance" class="approval-panel-body">
                <div class="approval-meta">
                  <span>审批策略：{{ approvalInstance.policy_name || '-' }}</span>
                  <span>当前阶段：{{ approvalInstance.current_stage_name || approvalInstance.current_stage_no || '-' }}</span>
                </div>
                <div class="approval-records">
                  <div v-for="record in approvalInstance.records || []" :key="record.id" class="approval-record">
                    <div class="approval-record-top">
                      <span class="approval-record-name">{{ record.approver_name || `用户#${record.approver_id}` }}</span>
                      <el-tag size="small" :type="(approvalRecordTagTypeMap[record.status] as any)">
                        {{ approvalRecordStatusMap[record.status] || record.status }}
                      </el-tag>
                    </div>
                    <div class="approval-record-meta">
                      <span>阶段 {{ record.stage_no }}</span>
                      <span>{{ record.acted_at || record.created_at }}</span>
                    </div>
                    <div v-if="record.comment" class="approval-record-comment">{{ record.comment }}</div>
                  </div>
                </div>
              </div>
              <div v-else class="approval-panel-empty">该工单暂未生成审批实例。</div>
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
        </el-col>
      </el-row>
    </template>
    <el-empty v-else :description="emptyDescription" :image-size="80">
      <el-button @click="router.back()">返回上一页</el-button>
    </el-empty>

    <!-- 操作弹窗 -->
    <el-dialog v-model="actionDialog" :title="{ assign: '分配处理人', process: '处理工单', close: '关闭工单', reopen: '重新打开', comment: '添加评论', transfer: '转交工单' }[actionType]" width="480px">
      <el-form :model="actionForm" label-width="80px">
        <el-form-item v-if="actionType === 'assign' || actionType === 'transfer'" label="处理人">
          <el-select v-model="actionForm.assignee_id" placeholder="选择处理人" clearable style="width: 100%;">
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

.related-assets {
  margin-top: 14px;
  padding: 14px;
  border: 1px solid #e5edf4;
  border-radius: 10px;
  background: linear-gradient(180deg, #fbfdff 0%, #f7fafc 100%);
}

.request-form-panel {
  margin-top: 14px;
  padding: 14px;
  border: 1px solid #e5edf4;
  border-radius: 10px;
  background: linear-gradient(180deg, #fdfefe 0%, #f8fbfd 100%);
}

.request-form-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  font-weight: 700;
  color: #334155;
}

.request-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.request-form-item {
  padding: 10px 12px;
  border-radius: 10px;
  background: #fff;
  border: 1px solid #e2e8f0;
}

.request-form-label {
  margin-bottom: 6px;
  font-size: 12px;
  color: #64748b;
}

.request-form-value {
  color: #1e293b;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.6;
}

.approval-panel {
  margin-top: 14px;
  padding: 14px;
  border: 1px solid #e5edf4;
  border-radius: 10px;
  background: linear-gradient(180deg, #fdfefe 0%, #f8fbfd 100%);
}

.approval-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  font-weight: 700;
  color: #334155;
}

.approval-meta {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  color: #64748b;
  font-size: 13px;
  margin-bottom: 12px;
}

.approval-records {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.approval-record {
  padding: 10px 12px;
  border-radius: 10px;
  background: #fff;
  border: 1px solid #e2e8f0;
}

.approval-record-top {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  align-items: center;
}

.approval-record-name {
  font-weight: 700;
  color: #1e293b;
}

.approval-record-meta {
  margin-top: 6px;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 12px;
  color: #94a3b8;
}

.approval-record-comment {
  margin-top: 8px;
  color: #475569;
  white-space: pre-wrap;
}

.approval-panel-empty {
  color: #94a3b8;
  font-size: 13px;
}

.related-assets-title {
  margin-bottom: 10px;
  color: #475467;
  font-size: 13px;
  font-weight: 700;
}

.related-assets-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.related-asset-item {
  padding: 10px 12px;
  border-radius: 10px;
  background: #fff;
  border: 1px solid #e7eef5;
}

.related-asset-top {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.related-asset-name {
  font-weight: 700;
  color: #223042;
}

.related-asset-meta {
  margin-top: 6px;
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  color: #667085;
  font-size: 12px;
}

@media (max-width: 1200px) {
  .request-form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
