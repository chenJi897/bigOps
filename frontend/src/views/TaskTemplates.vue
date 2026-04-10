<template>
  <div class="p-6">
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-800">任务模板</h1>
        <p class="text-gray-500 mt-1">管理可复用的任务模板，支持脚本编辑和类型约束</p>
      </div>
      <div class="flex gap-3">
        <el-button type="primary" @click="openCreateDialog">
          <el-icon><Plus /></el-icon>
          新建任务
        </el-button>
        <el-button @click="refreshList">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <div class="bg-white rounded-2xl shadow-sm p-4 mb-6 flex gap-4 flex-wrap">
      <el-input v-model="searchQuery" placeholder="搜索任务名称..." class="!w-72" clearable @keyup.enter="loadTasks">
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
      <el-select v-model="typeFilter" placeholder="任务类型" clearable class="!w-36" @change="loadTasks">
        <el-option label="全部类型" value="" />
        <el-option v-for="t in taskTypeOptions" :key="t.value" :label="t.label" :value="t.value" />
      </el-select>
      <el-select v-model="statusFilter" placeholder="状态" clearable class="!w-28" @change="loadTasks">
        <el-option label="全部" value="" />
        <el-option label="启用" value="1" />
        <el-option label="禁用" value="0" />
      </el-select>
    </div>

    <el-table :data="tasks" stripe v-loading="loading" class="rounded-2xl overflow-hidden" empty-text="暂无任务">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="任务名称" min-width="200" show-overflow-tooltip />
      <el-table-column label="类型" width="120">
        <template #default="{ row }">
          <el-tag size="small" :type="row.task_type === 'script' ? 'primary' : 'info'">{{ taskTypeLabel(row.task_type) }}</el-tag>
          <el-tag v-if="row.task_type === 'script' && row.script_type" size="small" type="warning" class="ml-1">{{ row.script_type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="timeout" label="超时(s)" width="80" align="center" />
      <el-table-column prop="run_as_user" label="执行用户" width="90" />
      <el-table-column label="风险" width="90" align="center">
        <template #default="{ row }">
          <el-tag :type="riskTagType(row.risk_level)" size="small">{{ riskLabel(row.risk_level) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="80" align="center">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="160" />
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link size="small" @click="openEditDialog(row)">编辑</el-button>
          <el-button v-if="row.require_approval === 1" type="warning" link size="small" @click="requestApproval(row)">申请审批</el-button>
          <el-button type="success" link size="small" @click="openExecuteDialog(row)">执行</el-button>
          <el-button type="danger" link size="small" @click="deleteTask(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="flex justify-between items-center mt-6">
      <div class="text-sm text-gray-500">共 {{ total }} 个任务</div>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
        v-model:current-page="currentPage" v-model:page-size="pageSize" :page-sizes="[15, 30, 50]"
        @current-change="loadTasks" @size-change="loadTasks" />
    </div>

    <!-- 创建/编辑弹窗 -->
    <el-dialog v-model="formVisible" :title="isEdit ? '编辑任务' : '新建任务'" width="780px" append-to-body destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="formRules" label-position="top">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="任务名称" prop="name">
              <el-input v-model="form.name" placeholder="例如：磁盘清理脚本" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="任务大类" prop="task_type">
              <el-select v-model="form.task_type" class="w-full" @change="onTaskTypeChange">
                <el-option v-for="t in taskTypeOptions" :key="t.value" :label="t.label" :value="t.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="脚本语言" prop="script_type" v-if="form.task_type === 'script'">
              <el-select v-model="form.script_type" class="w-full" @change="onScriptTypeChange">
                <el-option v-for="t in scriptTypeOptions" :key="t.value" :label="t.label" :value="t.value" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="超时时间(秒)">
              <el-input-number v-model="form.timeout" :min="5" :max="86400" :step="10" class="w-full" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="执行用户">
              <el-input v-model="form.run_as_user" placeholder="root" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="描述">
              <el-input v-model="form.description" placeholder="任务描述" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item prop="script_content" v-if="form.task_type === 'script'">
          <ScriptEditor v-model="form.script_content" :language="form.script_type" @validate="onEditorValidate" />
        </el-form-item>

        <div v-if="form.task_type !== 'script'" class="p-4 bg-slate-50 rounded-lg border border-slate-200 text-sm text-slate-500">
          {{ taskTypeLabel(form.task_type) }} 类型任务的配置区域待扩展（当前仅支持脚本类型的完整编辑）
        </div>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">{{ isEdit ? '保存' : '创建' }}</el-button>
      </template>
    </el-dialog>

    <!-- 执行弹窗 -->
    <el-dialog v-model="execVisible" title="执行任务" width="500px" append-to-body>
      <div class="mb-3 text-sm text-slate-600">任务：<strong>{{ execTask?.name }}</strong></div>
      <el-form label-position="top">
        <el-form-item label="目标主机 IP（每行一个）">
          <el-input v-model="execHostsText" type="textarea" :rows="5" placeholder="10.0.0.1&#10;10.0.0.2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="execVisible = false">取消</el-button>
        <el-button type="primary" @click="executeTask" :loading="execLoading">立即执行</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { taskApi } from '../api'
import ScriptEditor from '../components/ScriptEditor.vue'

const router = useRouter()
const loading = ref(false)
const submitting = ref(false)
const tasks = ref<any[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(15)
const searchQuery = ref('')
const typeFilter = ref('')
const statusFilter = ref('')

const formVisible = ref(false)
const isEdit = ref(false)
const editId = ref(0)
const formRef = ref<FormInstance>()
const editorWarnings = ref<any[]>([])

const taskTypeOptions = [
  { label: '脚本执行', value: 'script' },
  { label: '文件分发', value: 'file_transfer' },
  { label: 'API 调用', value: 'api_call' },
]
const scriptTypeOptions = [
  { label: 'Bash', value: 'bash' },
  { label: 'Python', value: 'python' },
  { label: 'Shell (sh)', value: 'sh' },
  { label: 'PowerShell', value: 'powershell' },
]

const form = reactive({
  name: '',
  task_type: 'script',
  script_type: 'bash',
  script_content: '',
  timeout: 60,
  run_as_user: 'root',
  description: '',
})

const formRules: FormRules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  task_type: [{ required: true, message: '请选择任务大类', trigger: 'change' }],
  script_type: [{ required: true, message: '请选择脚本语言', trigger: 'change' }],
  script_content: [{
    validator: (_rule: any, _value: any, callback: any) => {
      if (form.task_type !== 'script') return callback()
      if (!form.script_content.trim()) return callback(new Error('脚本内容不能为空'))
      if (editorWarnings.value.some(w => w.level === 'error')) return callback(new Error('脚本存在阻断性错误，请修复'))
      callback()
    },
    trigger: 'change',
  }],
}

const execVisible = ref(false)
const execTask = ref<any>(null)
const execHostsText = ref('')
const execLoading = ref(false)

function taskTypeLabel(t: string) {
  const m: Record<string, string> = { script: '脚本执行', shell: '脚本执行', file_transfer: '文件分发', api_call: 'API 调用', python: '脚本执行' }
  return m[t] || t
}

function riskLabel(level: string) {
  const m: Record<string, string> = { low: '低', medium: '中', high: '高', critical: '极高' }
  return m[level] || level || '低'
}

function riskTagType(level: string) {
  const m: Record<string, string> = { low: 'success', medium: 'warning', high: 'danger', critical: 'danger' }
  return m[level] || 'info'
}

async function requestApproval(row: any) {
  try {
    await ElMessageBox.confirm(
      `任务「${row.name}」为${riskLabel(row.risk_level)}风险，需要审批后才能执行。是否提交审批申请？`,
      '审批确认',
      { type: 'warning', confirmButtonText: '提交审批', cancelButtonText: '取消' }
    )
    await taskApi.requestApproval(row.id, [])
    ElMessage.success('审批申请已提交，请等待管理员审批')
  } catch (e: any) {
    if (e !== 'cancel' && e?.message) ElMessage.error(e.message)
  }
}

async function loadTasks() {
  loading.value = true
  try {
    const res = await taskApi.list({ page: currentPage.value, size: pageSize.value, keyword: searchQuery.value.trim(), task_type: typeFilter.value })
    tasks.value = (res as any).data?.list || []
    total.value = Number((res as any).data?.total || 0)
  } finally { loading.value = false }
}

function resetForm() {
  form.name = ''; form.task_type = 'script'; form.script_type = 'bash'; form.script_content = ''
  form.timeout = 60; form.run_as_user = 'root'; form.description = ''
  editorWarnings.value = []
}

function openCreateDialog() {
  isEdit.value = false; editId.value = 0
  resetForm()
  formVisible.value = true
}

function openEditDialog(row: any) {
  isEdit.value = true; editId.value = row.id
  let tt = row.task_type || 'script'
  if (['shell', 'bash', 'python', 'sh', 'powershell'].includes(tt)) { tt = 'script' }
  form.name = row.name
  form.task_type = tt
  form.script_type = row.script_type || (tt === 'script' ? 'bash' : '')
  form.script_content = row.script_content || ''
  form.timeout = row.timeout || 60
  form.run_as_user = row.run_as_user || 'root'
  form.description = row.description || ''
  editorWarnings.value = []
  formVisible.value = true
}

function onTaskTypeChange(newType: string) {
  if (newType !== 'script') {
    form.script_type = ''; form.script_content = ''
  } else {
    if (!form.script_type) form.script_type = 'bash'
  }
}

function onScriptTypeChange() {
  if (form.script_content.trim()) {
    ElMessageBox.confirm('切换脚本语言将清空已有内容，是否继续？', '确认', { type: 'warning' })
      .then(() => { form.script_content = '' })
      .catch(() => {})
  }
}

function onEditorValidate(w: any[]) { editorWarnings.value = w }

async function submitForm() {
  if (!formRef.value) return
  try { await formRef.value.validate() } catch { return }

  submitting.value = true
  try {
    const payload: any = {
      name: form.name,
      task_type: form.task_type,
      script_type: form.task_type === 'script' ? form.script_type : '',
      script_content: form.task_type === 'script' ? form.script_content : '',
      timeout: form.timeout,
      run_as_user: form.run_as_user,
      description: form.description,
    }
    if (isEdit.value) {
      await taskApi.update(editId.value, payload)
      ElMessage.success('更新成功')
    } else {
      await taskApi.create(payload)
      ElMessage.success('创建成功')
    }
    formVisible.value = false
    await loadTasks()
  } finally { submitting.value = false }
}

async function deleteTask(row: any) {
  try {
    await ElMessageBox.confirm(`确定删除任务「${row.name}」吗？删除后不可恢复。`, '警告', { type: 'warning' })
    await taskApi.delete(row.id)
    ElMessage.success('删除成功')
    await loadTasks()
  } catch {}
}

function openExecuteDialog(row: any) {
  if (row.require_approval === 1) {
    ElMessageBox.confirm(
      `任务「${row.name}」为${riskLabel(row.risk_level)}风险，需审批通过后才能执行。确定继续？`,
      '风险提示',
      { type: 'warning', confirmButtonText: '继续', cancelButtonText: '取消' }
    ).then(() => {
      execTask.value = row; execHostsText.value = ''; execVisible.value = true
    }).catch(() => {})
    return
  }
  execTask.value = row; execHostsText.value = ''; execVisible.value = true
}

async function executeTask() {
  if (!execTask.value) return
  const ips = execHostsText.value.split(/[\n,;]+/).map(s => s.trim()).filter(Boolean)
  if (!ips.length) { ElMessage.warning('请输入至少一个目标主机 IP'); return }
  execLoading.value = true
  try {
    const res = await taskApi.execute(execTask.value.id, { host_ips: ips })
    const execId = (res as any).data?.id
    ElMessage.success('执行已下发')
    execVisible.value = false
    if (execId) router.push(`/task/executions/${execId}`)
  } finally { execLoading.value = false }
}

function refreshList() { loadTasks(); ElMessage.success('刷新成功') }

onMounted(() => { loadTasks() })
</script>
