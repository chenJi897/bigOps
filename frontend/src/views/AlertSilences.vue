<script setup lang="ts">
defineOptions({ name: 'AlertSilences' })

import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { alertRuleApi, alertSilenceApi, serviceTreeApi, userApi } from '../api'

const loading = ref(false)
const dialogVisible = ref(false)
const submitting = ref(false)
const isEdit = ref(false)
const editId = ref(0)
const rows = ref<any[]>([])
const rules = ref<any[]>([])
const users = ref<any[]>([])
const serviceTrees = ref<any[]>([])

const form = ref<any>(createDefaultForm())

const userOptions = computed(() => users.value.map((item) => ({
  label: item.real_name || item.username,
  value: item.id,
})))

function createDefaultForm() {
  return {
    name: '',
    rule_id: null as number | null,
    agent_id: '',
    service_tree_id: null as number | null,
    owner_id: null as number | null,
    reason: '',
    enabled: 1,
    starts_at: '',
    ends_at: '',
  }
}

function flattenTree(nodes: any[], level = 0): any[] {
  const result: any[] = []
  nodes.forEach((node) => {
    result.push({
      id: node.id,
      label: `${'　'.repeat(level)}${level > 0 ? '└ ' : ''}${node.name}`,
    })
    if (Array.isArray(node.children) && node.children.length > 0) {
      result.push(...flattenTree(node.children, level + 1))
    }
  })
  return result
}

function createDefaultWindow() {
  const now = new Date()
  const end = new Date(now.getTime() + 2 * 60 * 60 * 1000)
  return {
    starts_at: formatDateTime(now),
    ends_at: formatDateTime(end),
  }
}

