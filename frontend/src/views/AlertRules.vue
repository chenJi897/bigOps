<script setup lang="ts">
defineOptions({ name: 'AlertRules' })

import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { alertRuleApi, onCallApi, serviceTreeApi, taskApi, userApi } from '../api'

const loading = ref(false)
const eventLoading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(0)
const autoRefresh = ref(true)
const tableData = ref<any[]>([])
const events = ref<any[]>([])
const users = ref<any[]>([])
const tasks = ref<any[]>([])
const oncallSchedules = ref<any[]>([])
const serviceTrees = ref<any[]>([])

const rulePager = ref({
  page: 1,
  size: 20,
  total: 0,
})

const eventPager = ref({
  page: 1,
  size: 20,
  total: 0,
})

const filters = ref({
  keyword: '',
  metric_type: '',
  severity: '',
  enabled: '',
})

const eventFilters = ref({
  keyword: '',
  status: '',
  severity: '',
})
const router = useRouter()

const form = ref<any>({
  name: '',
  metric_type: 'cpu_usage',
  operator: 'gt',
  threshold: 80,
  severity: 'warning',
  enabled: 1,
  description: '',
  notify_user_ids: [] as number[],
  notify_channels: ['in_app'] as string[],
  action: 'notify_only',
  repair_task_id: null as number | null,
  ticket_type_id: null as number | null,
  oncall_schedule_id: null as number | null,
  service_tree_id: null as number | null,
  owner_id: null as number | null,
})

let refreshTimer: number | null = null

const metricOptions = [
  { label: 'CPU 使用率', value: 'cpu_usage' },
  { label: '内存使用率', value: 'memory_usage' },
  { label: '磁盘使用率', value: 'disk_usage' },
  { label: 'Agent 离线', value: 'agent_offline' },
]

const operatorOptions = [
  { label: '大于', value: 'gt' },
  { label: '大于等于', value: 'gte' },
  { label: '小于', value: 'lt' },
  { label: '小于等于', value: 'lte' },
  { label: '等于', value: 'eq' },
  { label: '不等于', value: 'neq' },
]

const severityOptions = [
  { label: '提示', value: 'info' },
  { label: '警告', value: 'warning' },
  { label: '严重', value: 'critical' },
]

const enabledOptions = [
  { label: '全部状态', value: '' },
  { label: '启用', value: 1 },
  { label: '停用', value: 0 },
]

const eventStatusOptions = [
  { label: '全部状态', value: '' },
  { label: '触发中', value: 'firing' },
  { label: '已确认', value: 'acknowledged' },
  { label: '已恢复', value: 'resolved' },
]

const notifyChannelOptions = [
  { label: '站内通知', value: 'in_app' },
  { label: '邮件', value: 'email' },
  { label: 'Webhook', value: 'webhook' },
  { label: 'Message Pusher', value: 'message_pusher' },
]

async function fetchUsers() {
  const res = await userApi.list(1, 500)
  users.value = (res as any).data?.list || []
}

async function fetchTasks() {
  const res = await taskApi.list({ page: 1, size: 500 })
  tasks.value = (res as any).data?.list || []
}

async function fetchOnCallSchedules() {
  const res = await onCallApi.list()
  oncallSchedules.value = (res as any).data || []
}

async function fetchServiceTrees() {
  const res = await serviceTreeApi.tree()
  serviceTrees.value = flattenTree((res as any).data || [])
}

async function fetchRules(showLoading = true) {
  if (showLoading) {
    loading.value = true
  }
  try {
    const res = await alertRuleApi.list({
      page: rulePager.value.page,
      size: rulePager.value.size,
      keyword: filters.value.keyword.trim(),
      metric_type: filters.value.metric_type,
      severity: filters.value.severity,
      enabled: filters.value.enabled === '' ? undefined : Number(filters.value.enabled),
    })
    tableData.value = (res as any).data?.list || []
    rulePager.value.total = Number((res as any).data?.total || 0)
  } finally {
    loading.value = false
  }
}

