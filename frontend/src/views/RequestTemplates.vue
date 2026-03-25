<script setup lang="ts">
defineOptions({ name: 'RequestTemplates' })
import { computed, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { requestTemplateApi, userApi, departmentApi } from '../api'
import { useViewStateStore } from '../stores/viewState'

type TemplateNode = {
  node_id: string
  name: string
  approve_mode: 'or' | 'and' | 'none'
  handler_ids: number[]
  optional_handler_ids: number[]
  notify_user_ids: number[]
  node_form_schema: string
  callback_config: string
}

const loading = ref(false)
const tableData = ref<any[]>([])
const userOptions = ref<any[]>([])
const deptOptions = ref<any[]>([])
const dialogVisible = ref(false)
const dialogTitle = ref('新增工单模板')
const isEdit = ref(false)
const editId = ref(0)
const activeTab = ref<'basic' | 'nodes'>('basic')
const nodeDialogVisible = ref(false)
const nodeDialogTitle = ref('新增模板节点')
const nodeEditIndex = ref(-1)
const viewStateStore = useViewStateStore()

const form = ref<any>({
  name: '',
  code: '',
  category: 'other',
  project_name: '',
  environment_name: '',
  description: '',
  icon: '',
  type_id: undefined,
  form_schema: '{"fields":[]}',
  nodes_json: '[]',
  execution_template: '',
  ticket_kind: 'request',
  priority: 'medium',
  handle_dept_id: undefined,
  auto_assign_rule: 'manual',
  default_assignee: undefined,
  auto_create_order: 1,
  notify_applicant: 1,
  sort: 999,
  status: 1,
})

const nodeForm = ref<TemplateNode>({
  node_id: '',
  name: '',
  approve_mode: 'or',
  handler_ids: [],
  optional_handler_ids: [],
  notify_user_ids: [],
  node_form_schema: '{"fields":[]}',
  callback_config: '',
})

const categoryOptions = [
  { label: '发版申请', value: 'release' },
  { label: '权限申请', value: 'access' },
  { label: '数据库上线', value: 'db_release' },
  { label: '代码仓库', value: 'repo' },
  { label: '其他', value: 'other' },
]

const templateCategoryOptions = categoryOptions

const approveModeOptions = [
  { label: '或签(单一通过)', value: 'or' },
  { label: '会签(全部通过)', value: 'and' },
  { label: '无需审批', value: 'none' },
]

const categoryLabelMap = categoryOptions.reduce<Record<string, string>>((acc, item) => {
  acc[item.value] = item.label
  return acc
}, {})

const templateNodes = computed<TemplateNode[]>(() => {
  const raw = form.value.nodes_json
  if (!raw) return []
  try {
    const parsed = typeof raw === 'string' ? JSON.parse(raw) : raw
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
})

function generateTemplateCode(name: string) {
  const normalized = name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
  if (normalized) {
    return normalized.slice(0, 50)
  }
  return `template-${Date.now().toString(36)}`
}

function createNodeID() {
  return `node-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`
}

function syncNodesJSON(nodes: TemplateNode[]) {
  form.value.nodes_json = JSON.stringify(nodes, null, 2)
}

function buildStageSummary(row: any) {
  const raw = row?.nodes_json
  if (!raw) return '发起 => 处理'
  try {
    const nodes = Array.isArray(raw) ? raw : JSON.parse(raw)
    if (!Array.isArray(nodes) || nodes.length === 0) return '发起 => 处理'
    return ['发起', ...nodes.map((node: any, index: number) => node.name || `节点${index + 1}`), '处理'].join(' => ')
  } catch {
    return '发起 => 处理'
  }
}

function formatCategory(category?: string) {
  if (!category) return '-'
  return categoryLabelMap[category] || category
}

function getUserNames(ids: number[]) {
  if (!ids?.length) return '没有设置'
  return ids.map(id => {
    const user = userOptions.value.find((item: any) => item.id === id)
    return user ? (user.real_name || user.username) : `用户${id}`
  }).join('、')
}

function approveModeLabel(mode: string) {
  return approveModeOptions.find(item => item.value === mode)?.label || mode
}

function isEnabled(row: any) {
  return Number(row?.status) === 1
}

async function fetchData() {
  loading.value = true
  try {
    const [templateRes, userRes, deptRes] = await Promise.all([
      requestTemplateApi.list(),
      userApi.list(1, 200),
      departmentApi.all(),
    ])
    tableData.value = (templateRes as any).data || []
    userOptions.value = (userRes as any).data?.list || []
    deptOptions.value = (deptRes as any).data || []
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.value = {
    name: '',
    code: '',
    category: 'other',
    project_name: '',
    environment_name: '',
    description: '',
    icon: '',
    type_id: 0,
    form_schema: '{"fields":[]}',
    nodes_json: '[]',
    execution_template: '',
    ticket_kind: 'request',
    priority: 'medium',
    handle_dept_id: undefined,
    auto_assign_rule: 'manual',
    default_assignee: undefined,
    auto_create_order: 1,
    notify_applicant: 1,
    sort: 999,
    status: 1,
  }
  activeTab.value = 'basic'
}

function handleAdd() {
  isEdit.value = false
  dialogTitle.value = '新增工单模板'
  resetForm()
  dialogVisible.value = true
}

function handleEdit(row: any) {
  isEdit.value = true
  dialogTitle.value = '编辑工单模板'
  editId.value = row.id
  form.value = {
    name: row.name,
    code: row.code || '',
    category: row.category || 'other',
    project_name: row.project_name || '',
    environment_name: row.environment_name || '',
    description: row.description || '',
    icon: row.icon || '',
    type_id: row.type_id || 0,
    form_schema: row.form_schema || '{"fields":[]}',
    nodes_json: row.nodes_json || '[]',
    execution_template: row.execution_template || '',
    ticket_kind: row.ticket_kind || 'request',
    priority: row.priority || 'medium',
    handle_dept_id: row.handle_dept_id || undefined,
    auto_assign_rule: row.auto_assign_rule || 'manual',
    default_assignee: row.default_assignee || undefined,
    auto_create_order: row.auto_create_order ?? 1,
    notify_applicant: row.notify_applicant ?? 1,
    sort: row.sort || 999,
    status: Number(row.status) || 0,
  }
  activeTab.value = 'basic'
  dialogVisible.value = true
}

function openNodeDialog(index = -1) {
  nodeEditIndex.value = index
  if (index >= 0) {
    const node = templateNodes.value[index]
    nodeDialogTitle.value = '编辑模板节点'
    nodeForm.value = {
      node_id: node.node_id || createNodeID(),
      name: node.name,
      approve_mode: node.approve_mode || 'or',
      handler_ids: [...(node.handler_ids || [])],
      optional_handler_ids: [...(node.optional_handler_ids || [])],
      notify_user_ids: [...(node.notify_user_ids || [])],
      node_form_schema: node.node_form_schema || '{"fields":[]}',
      callback_config: node.callback_config || '',
    }
  } else {
    nodeDialogTitle.value = '新增模板节点'
    nodeForm.value = {
      node_id: createNodeID(),
      name: '',
      approve_mode: 'or',
      handler_ids: [],
      optional_handler_ids: [],
      notify_user_ids: [],
      node_form_schema: '{"fields":[]}',
      callback_config: '',
    }
  }
  nodeDialogVisible.value = true
}

function saveNode() {
  if (!nodeForm.value.name) {
    ElMessage.warning('请填写节点名')
    return
  }
  const nextNodes = [...templateNodes.value]
  const payload = { ...nodeForm.value }
  if (nodeEditIndex.value >= 0) {
    nextNodes.splice(nodeEditIndex.value, 1, payload)
  } else {
    nextNodes.push(payload)
  }
  syncNodesJSON(nextNodes)
  nodeDialogVisible.value = false
}

function deleteNode(index: number) {
  const nextNodes = [...templateNodes.value]
  nextNodes.splice(index, 1)
  syncNodesJSON(nextNodes)
}

async function submitForm() {
  if (!form.value.name) {
    ElMessage.warning('请填写模板名称')
    return
  }
  if (templateNodes.value.length < 2) {
    ElMessage.warning('至少需要配置 2 个节点，请切换到「节点配置」添加')
    activeTab.value = 'nodes'
    return
  }
  const payload = {
    ...form.value,
    code: form.value.code?.trim() || generateTemplateCode(form.value.name),
    type_id: form.value.type_id || 0,
    approval_policy_id: 0,
    nodes_json: form.value.nodes_json || '[]',
  }
  try {
    if (isEdit.value) {
      await requestTemplateApi.update(editId.value, payload)
      ElMessage.success('更新成功')
    } else {
      await requestTemplateApi.create(payload)
      ElMessage.success('创建成功')
    }
    viewStateStore.markRequestTemplateDirty()
    dialogVisible.value = false
    fetchData()
  } catch {}
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(`确定删除工单模板 "${row.name}" 吗？`, '提示', { type: 'warning' })
    await requestTemplateApi.delete(row.id)
    ElMessage.success('删除成功')
    viewStateStore.markRequestTemplateDirty()
    fetchData()
  } catch {}
}

async function toggleStatus(row: any, value: boolean) {
  const nextStatus = value ? 1 : 0
  const previousStatus = row.status
  row.status = nextStatus
  try {
    await requestTemplateApi.update(row.id, {
      name: row.name,
      code: row.code,
      category: row.category || 'other',
      project_name: row.project_name || '',
      environment_name: row.environment_name || '',
      description: row.description || '',
      icon: row.icon || '',
      type_id: row.type_id || 0,
      form_schema: row.form_schema || '{"fields":[]}',
      nodes_json: row.nodes_json || '[]',
      execution_template: row.execution_template || '',
      ticket_kind: row.ticket_kind || 'request',
      priority: row.priority || 'medium',
      handle_dept_id: row.handle_dept_id || 0,
      auto_assign_rule: row.auto_assign_rule || 'manual',
      default_assignee: row.default_assignee || 0,
      auto_create_order: row.auto_create_order ?? 1,
      notify_applicant: row.notify_applicant ?? 1,
      sort: row.sort || 999,
      status: nextStatus,
      approval_policy_id: 0,
    })
    viewStateStore.markRequestTemplateDirty()
    ElMessage.success(nextStatus === 1 ? '已启用' : '已停用')
  } catch {
    row.status = previousStatus
  }
}

onMounted(fetchData)
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div class="page-head">
          <div class="page-head-title">
            <span class="page-title">工单模板</span>
            <span class="page-subtitle">按模板维护基础信息与审批节点</span>
          </div>
          <div class="page-head-actions">
            <el-button @click="fetchData">刷新</el-button>
            <el-button type="primary" @click="handleAdd"><el-icon><Plus /></el-icon> 新增工单模板</el-button>
          </div>
        </div>
      </template>

      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="模板名" min-width="180" />
        <el-table-column prop="category" label="所属类别" width="120">
          <template #default="{ row }">{{ formatCategory(row.category) }}</template>
        </el-table-column>
        <el-table-column label="节点列表" min-width="280" show-overflow-tooltip>
          <template #default="{ row }">{{ buildStageSummary(row) }}</template>
        </el-table-column>
        <el-table-column label="启用" width="90" align="center">
          <template #default="{ row }">
            <el-switch :model-value="isEnabled(row)" @update:model-value="(value: boolean) => toggleStatus(row, value)" />
          </template>
        </el-table-column>
        <el-table-column prop="description" label="备注" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">{{ row.description || '-' }}</template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" width="180">
          <template #default="{ row }">{{ row.updated_at || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" min-width="160">
          <template #default="{ row }">
            <el-button link size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button link size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="980px">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="基础信息" name="basic" />
        <el-tab-pane label="节点配置" name="nodes" />
      </el-tabs>

      <div v-if="activeTab === 'basic'" class="template-section">
        <div class="section-title">基础信息</div>
        <el-form :model="form" label-width="100px">
          <el-form-item label="模板类别"><el-select v-model="form.category" style="width: 100%;"><el-option v-for="item in templateCategoryOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
          <el-form-item label="模板名称"><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="关联项目"><el-input v-model="form.project_name" placeholder="可选" /></el-form-item>
          <el-form-item label="关联环境"><el-input v-model="form.environment_name" placeholder="可选" /></el-form-item>
          <el-form-item label="模板排序"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
          <el-form-item label="默认优先级">
            <el-select v-model="form.priority" style="width: 100%;">
              <el-option label="低" value="low" />
              <el-option label="中" value="medium" />
              <el-option label="高" value="high" />
              <el-option label="紧急" value="urgent" />
            </el-select>
          </el-form-item>
          <el-form-item label="处理部门">
            <el-select v-model="form.handle_dept_id" placeholder="可选" clearable style="width: 100%;">
              <el-option v-for="d in deptOptions" :key="d.id" :label="d.name" :value="d.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="分派规则">
            <el-select v-model="form.auto_assign_rule" style="width: 100%;">
              <el-option label="手动指定" value="manual" />
              <el-option label="资产负责人" value="resource_owner" />
              <el-option label="服务树负责人" value="service_owner" />
              <el-option label="部门默认人" value="dept_default" />
            </el-select>
          </el-form-item>
          <el-form-item label="默认处理人" v-if="form.auto_assign_rule === 'dept_default'">
            <el-select v-model="form.default_assignee" placeholder="选择处理人" clearable style="width: 100%;">
              <el-option v-for="u in userOptions" :key="u.id" :label="u.real_name || u.username" :value="u.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="是否启用"><el-switch v-model="form.status" :active-value="1" :inactive-value="0" /></el-form-item>
          <el-form-item label="备注"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
        </el-form>
      </div>

      <div v-else class="template-section">
        <div class="section-title section-head">
          <span>节点配置</span>
          <el-button link type="primary" @click="openNodeDialog()">添加节点</el-button>
        </div>
        <el-table :data="templateNodes" border>
          <el-table-column prop="name" label="节点名" min-width="160" />
          <el-table-column label="审批方式" width="160">
            <template #default="{ row }">{{ approveModeLabel(row.approve_mode) }}</template>
          </el-table-column>
          <el-table-column label="字段数量" width="120">
            <template #default="{ row }">
              {{
                (() => {
                  try {
                    const parsed = JSON.parse(row.node_form_schema || '{"fields":[]}')
                    return Array.isArray(parsed?.fields) ? parsed.fields.length : 0
                  } catch {
                    return 0
                  }
                })()
              }}
            </template>
          </el-table-column>
          <el-table-column label="处理成员" min-width="220">
            <template #default="{ row }">{{ getUserNames(row.handler_ids) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="140">
            <template #default="{ $index }">
              <el-button link size="small" @click="openNodeDialog($index)">编辑</el-button>
              <el-button link size="small" type="danger" @click="deleteNode($index)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">提交</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="nodeDialogVisible" :title="nodeDialogTitle" width="860px">
      <div class="template-section">
        <div class="section-title">节点基础信息</div>
        <el-form :model="nodeForm" label-width="110px">
          <el-form-item label="节点名"><el-input v-model="nodeForm.name" /></el-form-item>
          <el-form-item label="审批方式">
            <el-radio-group v-model="nodeForm.approve_mode">
              <el-radio value="or">或签(单一通过)</el-radio>
              <el-radio value="and">会签(全部通过)</el-radio>
              <el-radio value="none">无需审批</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="绑定处理成员">
            <el-select v-model="nodeForm.handler_ids" multiple clearable style="width: 100%;">
              <el-option v-for="item in userOptions" :key="item.id" :label="item.real_name || item.username" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="可选处理成员">
            <el-select v-model="nodeForm.optional_handler_ids" multiple clearable style="width: 100%;">
              <el-option v-for="item in userOptions" :key="item.id" :label="item.real_name || item.username" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="绑定通知成员">
            <el-select v-model="nodeForm.notify_user_ids" multiple clearable style="width: 100%;">
              <el-option v-for="item in userOptions" :key="item.id" :label="item.real_name || item.username" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="表单可视化设计">
            <el-input v-model="nodeForm.node_form_schema" type="textarea" :rows="6" placeholder='{"fields":[]}' />
          </el-form-item>
          <el-form-item label="节点回调配置">
            <el-input v-model="nodeForm.callback_config" type="textarea" :rows="4" placeholder="可选，回调配置 JSON" />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="nodeDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveNode">应用</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
.page-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}
.page-head-title {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.page-title {
  font-size: 16px;
  font-weight: 700;
  color: #1f2937;
}
.page-subtitle {
  font-size: 12px;
  color: #6b7280;
}
.page-head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.template-section {
  padding-top: 8px;
}
.section-title {
  margin-bottom: 14px;
  padding-left: 10px;
  border-left: 4px solid #0ea5e9;
  font-size: 15px;
  font-weight: 700;
  color: #1f2937;
}
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-left: 0;
  border-left: 0;
}
</style>
