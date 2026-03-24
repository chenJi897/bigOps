<script setup lang="ts">
defineOptions({ name: 'RequestTemplates' })
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { approvalPolicyApi, requestTemplateApi, ticketTypeApi } from '../api'
import { useViewStateStore } from '../stores/viewState'

const loading = ref(false)
const tableData = ref<any[]>([])
const policyOptions = ref<any[]>([])
const ticketTypeOptions = ref<any[]>([])
const dialogVisible = ref(false)
const dialogTitle = ref('新增请求模板')
const isEdit = ref(false)
const editId = ref(0)
const viewStateStore = useViewStateStore()
const form = ref<any>({
  name: '', code: '', category: 'resource', description: '', icon: '',
  type_id: undefined, form_schema: '{"fields":[]}', approval_policy_id: undefined, execution_template: '',
  ticket_kind: 'request', auto_create_order: 1, notify_applicant: 1, sort: 0,
})
const schemaFields = ref<any[]>([])

const categoryOptions = [
  { label: '资源申请', value: 'resource' },
  { label: '权限申请', value: 'access' },
  { label: '变更申请', value: 'change' },
  { label: '其他', value: 'other' },
]

const ticketKindOptions = [
  { label: '请求单', value: 'request' },
  { label: '变更单', value: 'change' },
]

const schemaFieldTypeOptions = [
  { label: '单行文本', value: 'text' },
  { label: '多行文本', value: 'textarea' },
  { label: '数字', value: 'number' },
  { label: '下拉选择', value: 'select' },
  { label: '开关', value: 'switch' },
]

function createSchemaField(index = 0) {
  return {
    key: `field_${index + 1}`,
    label: `字段${index + 1}`,
    type: 'text',
    required: false,
    placeholder: '',
    rows: 3,
    default: '',
    options: [] as Array<{ label: string; value: string }>,
  }
}

function parseSchemaFields(raw?: string) {
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    const fields = Array.isArray(parsed?.fields) ? parsed.fields : []
    return fields.map((field: any, index: number) => ({
      key: field.key || `field_${index + 1}`,
      label: field.label || `字段${index + 1}`,
      type: field.type || 'text',
      required: !!field.required,
      placeholder: field.placeholder || '',
      rows: field.rows || 3,
      default: field.default ?? (field.type === 'switch' ? false : ''),
      options: Array.isArray(field.options) ? field.options.map((option: any) => ({
        label: option.label || '',
        value: option.value ?? '',
      })) : [],
    }))
  } catch {
    return []
  }
}

function syncSchemaFromFields() {
  const payload = {
    fields: schemaFields.value.map(field => {
      const nextField: any = {
        key: field.key,
        label: field.label,
        type: field.type,
        required: field.required,
      }
      if (field.placeholder) nextField.placeholder = field.placeholder
      if (field.type === 'textarea') nextField.rows = field.rows || 3
      if (field.default !== '' && field.default !== undefined && field.default !== null) nextField.default = field.default
      if (field.type === 'select') {
        nextField.options = (field.options || []).filter((option: any) => option.label && option.value !== '')
      }
      return nextField
    }),
  }
  form.value.form_schema = JSON.stringify(payload, null, 2)
}

function syncFieldsFromSchemaText() {
  const parsed = parseSchemaFields(form.value.form_schema)
  if (parsed.length > 0 || form.value.form_schema.trim() === '{"fields":[]}' || form.value.form_schema.trim() === '') {
    schemaFields.value = parsed
  }
}

async function fetchData() {
  loading.value = true
  try {
    const [templateRes, policyRes, typeRes] = await Promise.all([
      requestTemplateApi.list(),
      approvalPolicyApi.list(),
      ticketTypeApi.all(),
    ])
    tableData.value = (templateRes as any).data || []
    policyOptions.value = (policyRes as any).data || []
    ticketTypeOptions.value = (typeRes as any).data || []
  } finally {
    loading.value = false
  }
}

function handleAdd() {
  isEdit.value = false
  dialogTitle.value = '新增请求模板'
  form.value = {
    name: '', code: '', category: 'resource', description: '', icon: '',
    type_id: undefined, form_schema: '{"fields":[]}', approval_policy_id: undefined, execution_template: '',
    ticket_kind: 'request', auto_create_order: 1, notify_applicant: 1, sort: 0,
  }
  schemaFields.value = []
  dialogVisible.value = true
}

function handleEdit(row: any) {
  isEdit.value = true
  dialogTitle.value = '编辑请求模板'
  editId.value = row.id
  form.value = {
    name: row.name,
    code: row.code,
    category: row.category,
    description: row.description || '',
    icon: row.icon || '',
    type_id: row.type_id || undefined,
    form_schema: row.form_schema || '{"fields":[]}',
    approval_policy_id: row.approval_policy_id || undefined,
    execution_template: row.execution_template || '',
    ticket_kind: row.ticket_kind || 'request',
    auto_create_order: row.auto_create_order ?? 1,
    notify_applicant: row.notify_applicant ?? 1,
    sort: row.sort || 0,
    status: row.status,
  }
  schemaFields.value = parseSchemaFields(form.value.form_schema)
  dialogVisible.value = true
}

function addSchemaField() {
  schemaFields.value.push(createSchemaField(schemaFields.value.length))
  syncSchemaFromFields()
}

function removeSchemaField(index: number) {
  schemaFields.value.splice(index, 1)
  syncSchemaFromFields()
}

function addOption(field: any) {
  field.options.push({ label: '', value: '' })
  syncSchemaFromFields()
}

function removeOption(field: any, index: number) {
  field.options.splice(index, 1)
  syncSchemaFromFields()
}

