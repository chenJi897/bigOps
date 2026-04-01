<script setup lang="ts">
defineOptions({ name: 'AlertEvents' })

import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { alertRuleApi } from '../api'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const events = ref<any[]>([])
const selectedRows = ref<any[]>([])
const pager = ref({ page: 1, size: 20, total: 0 })
const filters = ref({
  keyword: String(route.query.keyword || ''),
  status: String(route.query.status || ''),
  severity: String(route.query.severity || ''),
  agent_id: String(route.query.agent_id || ''),
})

const statusOptions = [
  { label: '全部状态', value: '' },
  { label: '触发中', value: 'firing' },
  { label: '已确认', value: 'acknowledged' },
  { label: '已恢复', value: 'resolved' },
]

const severityOptions = [
  { label: '全部级别', value: '' },
  { label: '提示', value: 'info' },
  { label: '警告', value: 'warning' },
  { label: '严重', value: 'critical' },
]

const canBatchAck = computed(() => selectedRows.value.some(item => item.status === 'firing'))
const canBatchResolve = computed(() => selectedRows.value.some(item => item.status !== 'resolved'))

function severityTagType(severity: string) {
  const map: Record<string, string> = { info: 'info', warning: 'warning', critical: 'danger' }
  return map[severity] || 'info'
}

function statusTagType(status: string) {
  const map: Record<string, string> = { firing: 'danger', acknowledged: 'warning', resolved: 'info' }
  return map[status] || 'info'
}

async function fetchEvents() {
  loading.value = true
  try {
    const res = await alertRuleApi.events({
      page: pager.value.page,
      size: pager.value.size,
      status: filters.value.status,
      severity: filters.value.severity,
      keyword: filters.value.keyword.trim(),
      agent_id: filters.value.agent_id.trim(),
    })
    events.value = (res as any).data?.list || []
    pager.value.total = Number((res as any).data?.total || 0)
  } finally {
    loading.value = false
  }
}

function applyFilters() {
  pager.value.page = 1
  fetchEvents()
}

function resetFilters() {
  filters.value = { keyword: '', status: '', severity: '', agent_id: '' }
  pager.value.page = 1
  fetchEvents()
}

function handleSelectionChange(rows: any[]) {
  selectedRows.value = rows
}

async function acknowledge(row: any) {
  const { value } = await ElMessageBox.prompt('可填写确认备注', '确认告警', { inputPlaceholder: '例如：已知问题，处理中', confirmButtonText: '确认', cancelButtonText: '取消' })
  await alertRuleApi.ackEvent(row.id, value || '')
  ElMessage.success('事件已确认')
  fetchEvents()
}

async function resolve(row: any) {
  const { value } = await ElMessageBox.prompt('请填写关闭说明', '关闭告警', { inputPlaceholder: '例如：已恢复', confirmButtonText: '关闭', cancelButtonText: '取消' })
  await alertRuleApi.resolveEvent(row.id, value || '')
  ElMessage.success('事件已关闭')
  fetchEvents()
}

async function batchAcknowledge() {
  if (!canBatchAck.value) return
  await Promise.all(selectedRows.value.filter(item => item.status === 'firing').map(item => alertRuleApi.ackEvent(item.id, '批量确认')))
  ElMessage.success('已批量确认')
  fetchEvents()
}

async function batchResolve() {
  if (!canBatchResolve.value) return
  await Promise.all(selectedRows.value.filter(item => item.status !== 'resolved').map(item => alertRuleApi.resolveEvent(item.id, '批量关闭')))
  ElMessage.success('已批量关闭')
  fetchEvents()
}

function goTicket(id?: number) {
  if (!id) return
  router.push(`/ticket/detail/${id}`)
}

function goExecution(id?: number) {
  if (!id) return
  router.push(`/task/execution/${id}`)
}

