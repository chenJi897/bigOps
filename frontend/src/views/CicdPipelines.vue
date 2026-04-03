<script setup lang="ts">
defineOptions({ name: 'CicdPipelines' })

import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { cicdPipelineApi, cicdProjectApi, requestTemplateApi, taskApi } from '../api'
import NotifyConfigEditor from '../components/NotifyConfigEditor.vue'

const router = useRouter()
const loading = ref(false)
const formLoading = ref(false)
const detailLoading = ref(false)
const formVisible = ref(false)
const detailVisible = ref(false)
const isEdit = ref(false)
const editingId = ref<number | null>(null)
const page = ref(1)
const size = ref(20)
const total = ref(0)
const keyword = ref('')
const projectFilter = ref<number | null>(null)
const statusFilter = ref('')

const pipelines = ref<any[]>([])
const projects = ref<any[]>([])
const taskOptions = ref<any[]>([])
const templateOptions = ref<any[]>([])
const selectedRunDetail = ref<any>(null)

const form = ref({
  name: '',
  code: '',
  project_id: 0,
  description: '',
  schedule: 'manual',
  trigger_type: 'manual',
  branch: 'main',
  environment: 'test',
  build_task_id: undefined as number | undefined,
  deploy_task_id: undefined as number | undefined,
  request_template_id: undefined as number | undefined,
  target_hosts_text: '',
  build_hosts_text: '',
  variables_text: '',
  notify_channels: ['in_app'] as string[],
  notify_config: {} as Record<string, { webhook_url: string; secret: string }>,
  webhook_enabled: 0,
  webhook_secret: '',
  active: 1,
})

const webhookPreviewUrl = computed(() => {
  const origin = typeof window !== 'undefined' ? window.location.origin : ''
  const pipelineCode = (form.value.code || slugifyCode(form.value.name) || 'pipeline-code').trim()
  return `${origin}/api/v1/cicd/webhook/${pipelineCode}`
})

function normalizeId(value: any) {
  if (value === null || value === undefined || value === '') return undefined
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed <= 0) return undefined
  return parsed
}

function normalizeBoolFlag(value: any) {
  return value === true || value === 1 || value === '1'
}

function slugifyCode(value: string) {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

function parseTextList(text: string) {
  return text
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)
}

function formatTextList(items: any) {
  if (!Array.isArray(items)) return ''
  return items.map((item) => String(item || '').trim()).filter(Boolean).join('\n')
}

function variablesToText(value: any) {
  if (!value || typeof value !== 'object') return ''
  return Object.entries(value)
    .map(([key, val]) => `${key}=${val ?? ''}`)
    .join('\n')
}

function parseVariablesText(value: string) {
  const result: Record<string, string> = {}
  value
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)
    .forEach((line) => {
      const index = line.indexOf('=')
      if (index <= 0) return
      const key = line.slice(0, index).trim()
      const val = line.slice(index + 1).trim()
      if (!key) return
      result[key] = val
    })
  return result
}

function buildListParams() {
  const params: Record<string, any> = { page: page.value, size: size.value }
  if (keyword.value) params.keyword = keyword.value
  if (projectFilter.value) params.project_id = projectFilter.value
  if (statusFilter.value !== '') params.active = Number(statusFilter.value)
  return params
}

async function loadProjects() {
  const res: any = await cicdProjectApi.list({ page: 1, size: 200 })
  projects.value = res.data.list || []
}

async function loadTasks() {
  const res: any = await taskApi.list({ page: 1, size: 200 })
  taskOptions.value = res.data.list || []
}

async function loadTemplates() {
  const res: any = await requestTemplateApi.list(true)
  templateOptions.value = res.data || []
}

