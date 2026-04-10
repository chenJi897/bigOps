<script setup lang="ts">
defineOptions({ name: 'AlertEvents' })

import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { alertRuleApi } from '../api'

const route = useRoute()
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
let refreshTimer: number | null = null

const filters = ref({
  keyword: String(route.query.keyword || ''),
  status: String(route.query.status || ''),
  severity: String(route.query.severity || ''),
})

// 统计
const stats = computed(() => {
  const all = events.value
  return {
    total: pager.value.total,
    firing: all.filter(e => e.status === 'firing').length,
    acknowledged: all.filter(e => e.status === 'acknowledged').length,
    resolved: all.filter(e => e.status === 'resolved').length,
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

function applyFilters() {
  pager.value.page = 1
  fetchEvents()
}

function resetFilters() {
  filters.value = { keyword: '', status: '', severity: '' }
  pager.value.page = 1
  fetchEvents()
}

function handleSelectionChange(rows: any[]) {
  selectedRows.value = rows
}

async function acknowledge(row: any) {
  try {
    const { value } = await ElMessageBox.prompt('确认备注（可选）', '确认告警', {
      inputPlaceholder: '例如：已知问题，处理中',
      confirmButtonText: '确认',
      cancelButtonText: '取消',
    })
    await alertRuleApi.ackEvent(row.id, value || '')
    ElMessage.success('已确认')
    fetchEvents()
  } catch {}
}

async function resolve(row: any) {
  try {
    const { value } = await ElMessageBox.prompt('关闭说明（可选）', '关闭告警', {
      inputPlaceholder: '例如：已恢复',
      confirmButtonText: '关闭',
      cancelButtonText: '取消',
    })
    await alertRuleApi.resolveEvent(row.id, value || '')
    ElMessage.success('已关闭')
    fetchEvents()
  } catch {}
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
  } finally {
    timelineLoading.value = false
  }
}

async function openRootCauseByEventID(eventID: number) {
  rootCauseVisible.value = true
  rootCauseLoading.value = true
  contextLoading.value = true
  try {
    const [res, ctxRes] = await Promise.all([
      alertRuleApi.eventRootCause(eventID),
      alertRuleApi.eventContext(eventID),
    ])
    rootCauseData.value = (res as any).data
    contextData.value = (ctxRes as any).data
  } finally {
    rootCauseLoading.value = false
    contextLoading.value = false
  }
}

function setupTimer() {
  if (refreshTimer) { clearInterval(refreshTimer); refreshTimer = null }
  if (autoRefresh.value) {
    refreshTimer = window.setInterval(() => fetchEvents(true), 15000)
  }
}

watch(autoRefresh, setupTimer)
watch(viewMode, () => {
  pager.value.page = 1
  selectedRows.value = []
  fetchEvents()
})
onMounted(() => { fetchEvents(); setupTimer() })
onUnmounted(() => { if (refreshTimer) clearInterval(refreshTimer) })
</script>

<template>
  <div class="bg-slate-50 -m-5 alert-events-root">
    <!-- 顶部：标题 + 操作 -->
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
      <!-- 统计卡片 -->
      <div class="grid grid-cols-4 gap-4 mb-5">
        <div class="bg-white rounded-xl border border-slate-200 px-4 py-3 cursor-pointer hover:shadow-md transition-shadow"
             :class="{ 'ring-2 ring-slate-400': !filters.status }"
             @click="setStatusFilter('')">
          <div class="text-xs text-slate-400 mb-1">全部事件</div>
          <div class="text-2xl font-bold text-slate-700">{{ stats.total }}</div>
        </div>
        <div class="bg-white rounded-xl border border-red-200 px-4 py-3 cursor-pointer hover:shadow-md transition-shadow"
             :class="{ 'ring-2 ring-red-400': filters.status === 'firing' }"
             @click="setStatusFilter('firing')">
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full bg-red-500 animate-pulse"></span>
            <span class="text-xs text-red-500">触发中</span>
          </div>
          <div class="text-2xl font-bold text-red-600 mt-1">{{ stats.firing }}</div>
        </div>
        <div class="bg-white rounded-xl border border-amber-200 px-4 py-3 cursor-pointer hover:shadow-md transition-shadow"
             :class="{ 'ring-2 ring-amber-400': filters.status === 'acknowledged' }"
             @click="setStatusFilter('acknowledged')">
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full bg-amber-500"></span>
            <span class="text-xs text-amber-600">已确认</span>
          </div>
          <div class="text-2xl font-bold text-amber-600 mt-1">{{ stats.acknowledged }}</div>
        </div>
        <div class="bg-white rounded-xl border border-green-200 px-4 py-3 cursor-pointer hover:shadow-md transition-shadow"
             :class="{ 'ring-2 ring-green-400': filters.status === 'resolved' }"
             @click="setStatusFilter('resolved')">
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full bg-green-500"></span>
            <span class="text-xs text-green-600">已恢复</span>
          </div>
          <div class="text-2xl font-bold text-green-600 mt-1">{{ stats.resolved }}</div>
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
        <el-button size="small" type="primary" @click="applyFilters">筛选</el-button>
        <el-button size="small" @click="resetFilters">重置</el-button>
        <div class="flex-1"></div>
        <span class="text-xs text-slate-400">共 {{ pager.total }} 条</span>
      </div>

      <!-- 事件列表 -->
      <div class="bg-white rounded-xl border border-slate-200 overflow-hidden" v-if="viewMode === 'events'">
        <el-table :data="events" v-loading="loading" stripe class="w-full" @selection-change="handleSelectionChange" :row-class-name="(r: any) => r.row.status === 'firing' ? 'firing-row' : ''">
          <el-table-column type="selection" width="40" align="center" />
          <el-table-column label="状态" width="90" align="center">
            <template #default="{ row }">
              <div class="flex items-center justify-center gap-1.5">
                <span class="w-2 h-2 rounded-full shrink-0" :class="statusMap[row.status]?.dot || 'bg-gray-400'" :style="row.status === 'firing' ? 'animation: pulse 2s infinite' : ''"></span>
                <span class="text-xs font-medium" :class="{ 'text-red-600': row.status === 'firing', 'text-amber-600': row.status === 'acknowledged', 'text-green-600': row.status === 'resolved' }">
                  {{ statusMap[row.status]?.label || row.status }}
                </span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="级别" width="70" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="severityMap[row.severity]?.tagType || 'info'" effect="dark" round>
                {{ severityMap[row.severity]?.label || row.severity }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="告警规则 / 主机" min-width="240" show-overflow-tooltip>
            <template #default="{ row }">
              <div>
                <div class="font-medium text-slate-800">{{ row.rule_name }}</div>
                <div class="text-xs text-slate-400 mt-0.5">{{ row.hostname || row.agent_id || '—' }}</div>
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
          <el-table-column label="触发时间" width="155" align="center">
            <template #default="{ row }">
              <span class="text-xs text-slate-500">{{ row.triggered_at }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right" align="center">
            <template #default="{ row }">
              <template v-if="row.status === 'firing'">
                <el-button link type="primary" size="small" @click="acknowledge(row)">确认</el-button>
                <el-button link type="danger" size="small" @click="resolve(row)">关闭</el-button>
              </template>
              <template v-else-if="row.status === 'acknowledged'">
                <el-button link type="danger" size="small" @click="resolve(row)">关闭</el-button>
              </template>
              <template v-else>
                <el-button link size="small" @click="openTimelineByEventID(row.id)">时间轴</el-button>
                <el-button link type="warning" size="small" @click="openRootCauseByEventID(row.id)">根因</el-button>
              </template>
            </template>
          </el-table-column>
        </el-table>

        <div class="px-4 py-3 border-t border-slate-100 flex justify-end">
          <el-pagination
            background
            size="small"
            layout="total, sizes, prev, pager, next"
            :total="pager.total"
            :current-page="pager.page"
            :page-size="pager.size"
            :page-sizes="[20, 50, 100]"
            @current-change="(page: number) => { pager.page = page; fetchEvents() }"
            @size-change="(size: number) => { pager.size = size; pager.page = 1; fetchEvents() }"
          />
        </div>
      </div>

      <div class="bg-white rounded-xl border border-slate-200 overflow-hidden" v-else>
        <el-table :data="groups" v-loading="loading" stripe class="w-full">
          <el-table-column label="状态" width="110" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="statusMap[row.status]?.tagType || 'info'">{{ statusMap[row.status]?.label || row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="规则" min-width="220" prop="rule_name" show-overflow-tooltip />
          <el-table-column label="级别" width="80" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="severityMap[row.severity]?.tagType || 'info'" effect="dark" round>
                {{ severityMap[row.severity]?.label || row.severity }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="聚合数" width="90" prop="total_count" align="center" />
          <el-table-column label="最新主机" width="160" prop="latest_host" show-overflow-tooltip />
          <el-table-column label="首触发" width="160" prop="first_triggered" />
          <el-table-column label="末触发" width="160" prop="last_triggered" />
          <el-table-column label="时长(s)" width="90" prop="duration_sec" align="right" />
          <el-table-column label="操作" width="100" fixed="right" align="center">
            <template #default="{ row }">
              <el-button link size="small" @click="openTimelineByEventID(row.latest_event_id)">时间轴</el-button>
              <el-button link type="warning" size="small" @click="openRootCauseByEventID(row.latest_event_id)">根因</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="px-4 py-3 border-t border-slate-100 flex justify-end">
          <el-pagination
            background
            size="small"
            layout="total, sizes, prev, pager, next"
            :total="pager.total"
            :current-page="pager.page"
            :page-size="pager.size"
            :page-sizes="[20, 50, 100]"
            @current-change="(page: number) => { pager.page = page; fetchEvents() }"
            @size-change="(size: number) => { pager.size = size; pager.page = 1; fetchEvents() }"
          />
        </div>
      </div>
    </div>

  <el-drawer v-model="timelineVisible" title="告警时间轴" size="36%" append-to-body>
    <div v-loading="timelineLoading">
      <template v-if="timelineData">
        <div class="mb-3 text-xs text-slate-500">规则：{{ timelineData.rule_name }}，关联事件数：{{ timelineData.related_count }}</div>
        <el-timeline>
          <el-timeline-item
            v-for="(item, idx) in timelineData.items || []"
            :key="idx"
            :timestamp="item.timestamp || '-'"
            :type="item.type === 'resolved' ? 'success' : item.type === 'acknowledged' ? 'warning' : 'danger'"
          >
            <div class="font-medium">{{ item.type }}</div>
            <div class="text-xs text-slate-500">{{ item.note || '—' }}</div>
          </el-timeline-item>
        </el-timeline>
      </template>
      <el-empty v-else description="暂无数据" />
    </div>
  </el-drawer>

  <el-drawer v-model="rootCauseVisible" title="根因分析" size="36%" append-to-body>
    <div v-loading="rootCauseLoading">
      <template v-if="rootCauseData">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="事件ID">{{ rootCauseData.event_id }}</el-descriptions-item>
          <el-descriptions-item label="嫌疑模式">{{ rootCauseData.primary_suspect }}</el-descriptions-item>
          <el-descriptions-item label="置信度">{{ Number(rootCauseData.confidence || 0).toFixed(2) }}</el-descriptions-item>
          <el-descriptions-item label="关联事件数">{{ rootCauseData.related_event_count }}</el-descriptions-item>
          <el-descriptions-item label="服务树">{{ rootCauseData.service_tree_id || '-' }}</el-descriptions-item>
          <el-descriptions-item label="负责人">{{ rootCauseData.owner_id || '-' }}</el-descriptions-item>
        </el-descriptions>
        <div class="mt-3 text-sm font-medium">证据链</div>
        <el-timeline class="mt-2">
          <el-timeline-item v-for="(ev, idx) in rootCauseData.evidence || []" :key="idx">
            <span class="text-xs text-slate-600">{{ ev }}</span>
          </el-timeline-item>
        </el-timeline>
        <div class="mt-3" v-loading="contextLoading">
          <div class="text-sm font-medium mb-1">处置建议</div>
          <el-tag v-if="contextData?.task_execution_id" type="warning" class="mr-2 mb-2">修复任务: {{ contextData.task_execution_id }}</el-tag>
          <el-tag v-if="contextData?.ticket_id" type="info" class="mr-2 mb-2">工单: {{ contextData.ticket_id }}</el-tag>
          <ul class="text-xs text-slate-600 list-disc pl-5">
            <li v-for="(s, idx) in contextData?.suggestions || []" :key="idx">{{ s }}</li>
          </ul>
        </div>
      </template>
      <el-empty v-else description="暂无数据" />
    </div>
  </el-drawer>
  </div>
</template>

<style scoped>
:deep(.firing-row) {
  background-color: #fef2f2 !important;
}
:deep(.firing-row:hover > td) {
  background-color: #fee2e2 !important;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
</style>