async function submitForm() {
  if (!form.value.name || !form.value.code) {
    ElMessage.warning('请填写名称和编码')
    return
  }
  if (!form.value.type_id) {
    ElMessage.warning('请选择绑定的工单类型')
    return
  }
  try {
    const payload = {
      ...form.value,
      type_id: form.value.type_id,
      approval_policy_id: form.value.approval_policy_id || 0,
    }
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
    await ElMessageBox.confirm(`确定删除请求模板 "${row.name}" 吗？`, '提示', { type: 'warning' })
    await requestTemplateApi.delete(row.id)
    ElMessage.success('删除成功')
    viewStateStore.markRequestTemplateDirty()
    fetchData()
  } catch {}
}

onMounted(fetchData)
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div class="page-head">
          <span>请求模板</span>
          <el-button type="primary" @click="handleAdd"><el-icon><Plus /></el-icon> 新增</el-button>
        </div>
      </template>

      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="code" label="编码" width="120" />
        <el-table-column prop="category" label="分类" width="110" />
        <el-table-column prop="ticket_kind" label="单据类型" width="100" />
        <el-table-column prop="type_name" label="绑定类型" width="140">
          <template #default="{ row }">{{ row.type_name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="approval_policy_name" label="审批策略" width="150">
          <template #default="{ row }">{{ row.approval_policy_name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="execution_template" label="执行模板" width="140">
          <template #default="{ row }">{{ row.execution_template || '-' }}</template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column label="操作" fixed="right" min-width="160">
          <template #default="{ row }">
            <el-button link size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button link size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="620px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="编码"><el-input v-model="form.code" /></el-form-item>
        <el-form-item label="分类">
          <el-select v-model="form.category" style="width: 100%;">
            <el-option v-for="item in categoryOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="单据类型">
          <el-select v-model="form.ticket_kind" style="width: 100%;">
            <el-option v-for="item in ticketKindOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="工单类型">
          <el-select v-model="form.type_id" placeholder="绑定一个底层工单类型" style="width: 100%;">
            <el-option v-for="item in ticketTypeOptions" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="审批策略">
          <el-select v-model="form.approval_policy_id" clearable placeholder="可不绑定审批策略" style="width: 100%;">
            <el-option v-for="item in policyOptions" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="执行模板"><el-input v-model="form.execution_template" /></el-form-item>
        <el-form-item label="图标"><el-input v-model="form.icon" /></el-form-item>
        <el-form-item label="表单字段">
          <div class="schema-builder">
            <div class="schema-builder-head">
              <span>可视化字段编辑</span>
              <el-button type="primary" plain @click="addSchemaField">新增字段</el-button>
            </div>
            <div v-if="schemaFields.length" class="schema-field-list">
              <div v-for="(field, index) in schemaFields" :key="index" class="schema-field-card">
                <div class="schema-field-head">
                  <span>字段 {{ index + 1 }}</span>
                  <el-button link type="danger" @click="removeSchemaField(index)">删除</el-button>
                </div>
                <el-form label-width="90px">
                  <el-form-item label="字段Key"><el-input v-model="field.key" @input="syncSchemaFromFields" /></el-form-item>
                  <el-form-item label="显示名称"><el-input v-model="field.label" @input="syncSchemaFromFields" /></el-form-item>
                  <el-form-item label="类型">
                    <el-select v-model="field.type" style="width: 100%;" @change="syncSchemaFromFields">
                      <el-option v-for="item in schemaFieldTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
                    </el-select>
                  </el-form-item>
                  <el-form-item label="占位文案"><el-input v-model="field.placeholder" @input="syncSchemaFromFields" /></el-form-item>
                  <el-form-item label="必填">
                    <el-switch v-model="field.required" @change="syncSchemaFromFields" />
                  </el-form-item>
                  <el-form-item label="默认值">
                    <el-input v-if="field.type !== 'switch'" v-model="field.default" @input="syncSchemaFromFields" />
                    <el-switch v-else v-model="field.default" @change="syncSchemaFromFields" />
                  </el-form-item>
                  <el-form-item label="行数" v-if="field.type === 'textarea'">
                    <el-input-number v-model="field.rows" :min="2" @change="syncSchemaFromFields" />
                  </el-form-item>
                  <el-form-item label="选项" v-if="field.type === 'select'">
                    <div class="schema-options">
                      <div v-for="(option, optionIndex) in field.options" :key="Number(optionIndex)" class="schema-option-row">
                        <el-input v-model="option.label" placeholder="标签" @input="syncSchemaFromFields" />
                        <el-input v-model="option.value" placeholder="值" @input="syncSchemaFromFields" />
                        <el-button link type="danger" @click="removeOption(field, Number(optionIndex))">删除</el-button>
                      </div>
                      <el-button plain @click="addOption(field)">新增选项</el-button>
                    </div>
                  </el-form-item>
                </el-form>
              </div>
            </div>
            <el-empty v-else description="暂未配置动态字段" :image-size="60" />
          </div>
        </el-form-item>
        <el-form-item label="表单Schema">
          <el-input v-model="form.form_schema" type="textarea" :rows="6" @blur="syncFieldsFromSchemaText" />
        </el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
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
.page-head { display: flex; justify-content: space-between; align-items: center; }
.schema-builder {
  width: 100%;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 12px;
  background: #fafafa;
}
.schema-builder-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-weight: 700;
}
.schema-field-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.schema-field-card {
  border: 1px solid #dbe4ee;
  border-radius: 12px;
  padding: 12px;
  background: #fff;
}
.schema-field-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-weight: 700;
}
.schema-options {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.schema-option-row {
  display: grid;
  grid-template-columns: 1fr 1fr auto;
  gap: 8px;
  align-items: center;
}
</style>
