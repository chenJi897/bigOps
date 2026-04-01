<script setup lang="ts">
defineOptions({ name: 'CicdRuns' })
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { cicdPipelineApi, cicdProjectApi } from '../api'

const runs = ref<any[]>([])
const projects = ref<any[]>([])
const pipelines = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const loading = ref(false)
const pipelineLoading = ref(false)
const projectFilter = ref<number | null>(null)
const pipelineFilter = ref<number | null>(null)
const statusFilter = ref('')
const detailDrawerVisible = ref(false)
const detailLoading = ref(false)
const selectedRunDetail = ref<any>(null)
const runVariablesText = ref('')
const route = useRoute()
const router = useRouter()

const statusOptions = [
  { label: '全部状态', value: '' },
  { label: 'created', value: 'created' },
  { label: 'waiting_approval', value: 'waiting_approval' },
  { label: 'running', value: 'running' },
  { label: 'success', value: 'success' },
  { label: 'failed', value: 'failed' },
  { label: 'pending', value: 'pending' },
  { label: 'canceled', value: 'canceled' },
]

async function loadProjects() {
  try {
    const res: any = await cicdProjectApi.list({ page: 1, size: 200 })
    projects.value = res.data.list || []
  } catch {}
}

async function loadPipelineOptions() {
  pipelineLoading.value = true
  try {
    const params: Record<string, any> = { page: 1, size: 200 }
    if (projectFilter.value) {
      params.project_id = projectFilter.value
    }
    const res: any = await cicdPipelineApi.list(params)
    pipelines.value = res.data.list || []
  } catch {} finally {
    pipelineLoading.value = false
  }
}

function buildListParams() {
  const params: Record<string, any> = { page: page.value, size: size.value }
  if (projectFilter.value) params.project_id = projectFilter.value
  if (pipelineFilter.value) params.pipeline_id = pipelineFilter.value
  else if (route.query.pipeline_id) params.pipeline_id = Number(route.query.pipeline_id)
  if (statusFilter.value) params.status = statusFilter.value
  return params
}

