<script setup lang="ts">
defineOptions({ name: 'TicketTypes' })
import { ref, onActivated, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ticketTypeApi, departmentApi, userApi } from '../api'
import { useViewStateStore } from '../stores/viewState'

const router = useRouter()
const loading = ref(false)
const tableData = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)

const dialogVisible = ref(false)
const dialogTitle = ref('新增工单类型')
const isEdit = ref(false)
const editId = ref(0)
const form = ref<any>({ name: '', code: '', icon: '', description: '', handle_dept_id: undefined, default_assignee: undefined, priority: 'medium', auto_assign_rule: 'manual', sort: 0 })
const viewStateStore = useViewStateStore()
const seenTicketTypeVersion = ref(0)

const allDepts = ref<any[]>([])
const allUsers = ref<any[]>([])

const assignRuleOptions = [
  { label: '手动指定', value: 'manual' },
  { label: '资产负责人', value: 'resource_owner' },
  { label: '服务树负责人', value: 'service_owner' },
  { label: '部门默认人', value: 'dept_default' },
]

const priorityOptions = [
  { label: '低', value: 'low' },
  { label: '中', value: 'medium' },
  { label: '高', value: 'high' },
  { label: '紧急', value: 'urgent' },
]

async function fetchData() {
  loading.value = true
  try {
    const res: any = await ticketTypeApi.list(page.value, size.value)
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally { loading.value = false }
}

function handleAdd() {
  isEdit.value = false
  dialogTitle.value = '新增工单类型'
  form.value = { name: '', code: '', icon: '', description: '', handle_dept_id: undefined, default_assignee: undefined, priority: 'medium', auto_assign_rule: 'manual', sort: 0 }
  dialogVisible.value = true
}

function handleEdit(row: any) {
  isEdit.value = true
  dialogTitle.value = '编辑工单类型'
  editId.value = row.id
  form.value = {
    name: row.name,
    code: row.code || '',
    icon: row.icon || '',
    description: row.description || '',
    handle_dept_id: row.handle_dept_id || undefined,
    default_assignee: row.default_assignee || undefined,
    priority: row.priority || 'medium',
    auto_assign_rule: row.auto_assign_rule || 'manual',
    sort: row.sort || 0,
  }
  dialogVisible.value = true
}

async function submitForm() {
  if (!form.value.name) { ElMessage.warning('请输入类型名称'); return }
  try {
    const payload = {
      ...form.value,
      handle_dept_id: form.value.handle_dept_id || 0,
      default_assignee: form.value.default_assignee || 0,
    }
    if (isEdit.value) {
      await ticketTypeApi.update(editId.value, payload)
      ElMessage.success('更新成功')
    } else {
      await ticketTypeApi.create(payload)
      ElMessage.success('创建成功')
    }
    viewStateStore.markTicketTypeDirty()
    dialogVisible.value = false
    fetchData()
  } catch {}
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(`确定删除工单类型 "${row.name}" 吗？`, '提示', { type: 'warning' })
    await ticketTypeApi.delete(row.id)
    ElMessage.success('删除成功')
    viewStateStore.markTicketTypeDirty()
    fetchData()
  } catch {}
}

function ruleLabel(v: string) { return assignRuleOptions.find(o => o.value === v)?.label || v }
function priorityLabel(v: string) { return priorityOptions.find(o => o.value === v)?.label || v }

onMounted(() => {
  fetchData()
  departmentApi.all().then((res: any) => { allDepts.value = res.data || [] }).catch(() => {})
  userApi.list(1, 200).then((res: any) => { allUsers.value = res.data?.list || [] }).catch(() => {})
  seenTicketTypeVersion.value = viewStateStore.ticketTypeVersion
})

onActivated(() => {
  if (seenTicketTypeVersion.value !== viewStateStore.ticketTypeVersion) {
    seenTicketTypeVersion.value = viewStateStore.ticketTypeVersion
    fetchData()
  }
})
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>工单类型管理</span>
          <div style="display: flex; gap: 8px;">
            <el-button plain @click="router.push('/request/templates')">请求模板</el-button>
            <el-button plain @click="router.push('/approval/policies')">审批策略</el-button>
            <el-button plain @click="router.push('/notification/console')">通知联调</el-button>
            <el-button type="primary" @click="handleAdd"><el-icon><Plus /></el-icon> 新增</el-button>
          </div>
        </div>
      </template>

      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="类型名称" min-width="120" />
        <el-table-column prop="code" label="编码" width="100" />
        <el-table-column label="默认优先级" width="100">
          <template #default="{ row }">
            <el-tag :type="(row.priority === 'urgent' ? 'danger' : row.priority === 'high' ? 'warning' : 'info') as any" size="small">{{ priorityLabel(row.priority) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="分派规则" width="120">
          <template #default="{ row }">{{ ruleLabel(row.auto_assign_rule) }}</template>
        </el-table-column>
        <el-table-column prop="handle_dept_name" label="默认处理部门" width="130">
          <template #default="{ row }">{{ row.handle_dept_name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="180" show-overflow-tooltip />
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column label="操作" min-width="140" fixed="right">
          <template #default="{ row }">
            <el-button link size="small" @click="handleEdit(row)"><el-icon><Edit /></el-icon> 编辑</el-button>
            <el-button link size="small" type="danger" @click="handleDelete(row)"><el-icon><Delete /></el-icon> 删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination v-if="total > 0" style="margin-top: 16px; justify-content: flex-end;" background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page" @current-change="(p: number) => { page = p; fetchData() }" />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="520px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="类型名称"><el-input v-model="form.name" placeholder="如：故障报修" /></el-form-item>
        <el-form-item label="编码"><el-input v-model="form.code" placeholder="如：incident" /></el-form-item>
        <el-form-item label="图标"><el-input v-model="form.icon" placeholder="Element Plus 图标名" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item>
        <el-form-item label="默认优先级">
          <el-select v-model="form.priority" style="width: 100%;">
            <el-option v-for="o in priorityOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="分派规则">
          <el-select v-model="form.auto_assign_rule" style="width: 100%;">
            <el-option v-for="o in assignRuleOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="处理部门">
          <el-select v-model="form.handle_dept_id" placeholder="选择部门" clearable style="width: 100%;">
            <el-option v-for="d in allDepts" :key="d.id" :label="d.name" :value="d.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="默认处理人" v-if="form.auto_assign_rule === 'dept_default'">
          <el-select v-model="form.default_assignee" placeholder="选择处理人" clearable style="width: 100%;">
            <el-option v-for="u in allUsers" :key="u.id" :label="u.real_name || u.username" :value="u.id" />
          </el-select>
        </el-form-item>
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
</style>
