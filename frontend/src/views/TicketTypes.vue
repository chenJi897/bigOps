<script setup lang="ts">
defineOptions({ name: 'TicketTypes' })
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { requestTemplateApi, ticketTypeApi, approvalPolicyApi } from '../api'
import { useViewStateStore } from '../stores/viewState'

const router = useRouter()
const viewStateStore = useViewStateStore()
const loading = ref(false)
const tableData = ref<any[]>([])
const ticketTypeOptions = ref<any[]>([])
const policyOptions = ref<any[]>([])

// 新增/编辑对话框
const dialogVisible = ref(false)
const dialogTitle = ref('新增工单模板')
const isEdit = ref(false)
const editId = ref(0)
const form = ref<any>({
  name: '', code: '', category: 'resource', description: '', icon: '',
  type_id: undefined, form_schema: '{"fields":[]}', approval_policy_id: undefined,
  ticket_kind: 'request', auto_create_order: 1, notify_applicant: 1, sort: 0, status: 1,
})
const schemaFields = ref<any[]>([])

const ticketKindOptions = [
  { label: '请求单', value: 'request' },
  { label: '变更单', value: 'change' },
  { label: '事件单', value: 'incident' },
]

const schemaFieldTypeOptions = [
  { label: '单行文本', value: 'text' },
  { label: '多行文本', value: 'textarea' },
  { label: '数字', value: 'number' },
  { label: '下拉选择', value: 'select' },
  { label: '开关', value: 'switch' },
]

// 节点列表：根据审批策略推导
function getNodeList(row: any) {
  const nodes = ['发起']
  if (row.approval_policy_name) nodes.push('审批')
  nodes.push(row.ticket_kind === 'incident' ? '处理' : '审批')
  return nodes.join(' => ')
}

// 所属类别名称
function getTypeName(row: any) {
  const t = ticketTypeOptions.value.find((item: any) => item.id === row.type_id)
  return t?.name || row.type_name || '-'
}

async function fetchData() {
  loading.value = true
  try {
    const [templateRes, typeRes, policyRes] = await Promise.all([
      requestTemplateApi.list(),
      ticketTypeApi.all(),
      approvalPolicyApi.list(),
    ])
    tableData.value = (templateRes as any).data || []
    ticketTypeOptions.value = (typeRes as any).data || []
    policyOptions.value = (policyRes as any).data || []
  } finally { loading.value = false }
}

function handleAdd() {
  isEdit.value = false
  dialogTitle.value = '新增工单模板'
  form.value = {
    name: '', code: '', category: 'resource', description: '', icon: '',
    type_id: undefined, form_schema: '{"fields":[]}', approval_policy_id: undefined,
    ticket_kind: 'request', auto_create_order: 1, notify_applicant: 1, sort: 0, status: 1,
  }
  schemaFields.value = []
  dialogVisible.value = true
}

function handleEdit(row: any) {
  isEdit.value = true
  dialogTitle.value = '编辑工单模板'
  editId.value = row.id
  form.value = {
    name: row.name, code: row.code, category: row.category,
    description: row.description || '', icon: row.icon || '',
    type_id: row.type_id || undefined,
    form_schema: row.form_schema || '{"fields":[]}',
    approval_policy_id: row.approval_policy_id || undefined,
    execution_template: row.execution_template || '',
    ticket_kind: row.ticket_kind || 'request',
    auto_create_order: row.auto_create_order ?? 1,
    notify_applicant: row.notify_applicant ?? 1,
    sort: row.sort || 0, status: row.status ?? 1,
  }
  schemaFields.value = parseSchemaFields(form.value.form_schema)
  dialogVisible.value = true
}

async function submitForm() {
  if (!form.value.name) { ElMessage.warning('请输入模板名称'); return }
  if (!form.value.type_id) { ElMessage.warning('请选择所属类别'); return }
  if (!form.value.code) { form.value.code = form.value.name }
  try {
    const payload = { ...form.value, approval_policy_id: form.value.approval_policy_id || 0 }
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
  await ElMessageBox.confirm(`确定删除模板「${row.name}」？`, '提示', { type: 'warning' })
  await requestTemplateApi.delete(row.id)
  ElMessage.success('删除成功')
  viewStateStore.markRequestTemplateDirty()
  fetchData()
}

function parseSchemaFields(raw?: string) {
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed?.fields) ? parsed.fields.map((f: any, i: number) => ({
      key: f.key || `field_${i + 1}`, label: f.label || '', type: f.type || 'text',
      required: !!f.required, placeholder: f.placeholder || '', rows: f.rows || 3,
      default: f.default ?? '', options: Array.isArray(f.options) ? f.options : [],
    })) : []
  } catch { return [] }
}