function formatDateTime(date: Date) {
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

function serviceTreeLabel(id: number) {
  if (!id) return '—'
  return serviceTrees.value.find((item) => item.id === id)?.label || `#${id}`
}

function ownerLabel(id: number) {
  if (!id) return '—'
  return userOptions.value.find((item) => item.value === id)?.label || `#${id}`
}

function ruleLabel(id: number) {
  if (!id) return '全部规则'
  return rules.value.find((item) => item.id === id)?.name || `#${id}`
}

function silenceScope(row: any) {
  const scopes = []
  if (row.rule_id) scopes.push(ruleLabel(Number(row.rule_id)))
  if (row.agent_id) scopes.push(`Agent:${row.agent_id}`)
  if (row.service_tree_id) scopes.push(`服务树:${serviceTreeLabel(Number(row.service_tree_id))}`)
  if (row.owner_id) scopes.push(`负责人:${ownerLabel(Number(row.owner_id))}`)
  return scopes.length ? scopes.join(' / ') : '全局'
}

async function fetchBaseOptions() {
  const [ruleRes, userRes, treeRes] = await Promise.all([
    alertRuleApi.list({ page: 1, size: 500 }),
    userApi.list(1, 500),
    serviceTreeApi.tree(),
  ])
  rules.value = (ruleRes as any).data?.list || []
  users.value = (userRes as any).data?.list || []
  serviceTrees.value = flattenTree((treeRes as any).data || [])
}

async function fetchData() {
  loading.value = true
  try {
    const res = await alertSilenceApi.list()
    rows.value = (res as any).data || []
  } finally {
    loading.value = false
  }
}

function openAdd() {
  isEdit.value = false
  editId.value = 0
  form.value = { ...createDefaultForm(), ...createDefaultWindow() }
  dialogVisible.value = true
}

function openEdit(row: any) {
  isEdit.value = true
  editId.value = row.id
  form.value = {
    name: row.name || '',
    rule_id: row.rule_id ? Number(row.rule_id) : null,
    agent_id: row.agent_id || '',
    service_tree_id: row.service_tree_id ? Number(row.service_tree_id) : null,
    owner_id: row.owner_id ? Number(row.owner_id) : null,
    reason: row.reason || '',
    enabled: Number(row.enabled ?? 1),
    starts_at: row.starts_at || '',
    ends_at: row.ends_at || '',
  }
  dialogVisible.value = true
}

async function submit() {
  if (!form.value.name.trim()) {
    ElMessage.warning('请填写静默名称')
    return
  }
  submitting.value = true
  try {
    const payload = {
      ...form.value,
      name: form.value.name.trim(),
      rule_id: Number(form.value.rule_id || 0),
      service_tree_id: Number(form.value.service_tree_id || 0),
      owner_id: Number(form.value.owner_id || 0),
    }
    if (isEdit.value) {
      await alertSilenceApi.update(editId.value, payload)
    } else {
      await alertSilenceApi.create(payload)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function removeRow(row: any) {
  try {
    await ElMessageBox.confirm(`确定删除静默「${row.name}」？`, '提示', { type: 'warning' })
    await alertSilenceApi.delete(row.id)
    ElMessage.success('删除成功')
    await fetchData()
  } catch {}
}

onMounted(async () => {
  await Promise.all([fetchBaseOptions(), fetchData()])
})
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50">
    <div class="bg-white border-b border-gray-200 px-6 py-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold text-gray-900">告警静默</h1>
        <p class="text-sm text-gray-500 mt-1">按规则、Agent、服务树或负责人设置静默窗口，降低值守噪音。</p>
      </div>
      <div class="flex items-center gap-3">
        <el-button plain @click="fetchData">刷新</el-button>
        <el-button type="primary" @click="openAdd">新增静默</el-button>
      </div>
    </div>

    <div class="flex-1 overflow-auto p-6">
      <el-card shadow="never" class="border-gray-200">
        <el-table :data="rows" v-loading="loading" stripe border class="w-full">
          <el-table-column prop="name" label="静默名称" min-width="180" show-overflow-tooltip />
          <el-table-column label="作用范围" min-width="320" show-overflow-tooltip>
            <template #default="{ row }">{{ silenceScope(row) }}</template>
          </el-table-column>
          <el-table-column prop="starts_at" label="开始时间" width="180" align="center" />
          <el-table-column prop="ends_at" label="结束时间" width="180" align="center" />
          <el-table-column label="启用" width="90" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="Number(row.enabled) === 1 ? 'success' : 'info'">
                {{ Number(row.enabled) === 1 ? '启用' : '停用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" min-width="220" show-overflow-tooltip />
          <el-table-column label="操作" width="140" fixed="right" align="center">
            <template #default="{ row }">
              <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
              <el-button link type="danger" @click="removeRow(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑静默' : '新增静默'" width="640px" destroy-on-close align-center>
      <el-form label-width="100px" class="pr-6">
        <el-form-item label="静默名称" required>
          <el-input v-model="form.name" placeholder="例如：数据库维护窗口" />
        </el-form-item>
        <el-form-item label="规则范围">
          <el-select v-model="form.rule_id" clearable filterable class="w-full" placeholder="不限制规则">
            <el-option v-for="item in rules" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="Agent ID">
          <el-input v-model="form.agent_id" placeholder="可选，指定某台 Agent" />
        </el-form-item>
        <el-form-item label="服务树">
          <el-select v-model="form.service_tree_id" clearable filterable class="w-full" placeholder="不限制服务树">
            <el-option v-for="item in serviceTrees" :key="item.id" :label="item.label" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="负责人">
          <el-select v-model="form.owner_id" clearable filterable class="w-full" placeholder="不限制负责人">
            <el-option v-for="item in userOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="开始时间">
          <el-date-picker
            v-model="form.starts_at"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            placeholder="选择开始时间"
            class="!w-full"
          />
        </el-form-item>
        <el-form-item label="结束时间">
          <el-date-picker
            v-model="form.ends_at"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            placeholder="选择结束时间"
            class="!w-full"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.enabled" :active-value="1" :inactive-value="0" />
        </el-form-item>
        <el-form-item label="静默原因">
          <el-input v-model="form.reason" type="textarea" :rows="4" placeholder="例如：业务窗口期发布、节假日演练等" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end pt-4">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submit">保存</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
/* Scoped styles replaced with Tailwind utility classes */
</style>
