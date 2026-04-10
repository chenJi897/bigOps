<script setup lang="ts">
defineOptions({ name: 'AlertEvents' })

import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { alertRuleApi } from '../api'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const events = ref<any[]>([])
const groups = ref<any[]>([])
const selectedRows = ref<any[]>([])
const pager = ref({ page: 1, size: 20, total: 0 })
const autoRefresh = ref(true)
const viewMode = ref<'events' | 'groups'>('events')
const groupWindowMinutes = ref(5)
const timelineVisible = ref(false)
const timelineLoading = ref(false)
const timelineData = ref<any>(null)
const rootCauseVisible = ref(false)
const rootCauseLoading = ref(false)
const rootCauseData = ref<any>(null)
const contextLoading = ref(false)
const contextData = ref<any>(null)
const topologyVisible = ref(false)
const topologyLoading = ref(false)
const topologyData = ref<any[]>([])
const detailVisible = ref(false)
const detailData = ref<any>(null)
const detailLoading = ref(false)
let refreshTimer: number | null = null

const filters = ref({
  keyword: String(route.query.keyword || ''),
  status: String(route.query.status || ''),
  severity: String(route.query.severity || ''),
})

const stats = computed(() => {
  const all = events.value
  return {
    total: pager.value.total,
    firing: all.filter(e => e.status === 'firing').length,
    acknowledged: all.filter(e => e.status === 'acknowledged').length,
    resolved: all.filter(e => e.status === 'resolved').length,
    suppressed: all.filter(e => e.status === 'suppressed').length,
  }
})

const severityMap: Record<string, { label: string; color: string; tagType: string }> = {
  info: { label: '提示', color: '#909399', tagType: 'info' },
  warning: { label: '警告', color: '#E6A23C', tagType: 'warning' },
  critical: { label: '严重', color: '#F56C6C', tagType: 'danger' },
}

const statusMap: Record<string, { label: string; dot: string; tagType: string }> = {
  firing: { label: '触发中', dot: 'bg-red-500', tagType: 'danger' },
  acknowledged: { label: '已确认', dot: 'bg-amber-500', tagType: 'warning' },
  resolved: { label: '已恢复', dot: 'bg-green-500', tagType: 'success' },
  suppressed: { label: '已抑制', dot: 'bg-slate-400', tagType: 'info' },
}

const timelineTypeMap: Record<string, { label: string; color: string; icon: string }> = {
  triggered: { label: '触发', color: 'danger', icon: '🔴' },
  acknowledged: { label: '确认', color: 'warning', icon: '🟡' },
  resolved: { label: '恢复', color: 'success', icon: '🟢' },
  related_triggered: { label: '关联触发', color: 'danger', icon: '🔗' },
  related_resolved: { label: '关联恢复', color: 'success', icon: '🔗' },
  activity: { label: '活动', color: 'primary', icon: '📝' },
  escalation: { label: '升级', color: 'danger', icon: '⬆️' },
}

const canBatchAck = computed(() => selectedRows.value.some(e => e.status === 'firing'))
const canBatchResolve = computed(() => selectedRows.value.some(e => e.status !== 'resolved'))

async function fetchEvents(silent = false) {
  if (!silent) loading.value = true
  try {
    let total = 0
    if (viewMode.value === 'groups') {
      const res = await alertRuleApi.eventGroups({
        page: pager.value.page,
        size: pager.value.size,
        status: filters.value.status,
        severity: filters.value.severity,
        keyword: filters.value.keyword.trim(),
        window_minutes: groupWindowMinutes.value,
      })
      groups.value = (res as any).data?.list || []
      events.value = []
      total = Number((res as any).data?.total || 0)
    } else {
      const res = await alertRuleApi.events({
        page: pager.value.page,
        size: pager.value.size,
        status: filters.value.status,
        severity: filters.value.severity,
        keyword: filters.value.keyword.trim(),
      })
      events.value = (res as any).data?.list || []
      groups.value = []
      total = Number((res as any).data?.total || 0)
    }
    pager.value.total = total
  } finally {
    loading.value = false
  }
}

function setStatusFilter(status: string) {
  filters.value.status = filters.value.status === status ? '' : status
  pager.value.page = 1
  fetchEvents()
}

