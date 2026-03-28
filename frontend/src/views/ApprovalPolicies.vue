<script setup lang="ts">
defineOptions({ name: 'ApprovalPolicies' })
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { approvalPolicyApi, roleApi, userApi } from '../api'

const loading = ref(false)
const tableData = ref<any[]>([])
const roleOptions = ref<any[]>([])
const userOptions = ref<any[]>([])
const dialogVisible = ref(false)
const dialogTitle = ref('新增审批策略')
const isEdit = ref(false)
const editId = ref(0)
const form = ref<any>({
  name: '', code: '', description: '', scope: 'request',
  stages: [
    { stage_no: 1, name: '一级审批', stage_type: 'serial', approver_type: 'dept_leader', approver_config: { user_ids: [], role_names: [] }, pass_rule: 'all', timeout_hours: 24, required: 1, sort: 0 },
  ],
})

const scopeOptions = [
  { label: '请求单', value: 'request' },
  { label: '变更单', value: 'change' },
]

const approverTypeOptions = [
  { label: '部门负责人', value: 'dept_leader' },
  { label: '固定用户', value: 'fixed_user' },
  { label: '固定角色', value: 'fixed_role' },
  { label: '服务负责人', value: 'service_owner' },
]

function createStage(stageNo = 1) {
  return {
    stage_no: stageNo,
    name: `第${stageNo}级审批`,
    stage_type: 'serial',
    approver_type: 'dept_leader',
    approver_config: { user_ids: [] as number[], role_names: [] as string[] },
    pass_rule: 'all',
    timeout_hours: 24,
    required: 1,
    sort: 0,
  }
}

async function fetchData() {
  loading.value = true
  try {
    const [policyRes, roleRes, userRes] = await Promise.all([
      approvalPolicyApi.list(),
      roleApi.list(1, 200),
      userApi.list(1, 200),
    ])
    tableData.value = (policyRes as any).data || []
    roleOptions.value = (roleRes as any).data?.list || []
    userOptions.value = (userRes as any).data?.list || []
  } finally {
    loading.value = false
  }
}

function handleAdd() {
  isEdit.value = false
  dialogTitle.value = '新增审批策略'
  form.value = {
    name: '', code: '', description: '', scope: 'request',
    stages: [createStage(1)],
  }
  dialogVisible.value = true
}

function handleEdit(row: any) {
  isEdit.value = true
  dialogTitle.value = '编辑审批策略'
  editId.value = row.id
  form.value = {
    name: row.name,
    code: row.code,
    description: row.description || '',
    scope: row.scope || 'request',
    enabled: row.enabled,
    stages: (row.stages || []).map((stage: any) => ({
      stage_no: stage.stage_no,
      name: stage.name,
      stage_type: stage.stage_type || 'serial',
      approver_type: stage.approver_type,
      approver_config: parseApproverConfig(stage.approver_config),
      pass_rule: stage.pass_rule || 'all',
      timeout_hours: stage.timeout_hours || 24,
      required: stage.required ?? 1,
      sort: stage.sort || 0,
    })),
  }
  dialogVisible.value = true
}

function parseApproverConfig(raw: any) {
  if (!raw) return { user_ids: [], role_names: [] }
  if (typeof raw === 'string') {
    try {
      const parsed = JSON.parse(raw)
      return { user_ids: parsed.user_ids || [], role_names: parsed.role_names || [] }
    } catch {
      return { user_ids: [], role_names: [] }
    }
  }
  return { user_ids: raw.user_ids || [], role_names: raw.role_names || [] }
}

function addStage() {
  const nextNo = form.value.stages.length + 1
  form.value.stages.push(createStage(nextNo))
}

function removeStage(index: number) {
  form.value.stages.splice(index, 1)
  form.value.stages.forEach((stage: any, idx: number) => {
    stage.stage_no = idx + 1
    if (!stage.name) {
      stage.name = `第${idx + 1}级审批`
    }
  })
}

async function submitForm() {
  if (!form.value.name || !form.value.code) {
    ElMessage.warning('请填写名称和编码')
    return
  }
  if (!form.value.stages.length) {
    ElMessage.warning('请至少配置一个审批阶段')
    return
  }
  const payload = {
    ...form.value,
    stages: form.value.stages.map((stage: any, index: number) => ({
      ...stage,
      stage_no: index + 1,
      approver_config: JSON.stringify(stage.approver_config || { user_ids: [], role_names: [] }),
    })),
  }
  try {
    if (isEdit.value) {
      await approvalPolicyApi.update(editId.value, payload)
      ElMessage.success('更新成功')
    } else {
      await approvalPolicyApi.create(payload)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchData()
  } catch {}
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(`确定删除审批策略 "${row.name}" 吗？`, '提示', { type: 'warning' })
    await approvalPolicyApi.delete(row.id)
    ElMessage.success('删除成功')
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
          <span>审批策略</span>
          <el-button type="primary" @click="handleAdd"><el-icon><Plus /></el-icon> 新增</el-button>
        </div>
      </template>

      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="code" label="编码" width="140" />
        <el-table-column prop="scope" label="范围" width="100" />
        <el-table-column label="阶段数" width="90">
          <template #default="{ row }">{{ row.stages?.length || 0 }}</template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip />
        <el-table-column label="操作" fixed="right" min-width="160">
          <template #default="{ row }">
            <el-button link size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button link size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="760px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="编码"><el-input v-model="form.code" /></el-form-item>
        <el-form-item label="范围">
          <el-select v-model="form.scope" style="width: 100%;">
            <el-option v-for="item in scopeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item>
      </el-form>

      <div class="stage-head">
        <span>审批阶段</span>
        <el-button type="primary" plain @click="addStage">新增阶段</el-button>
      </div>

      <div class="stage-list">
        <div v-for="(stage, index) in form.stages" :key="Number(index)" class="stage-card">
          <div class="stage-card-head">
            <span>阶段 {{ Number(index) + 1 }}</span>
            <el-button v-if="form.stages.length > 1" link type="danger" @click="removeStage(Number(index))">删除</el-button>
          </div>
          <el-form :model="stage" label-width="90px">
            <el-form-item label="名称"><el-input v-model="stage.name" /></el-form-item>
            <el-form-item label="审批人类型">
              <el-select v-model="stage.approver_type" style="width: 100%;">
                <el-option v-for="item in approverTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
            <el-form-item label="固定用户" v-if="stage.approver_type === 'fixed_user'">
              <el-select v-model="stage.approver_config.user_ids" multiple clearable style="width: 100%;">
                <el-option v-for="item in userOptions" :key="item.id" :label="item.real_name || item.username" :value="item.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="固定角色" v-if="stage.approver_type === 'fixed_role'">
              <el-select v-model="stage.approver_config.role_names" multiple clearable style="width: 100%;">
                <el-option v-for="item in roleOptions" :key="item.id" :label="item.name" :value="item.name" />
              </el-select>
            </el-form-item>
            <el-form-item label="通过规则">
              <el-radio-group v-model="stage.pass_rule">
                <el-radio value="all">全部通过</el-radio>
                <el-radio value="any">任一通过</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="超时小时"><el-input-number v-model="stage.timeout_hours" :min="1" /></el-form-item>
          </el-form>
        </div>
      </div>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
.page-head, .stage-head { display: flex; justify-content: space-between; align-items: center; }
.stage-head { margin: 12px 0; }
.stage-list { display: flex; flex-direction: column; gap: 12px; max-height: 420px; overflow: auto; }
.stage-card { border: 1px solid #e5e7eb; border-radius: 12px; padding: 14px; background: #fafafa; }
.stage-card-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; font-weight: 700; }
</style>