async function fetchEvents(showLoading = true) {
  if (showLoading) {
    eventLoading.value = true
  }
  try {
    const res = await alertRuleApi.events({
      page: eventPager.value.page,
      size: eventPager.value.size,
      status: eventFilters.value.status,
      severity: eventFilters.value.severity,
      keyword: eventFilters.value.keyword.trim(),
    })
    events.value = (res as any).data?.list || []
    eventPager.value.total = Number((res as any).data?.total || 0)
  } finally {
    eventLoading.value = false
  }
}

async function fetchData(showLoading = true) {
  await Promise.all([
    fetchRules(showLoading),
    fetchEvents(showLoading),
    users.value.length ? Promise.resolve() : fetchUsers(),
    tasks.value.length ? Promise.resolve() : fetchTasks(),
    oncallSchedules.value.length ? Promise.resolve() : fetchOnCallSchedules(),
    serviceTrees.value.length ? Promise.resolve() : fetchServiceTrees(),
  ])
}

function openAdd() {
  isEdit.value = false
  editId.value = 0
  form.value = {
    name: '',
    metric_type: 'cpu_usage',
    operator: 'gt',
    threshold: 80,
    severity: 'warning',
    enabled: 1,
    description: '',
    notify_user_ids: [],
    notify_channels: ['in_app'],
    action: 'notify_only',
    repair_task_id: null,
    ticket_type_id: null,
    oncall_schedule_id: null,
    service_tree_id: null,
    owner_id: null,
  }
  dialogVisible.value = true
}

function openEdit(row: any) {
  isEdit.value = true
  editId.value = row.id
  form.value = {
    name: row.name,
    metric_type: row.metric_type,
    operator: row.operator,
    threshold: Number(row.threshold || 0),
    severity: row.severity,
    enabled: Number(row.enabled ?? 1),
    description: row.description || '',
    notify_user_ids: row.notify_user_ids ? JSON.parse(row.notify_user_ids) : [],
    notify_channels: row.notify_channels ? JSON.parse(row.notify_channels) : ['in_app'],
    action: row.action || 'notify_only',
    repair_task_id: row.repair_task_id ? Number(row.repair_task_id) : null,
    ticket_type_id: row.ticket_type_id ? Number(row.ticket_type_id) : null,
    oncall_schedule_id: row.oncall_schedule_id ? Number(row.oncall_schedule_id) : null,
    service_tree_id: row.service_tree_id ? Number(row.service_tree_id) : null,
    owner_id: row.owner_id ? Number(row.owner_id) : null,
  }
  dialogVisible.value = true
}