async function loadPipelines() {
  loading.value = true
  try {
    const res: any = await cicdPipelineApi.list(buildListParams())
    pipelines.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  loadPipelines()
}

function handleReset() {
  keyword.value = ''
  projectFilter.value = null
  statusFilter.value = ''
  page.value = 1
  loadPipelines()
}

function handlePageChange(current: number) {
  page.value = current
  loadPipelines()
}

function handleSizeChange(current: number) {
  size.value = current
  page.value = 1
  loadPipelines()
}

function resetForm() {
  form.value = {
    name: '',
    code: '',
    project_id: projects.value[0]?.id || 0,
    description: '',
    schedule: 'manual',
    trigger_type: 'manual',
    branch: 'main',
    environment: 'test',
    build_task_id: undefined,
    deploy_task_id: undefined,
    request_template_id: undefined,
    target_hosts_text: '',
    build_hosts_text: '',
    variables_text: '',
    notify_channels: ['in_app'],
    notify_config: {},
    webhook_enabled: 0,
    webhook_secret: '',
    active: 1,
  }
}

function openCreate() {
  isEdit.value = false
  editingId.value = null
  resetForm()
  formVisible.value = true
}

function openEdit(row: any) {
  isEdit.value = true
  editingId.value = row.id
  form.value = {
    name: row.name || '',
    code: row.code || '',
    project_id: row.project_id || 0,
    description: row.description || '',
    schedule: row.schedule || 'manual',
    trigger_type: row.trigger_type || row.schedule || 'manual',
    branch: row.branch || 'main',
    environment: row.environment || 'test',
    build_task_id: normalizeId(row.build_task_id),
    deploy_task_id: normalizeId(row.deploy_task_id),
    request_template_id: normalizeId(row.request_template_id),
    target_hosts_text: formatTextList(row.target_hosts_list),
    build_hosts_text: formatTextList(row.build_hosts_list),
    variables_text: variablesToText(row.variables || row.pipeline_variables),
    notify_channels: Array.isArray(row.notify_channels) ? row.notify_channels : [],
    notify_config: row.notify_config ? (typeof row.notify_config === 'string' ? JSON.parse(row.notify_config) : row.notify_config) : {},
    webhook_enabled: normalizeBoolFlag(row.webhook_enabled) ? 1 : 0,
    webhook_secret: row.webhook_secret || '',
    active: row.active === 1 ? 1 : 0,
  }
  formVisible.value = true
}

async function submitForm() {
  if (!form.value.name || !form.value.project_id) {
    ElMessage.warning('请填写流水线名称并选择项目')
    return
  }
  formLoading.value = true
  try {
    const payload: Record<string, any> = {
      name: form.value.name,
      code: form.value.code,
      project_id: form.value.project_id,
      description: form.value.description,
      schedule: form.value.schedule,
      trigger_type: form.value.trigger_type,
      branch: form.value.branch,
      environment: form.value.environment,
      target_hosts: parseTextList(form.value.target_hosts_text),
      build_hosts: parseTextList(form.value.build_hosts_text),
      variables: parseVariablesText(form.value.variables_text),
      notify_channels: form.value.notify_channels,
      notify_config: form.value.notify_config,
      webhook_enabled: form.value.webhook_enabled === 1,
      webhook_secret: form.value.webhook_secret.trim(),
      active: form.value.active,
    }
    if (form.value.build_task_id) payload.build_task_id = form.value.build_task_id
    if (form.value.deploy_task_id) payload.deploy_task_id = form.value.deploy_task_id
    if (form.value.request_template_id) payload.request_template_id = form.value.request_template_id
    if (isEdit.value && editingId.value) {
      await cicdPipelineApi.update(editingId.value, payload)
      ElMessage.success('流水线更新成功')
    } else {
      await cicdPipelineApi.create(payload)
      ElMessage.success('流水线创建成功')
    }
    formVisible.value = false
    loadPipelines()
  } finally {
    formLoading.value = false
  }
}

async function toggleStatus(row: any) {
  await cicdPipelineApi.update(row.id, { ...row, active: row.active === 1 ? 0 : 1 })
  ElMessage.success(row.active === 1 ? '流水线已停用' : '流水线已启用')
  loadPipelines()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确定删除流水线 ${row.name}？`, '提示', { type: 'warning' })
  await cicdPipelineApi.delete(row.id)
  ElMessage.success('流水线已删除')
  loadPipelines()
}

async function handleTrigger(row: any) {
  const res: any = await cicdPipelineApi.trigger(row.id)
  ElMessage.success(`已触发流水线 #${res?.data?.run_number || row.id}`)
  loadPipelines()
}

function statusLabel(status?: string) {
  const map: Record<string, string> = {
    success: '成功',
    failed: '失败',
    running: '运行中',
    waiting_approval: '审批中',
    created: '已创建',
    pending: '排队',
    canceled: '已取消',
  }
  return status ? (map[status] || status) : '—'
}

function statusTagType(status?: string) {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'running') return 'primary'
  if (status === 'waiting_approval') return 'warning'
  return 'info'
}

