<script setup lang="ts">
defineOptions({ name: 'InspectionCenter' })

import { onMounted, ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
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

const editingTemplateId = ref<number | null>(null)
const editingPlanId = ref<number | null>(null)

const templateForm = ref<any>({
  name: '',
  description: '',
  task_id: undefined as number | undefined,
  default_hosts_text: '',
  enabled: 1,
})

const planForm = ref<any>({
  name: '',
  template_id: undefined as number | undefined,
  cron_expr: '0 */6 * * *',
  enabled: 1,
})

const templateFormValid = computed(() => !!templateForm.value.name && !!templateForm.value.task_id)
const planFormValid = computed(() => !!planForm.value.name && !!planForm.value.template_id && !!planForm.value.cron_expr)

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

function resetTemplateForm() {
  editingTemplateId.value = null
  templateForm.value = { name: '', description: '', task_id: undefined, default_hosts_text: '', enabled: 1 }
}

function editTemplate(row: any) {
  editingTemplateId.value = row.id
  let hosts: string[] = []
  try { hosts = JSON.parse(row.default_hosts || '[]') } catch { hosts = [] }
  templateForm.value = {
    name: row.name,
    description: row.description || '',
    task_id: row.task_id,
    default_hosts_text: hosts.join('\n'),
    enabled: row.enabled,
  }
}

async function submitTemplate() {
  if (!templateFormValid.value) {
    ElMessage.warning('请填写模板名称并选择任务模板')
    return
  }
  const hosts = templateForm.value.default_hosts_text
    .split('\n')
    .map((s: string) => s.trim())
    .filter(Boolean)
  const payload = {
    name: templateForm.value.name,
    description: templateForm.value.description,
    task_id: templateForm.value.task_id,
    default_hosts: hosts,
    enabled: templateForm.value.enabled,
  }
  if (editingTemplateId.value) {
    await inspectionApi.updateTemplate(editingTemplateId.value, payload)
    ElMessage.success('模板已更新')
  } else {
    await inspectionApi.createTemplate(payload)
    ElMessage.success('模板已创建')
  }
  resetTemplateForm()
  await loadAll()
}

function resetPlanForm() {
  editingPlanId.value = null
  planForm.value = { name: '', template_id: undefined, cron_expr: '0 */6 * * *', enabled: 1 }
}

function editPlan(row: any) {
  editingPlanId.value = row.id
  planForm.value = {
    name: row.name,
    template_id: row.template_id,
    cron_expr: row.cron_expr || '0 */6 * * *',
    enabled: row.enabled,
  }
}

async function submitPlan() {
  if (!planFormValid.value) {
    ElMessage.warning('请填写计划名称、选择模板并输入 Cron 表达式')
    return
  }
  const payload = {
    name: planForm.value.name,
    template_id: planForm.value.template_id,
    cron_expr: planForm.value.cron_expr,
    enabled: planForm.value.enabled,
  }
  if (editingPlanId.value) {
    await inspectionApi.updatePlan(editingPlanId.value, payload)
    ElMessage.success('计划已更新')
  } else {
    await inspectionApi.createPlan(payload)
    ElMessage.success('计划已创建')
  }
  resetPlanForm()
  await loadAll()
}

async function runPlan(id: number) {
  try {
    await ElMessageBox.confirm('确认立即执行该巡检计划？', '确认', { type: 'info' })
    await inspectionApi.runPlan(id)
    ElMessage.success('巡检执行已发起')
    await loadAll()
  } catch { /* cancelled */ }
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

function statusType(status: string) {
  if (status === 'success') return 'success'
  if (status === 'failed' || status === 'canceled') return 'danger'
  if (status === 'running') return 'warning'
  return 'info'
}

function taskName(taskId: number) {
  return tasks.value.find((t: any) => t.id === taskId)?.name || `#${taskId}`
}

function templateName(tplId: number) {
  return templates.value.find((t: any) => t.id === tplId)?.name || `#${tplId}`
}

const parsedReport = computed(() => {
  if (!reportData.value) return null
  const d = reportData.value.detail
  if (!d) return null
  let detail = d
  if (typeof d === 'string') {
    try { detail = JSON.parse(d) } catch { return null }
  }
  return detail
})

onMounted(loadAll)
</script>

<template>
  <div class="p-5" v-loading="loading">
    <h1 class="text-lg font-bold text-slate-800 mb-4">巡检管理</h1>

    <el-row :gutter="16">
      <!-- 巡检模板 -->
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="font-medium">巡检模板</span>
              <el-button v-if="editingTemplateId" link type="info" @click="resetTemplateForm">取消编辑</el-button>
            </div>
          </template>

          <el-form label-position="top" size="small">
            <el-form-item label="模板名称" required>
              <el-input v-model="templateForm.name" placeholder="例如：磁盘巡检" />
            </el-form-item>
            <el-form-item label="绑定任务模板" required>
              <el-select v-model="templateForm.task_id" placeholder="请选择任务模板" class="w-full" filterable>
                <el-option v-for="t in tasks" :key="t.id" :label="t.name" :value="t.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="描述">
              <el-input v-model="templateForm.description" placeholder="可选描述" />
            </el-form-item>
            <el-form-item label="默认巡检主机">
              <el-input
                v-model="templateForm.default_hosts_text"
                type="textarea"
                :rows="3"
                placeholder="每行一个主机名或 IP"
              />
            </el-form-item>
            <el-button type="primary" :disabled="!templateFormValid" @click="submitTemplate">
              {{ editingTemplateId ? '保存修改' : '创建模板' }}
            </el-button>
          </el-form>

          <el-table :data="templates" class="mt-4" size="small" stripe>
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="name" label="模板名" min-width="120" show-overflow-tooltip />
            <el-table-column label="任务模板" min-width="120" show-overflow-tooltip>
              <template #default="{ row }">{{ taskName(row.task_id) }}</template>
            </el-table-column>
            <el-table-column label="启用" width="60">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="130">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="editTemplate(row)">编辑</el-button>
                <el-button link type="warning" size="small" @click="viewTemplateTrend(row.id)">趋势</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <!-- 巡检计划 -->
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="font-medium">巡检计划</span>
              <el-button v-if="editingPlanId" link type="info" @click="resetPlanForm">取消编辑</el-button>
            </div>
          </template>

          <el-form label-position="top" size="small">
            <el-form-item label="计划名称" required>
              <el-input v-model="planForm.name" placeholder="例如：每日磁盘巡检" />
            </el-form-item>
            <el-form-item label="绑定模板" required>
              <el-select v-model="planForm.template_id" placeholder="请选择巡检模板" class="w-full" filterable>
                <el-option v-for="t in templates" :key="t.id" :label="t.name" :value="t.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="Cron 表达式" required>
              <el-input v-model="planForm.cron_expr" placeholder="0 */6 * * *" />
            </el-form-item>
            <el-button type="primary" :disabled="!planFormValid" @click="submitPlan">
              {{ editingPlanId ? '保存修改' : '创建计划' }}
            </el-button>
          </el-form>

          <el-table :data="plans" class="mt-4" size="small" stripe>
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="name" label="计划名" min-width="120" show-overflow-tooltip />
            <el-table-column label="模板" min-width="100" show-overflow-tooltip>
              <template #default="{ row }">{{ templateName(row.template_id) }}</template>
            </el-table-column>
            <el-table-column prop="cron_expr" label="Cron" min-width="100" />
            <el-table-column label="操作" width="150">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="editPlan(row)">编辑</el-button>
                <el-button link type="success" size="small" @click="runPlan(row.id)">执行</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <!-- 执行记录 -->
    <el-card shadow="never" class="mt-4">
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium">执行记录</span>
          <el-button size="small" plain @click="loadAll">刷新</el-button>
        </div>
      </template>
      <el-table :data="records" size="small" stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column label="计划" min-width="100" show-overflow-tooltip>
          <template #default="{ row }">{{ plans.find((p: any) => p.id === row.plan_id)?.name || `#${row.plan_id}` }}</template>
        </el-table-column>
        <el-table-column label="模板" min-width="100" show-overflow-tooltip>
          <template #default="{ row }">{{ templateName(row.template_id) }}</template>
        </el-table-column>
        <el-table-column prop="task_execution_id" label="执行ID" width="80" />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="开始时间" min-width="150" />
        <el-table-column prop="finished_at" label="完成时间" min-width="150">
          <template #default="{ row }">{{ row.finished_at || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="220">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="viewReport(row.id)">报告</el-button>
            <el-button link type="success" size="small" @click="exportReport(row.id, 'json')">JSON</el-button>
            <el-button link type="warning" size="small" @click="exportReport(row.id, 'csv')">CSV</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 报告 Drawer -->
    <el-drawer v-model="reportVisible" title="巡检报告" size="50%" append-to-body>
      <div v-loading="reportLoading">
        <template v-if="parsedReport">
          <!-- 概览 -->
          <div class="grid grid-cols-4 gap-3 mb-4">
            <div class="bg-slate-50 rounded-lg p-3 text-center">
              <div class="text-xs text-slate-400">状态</div>
              <el-tag :type="statusType(parsedReport.status)" class="mt-1">{{ parsedReport.status }}</el-tag>
            </div>
            <div class="bg-green-50 rounded-lg p-3 text-center">
              <div class="text-xs text-slate-400">成功</div>
              <div class="text-xl font-bold text-green-600">{{ parsedReport.success_count ?? '-' }}</div>
            </div>
            <div class="bg-red-50 rounded-lg p-3 text-center">
              <div class="text-xs text-slate-400">失败</div>
              <div class="text-xl font-bold text-red-600">{{ parsedReport.fail_count ?? '-' }}</div>
            </div>
            <div class="bg-blue-50 rounded-lg p-3 text-center">
              <div class="text-xs text-slate-400">总计</div>
              <div class="text-xl font-bold text-blue-600">{{ parsedReport.total_count ?? '-' }}</div>
            </div>
          </div>
          <div class="text-xs text-slate-400 mb-3">
            {{ parsedReport.started_at }} ~ {{ parsedReport.finished_at || '进行中' }}
          </div>

          <!-- 主机结果 -->
          <el-table v-if="parsedReport.host_results?.length" :data="parsedReport.host_results" size="small" stripe>
            <el-table-column prop="host_ip" label="主机" min-width="120" show-overflow-tooltip>
              <template #default="{ row }">{{ row.hostname || row.host_ip }}</template>
            </el-table-column>
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="statusType(row.status)" size="small">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="exit_code" label="Exit" width="60" />
            <el-table-column label="耗时" width="80">
              <template #default="{ row }">{{ row.duration_ms ? (row.duration_ms / 1000).toFixed(1) + 's' : '-' }}</template>
            </el-table-column>
            <el-table-column label="输出" min-width="200">
              <template #default="{ row }">
                <div class="text-xs text-slate-600 line-clamp-2">{{ row.stdout || row.stdout_tail || '-' }}</div>
                <div v-if="row.stderr || row.stderr_tail" class="text-xs text-red-500 line-clamp-1 mt-0.5">{{ row.stderr || row.stderr_tail }}</div>
              </template>
            </el-table-column>
          </el-table>

          <div v-else class="text-sm text-slate-400 mt-4">暂无主机执行结果（任务可能仍在运行中）</div>
        </template>

        <!-- 回退：原始 JSON -->
        <template v-else-if="reportData">
          <pre class="text-xs bg-slate-50 p-3 rounded border border-slate-200 whitespace-pre-wrap">{{ JSON.stringify(reportData, null, 2) }}</pre>
        </template>

        <el-empty v-else description="暂无报告数据" />
      </div>
    </el-drawer>

    <!-- 趋势 Drawer -->
    <el-drawer v-model="trendVisible" title="模板执行趋势" size="36%" append-to-body>
      <div v-loading="trendLoading">
        <template v-if="trendData">
          <div class="flex gap-6 mb-4">
            <el-statistic title="成功" :value="trendData.success || 0" />
            <el-statistic title="失败" :value="trendData.failed || 0" />
            <el-statistic title="进行中" :value="trendData.running || 0" />
          </div>
          <el-table :data="trendData.series || []" size="small" stripe>
            <el-table-column prop="id" label="记录ID" width="80" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="statusType(row.status)" size="small">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="时间" />
          </el-table>
        </template>
        <el-empty v-else description="暂无趋势数据" />
      </div>
    </el-drawer>
  </div>
</template>