function applyFilters() { pager.value.page = 1; fetchEvents() }
function resetFilters() { filters.value = { keyword: '', status: '', severity: '' }; pager.value.page = 1; fetchEvents() }
function handleSelectionChange(rows: any[]) { selectedRows.value = rows }

async function acknowledge(row: any) {
  try {
    const { value } = await ElMessageBox.prompt('确认备注（可选）', '确认告警', { inputPlaceholder: '例如：已知问题，处理中' })
    await alertRuleApi.ackEvent(row.id, value || '')
    ElMessage.success('已确认')
    fetchEvents()
  } catch {}
}

async function resolve(row: any) {
  try {
    const { value } = await ElMessageBox.prompt('关闭说明（可选）', '关闭告警', { inputPlaceholder: '例如：已恢复' })
    await alertRuleApi.resolveEvent(row.id, value || '')
    ElMessage.success('已关闭')
    fetchEvents()
  } catch {}
}

async function commentEvent(row: any) {
  try {
    const { value } = await ElMessageBox.prompt('添加评论', '告警评论', { inputPlaceholder: '输入评论内容...', inputType: 'textarea' })
    if (!value) return
    await alertRuleApi.commentEvent(row.id, value)
    ElMessage.success('评论已添加')
  } catch {}
}

async function assignEvent(row: any) {
  try {
    const { value } = await ElMessageBox.prompt('输入指派用户 ID', '指派告警', { inputPlaceholder: '例如：1', inputPattern: /^\d+$/, inputErrorMessage: '请输入数字' })
    if (!value) return
    await alertRuleApi.assignEvent(row.id, Number(value))
    ElMessage.success('已指派')
    fetchEvents()
  } catch {}
}

async function openDetail(eventID: number) {
  detailVisible.value = true
  detailLoading.value = true
  try {
    const res = await alertRuleApi.getEvent(eventID)
    detailData.value = (res as any).data
  } finally {
    detailLoading.value = false
  }
}

async function openTopology(eventID: number) {
  topologyVisible.value = true
  topologyLoading.value = true
  try {
    const res = await alertRuleApi.eventTopology(eventID)
    topologyData.value = (res as any).data || []
  } catch { topologyData.value = [] }
  finally { topologyLoading.value = false }
}

async function batchAcknowledge() {
  if (!canBatchAck.value) return
  try {
    await ElMessageBox.confirm(`确认批量确认 ${selectedRows.value.filter(e => e.status === 'firing').length} 条告警？`, '批量确认')
    await Promise.all(selectedRows.value.filter(e => e.status === 'firing').map(e => alertRuleApi.ackEvent(e.id, '批量确认')))
    ElMessage.success('已批量确认')
    fetchEvents()
  } catch {}
}

async function batchResolve() {
  if (!canBatchResolve.value) return
  try {
    await ElMessageBox.confirm(`确认批量关闭 ${selectedRows.value.filter(e => e.status !== 'resolved').length} 条告警？`, '批量关闭')
    await Promise.all(selectedRows.value.filter(e => e.status !== 'resolved').map(e => alertRuleApi.resolveEvent(e.id, '批量关闭')))
    ElMessage.success('已批量关闭')
    fetchEvents()
  } catch {}
}

async function openTimelineByEventID(eventID: number) {
  timelineVisible.value = true
  timelineLoading.value = true
  try {
    const res = await alertRuleApi.eventTimeline(eventID)
    timelineData.value = (res as any).data
  } finally { timelineLoading.value = false }
}

async function openRootCauseByEventID(eventID: number) {
  rootCauseVisible.value = true
  rootCauseLoading.value = true
  contextLoading.value = true
  try {
    const [res, ctxRes] = await Promise.all([alertRuleApi.eventRootCause(eventID), alertRuleApi.eventContext(eventID)])
    rootCauseData.value = (res as any).data
    contextData.value = (ctxRes as any).data
  } finally { rootCauseLoading.value = false; contextLoading.value = false }
}

function goTicket(id: number) { if (id) router.push(`/tickets/${id}`) }
function goExecution(id: number) { if (id) router.push(`/task/executions/${id}`) }

function timelineItemType(t: string) { return timelineTypeMap[t]?.color || 'info' }
function timelineItemLabel(t: string) { return timelineTypeMap[t]?.label || t }