async function submitForm() {
  if (!form.value.name?.trim()) {
    ElMessage.warning('请填写规则名称')
    return
  }
  submitting.value = true
  try {
    const payload = {
      ...form.value,
      name: form.value.name.trim(),
      repair_task_id: Number(form.value.repair_task_id || 0),
      ticket_type_id: Number(form.value.ticket_type_id || 0),
      oncall_schedule_id: Number(form.value.oncall_schedule_id || 0),
      service_tree_id: Number(form.value.service_tree_id || 0),
      owner_id: Number(form.value.owner_id || 0),
    }
    if (isEdit.value) {
      await alertRuleApi.update(editId.value, payload)
    } else {
      await alertRuleApi.create(payload)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    await fetchRules(true)
  } finally {
    submitting.value = false
  }
}

async function removeRule(row: any) {
  try {
    await ElMessageBox.confirm(`确定删除规则「${row.name}」？`, '提示', { type: 'warning' })
    await alertRuleApi.delete(row.id)
    ElMessage.success('删除成功')
    await fetchRules(true)
  } catch {}
}

async function toggleEnabled(row: any, enabled: number) {
  try {
    await alertRuleApi.update(row.id, {
      name: row.name,
      metric_type: row.metric_type,
      operator: row.operator,
      threshold: row.threshold,
      severity: row.severity,
      enabled,
      description: row.description,
      notify_user_ids: row.notify_user_ids ? JSON.parse(row.notify_user_ids) : [],
      notify_channels: row.notify_channels ? JSON.parse(row.notify_channels) : [],
      action: row.action,
      repair_task_id: row.repair_task_id,
      ticket_type_id: row.ticket_type_id,
      oncall_schedule_id: row.oncall_schedule_id,
      service_tree_id: row.service_tree_id,
      owner_id: row.owner_id,
    })
    ElMessage.success(enabled === 1 ? '规则已启用' : '规则已停用')
    await fetchRules(false)
  } catch {
    row.enabled = row.enabled === 1 ? 0 : 1
  }
}

async function runEvaluate() {
  loading.value = true
  try {
    const res = await alertRuleApi.evaluate()
    const data = (res as any).data || {}
    ElMessage.success(`巡检完成：新触发 ${data.triggered_count || 0}，已恢复 ${data.resolved_count || 0}`)
    await fetchData(false)
  } finally {
    loading.value = false
  }
}

async function acknowledgeEvent(row: any) {
  try {
    const { value } = await ElMessageBox.prompt('可填写确认备注', '确认告警', {
      inputPlaceholder: '例如：已知问题，待窗口期处理',
      confirmButtonText: '确认',
      cancelButtonText: '取消',
    })
    await alertRuleApi.ackEvent(row.id, value || '')
    ElMessage.success('事件已确认')
    await fetchEvents(false)
  } catch {}
}

async function resolveEvent(row: any) {
  try {
    const { value } = await ElMessageBox.prompt('请填写恢复说明', '关闭告警', {
      inputPlaceholder: '例如：服务已恢复，告警关闭',
      confirmButtonText: '关闭事件',
      cancelButtonText: '取消',
    })
    await alertRuleApi.resolveEvent(row.id, value || '')
    ElMessage.success('事件已关闭')
    await fetchEvents(false)
  } catch {}
}

function applyRuleFilters() {
  rulePager.value.page = 1
  fetchRules(true)
}

function resetRuleFilters() {
  filters.value = {
    keyword: '',
    metric_type: '',
    severity: '',
    enabled: '',
  }
  rulePager.value.page = 1
  fetchRules(true)
}

function applyEventFilters() {
  eventPager.value.page = 1
  fetchEvents(true)
}

function resetEventFilters() {
  eventFilters.value = {
    keyword: '',
    status: '',
    severity: '',
  }
  eventPager.value.page = 1
  fetchEvents(true)
}

function handleRulePageChange(page: number) {
  rulePager.value.page = page
  fetchRules(true)
}

function handleRuleSizeChange(size: number) {
  rulePager.value.size = size
  rulePager.value.page = 1
  fetchRules(true)
}

function handleEventPageChange(page: number) {
  eventPager.value.page = page
  fetchEvents(true)
}

function handleEventSizeChange(size: number) {
  eventPager.value.size = size
  eventPager.value.page = 1
  fetchEvents(true)
}

function setupRefreshTimer() {
  if (refreshTimer) {
    window.clearInterval(refreshTimer)
    refreshTimer = null
  }
  if (!autoRefresh.value) {
    return
  }
  refreshTimer = window.setInterval(() => {
    fetchEvents(false)
    fetchRules(false)
  }, 30000)
}

function metricLabel(metricType: string) {
  const match = metricOptions.find(item => item.value === metricType)
  return match?.label || metricType
}

function severityTagType(severity: string) {
  const map: Record<string, string> = {
    info: 'info',
    warning: 'warning',
    critical: 'danger',
  }
  return map[severity] || 'info'
}

function severityLabel(severity: string) {
  const match = severityOptions.find(item => item.value === severity)
  return match?.label || severity
}

function statusTagType(status: string) {
  const map: Record<string, string> = {
    firing: 'danger',
    acknowledged: 'warning',
    resolved: 'info',
  }
  return map[status] || 'info'
}

function statusLabel(status: string) {
  const match = eventStatusOptions.find(item => item.value === status)
  return match?.label || status
}

function actionLabel(action: string) {
  const map: Record<string, string> = {
    notify_only: '仅通知',
    create_ticket: '自动建单',
    execute_task: '自动修复',
  }
  return map[action] || action
}

const userOptions = computed(() => users.value.map(item => ({
  label: item.real_name || item.username,
  value: item.id,
})))

function treeLabel(id: number) {
  if (!id) return '—'
  return serviceTrees.value.find(item => item.id === id)?.label || `#${id}`
}

function ownerLabel(id: number) {
  if (!id) return '—'
  return userOptions.value.find(item => item.value === id)?.label || `#${id}`
}

function oncallLabel(id: number) {
  if (!id) return '—'
  return oncallSchedules.value.find(item => item.id === id)?.name || `#${id}`
}

function goAlertEvents() {
  router.push('/monitor/alerts')
}

function goDatasourcePage() {
  router.push('/monitor/datasources')
}

function goQueryPage() {
  router.push('/monitor/query')
}

function goSilencePage() {
  router.push('/monitor/silences')
}

function goOnCallPage() {
  router.push('/monitor/oncall')
}

function flattenTree(nodes: any[], level = 0): any[] {
  const result: any[] = []
  nodes.forEach((node) => {
    result.push({
      id: node.id,
      label: `${'　'.repeat(level)}${level > 0 ? '└ ' : ''}${node.name}`,
      rawName: node.name,
    })
    if (Array.isArray(node.children) && node.children.length > 0) {
      result.push(...flattenTree(node.children, level + 1))
    }
  })
  return result
}

onMounted(async () => {
  await fetchData(true)
  setupRefreshTimer()
})

onUnmounted(() => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer)
    refreshTimer = null
  }
})
</script>

