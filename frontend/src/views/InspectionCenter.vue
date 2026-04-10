<script setup lang="ts">
defineOptions({ name: 'InspectionCenter' })

import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { inspectionApi, taskApi } from '../api'

const loading = ref(false)
const tasks = ref<any[]>([])
const templates = ref<any[]>([])
const plans = ref<any[]>([])
const records = ref<any[]>([])
const reportVisible = ref(false)
const reportLoading = ref(false)
const reportData = ref<any>(null)
const trendVisible = ref(false)
const trendLoading = ref(false)
const trendData = ref<any>(null)

const templateForm = ref<any>({
  name: '',
  description: '',
  task_id: undefined,
  default_hosts_text: '',
  enabled: 1,
})

const planForm = ref<any>({
  name: '',
  template_id: undefined,
  cron_expr: '0 */6 * * *',
  enabled: 1,
})

async function loadAll() {
  loading.value = true
  try {
    const [tplRes, planRes, recRes, taskRes] = await Promise.all([
      inspectionApi.templates({ page: 1, size: 100 }),
      inspectionApi.plans({ page: 1, size: 100 }),
      inspectionApi.records({ page: 1, size: 100 }),
      taskApi.list({ page: 1, size: 200 }),
    ])
    templates.value = (tplRes as any).data?.list || []
    plans.value = (planRes as any).data?.list || []
    records.value = (recRes as any).data?.list || []
    tasks.value = (taskRes as any).data?.list || []
  } finally {
    loading.value = false
  }
}

async function createTemplate() {
  const hosts = templateForm.value.default_hosts_text
    .split('\n')
    .map((s: string) => s.trim())
    .filter(Boolean)
  await inspectionApi.createTemplate({
    name: templateForm.value.name,
    description: templateForm.value.description,
    task_id: templateForm.value.task_id,
    default_hosts: hosts,
    enabled: templateForm.value.enabled,
  })
  ElMessage.success('模板已创建')
  templateForm.value = { name: '', description: '', task_id: undefined, default_hosts_text: '', enabled: 1 }
  await loadAll()
}

async function createPlan() {
  await inspectionApi.createPlan({
    name: planForm.value.name,
    template_id: planForm.value.template_id,
    cron_expr: planForm.value.cron_expr,
    enabled: planForm.value.enabled,
  })
  ElMessage.success('计划已创建')
  planForm.value = { name: '', template_id: undefined, cron_expr: '0 */6 * * *', enabled: 1 }
  await loadAll()
}

async function runPlan(id: number) {
  await inspectionApi.runPlan(id)
  ElMessage.success('巡检执行已发起')
  await loadAll()
}

async function viewReport(id: number) {
  reportVisible.value = true
  reportLoading.value = true
  try {
    const res = await inspectionApi.recordReport(id)
    reportData.value = (res as any).data
  } finally {
    reportLoading.value = false
  }
}

async function exportReport(id: number, format: 'json' | 'csv') {
  const token = localStorage.getItem('token') || ''
  const url = inspectionApi.recordReportExportUrl(id, format)
  const response = await fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!response.ok) {
    throw new Error(`导出失败(${response.status})`)
  }
  const blob = await response.blob()
  const objectUrl = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = objectUrl
  link.download = `inspection-record-${id}.${format}`
  document.body.appendChild(link)
  link.click()
  link.remove()
  window.URL.revokeObjectURL(objectUrl)
  ElMessage.success(`已导出${format.toUpperCase()}文件`)
}

async function viewTemplateTrend(id: number) {
  trendVisible.value = true
  trendLoading.value = true
  try {
    const res = await inspectionApi.templateTrend(id)
    trendData.value = (res as any).data
  } finally {
    trendLoading.value = false
  }
}

onMounted(loadAll)
</script>

<template>
  <div class="p-5" v-loading="loading">
    <el-page-header content="巡检管理系统" />
    <el-row :gutter="16" class="mt-4">
      <el-col :span="12">
        <el-card header="巡检模板">
          <div class="grid grid-cols-2 gap-2 mb-3">
            <el-input v-model="templateForm.name" placeholder="模板名称" />
            <el-select v-model="templateForm.task_id" placeholder="绑定任务模板">
              <el-option v-for="t in tasks" :key="t.id" :label="t.name" :value="t.id" />
            </el-select>
            <el-input v-model="templateForm.description" placeholder="描述" class="col-span-2" />
            <el-input
              v-model="templateForm.default_hosts_text"
              type="textarea"
              :rows="4"
              class="col-span-2"
              placeholder="每行一个主机/IP"
            />
          </div>
          <el-button type="primary" @click="createTemplate">创建模板</el-button>
          <el-table :data="templates" class="mt-4" size="small">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="name" label="模板名" />
            <el-table-column prop="task_id" label="任务ID" width="100" />
            <el-table-column prop="enabled" label="启用" width="80" />
            <el-table-column label="趋势" width="100">
              <template #default="{ row }">
                <el-button link type="warning" @click="viewTemplateTrend(row.id)">查看趋势</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card header="巡检计划">
          <div class="grid grid-cols-2 gap-2 mb-3">
            <el-input v-model="planForm.name" placeholder="计划名称" />
            <el-select v-model="planForm.template_id" placeholder="绑定模板">
              <el-option v-for="t in templates" :key="t.id" :label="t.name" :value="t.id" />
            </el-select>
            <el-input v-model="planForm.cron_expr" placeholder="Cron表达式" class="col-span-2" />
          </div>
          <el-button type="primary" @click="createPlan">创建计划</el-button>
          <el-table :data="plans" class="mt-4" size="small">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="name" label="计划名" />
            <el-table-column prop="cron_expr" label="Cron" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button link type="primary" @click="runPlan(row.id)">立即执行</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
    <el-card header="执行记录" class="mt-4">
      <el-table :data="records" size="small">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="plan_id" label="计划ID" width="100" />
        <el-table-column prop="template_id" label="模板ID" width="100" />
        <el-table-column prop="task_execution_id" label="任务执行ID" width="140" />
        <el-table-column prop="status" label="状态" width="120" />
        <el-table-column prop="created_at" label="创建时间" />
        <el-table-column label="操作" width="220">
          <template #default="{ row }">
            <el-button link type="primary" @click="viewReport(row.id)">查看报告</el-button>
            <el-button link type="success" @click="exportReport(row.id, 'json')">导出JSON</el-button>
            <el-button link type="warning" @click="exportReport(row.id, 'csv')">导出CSV</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-drawer v-model="reportVisible" title="巡检报告" size="42%">
      <div v-loading="reportLoading">
        <pre class="text-xs bg-slate-50 p-3 rounded border border-slate-200 whitespace-pre-wrap">{{ JSON.stringify(reportData, null, 2) }}</pre>
      </div>
    </el-drawer>

    <el-drawer v-model="trendVisible" title="模板执行趋势" size="36%">
      <div v-loading="trendLoading">
        <template v-if="trendData">
          <el-statistic title="成功" :value="trendData.success || 0" />
          <el-statistic title="失败" :value="trendData.failed || 0" class="mt-2" />
          <el-statistic title="进行中" :value="trendData.running || 0" class="mt-2" />
          <el-table :data="trendData.series || []" size="small" class="mt-3">
            <el-table-column prop="id" label="记录ID" width="90" />
            <el-table-column prop="status" label="状态" width="120" />
            <el-table-column prop="created_at" label="时间" />
          </el-table>
        </template>
      </div>
    </el-drawer>
  </div>
</template>