function setupTimer() {
  if (refreshTimer) { clearInterval(refreshTimer); refreshTimer = null }
  if (autoRefresh.value) { refreshTimer = window.setInterval(() => fetchEvents(true), 15000) }
}

watch(autoRefresh, setupTimer)
watch(viewMode, () => { pager.value.page = 1; selectedRows.value = []; fetchEvents() })
onMounted(() => { fetchEvents(); setupTimer() })
onUnmounted(() => { if (refreshTimer) clearInterval(refreshTimer) })
</script>

<template>
  <div class="bg-slate-50 -m-5 alert-events-root">
    <div class="bg-white border-b border-slate-200 px-6 py-3 flex items-center justify-between gap-4">
      <h1 class="text-lg font-bold text-slate-800 whitespace-nowrap">告警事件</h1>
      <div class="flex items-center gap-2 flex-wrap">
        <el-radio-group v-model="viewMode" size="small">
          <el-radio-button value="events">事件视图</el-radio-button>
          <el-radio-button value="groups">收敛分组</el-radio-button>
        </el-radio-group>
        <el-select v-if="viewMode === 'groups'" v-model="groupWindowMinutes" class="!w-32" size="small" @change="fetchEvents()">
          <el-option :value="1" label="1分钟窗口" />
          <el-option :value="5" label="5分钟窗口" />
          <el-option :value="10" label="10分钟窗口" />
          <el-option :value="30" label="30分钟窗口" />
          <el-option :value="60" label="60分钟窗口" />
        </el-select>
        <div class="flex items-center gap-1.5 bg-slate-100 px-2.5 py-1 rounded-lg">
          <span class="text-xs text-slate-500 whitespace-nowrap">自动刷新</span>
          <el-switch v-model="autoRefresh" size="small" />
        </div>
        <el-button plain size="small" :disabled="!canBatchAck" @click="batchAcknowledge">批量确认</el-button>
        <el-button plain size="small" type="warning" :disabled="!canBatchResolve" @click="batchResolve">批量关闭</el-button>
        <el-button size="small" type="primary" plain :loading="loading" @click="fetchEvents()">刷新</el-button>
      </div>
    </div>

    <div class="p-5">
      <!-- 统计卡片（增加 suppressed） -->
      <div class="grid grid-cols-5 gap-4 mb-5">
        <div class="bg-white rounded-xl border border-slate-200 px-4 py-3 cursor-pointer hover:shadow-md transition-shadow"
             :class="{ 'ring-2 ring-slate-400': !filters.status }" @click="setStatusFilter('')">
          <div class="text-xs text-slate-400 mb-1">全部事件</div>
          <div class="text-2xl font-bold text-slate-700">{{ stats.total }}</div>
        </div>
        <div class="bg-white rounded-xl border border-red-200 px-4 py-3 cursor-pointer hover:shadow-md transition-shadow"
             :class="{ 'ring-2 ring-red-400': filters.status === 'firing' }" @click="setStatusFilter('firing')">
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full bg-red-500 animate-pulse"></span>
            <span class="text-xs text-red-500">触发中</span>
          </div>
          <div class="text-2xl font-bold text-red-600 mt-1">{{ stats.firing }}</div>
        </div>
        <div class="bg-white rounded-xl border border-amber-200 px-4 py-3 cursor-pointer hover:shadow-md transition-shadow"
             :class="{ 'ring-2 ring-amber-400': filters.status === 'acknowledged' }" @click="setStatusFilter('acknowledged')">
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full bg-amber-500"></span>
            <span class="text-xs text-amber-600">已确认</span>
          </div>
          <div class="text-2xl font-bold text-amber-600 mt-1">{{ stats.acknowledged }}</div>
        </div>
        <div class="bg-white rounded-xl border border-green-200 px-4 py-3 cursor-pointer hover:shadow-md transition-shadow"
             :class="{ 'ring-2 ring-green-400': filters.status === 'resolved' }" @click="setStatusFilter('resolved')">
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full bg-green-500"></span>
            <span class="text-xs text-green-600">已恢复</span>
          </div>
          <div class="text-2xl font-bold text-green-600 mt-1">{{ stats.resolved }}</div>
        </div>
        <div class="bg-white rounded-xl border border-slate-200 px-4 py-3 cursor-pointer hover:shadow-md transition-shadow"
             :class="{ 'ring-2 ring-slate-400': filters.status === 'suppressed' }" @click="setStatusFilter('suppressed')">
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full bg-slate-400"></span>
            <span class="text-xs text-slate-500">已抑制</span>
          </div>
          <div class="text-2xl font-bold text-slate-500 mt-1">{{ stats.suppressed }}</div>
        </div>
      </div>

      <!-- 筛选栏 -->
      <div class="bg-white rounded-xl border border-slate-200 px-4 py-3 mb-4 flex items-center gap-3 flex-wrap">
        <el-input v-model="filters.keyword" placeholder="搜索规则 / 主机 / IP" clearable class="!w-52" size="small" @keyup.enter="applyFilters">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="filters.severity" placeholder="告警级别" clearable class="!w-28" size="small" @change="applyFilters">
          <el-option label="全部级别" value="" />
          <el-option label="提示" value="info" />
          <el-option label="警告" value="warning" />
          <el-option label="严重" value="critical" />
        </el-select>
        <el-select v-model="filters.status" placeholder="状态" clearable class="!w-28" size="small" @change="applyFilters">
          <el-option label="全部状态" value="" />
          <el-option label="触发中" value="firing" />
          <el-option label="已确认" value="acknowledged" />
          <el-option label="已恢复" value="resolved" />
          <el-option label="已抑制" value="suppressed" />
        </el-select>
        <el-button size="small" type="primary" @click="applyFilters">筛选</el-button>
        <el-button size="small" @click="resetFilters">重置</el-button>
        <div class="flex-1"></div>
        <span class="text-xs text-slate-400">共 {{ pager.total }} 条</span>
      </div>

      <!-- 事件列表 -->
      <div class="bg-white rounded-xl border border-slate-200 overflow-hidden" v-if="viewMode === 'events'">
        <el-table :data="events" v-loading="loading" stripe class="w-full" @selection-change="handleSelectionChange"
                  :row-class-name="(r: any) => r.row.status === 'firing' ? 'firing-row' : r.row.status === 'suppressed' ? 'suppressed-row' : ''">
          <el-table-column type="selection" width="40" align="center" />
          <el-table-column label="状态" width="90" align="center">
            <template #default="{ row }">
              <div class="flex items-center justify-center gap-1.5">
                <span class="w-2 h-2 rounded-full shrink-0" :class="statusMap[row.status]?.dot || 'bg-gray-400'"
                      :style="row.status === 'firing' ? 'animation: pulse 2s infinite' : ''"></span>
                <span class="text-xs font-medium" :class="{
                  'text-red-600': row.status === 'firing',
                  'text-amber-600': row.status === 'acknowledged',
                  'text-green-600': row.status === 'resolved',
                  'text-slate-400': row.status === 'suppressed',
                }">{{ statusMap[row.status]?.label || row.status }}</span>
              </div>
              <div v-if="row.escalated" class="text-[10px] text-red-400 mt-0.5">已升级</div>
            </template>
          </el-table-column>
          <el-table-column label="级别" width="70" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="severityMap[row.severity]?.tagType || 'info'" effect="dark" round>
                {{ severityMap[row.severity]?.label || row.severity }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="告警规则 / 主机" min-width="220" show-overflow-tooltip>
            <template #default="{ row }">
              <div class="font-medium text-slate-800 cursor-pointer hover:text-blue-600" @click="openDetail(row.id)">{{ row.rule_name }}</div>
              <div class="text-xs text-slate-400 mt-0.5">{{ row.hostname || row.agent_id || '—' }}
                <span v-if="row.ip" class="ml-1 text-slate-300">{{ row.ip }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="值 / 阈值" width="110" align="center">
            <template #default="{ row }">
              <span class="font-mono text-xs">
                <span :class="row.status === 'firing' ? 'text-red-600 font-bold' : 'text-slate-600'">{{ Number(row.metric_value || 0).toFixed(1) }}</span>
                <span class="text-slate-300 mx-0.5">/</span>
                <span class="text-slate-500">{{ row.threshold }}</span>
              </span>
            </template>
          </el-table-column>
          <el-table-column label="关联" width="90" align="center">
            <template #default="{ row }">
              <el-button v-if="row.ticket_id" link type="primary" size="small" @click="goTicket(row.ticket_id)" title="关联工单">
                工单#{{ row.ticket_id }}
              </el-button>
              <el-button v-if="row.task_execution_id" link type="warning" size="small" @click="goExecution(row.task_execution_id)" title="修复任务">
                任务#{{ row.task_execution_id }}
              </el-button>
              <span v-if="!row.ticket_id && !row.task_execution_id" class="text-xs text-slate-300">—</span>
            </template>
          </el-table-column>
          <el-table-column label="触发时间" width="150" align="center">
            <template #default="{ row }">
              <span class="text-xs text-slate-500">{{ row.triggered_at }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="260" fixed="right" align="center">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="openDetail(row.id)">详情</el-button>
              <template v-if="row.status === 'firing'">
                <el-button link type="primary" size="small" @click="acknowledge(row)">确认</el-button>
                <el-button link type="danger" size="small" @click="resolve(row)">关闭</el-button>
              </template>
              <template v-else-if="row.status === 'acknowledged'">
                <el-button link type="danger" size="small" @click="resolve(row)">关闭</el-button>
              </template>
              <el-button link size="small" @click="openTimelineByEventID(row.id)">时间轴</el-button>
              <el-button link type="warning" size="small" @click="openRootCauseByEventID(row.id)">根因</el-button>
              <el-button link type="info" size="small" @click="commentEvent(row)">评论</el-button>
              <el-button link size="small" @click="assignEvent(row)">指派</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="px-4 py-3 border-t border-slate-100 flex justify-end">
          <el-pagination background size="small" layout="total, sizes, prev, pager, next" :total="pager.total"
            :current-page="pager.page" :page-size="pager.size" :page-sizes="[20, 50, 100]"
            @current-change="(page: number) => { pager.page = page; fetchEvents() }"
            @size-change="(size: number) => { pager.size = size; pager.page = 1; fetchEvents() }" />
        </div>
      </div>

      <!-- 收敛分组 -->
      <div class="bg-white rounded-xl border border-slate-200 overflow-hidden" v-else>
        <el-table :data="groups" v-loading="loading" stripe class="w-full">
          <el-table-column label="状态" width="90" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="statusMap[row.status]?.tagType || 'info'">{{ statusMap[row.status]?.label || row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="规则" min-width="200" prop="rule_name" show-overflow-tooltip />
          <el-table-column label="级别" width="70" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="severityMap[row.severity]?.tagType || 'info'" effect="dark" round>{{ severityMap[row.severity]?.label || row.severity }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="聚合数" width="70" prop="total_count" align="center" />
          <el-table-column label="最新主机" width="130" prop="latest_host" show-overflow-tooltip />
          <el-table-column label="首触发" width="150">
            <template #default="{ row }">{{ row.first_triggered || '—' }}</template>
          </el-table-column>
          <el-table-column label="末触发" width="150">
            <template #default="{ row }">{{ row.last_triggered || '—' }}</template>
          </el-table-column>
          <el-table-column label="持续" width="80" align="right">
            <template #default="{ row }">
              <span v-if="row.duration_sec > 0">{{ row.duration_sec }}s</span>
              <span v-else class="text-slate-300">—</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120" fixed="right" align="center">
            <template #default="{ row }">
              <el-button link size="small" @click="openDetail(row.latest_event_id)">详情</el-button>
              <el-button link size="small" @click="openTimelineByEventID(row.latest_event_id)">时间轴</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="px-4 py-3 border-t border-slate-100 flex justify-end">
          <el-pagination background size="small" layout="total, sizes, prev, pager, next" :total="pager.total"
            :current-page="pager.page" :page-size="pager.size" :page-sizes="[20, 50, 100]"
            @current-change="(page: number) => { pager.page = page; fetchEvents() }"
            @size-change="(size: number) => { pager.size = size; pager.page = 1; fetchEvents() }" />
        </div>
      </div>
    </div>

    <!-- 事件详情 Drawer -->
    <el-drawer v-model="detailVisible" title="告警事件详情" size="45%" append-to-body>
      <div v-loading="detailLoading">
        <template v-if="detailData">
          <div class="flex items-center gap-2 mb-4">
            <el-tag :type="statusMap[detailData.status]?.tagType" effect="dark">{{ statusMap[detailData.status]?.label || detailData.status }}</el-tag>
            <el-tag :type="severityMap[detailData.severity]?.tagType" effect="dark" round>{{ severityMap[detailData.severity]?.label || detailData.severity }}</el-tag>
            <el-tag v-if="detailData.escalated" type="danger" effect="plain" size="small">已升级</el-tag>
          </div>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="事件ID">{{ detailData.id }}</el-descriptions-item>
            <el-descriptions-item label="规则">{{ detailData.rule_name }}</el-descriptions-item>
            <el-descriptions-item label="主机">{{ detailData.hostname || '—' }}</el-descriptions-item>
            <el-descriptions-item label="IP">{{ detailData.ip || '—' }}</el-descriptions-item>
            <el-descriptions-item label="Agent">{{ detailData.agent_id }}</el-descriptions-item>
            <el-descriptions-item label="指标">{{ detailData.metric_type }}</el-descriptions-item>
            <el-descriptions-item label="当前值">{{ Number(detailData.metric_value || 0).toFixed(2) }}</el-descriptions-item>
            <el-descriptions-item label="阈值">{{ detailData.threshold }} ({{ detailData.operator }})</el-descriptions-item>
            <el-descriptions-item label="触发时间">{{ detailData.triggered_at }}</el-descriptions-item>
            <el-descriptions-item label="恢复时间">{{ detailData.resolved_at || '—' }}</el-descriptions-item>
            <el-descriptions-item label="确认人">{{ detailData.acknowledged_by || '—' }}</el-descriptions-item>
            <el-descriptions-item label="确认时间">{{ detailData.acknowledged_at || '—' }}</el-descriptions-item>
            <el-descriptions-item label="确认备注" :span="2">{{ detailData.acknowledgement_note || '—' }}</el-descriptions-item>
            <el-descriptions-item label="处理说明" :span="2">{{ detailData.resolution_note || '—' }}</el-descriptions-item>
            <el-descriptions-item label="描述" :span="2">{{ detailData.description || '—' }}</el-descriptions-item>
          </el-descriptions>
          <div class="mt-4 flex gap-2 flex-wrap">
            <el-button v-if="detailData.ticket_id" type="primary" size="small" @click="goTicket(detailData.ticket_id)">查看关联工单 #{{ detailData.ticket_id }}</el-button>
            <el-button v-if="detailData.task_execution_id" type="warning" size="small" @click="goExecution(detailData.task_execution_id)">查看修复任务 #{{ detailData.task_execution_id }}</el-button>
            <el-button size="small" @click="openTimelineByEventID(detailData.id)">查看时间轴</el-button>
            <el-button size="small" @click="openRootCauseByEventID(detailData.id)">根因分析</el-button>
            <el-button size="small" @click="openTopology(detailData.id)">拓扑视图</el-button>
          </div>
        </template>
        <el-empty v-else description="加载中..." />
      </div>
    </el-drawer>

    <!-- 时间轴 Drawer（增强 activity 图标） -->
    <el-drawer v-model="timelineVisible" title="告警时间轴" size="36%" append-to-body>
      <div v-loading="timelineLoading">
        <template v-if="timelineData">
          <div class="mb-3 text-xs text-slate-500">规则：{{ timelineData.rule_name }}，关联事件数：{{ timelineData.related_count }}</div>
          <el-timeline>
            <el-timeline-item v-for="(item, idx) in timelineData.items || []" :key="idx"
              :timestamp="item.timestamp || '-'" :type="timelineItemType(item.type)">
              <div class="font-medium text-sm">
                <span class="mr-1">{{ timelineTypeMap[item.type]?.icon || '📌' }}</span>
                {{ timelineItemLabel(item.type) }}
              </div>
              <div class="text-xs text-slate-500 mt-0.5">{{ item.note || '—' }}</div>
              <div v-if="item.operator" class="text-xs text-slate-400 mt-0.5">操作人: #{{ item.operator }}</div>
            </el-timeline-item>
          </el-timeline>
        </template>
        <el-empty v-else description="暂无数据" />
      </div>
    </el-drawer>

    <!-- 根因分析 Drawer -->
    <el-drawer v-model="rootCauseVisible" title="根因分析" size="40%" append-to-body>
      <div v-loading="rootCauseLoading">
        <template v-if="rootCauseData">
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="事件ID">{{ rootCauseData.event_id }}</el-descriptions-item>
            <el-descriptions-item label="嫌疑模式">
              <el-tag size="small" :type="rootCauseData.confidence > 0.7 ? 'danger' : rootCauseData.confidence > 0.4 ? 'warning' : 'info'">
                {{ rootCauseData.primary_suspect }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="置信度">
              <el-progress :percentage="Math.round(rootCauseData.confidence * 100)" :stroke-width="12" :text-inside="true" class="!w-32" />
            </el-descriptions-item>
            <el-descriptions-item label="关联事件数">{{ rootCauseData.related_event_count }}</el-descriptions-item>
          </el-descriptions>
          <div class="mt-4 text-sm font-medium mb-2">证据链</div>
          <el-timeline>
            <el-timeline-item v-for="(ev, idx) in rootCauseData.evidence || []" :key="idx" size="small">
              <span class="text-xs text-slate-600">{{ ev }}</span>
            </el-timeline-item>
          </el-timeline>
          <div class="mt-3" v-loading="contextLoading">
            <div class="text-sm font-medium mb-2">处置建议</div>
            <div class="flex gap-2 flex-wrap mb-2">
              <el-button v-if="contextData?.task_execution_id" type="warning" size="small" @click="goExecution(contextData.task_execution_id)">修复任务 #{{ contextData.task_execution_id }}</el-button>
              <el-button v-if="contextData?.ticket_id" type="primary" size="small" @click="goTicket(contextData.ticket_id)">关联工单 #{{ contextData.ticket_id }}</el-button>
            </div>
            <ul class="text-xs text-slate-600 list-disc pl-5">
              <li v-for="(s, idx) in contextData?.suggestions || []" :key="idx">{{ s }}</li>
            </ul>
          </div>
        </template>
        <el-empty v-else description="暂无数据" />
      </div>
    </el-drawer>

    <!-- 拓扑视图 Drawer -->
    <el-drawer v-model="topologyVisible" title="告警关联拓扑" size="50%" append-to-body>
      <div v-loading="topologyLoading">
        <template v-if="topologyData.length">
          <div class="text-xs text-slate-500 mb-3">同服务树下的主机健康状况</div>
          <el-table :data="topologyData" size="small" stripe>
            <el-table-column label="主机" min-width="140" show-overflow-tooltip>
              <template #default="{ row }">{{ row.hostname || row.agent_id }}</template>
            </el-table-column>
            <el-table-column prop="ip" label="IP" width="120" />
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="row.status === 'online' ? 'success' : 'danger'" size="small">{{ row.status === 'online' ? '在线' : '离线' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="CPU" width="70" align="right">
              <template #default="{ row }"><span :class="row.cpu_pct > 80 ? 'text-red-500 font-medium' : ''">{{ Number(row.cpu_pct || 0).toFixed(1) }}%</span></template>
            </el-table-column>
            <el-table-column label="内存" width="70" align="right">
              <template #default="{ row }"><span :class="row.mem_pct > 80 ? 'text-red-500 font-medium' : ''">{{ Number(row.mem_pct || 0).toFixed(1) }}%</span></template>
            </el-table-column>
            <el-table-column label="磁盘" width="70" align="right">
              <template #default="{ row }"><span :class="row.disk_pct > 80 ? 'text-red-500 font-medium' : ''">{{ Number(row.disk_pct || 0).toFixed(1) }}%</span></template>
            </el-table-column>
            <el-table-column label="告警" width="60" align="center">
              <template #default="{ row }">
                <el-tag v-if="row.alert_count > 0" type="danger" size="small">{{ row.alert_count }}</el-tag>
                <span v-else class="text-slate-400">0</span>
              </template>
            </el-table-column>
          </el-table>
        </template>
        <el-empty v-else description="该告警未关联服务树或暂无关联主机" />
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
:deep(.firing-row) { background-color: #fef2f2 !important; }
:deep(.firing-row:hover > td) { background-color: #fee2e2 !important; }
:deep(.suppressed-row) { background-color: #f8fafc !important; opacity: 0.7; }
@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
</style>