async function loadRuns() {
  loading.value = true
  try {
    const res: any = await cicdPipelineApi.runs(buildListParams())
    runs.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

watch(projectFilter, () => {
  pipelineFilter.value = null
  page.value = 1
  loadPipelineOptions()
  loadRuns()
})

watch(
  selectedRunDetail,
  () => {
    runVariablesText.value = variablesToText(selectedRunDetail.value?.run?.variables)
  },
  { immediate: true }
)

function handleSearch() {
  page.value = 1
  loadRuns()
}

function handleReset() {
  projectFilter.value = null
  pipelineFilter.value = null
  statusFilter.value = ''
  page.value = 1
  loadPipelineOptions()
  loadRuns()
}

function handlePageChange(current: number) {
  page.value = current
  loadRuns()
}

function handleSizeChange(currentSize: number) {
  size.value = currentSize
  page.value = 1
  loadRuns()
}

const runStatusLabels: Record<string, string> = {
  success: '成功',
  failed: '失败',
  running: '运行中',
  waiting_approval: '审批中',
  waiting: '等待中',
  created: '已创建',
  pending: '排队',
  canceled: '已取消',
  skipped: '已跳过',
  approved: '已通过',
  rejected: '已拒绝',
}

function statusLabel(status: string) {
  if (!status) return ''
  return runStatusLabels[status] || status
}

function statusTagType(status: string) {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'waiting_approval') return 'warning'
  if (status === 'running') return 'primary'
  return 'info'
}

function durationLabel(seconds: number) {
  if (seconds === null || seconds === undefined) return '—'
  const value = Number(seconds)
  if (isNaN(value)) return '—'
  if (value < 60) return `${value}s`
  const mins = Math.floor(value / 60)
  const secs = value % 60
  return `${mins}m ${secs}s`
}

function queueLabel(seconds: number) {
  if (seconds === null || seconds === undefined) return '—'
  return durationLabel(seconds)
}

function formatTimestamp(value?: string) {
  if (!value) return '—'
  if (value.includes('T')) {
    return value.replace('T', ' ').split('.')[0]
  }
  return value
}

function hostStatusTagType(status?: string) {
  if (status === 'success') return 'success'
  if (status === 'failed' || status === 'timeout') return 'danger'
  if (status === 'running') return 'primary'
  if (status === 'pending') return 'info'
  return 'info'
}

function variablesToText(value: any) {
  if (!value) return ''
  if (Array.isArray(value)) {
    return value
      .map((item: any) => {
        if (!item) return ''
        if (typeof item === 'string') return item
        if (typeof item === 'object' && item.key !== undefined) return `${item.key}=${item.value ?? ''}`
        return ''
      })
      .filter(Boolean)
      .join('\n')
  }
  if (typeof value === 'object') {
    return Object.entries(value)
      .map(([key, val]) => `${key}=${val ?? ''}`)
      .join('\n')
  }
  return ''
}

function copyRunVariables() {
  if (!runVariablesText.value) {
    ElMessage.warning('当前运行暂无环境变量可复制')
    return
  }
  const text = runVariablesText.value
  if (navigator.clipboard) {
    navigator.clipboard
      .writeText(text)
      .then(() => ElMessage.success('环境变量已复制'))
      .catch(() => ElMessage.error('复制失败，请手动选择后复制'))
  } else {
    ElMessage.info('当前浏览器不支持自动复制，请手动选择文本')
  }
}

async function openRunDetail(runId?: number) {
  if (!runId) return
  detailLoading.value = true
  try {
    const res: any = await cicdPipelineApi.runDetail(runId)
    selectedRunDetail.value = res.data
    detailDrawerVisible.value = true
  } finally {
    detailLoading.value = false
  }
}

async function refreshCurrentDetail() {
  const runId = selectedRunDetail.value?.run?.id
  if (!runId) return
  detailLoading.value = true
  try {
    const res: any = await cicdPipelineApi.runDetail(runId)
    selectedRunDetail.value = res.data
  } finally {
    detailLoading.value = false
  }
}

function closeDetailDrawer(done?: () => void) {
  detailDrawerVisible.value = false
  selectedRunDetail.value = null
  if (done) done()
}

async function retryRun(row: any) {
  await ElMessageBox.confirm(
    `将重新执行运行 #${row.run_number || row.id}，会基于当前流水线配置创建新的执行记录，确认继续吗？`,
    '确认重试',
    { type: 'warning', confirmButtonText: '确认重试', cancelButtonText: '取消' },
  )
  await cicdPipelineApi.retryRun(row.id)
  ElMessage.success(`已提交重试请求：运行 #${row.run_number || row.id}`)
  await loadRuns()
  await refreshCurrentDetail()
}

async function rollbackRun(row: any) {
  await ElMessageBox.confirm(
    `将基于运行 #${row.run_number || row.id} 发起回滚任务，建议确认目标环境与版本信息后再继续，是否执行？`,
    '确认回滚',
    { type: 'warning', confirmButtonText: '确认回滚', cancelButtonText: '取消' },
  )
  await cicdPipelineApi.rollbackRun(row.id)
  ElMessage.success(`已提交回滚请求：运行 #${row.run_number || row.id}`)
  await loadRuns()
  await refreshCurrentDetail()
}

function goToTaskExecution(id?: number) {
  if (!id) return
  router.push(`/task/execution/${id}`)
}

function goToTicketDetail(id?: number) {
  if (!id) return
  router.push(`/ticket/detail/${id}`)
}

function stageStatusTagType(status?: string) {
  if (!status) return 'info'
  if (status === 'success' || status === 'approved' || status === 'completed') return 'success'
  if (status === 'failed' || status === 'rejected' || status === 'timeout') return 'danger'
  if (status === 'running' || status === 'in_progress') return 'primary'
  if (status === 'waiting' || status === 'pending' || status === 'waiting_approval') return 'warning'
  return 'info'
}

function stageText(detail: any, stage: 'build' | 'approval' | 'deploy') {
  if (!detail?.run) return null
  const run = detail.run
  const artifact = run.artifact_summary_map || {}
  const buildStage = artifact.build || {}
  const approvalStage = artifact.approval || {}
  const deployStage = artifact.deploy || {}
  const taskExecution = detail.task_execution
  const buildExecutionId = Number(run.build_execution_id || buildStage.execution_id || 0)
  const deployExecutionId = Number(run.deploy_execution_id || deployStage.execution_id || run.task_execution_id || taskExecution?.id || 0)
  const approvalTicketId = Number(run.approval_ticket_id || run.approval_ticket_id_stage || approvalStage.ticket_id || 0)
  if (stage === 'build') {
    const status = run.build_status || run.build_stage_status || buildStage.status || (buildExecutionId ? 'pending' : 'skipped')
    return {
      status: status || 'pending',
      idLabel: buildExecutionId ? `执行 ID: ${buildExecutionId}` : '执行 ID: —',
      summary: run.build_summary || buildStage.summary || (buildExecutionId ? '构建阶段已创建执行任务' : '未配置构建阶段'),
      error: run.build_error || buildStage.error || '',
      action: buildExecutionId ? { label: '查看构建执行', fn: () => goToTaskExecution(buildExecutionId) } : null,
    }
  }
  if (stage === 'approval') {
    const status = run.approval_status || run.approval_stage_status || approvalStage.status || (approvalTicketId ? 'waiting_approval' : 'skipped')
    return {
      status,
      idLabel: approvalTicketId ? `审批单 ID: ${approvalTicketId}` : '审批单 ID: —',
      summary: run.approval_summary || approvalStage.summary || (approvalTicketId ? '等待审批结果' : '无需审批'),
      error: run.approval_error || approvalStage.error || '',
      action: approvalTicketId ? { label: '查看审批单', fn: () => goToTicketDetail(approvalTicketId) } : null,
    }
  }
  const status = run.deploy_status || run.deploy_stage_status || deployStage.status || run.status || 'pending'
  return {
    status,
    idLabel: `执行 ID: ${deployExecutionId || '—'}`,
    summary: run.deploy_summary || deployStage.summary || run.result || run.summary || '待执行',
    error: run.deploy_error || deployStage.error || run.error_message || '',
    action: deployExecutionId ? { label: '查看部署执行', fn: () => goToTaskExecution(deployExecutionId) } : null,
  }
}

watch(
  () => route.query.pipeline_id,
  (newVal) => {
    if (newVal) {
      const pipelineId = Number(newVal)
      if (Number.isFinite(pipelineId) && pipelineId > 0) {
        pipelineFilter.value = pipelineId
        page.value = 1
        loadRuns()
      }
    } else {
      pipelineFilter.value = null
      page.value = 1
      loadRuns()
    }
  }
)

onMounted(() => {
  if (route.query.pipeline_id) {
    const pipelineId = Number(route.query.pipeline_id)
    if (Number.isFinite(pipelineId) && pipelineId > 0) pipelineFilter.value = pipelineId
  }
  loadProjects()
  loadPipelineOptions()
  loadRuns()
})
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50">
    <div class="bg-white border-b border-gray-200 px-6 py-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold text-gray-900">CI/CD 运行记录</h1>
        <p class="text-sm text-gray-500 mt-1">查看和管理所有的流水线执行记录、部署历史与构建日志。</p>
      </div>
      <div class="flex items-center gap-3">
        <el-button plain @click="loadRuns">刷新</el-button>
      </div>
    </div>

    <div class="flex-1 overflow-auto p-6 space-y-6">
      <el-alert
        title="回滚说明"
        type="warning"
        effect="light"
        show-icon
        class="border border-yellow-200 rounded-lg shadow-sm"
      >
        <template #default>
          <div class="text-sm text-gray-600 mt-1">
            回滚会尝试重新部署当前运行记录的 commit，若需要针对某个主机或变量进行调整，请先复制环境变量并手动触发新运行。
          </div>
        </template>
      </el-alert>

      <el-card shadow="never" class="border-gray-200">
        <el-form label-width="0" :inline="true" class="mb-4 flex flex-wrap gap-2" @submit.prevent="handleSearch">
          <el-form-item class="mb-0">
            <el-select v-model="projectFilter" placeholder="项目" clearable class="w-48">
              <el-option v-for="project in projects" :key="project.id" :label="project.name" :value="project.id" />
            </el-select>
          </el-form-item>
          <el-form-item class="mb-0">
            <el-select
              v-model="pipelineFilter"
              placeholder="流水线"
              clearable
              filterable
              class="w-56"
              :loading="pipelineLoading"
            >
              <el-option v-for="pipeline in pipelines" :key="pipeline.id" :label="pipeline.name" :value="pipeline.id" />
            </el-select>
          </el-form-item>
          <el-form-item class="mb-0">
            <el-select v-model="statusFilter" placeholder="状态" clearable class="w-32">
              <el-option v-for="item in statusOptions" :key="item.value || 'all'" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item class="mb-0">
            <el-button type="primary" @click="handleSearch">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>

        <el-table :data="runs" v-loading="loading" stripe border class="w-full">
          <el-table-column prop="id" label="ID" width="72" align="center" />
          <el-table-column prop="project_name" label="项目" min-width="160" show-overflow-tooltip />
          <el-table-column prop="pipeline_name" label="流水线" min-width="180" show-overflow-tooltip />
          <el-table-column prop="run_number" label="运行号" width="90" align="center" />
          <el-table-column label="分支 / 触发" min-width="220">
            <template #default="{ row }">
              <div class="flex flex-col gap-1">
                <div class="flex items-center gap-2 font-medium text-gray-800">
                  <span>{{ row.branch || row.trigger_ref || '—' }}</span>
                  <el-tag size="small" type="info">{{ row.trigger_type || '—' }}</el-tag>
                </div>
                <div class="text-xs text-gray-500 truncate">{{ row.trigger_ref || '—' }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="提交" min-width="220">
            <template #default="{ row }">
              <div class="flex flex-col gap-1">
                <span class="text-xs font-mono text-gray-500">{{ row.commit_id ? row.commit_id.substring(0, 8) : '—' }}</span>
                <div class="text-xs text-gray-700 truncate" :title="row.commit_message">{{ row.commit_message || '—' }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="140" align="center">
            <template #default="{ row }">
              <div class="flex flex-col gap-1 items-center">
                <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) || row.status }}</el-tag>
                <span class="text-xs text-gray-500">{{ row.result || '—' }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="耗时 / 排队" width="140" align="center">
            <template #default="{ row }">
              <div class="flex flex-col gap-1 text-xs">
                <span>耗时 {{ durationLabel(row.duration_seconds) }}</span>
                <span class="text-gray-500">排队 {{ queueLabel(row.queued_seconds) }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="triggered_by_name" label="触发人" width="140" align="center" />
          <el-table-column prop="created_at" label="开始时间" width="170" align="center" />
          <el-table-column label="摘要 / 错误" min-width="260">
            <template #default="{ row }">
              <div class="flex flex-col gap-1">
                <div class="text-sm text-gray-700">{{ row.summary || row.result || '—' }}</div>
                <div v-if="row.error_message" class="text-xs text-red-600 break-words">{{ row.error_message }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="220" fixed="right" align="center">
            <template #default="{ row }">
              <el-button link type="primary" @click="openRunDetail(row.id)">详情</el-button>
              <el-button link type="warning" @click="retryRun(row)">重新执行</el-button>
              <el-button link type="danger" @click="rollbackRun(row)">发起回滚</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div v-if="total > 0" class="mt-6 flex justify-end">
          <el-pagination
            background
            :current-page="page"
            :page-size="size"
            :page-sizes="[10, 20, 50, 100]"
            :total="total"
            layout="total, sizes, prev, pager, next"
            @size-change="handleSizeChange"
            @current-change="handlePageChange"
          />
        </div>
      </el-card>

      <el-drawer
        v-model="detailDrawerVisible"
        size="600px"
        direction="rtl"
        :with-header="false"
        destroy-on-close
        @close="closeDetailDrawer"
      >
        <div v-if="selectedRunDetail" class="flex justify-between items-center mb-4 pb-4 border-b border-gray-100 px-6 pt-6">
          <div>
            <div class="text-lg font-bold text-gray-900">运行详情 #{{ selectedRunDetail.run?.run_number || '—' }}</div>
            <div class="text-sm text-gray-500 mt-1">{{ selectedRunDetail.run?.pipeline_name || '—' }}</div>
          </div>
          <el-tag :type="statusTagType(selectedRunDetail.run?.status)" size="large">
            {{ statusLabel(selectedRunDetail.run?.status) || selectedRunDetail.run?.status || '—' }}
          </el-tag>
        </div>

        <div v-loading="detailLoading" class="px-6 pb-6 space-y-6">
          <template v-if="selectedRunDetail?.run">
            <el-descriptions size="small" :column="2" title="基本信息" border>
              <el-descriptions-item label="流水线">{{ selectedRunDetail.run.pipeline_name || '—' }}</el-descriptions-item>
              <el-descriptions-item label="分支">{{ selectedRunDetail.run.branch || '—' }}</el-descriptions-item>
              <el-descriptions-item label="触发方式">{{ selectedRunDetail.run.trigger_type || '—' }}</el-descriptions-item>
              <el-descriptions-item label="触发事件">{{ selectedRunDetail.run.trigger_ref || '—' }}</el-descriptions-item>
              <el-descriptions-item label="开始时间">{{ formatTimestamp(selectedRunDetail.run.started_at) }}</el-descriptions-item>
              <el-descriptions-item label="结束时间">{{ formatTimestamp(selectedRunDetail.run.finished_at) }}</el-descriptions-item>
              <el-descriptions-item label="耗时">{{ durationLabel(selectedRunDetail.run.duration_seconds) }}</el-descriptions-item>
              <el-descriptions-item label="排队">{{ queueLabel(selectedRunDetail.run.queued_seconds) }}</el-descriptions-item>
              <el-descriptions-item label="触发人">{{ selectedRunDetail.run.triggered_by_name || '—' }}</el-descriptions-item>
              <el-descriptions-item label="提交" class="font-mono">{{ selectedRunDetail.run.commit_id || '—' }}</el-descriptions-item>
            </el-descriptions>

            <div v-if="selectedRunDetail.run.summary || selectedRunDetail.run.error_message" class="flex flex-col gap-2">
              <div class="font-semibold text-gray-900">摘要 / 错误</div>
              <div class="text-sm text-gray-700 bg-gray-50 p-3 rounded-lg border border-gray-200">
                <div>{{ selectedRunDetail.run.summary || '—' }}</div>
                <div v-if="selectedRunDetail.run.error_message" class="text-red-600 mt-2 text-xs font-mono whitespace-pre-wrap">
                  {{ selectedRunDetail.run.error_message }}
                </div>
              </div>
            </div>

            <div class="flex flex-col gap-3">
              <div class="font-semibold text-gray-900">阶段视图</div>
              <div class="grid grid-cols-1 gap-3">
                <div v-for="stage in ['build', 'approval', 'deploy']" :key="stage" class="border border-gray-200 rounded-lg p-4 bg-white shadow-sm">
                  <div class="flex justify-between items-center mb-2">
                    <span class="font-medium text-gray-900 capitalize">{{ stage }}</span>
                    <el-tag size="small" :type="stageStatusTagType(stageText(selectedRunDetail, stage as any)?.status)">
                      {{ stageText(selectedRunDetail, stage as any)?.status || 'pending' }}
                    </el-tag>
                  </div>
                  <div class="text-xs text-gray-500 mb-1">{{ stageText(selectedRunDetail, stage as any)?.idLabel }}</div>
                  <div class="text-sm text-gray-700 mb-2">{{ stageText(selectedRunDetail, stage as any)?.summary }}</div>
                  <div v-if="stageText(selectedRunDetail, stage as any)?.error" class="text-xs text-red-600 mb-2 whitespace-pre-wrap">
                    {{ stageText(selectedRunDetail, stage as any)?.error }}
                  </div>
                  <el-button v-if="stageText(selectedRunDetail, stage as any)?.action" size="small" type="primary" link @click="stageText(selectedRunDetail, stage as any)?.action?.fn()">
                    {{ stageText(selectedRunDetail, stage as any)?.action?.label }}
                  </el-button>
                </div>
              </div>
            </div>

            <div class="flex flex-col gap-2">
              <div class="font-semibold text-gray-900">日志片段</div>
              <pre class="bg-gray-900 text-gray-100 p-4 rounded-lg text-xs font-mono overflow-auto max-h-64 whitespace-pre-wrap">{{ selectedRunDetail.run.log_snippet || '暂无可用日志' }}</pre>
            </div>

            <div class="flex flex-col gap-2">
              <div class="flex justify-between items-center">
                <span class="font-semibold text-gray-900">环境变量</span>
                <el-button link type="primary" @click="copyRunVariables">复制</el-button>
              </div>
              <el-input type="textarea" v-model="runVariablesText" :rows="4" readonly placeholder="当前运行未记录环境变量" class="font-mono text-xs" />
              <div class="text-xs text-gray-500 mt-1">复制后可将变量粘贴在新运行或 Webhook 请求中。</div>
            </div>

            <template v-if="selectedRunDetail.task_execution">
              <el-divider />
              <div class="flex flex-col gap-4">
                <div class="flex justify-between items-center">
                  <span class="font-semibold text-gray-900">任务执行</span>
                  <span class="text-sm text-gray-500">
                    {{ selectedRunDetail.task_execution.success_count || 0 }}/{{ selectedRunDetail.task_execution.total_count || 0 }} 成功
                  </span>
                </div>
                <el-descriptions size="small" :column="2" border>
                  <el-descriptions-item label="ID">{{ selectedRunDetail.task_execution.id }}</el-descriptions-item>
                  <el-descriptions-item label="状态">{{ selectedRunDetail.task_execution.status }}</el-descriptions-item>
                  <el-descriptions-item label="执行人">{{ selectedRunDetail.task_execution.operator_name || '—' }}</el-descriptions-item>
                  <el-descriptions-item label="目标主机" :span="2">{{ selectedRunDetail.task_execution.target_hosts || '—' }}</el-descriptions-item>
                </el-descriptions>
                
                <div v-if="selectedRunDetail.task_execution.host_results?.length" class="flex flex-wrap gap-2 mt-2">
                  <div v-for="host in selectedRunDetail.task_execution.host_results" :key="host.id" class="flex items-center gap-2 px-3 py-1.5 bg-gray-50 border border-gray-200 rounded-md text-xs">
                    <span class="font-medium text-gray-700">{{ host.hostname || host.host_ip }}</span>
                    <el-tag size="small" :type="hostStatusTagType(host.status)">{{ host.status }}</el-tag>
                  </div>
                </div>
              </div>
            </template>
          </template>
        </div>
      </el-drawer>
    </div>
  </div>
</template>

<style scoped>
/* Scoped styles replaced with Tailwind utility classes */
</style>