onMounted(fetchEvents)
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50">
    <div class="bg-white border-b border-gray-200 px-6 py-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold text-gray-900">告警事件中心</h1>
        <p class="text-sm text-gray-500 mt-1">统一处理触发中的告警事件，支持确认、关闭和业务跳转。</p>
      </div>
      <div class="flex items-center gap-3">
        <el-button plain :disabled="!canBatchAck" @click="batchAcknowledge">批量确认</el-button>
        <el-button type="warning" plain :disabled="!canBatchResolve" @click="batchResolve">批量关闭</el-button>
        <el-button type="primary" plain @click="fetchEvents">刷新</el-button>
      </div>
    </div>

    <div class="flex-1 overflow-auto p-6">
      <el-card shadow="never" class="border-gray-200">
        <el-form inline class="mb-4 flex flex-wrap gap-2">
          <el-form-item class="mb-0">
            <el-input v-model="filters.keyword" placeholder="规则 / 主机 / IP" clearable class="w-56" @keyup.enter="applyFilters" />
          </el-form-item>
          <el-form-item class="mb-0">
            <el-input v-model="filters.agent_id" placeholder="Agent ID" clearable class="w-56" @keyup.enter="applyFilters" />
          </el-form-item>
          <el-form-item class="mb-0">
            <el-select v-model="filters.status" placeholder="状态" clearable class="w-36">
              <el-option v-for="item in statusOptions" :key="item.label" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item class="mb-0">
            <el-select v-model="filters.severity" placeholder="级别" clearable class="w-36">
              <el-option v-for="item in severityOptions" :key="item.label" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item class="mb-0">
            <el-button type="primary" @click="applyFilters">筛选</el-button>
            <el-button @click="resetFilters">重置</el-button>
          </el-form-item>
        </el-form>

        <el-table :data="events" v-loading="loading" stripe border class="w-full" @selection-change="handleSelectionChange">
          <el-table-column type="selection" width="48" align="center" />
          <el-table-column prop="rule_name" label="规则" min-width="180" show-overflow-tooltip />
          <el-table-column prop="hostname" label="主机" min-width="160" show-overflow-tooltip />
          <el-table-column prop="agent_id" label="Agent ID" min-width="220" show-overflow-tooltip />
          <el-table-column label="级别" width="90" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="severityTagType(row.severity)">{{ row.severity }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="statusTagType(row.status)">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="值 / 阈值" width="150" align="right">
            <template #default="{ row }">{{ Number(row.metric_value || 0).toFixed(1) }} / {{ row.threshold }}</template>
          </el-table-column>
          <el-table-column prop="triggered_at" label="触发时间" width="180" align="center" />
          <el-table-column label="关联" min-width="180">
            <template #default="{ row }">
              <el-button v-if="row.ticket_id" link type="primary" @click="goTicket(row.ticket_id)">工单 #{{ row.ticket_id }}</el-button>
              <el-button v-if="row.task_execution_id" link type="warning" @click="goExecution(row.task_execution_id)">执行 #{{ row.task_execution_id }}</el-button>
              <span v-if="!row.ticket_id && !row.task_execution_id" class="text-gray-400">—</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="140" fixed="right" align="center">
            <template #default="{ row }">
              <el-button link type="primary" :disabled="row.status !== 'firing'" @click="acknowledge(row)">确认</el-button>
              <el-button link type="danger" :disabled="row.status === 'resolved'" @click="resolve(row)">关闭</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="mt-6 flex justify-end">
          <el-pagination
            background
            layout="total, sizes, prev, pager, next"
            :total="pager.total"
            :current-page="pager.page"
            :page-size="pager.size"
            :page-sizes="[10, 20, 50]"
            @current-change="(page: number) => { pager.page = page; fetchEvents() }"
            @size-change="(size: number) => { pager.size = size; pager.page = 1; fetchEvents() }"
          />
        </div>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
/* Scoped styles replaced with Tailwind utility classes */
</style>