<template>
  <div class="alert-page">
    <el-card shadow="never" class="section-card">
      <template #header>
        <div class="section-head">
          <div>
            <div class="section-title">告警规则</div>
            <div class="section-subtitle">配置阈值规则、手动触发巡检，并管理通知对象。</div>
          </div>
          <div class="section-actions">
            <el-switch
              v-model="autoRefresh"
              inline-prompt
              active-text="自动刷新"
              inactive-text="手动"
              @change="setupRefreshTimer"
            />
            <el-button plain @click="fetchData(true)">
              <el-icon><RefreshRight /></el-icon>
              刷新
            </el-button>
            <el-button type="warning" plain @click="runEvaluate">
              <el-icon><Bell /></el-icon>
              立即巡检
            </el-button>
            <el-button plain @click="goAlertEvents">事件中心</el-button>
            <el-button plain @click="goSilencePage">告警静默</el-button>
            <el-button plain @click="goOnCallPage">OnCall</el-button>
            <el-button plain @click="goDatasourcePage">数据源</el-button>
            <el-button plain @click="goQueryPage">查询台</el-button>
            <el-button type="primary" @click="openAdd">
              <el-icon><Plus /></el-icon>
              新增规则
            </el-button>
          </div>
        </div>
      </template>

      <el-form inline class="filter-row">
        <el-form-item>
          <el-input v-model="filters.keyword" placeholder="搜索规则名称" clearable style="width: 220px" @keyup.enter="applyRuleFilters" />
        </el-form-item>
        <el-form-item>
          <el-select v-model="filters.metric_type" placeholder="监控项" clearable style="width: 160px">
            <el-option v-for="item in metricOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="filters.severity" placeholder="级别" clearable style="width: 140px">
            <el-option v-for="item in severityOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="filters.enabled" placeholder="启用状态" clearable style="width: 140px">
            <el-option v-for="item in enabledOptions" :key="`${item.label}-${item.value}`" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="applyRuleFilters">筛选</el-button>
          <el-button @click="resetRuleFilters">重置</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="tableData" stripe border v-loading="loading">
        <el-table-column prop="name" label="规则名称" min-width="220" show-overflow-tooltip />
        <el-table-column label="监控项" width="140">
          <template #default="{ row }">{{ metricLabel(row.metric_type) }}</template>
        </el-table-column>
        <el-table-column label="条件" width="120">
          <template #default="{ row }">{{ row.operator }} {{ row.threshold }}</template>
        </el-table-column>
        <el-table-column label="级别" width="100">
          <template #default="{ row }">
            <el-tag :type="severityTagType(row.severity)">{{ severityLabel(row.severity) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="通知成员" min-width="200">
          <template #default="{ row }">
            <span>{{ row.notify_user_ids ? JSON.parse(row.notify_user_ids).length : 0 }} 人</span>
          </template>
        </el-table-column>
        <el-table-column label="通知渠道" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            <span>{{ row.notify_channels ? JSON.parse(row.notify_channels).join(' / ') : '默认' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="动作" width="120">
          <template #default="{ row }">{{ actionLabel(row.action) }}</template>
        </el-table-column>
        <el-table-column label="服务树" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">{{ treeLabel(Number(row.service_tree_id || 0)) }}</template>
        </el-table-column>
        <el-table-column label="负责人" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">{{ ownerLabel(Number(row.owner_id || 0)) }}</template>
        </el-table-column>
        <el-table-column label="OnCall" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">{{ oncallLabel(Number(row.oncall_schedule_id || 0)) }}</template>
        </el-table-column>
        <el-table-column label="启用" width="90">
          <template #default="{ row }">
            <el-switch
              :model-value="row.enabled === 1"
              @change="(value: boolean) => toggleEnabled(row, value ? 1 : 0)"
            />
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" width="180" />
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="removeRule(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="table-footer">
        <el-pagination
          background
          layout="total, sizes, prev, pager, next"
          :total="rulePager.total"
          :current-page="rulePager.page"
          :page-size="rulePager.size"
          :page-sizes="[10, 20, 50]"
          @current-change="handleRulePageChange"
          @size-change="handleRuleSizeChange"
        />
      </div>
    </el-card>

    <el-card shadow="never" class="section-card" style="margin-top: 16px;">
      <template #header>
        <div class="section-head">
          <div>
            <div class="section-title">告警事件</div>
            <div class="section-subtitle">支持确认、关闭和按状态/级别/关键字筛选，方便值守时快速处理。</div>
          </div>
        </div>
      </template>

      <el-form inline class="filter-row">
        <el-form-item>
          <el-input v-model="eventFilters.keyword" placeholder="搜索规则 / 主机 / IP" clearable style="width: 220px" @keyup.enter="applyEventFilters" />
        </el-form-item>
        <el-form-item>
          <el-select v-model="eventFilters.status" placeholder="事件状态" clearable style="width: 140px">
            <el-option v-for="item in eventStatusOptions" :key="`${item.label}-${item.value}`" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="eventFilters.severity" placeholder="事件级别" clearable style="width: 140px">
            <el-option v-for="item in severityOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="applyEventFilters">筛选</el-button>
          <el-button @click="resetEventFilters">重置</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="events" stripe border v-loading="eventLoading">
        <el-table-column prop="rule_name" label="规则" min-width="180" show-overflow-tooltip />
        <el-table-column prop="hostname" label="主机" min-width="160" show-overflow-tooltip />
        <el-table-column label="监控项" width="140">
          <template #default="{ row }">{{ metricLabel(row.metric_type) }}</template>
        </el-table-column>
        <el-table-column label="当前值/阈值" width="150">
          <template #default="{ row }">{{ Number(row.metric_value || 0).toFixed(1) }} / {{ row.threshold }}</template>
        </el-table-column>
        <el-table-column label="级别" width="100">
          <template #default="{ row }">
            <el-tag :type="severityTagType(row.severity)">{{ severityLabel(row.severity) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="triggered_at" label="触发时间" width="180" />
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'firing'"
              link
              type="warning"
              @click="acknowledgeEvent(row)"
            >
              确认
            </el-button>
            <el-button
              v-if="row.status === 'firing' || row.status === 'acknowledged'"
              link
              type="danger"
              @click="resolveEvent(row)"
            >
              关闭
            </el-button>
            <span v-if="row.status === 'resolved'" class="resolved-tip">已恢复</span>
          </template>
        </el-table-column>
      </el-table>

      <div class="table-footer">
        <el-pagination
          background
          layout="total, sizes, prev, pager, next"
          :total="eventPager.total"
          :current-page="eventPager.page"
          :page-size="eventPager.size"
          :page-sizes="[10, 20, 50]"
          @current-change="handleEventPageChange"
          @size-change="handleEventSizeChange"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑告警规则' : '新增告警规则'" width="620px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="规则名称" required>
          <el-input v-model="form.name" placeholder="例如：CPU 使用率持续过高" />
        </el-form-item>
        <el-form-item label="监控项">
          <el-select v-model="form.metric_type" style="width: 100%;">
            <el-option v-for="item in metricOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="比较方式">
          <el-select v-model="form.operator" style="width: 100%;">
            <el-option v-for="item in operatorOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="阈值">
          <el-input-number v-model="form.threshold" :min="0" :max="100" :step="1" />
        </el-form-item>
        <el-form-item label="告警级别">
          <el-select v-model="form.severity" style="width: 100%;">
            <el-option v-for="item in severityOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="通知成员">
          <el-select v-model="form.notify_user_ids" multiple clearable filterable style="width: 100%;">
            <el-option
              v-for="item in userOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="通知渠道">
          <el-select v-model="form.notify_channels" multiple clearable filterable style="width: 100%;">
            <el-option
              v-for="item in notifyChannelOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="服务树范围">
          <el-select v-model="form.service_tree_id" clearable filterable style="width: 100%;" placeholder="不限制">
            <el-option
              v-for="item in serviceTrees"
              :key="item.id"
              :label="item.label"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="负责人范围">
          <el-select v-model="form.owner_id" clearable filterable style="width: 100%;" placeholder="不限制">
            <el-option
              v-for="item in userOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="OnCall 值班">
          <el-select v-model="form.oncall_schedule_id" clearable filterable style="width: 100%;" placeholder="不启用值班升级">
            <el-option
              v-for="item in oncallSchedules"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="触发动作">
          <el-select v-model="form.action" style="width: 100%;">
            <el-option label="仅通知" value="notify_only" />
            <el-option label="自动建单" value="create_ticket" />
            <el-option label="自动修复" value="execute_task" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.action === 'create_ticket'" label="工单类型ID">
          <el-input-number v-model="form.ticket_type_id" :min="0" />
        </el-form-item>
        <el-form-item v-if="form.action === 'execute_task'" label="修复任务">
          <el-select v-model="form.repair_task_id" clearable filterable style="width: 100%;" placeholder="选择修复任务">
            <el-option
              v-for="item in tasks"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="是否启用">
          <el-switch v-model="form.enabled" :active-value="1" :inactive-value="0" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="4" placeholder="可填写触发背景、值守说明或排查建议" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.alert-page {
  padding: 20px;
  background:
    linear-gradient(180deg, rgba(249, 250, 251, 0.96), rgba(241, 245, 249, 0.92)),
    #f8fafc;
  min-height: 100%;
}

.section-card {
  border: 1px solid #e7ecf3;
  border-radius: 18px;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.04);
}

.section-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  flex-wrap: wrap;
}

.section-title {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
}

.section-subtitle {
  margin-top: 8px;
  font-size: 13px;
  color: #64748b;
}

.section-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.filter-row {
  margin-bottom: 12px;
}

.table-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.resolved-tip {
  font-size: 12px;
  color: #64748b;
}

@media (max-width: 960px) {
  .section-head {
    flex-direction: column;
  }
}
</style>
