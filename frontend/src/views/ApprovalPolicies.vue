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
  <div class="h-full flex flex-col">
    <!-- Header -->
    <div class="flex justify-between items-center mb-4">
      <div class="text-lg font-bold text-gray-900">审批策略</div>
      <el-button type="primary" @click="handleAdd"><el-icon class="mr-1"><Plus /></el-icon> 新增策略</el-button>
    </div>

    <!-- Table -->
    <div class="flex-1 bg-white border border-gray-200 rounded-lg shadow-sm flex flex-col overflow-hidden">
      <el-table :data="tableData" v-loading="loading" class="flex-1 w-full" stripe>
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="code" label="编码" width="140" />
        <el-table-column prop="scope" label="范围" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.scope === 'change' ? '变更单' : '请求单' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="阶段数" width="90" align="center">
          <template #default="{ row }">
            <span class="font-bold text-gray-700">{{ row.stages?.length || 0 }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="text-gray-500 text-sm">{{ row.description || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="140" align="center">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Dialog -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="760px" destroy-on-close align-center>
      <div class="mb-4 pl-3 border-l-4 border-blue-500 text-sm font-bold text-gray-800">基础信息</div>
      <el-form :model="form" label-width="100px" @submit.prevent>
        <div class="grid grid-cols-2 gap-x-6 gap-y-2">
          <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="编码"><el-input v-model="form.code" /></el-form-item>
          <el-form-item label="范围">
            <el-select v-model="form.scope" class="w-full">
              <el-option v-for="item in scopeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
        </div>
        <el-form-item label="描述" class="mt-2"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item>
      </el-form>

      <div class="flex items-center justify-between mb-4 mt-6">
        <div class="pl-3 border-l-4 border-blue-500 text-sm font-bold text-gray-800">审批阶段</div>
        <el-button link type="primary" @click="addStage"><el-icon class="mr-1"><Plus /></el-icon>新增阶段</el-button>
      </div>

      <div class="flex flex-col gap-4 max-h-[420px] overflow-y-auto pr-2">
        <div v-for="(stage, index) in form.stages" :key="Number(index)" class="border border-gray-200 rounded-xl p-4 bg-gray-50/50">
          <div class="flex justify-between items-center mb-4">
            <span class="font-bold text-gray-700 text-sm">阶段 {{ Number(index) + 1 }}</span>
            <el-button v-if="form.stages.length > 1" link type="danger" @click="removeStage(Number(index))">删除</el-button>
          </div>
          <el-form :model="stage" label-width="90px" @submit.prevent>
            <div class="grid grid-cols-2 gap-x-6">
              <el-form-item label="名称"><el-input v-model="stage.name" /></el-form-item>
              <el-form-item label="审批人类型">
                <el-select v-model="stage.approver_type" class="w-full">
                  <el-option v-for="item in approverTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="固定用户" v-if="stage.approver_type === 'fixed_user'">
                <el-select v-model="stage.approver_config.user_ids" multiple clearable class="w-full">
                  <el-option v-for="item in userOptions" :key="item.id" :label="item.real_name || item.username" :value="item.id" />
                </el-select>
              </el-form-item>
              <el-form-item label="固定角色" v-if="stage.approver_type === 'fixed_role'">
                <el-select v-model="stage.approver_config.role_names" multiple clearable class="w-full">
                  <el-option v-for="item in roleOptions" :key="item.id" :label="item.name" :value="item.name" />
                </el-select>
              </el-form-item>
              <el-form-item label="通过规则">
                <el-radio-group v-model="stage.pass_rule">
                  <el-radio value="all">全部通过</el-radio>
                  <el-radio value="any">任一通过</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="超时小时"><el-input-number v-model="stage.timeout_hours" :min="1" class="w-full" /></el-form-item>
            </div>
          </el-form>
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end gap-2 pt-2">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitForm">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
</style>