function configSummary(row: any) {
  const parts = [
    row.build_task_id ? '构建' : '无构建',
    row.request_template_id ? '需审批' : '免审批',
    row.deploy_task_id ? '部署' : '无部署',
    normalizeBoolFlag(row.webhook_enabled) ? 'Webhook' : '手动',
  ]
  return parts.join(' / ')
}

function goToRuns(row: any) {
  router.push({ path: '/cicd/runs', query: { pipeline_id: row.id } })
}

async function openLatestDetail(runId?: number) {
  if (!runId) return
  detailLoading.value = true
  try {
    const res: any = await cicdPipelineApi.runDetail(runId)
    selectedRunDetail.value = res.data
    detailVisible.value = true
  } finally {
    detailLoading.value = false
  }
}

function closeDetail() {
  detailVisible.value = false
  selectedRunDetail.value = null
}

onMounted(async () => {
  await Promise.all([loadProjects(), loadTasks(), loadTemplates()])
  await loadPipelines()
})
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50">
    <div class="bg-white border-b border-gray-200 px-6 py-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold text-gray-900">CI/CD 流水线</h1>
        <p class="text-sm text-gray-500 mt-1">配置流水线的构建、部署任务，审批模板以及环境变量和触发规则。</p>
      </div>
      <div class="flex items-center gap-3">
        <el-button v-permission="'cicd_pipeline:create'" type="primary" @click="openCreate">新增流水线</el-button>
      </div>
    </div>

    <div class="flex-1 overflow-auto p-6 space-y-6">
      <el-card shadow="never" class="border-gray-200">
        <el-form :inline="true" class="mb-4 flex flex-wrap gap-2" label-width="0" @submit.prevent="handleSearch">
          <el-form-item class="mb-0">
            <el-input v-model="keyword" placeholder="流水线/项目" clearable class="w-56" @keyup.enter="handleSearch" />
          </el-form-item>
          <el-form-item class="mb-0">
            <el-select v-model="projectFilter" placeholder="项目" clearable class="w-48">
              <el-option v-for="project in projects" :key="project.id" :label="project.name" :value="project.id" />
            </el-select>
          </el-form-item>
          <el-form-item class="mb-0">
            <el-select v-model="statusFilter" placeholder="状态" clearable class="w-32">
              <el-option label="全部" value="" />
              <el-option label="启用" value="1" />
              <el-option label="停用" value="0" />
            </el-select>
          </el-form-item>
          <el-form-item class="mb-0">
            <el-button type="primary" @click="handleSearch">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>

        <el-table :data="pipelines" v-loading="loading" stripe border class="w-full">
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column prop="name" label="流水线名称" min-width="180" show-overflow-tooltip />
          <el-table-column prop="project_name" label="项目" min-width="160" show-overflow-tooltip />
          <el-table-column label="配置概览" min-width="240">
            <template #default="{ row }">
              <div class="flex flex-col gap-1.5">
                <div class="flex flex-wrap gap-1.5">
                  <el-tag size="small" :type="Array.isArray(row.notify_channels) && row.notify_channels.includes('wecom') ? 'warning' : 'info'">
                    {{ Array.isArray(row.notify_channels) && row.notify_channels.length ? row.notify_channels.join('/') : '默认通知' }}
                  </el-tag>
                  <el-tag size="small" :type="normalizeBoolFlag(row.webhook_enabled) ? 'success' : 'info'">
                    {{ normalizeBoolFlag(row.webhook_enabled) ? 'Webhook' : '手动' }}
                  </el-tag>
                  <el-tag size="small" :type="row.build_task_id ? 'primary' : 'info'">
                    {{ row.build_task_id ? '构建' : '无构建' }}
                  </el-tag>
                  <el-tag size="small" :type="row.request_template_id ? 'warning' : 'info'">
                    {{ row.request_template_id ? '审批' : '免审批' }}
                  </el-tag>
                  <el-tag size="small" :type="row.deploy_task_id ? 'success' : 'info'">
                    {{ row.deploy_task_id ? '部署' : '无部署' }}
                  </el-tag>
                </div>
                <div class="text-xs text-gray-500">{{ configSummary(row) }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="branch" label="分支" width="120" />
          <el-table-column prop="environment" label="环境" width="100" />
          <el-table-column label="最近运行" min-width="280">
            <template #default="{ row }">
              <div v-if="row.latest_run" class="flex flex-col gap-1.5">
                <div>
                  <el-tag size="small" :type="statusTagType(row.latest_run.status)">
                    #{{ row.latest_run.run_number }} {{ statusLabel(row.latest_run.status) }}
                  </el-tag>
                </div>
                <div class="text-sm text-gray-700 leading-snug">{{ row.latest_run.summary || '等待结果' }}</div>
                <div><el-button link type="primary" size="small" @click="openLatestDetail(row.latest_run.id)">查看详情</el-button></div>
              </div>
              <span v-else class="text-gray-400">—</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="90" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="row.active === 1 ? 'success' : 'info'">
                {{ row.active === 1 ? '启用' : '停用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="updated_at" label="更新时间" width="170" align="center" />
          <el-table-column label="操作" width="300" fixed="right" align="center">
            <template #default="{ row }">
              <el-button v-permission="'cicd_pipeline:edit'" link type="primary" @click="openEdit(row)">编辑</el-button>
              <el-button link :type="row.active === 1 ? 'warning' : 'success'" @click="toggleStatus(row)">{{ row.active === 1 ? '停用' : '启用' }}</el-button>
              <el-button link type="info" @click="handleTrigger(row)">触发</el-button>
              <el-button link type="primary" @click="goToRuns(row)">运行记录</el-button>
              <el-button v-permission="'cicd_pipeline:delete'" link type="danger" @click="handleDelete(row)">删除</el-button>
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
    </div>

    <el-dialog v-model="formVisible" :title="isEdit ? '编辑流水线' : '新增流水线'" width="720px" destroy-on-close align-center>
      <el-form label-width="100px" class="pr-6">
        <el-form-item label="流水线名称" required>
          <el-input v-model="form.name" placeholder="填写流水线名称" />
        </el-form-item>
        <el-form-item label="流水线编码">
          <el-input v-model="form.code" placeholder="留空自动生成" />
        </el-form-item>
        <el-form-item label="关联项目" required>
          <el-select v-model="form.project_id" placeholder="选择项目" class="w-full">
            <el-option v-for="project in projects" :key="project.id" :label="project.name" :value="project.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="环境 / 分支">
          <div class="flex gap-4 w-full">
            <el-select v-model="form.environment" placeholder="环境" class="w-1/3">
              <el-option label="测试" value="test" />
              <el-option label="预发" value="staging" />
              <el-option label="生产" value="prod" />
            </el-select>
            <el-input v-model="form.branch" placeholder="main" class="flex-1" />
          </div>
        </el-form-item>
        <el-form-item label="构建任务">
          <el-select v-model="form.build_task_id" clearable filterable placeholder="可选" class="w-full">
            <el-option v-for="task in taskOptions" :key="task.id" :label="task.name" :value="task.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="构建主机">
          <el-input v-model="form.build_hosts_text" type="textarea" :rows="3" placeholder="一行一个 IP，留空回退到目标主机" />
        </el-form-item>
        <el-form-item label="部署任务">
          <el-select v-model="form.deploy_task_id" clearable filterable placeholder="可选" class="w-full">
            <el-option v-for="task in taskOptions" :key="task.id" :label="task.name" :value="task.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="目标主机">
          <el-input v-model="form.target_hosts_text" type="textarea" :rows="3" placeholder="一行一个目标主机 IP" />
        </el-form-item>
        <el-form-item label="审批模板">
          <el-select v-model="form.request_template_id" clearable filterable placeholder="可选" class="w-full">
            <el-option v-for="item in templateOptions" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="环境变量">
          <el-input v-model="form.variables_text" type="textarea" :rows="4" placeholder="一行一个 KEY=VALUE" />
        </el-form-item>
        <el-form-item label="通知渠道">
          <NotifyConfigEditor v-model="form.notify_config" />
        </el-form-item>
        <el-form-item label="Webhook">
          <div class="flex flex-col gap-2 w-full">
            <div class="flex items-center gap-3">
              <el-switch v-model="form.webhook_enabled" :active-value="1" :inactive-value="0" />
              <span class="text-xs text-gray-500">启用后可通过公开地址触发</span>
            </div>
            <el-input v-model="form.webhook_secret" placeholder="可选，建议配置密钥" />
            <el-input :model-value="webhookPreviewUrl" readonly />
          </div>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="描述这条流水线的用途" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.active" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end pt-4">
          <el-button @click="formVisible = false">取消</el-button>
          <el-button type="primary" :loading="formLoading" @click="submitForm">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" size="500px" direction="rtl" :with-header="false" @close="closeDetail">
      <div v-if="selectedRunDetail?.run" class="flex justify-between items-center mb-4 pb-4 border-b border-gray-100 px-6 pt-6">
        <div>
          <div class="text-lg font-bold text-gray-900">运行详情 #{{ selectedRunDetail.run.run_number }}</div>
          <div class="text-sm text-gray-500 mt-1">{{ selectedRunDetail.run.pipeline_name }}</div>
        </div>
        <el-tag :type="statusTagType(selectedRunDetail.run.status)">{{ statusLabel(selectedRunDetail.run.status) }}</el-tag>
      </div>
      <div v-loading="detailLoading" class="px-6 pb-6">
        <template v-if="selectedRunDetail?.run">
          <el-descriptions size="small" :column="2" title="基本信息" border class="mb-6">
            <el-descriptions-item label="分支">{{ selectedRunDetail.run.branch || '—' }}</el-descriptions-item>
            <el-descriptions-item label="触发方式">{{ selectedRunDetail.run.trigger_type || '—' }}</el-descriptions-item>
            <el-descriptions-item label="提交">{{ selectedRunDetail.run.commit_id || '—' }}</el-descriptions-item>
            <el-descriptions-item label="审批单">{{ selectedRunDetail.run.approval_ticket_id || '—' }}</el-descriptions-item>
          </el-descriptions>
          
          <div class="flex flex-col gap-4">
            <div class="font-semibold text-gray-900">阶段状态</div>
            <div class="flex flex-col gap-3">
              <div class="flex justify-between items-center p-3 border border-gray-200 rounded-lg bg-gray-50">
                <span class="font-medium text-gray-800">Build</span>
                <el-tag size="small" :type="statusTagType(selectedRunDetail.run.build_status || selectedRunDetail.run.build_stage_status)">
                  {{ statusLabel(selectedRunDetail.run.build_status || selectedRunDetail.run.build_stage_status) }}
                </el-tag>
              </div>
              <div class="flex justify-between items-center p-3 border border-gray-200 rounded-lg bg-gray-50">
                <span class="font-medium text-gray-800">Approval</span>
                <el-tag size="small" :type="statusTagType(selectedRunDetail.run.approval_status || selectedRunDetail.run.approval_stage_status)">
                  {{ statusLabel(selectedRunDetail.run.approval_status || selectedRunDetail.run.approval_stage_status) }}
                </el-tag>
              </div>
              <div class="flex justify-between items-center p-3 border border-gray-200 rounded-lg bg-gray-50">
                <span class="font-medium text-gray-800">Deploy</span>
                <el-tag size="small" :type="statusTagType(selectedRunDetail.run.deploy_status || selectedRunDetail.run.deploy_stage_status)">
                  {{ statusLabel(selectedRunDetail.run.deploy_status || selectedRunDetail.run.deploy_stage_status) }}
                </el-tag>
              </div>
            </div>
          </div>
        </template>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
/* Scoped styles replaced with Tailwind utility classes */
</style>