function syncSchemaFromFields() {
  form.value.form_schema = JSON.stringify({
    fields: schemaFields.value.map(f => {
      const r: any = { key: f.key, label: f.label, type: f.type, required: f.required }
      if (f.placeholder) r.placeholder = f.placeholder
      if (f.type === 'textarea') r.rows = f.rows || 3
      if (f.default !== '' && f.default != null) r.default = f.default
      if (f.type === 'select') r.options = (f.options || []).filter((o: any) => o.label)
      return r
    })
  }, null, 2)
}

function addSchemaField() {
  schemaFields.value.push({ key: `field_${schemaFields.value.length + 1}`, label: '', type: 'text', required: false, placeholder: '', rows: 3, default: '', options: [] })
  syncSchemaFromFields()
}
function removeSchemaField(i: number) { schemaFields.value.splice(i, 1); syncSchemaFromFields() }
function addOption(f: any) { f.options.push({ label: '', value: '' }); syncSchemaFromFields() }
function removeOption(f: any, i: number) { f.options.splice(i, 1); syncSchemaFromFields() }

onMounted(() => { fetchData() })
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>工单模板</span>
          <div style="display: flex; gap: 8px;">
            <el-button plain @click="router.push('/approval/policies')">审批策略</el-button>
            <el-button type="primary" @click="handleAdd"><el-icon><Plus /></el-icon> 新增工单模板</el-button>
          </div>
        </div>
      </template>

      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="模板名" min-width="140" />
        <el-table-column label="所属类别" width="120">
          <template #default="{ row }">{{ getTypeName(row) }}</template>
        </el-table-column>
        <el-table-column label="节点列表" min-width="180">
          <template #default="{ row }">{{ getNodeList(row) }}</template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column label="启用" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="备注" min-width="140" show-overflow-tooltip />
        <el-table-column prop="updated_at" label="更新时间" width="170" />
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button link size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button link size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="640px" destroy-on-close>
      <el-form :model="form" label-width="100px">
        <el-form-item label="模板名称" required><el-input v-model="form.name" placeholder="如：生产发版" /></el-form-item>
        <el-form-item label="所属类别" required>
          <el-select v-model="form.type_id" placeholder="选择工单类别" style="width: 100%;">
            <el-option v-for="t in ticketTypeOptions" :key="t.id" :label="t.name" :value="t.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="单据类型">
          <el-select v-model="form.ticket_kind" style="width: 100%;">
            <el-option v-for="o in ticketKindOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="审批策略">
          <el-select v-model="form.approval_policy_id" clearable placeholder="不绑定则无需审批" style="width: 100%;">
            <el-option v-for="p in policyOptions" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item>

        <el-form-item label="表单字段">
          <div class="schema-builder">
            <div class="schema-builder-head">
              <span>字段配置</span>
              <el-button type="primary" plain size="small" @click="addSchemaField">新增字段</el-button>
            </div>
            <div v-if="schemaFields.length" class="schema-field-list">
              <div v-for="(field, index) in schemaFields" :key="index" class="schema-field-card">
                <div class="schema-field-head">
                  <span>字段 {{ index + 1 }}</span>
                  <el-button link type="danger" size="small" @click="removeSchemaField(index)">删除</el-button>
                </div>
                <el-form label-width="80px">
                  <el-form-item label="Key"><el-input v-model="field.key" size="small" @input="syncSchemaFromFields" /></el-form-item>
                  <el-form-item label="名称"><el-input v-model="field.label" size="small" @input="syncSchemaFromFields" /></el-form-item>
                  <el-form-item label="类型">
                    <el-select v-model="field.type" size="small" style="width:100%;" @change="syncSchemaFromFields">
                      <el-option v-for="o in schemaFieldTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
                    </el-select>
                  </el-form-item>
                  <el-form-item label="必填"><el-switch v-model="field.required" size="small" @change="syncSchemaFromFields" /></el-form-item>
                  <el-form-item label="选项" v-if="field.type === 'select'">
                    <div style="width:100%;">
                      <div v-for="(opt, oi) in field.options" :key="oi" style="display:flex;gap:6px;margin-bottom:4px;">
                        <el-input v-model="opt.label" placeholder="标签" size="small" @input="syncSchemaFromFields" />
                        <el-input v-model="opt.value" placeholder="值" size="small" @input="syncSchemaFromFields" />
                        <el-button link type="danger" size="small" @click="removeOption(field, Number(oi))">删</el-button>
                      </div>
                      <el-button plain size="small" @click="addOption(field)">新增选项</el-button>
                    </div>
                  </el-form-item>
                </el-form>
              </div>
            </div>
            <el-empty v-else description="暂未配置字段" :image-size="40" />
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
.schema-builder { width: 100%; border: 1px solid #e5e7eb; border-radius: 8px; padding: 12px; background: #fafafa; }
.schema-builder-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; font-weight: 600; }
.schema-field-list { display: flex; flex-direction: column; gap: 10px; }
.schema-field-card { border: 1px solid #dbe4ee; border-radius: 8px; padding: 10px; background: #fff; }
.schema-field-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; font-weight: 600; font-size: 13px; }
</style>
