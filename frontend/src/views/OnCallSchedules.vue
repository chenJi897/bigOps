<script setup lang="ts">
defineOptions({ name: 'OnCallSchedules' })

import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { onCallApi, userApi } from '../api'

const loading = ref(false)
const dialogVisible = ref(false)
const submitting = ref(false)
const isEdit = ref(false)
const editId = ref(0)
const rows = ref<any[]>([])
const users = ref<any[]>([])

const channelOptions = [
  { label: '站内信', value: 'in_app' },
  { label: '邮件', value: 'email' },
  { label: 'Webhook', value: 'webhook' },
  { label: 'Message Pusher', value: 'message_pusher' },
  { label: '企业微信', value: 'wecom' },
  { label: '飞书', value: 'feishu' },
  { label: '钉钉', value: 'dingtalk' },
]

const form = ref<any>(createDefaultForm())

const userOptions = computed(() => users.value.map((item) => ({
  label: item.real_name || item.username,
  value: item.id,
})))

function createDefaultForm() {
  return {
    name: '',
    description: '',
    timezone: 'Asia/Shanghai',
    user_ids: [] as number[],
    rotation_days: 1,
    notify_channels: ['in_app'] as string[],
    escalation_minutes: 0,
    enabled: 1,
  }
}

function parseJSON(value: string, fallback: any[] = []) {
  try {
    const parsed = JSON.parse(value || '[]')
    return Array.isArray(parsed) ? parsed : fallback
  } catch {
    return fallback
  }
}

function userNames(ids: number[]) {
  if (!ids.length) return '—'
  return ids
    .map((id) => userOptions.value.find((item) => item.value === id)?.label || `#${id}`)
    .join('、')
}

function currentOnCall(row: any) {
  const ids = parseJSON(row.users_json)
  if (!ids.length) return '—'
  const rotationDays = Number(row.rotation_days || 1) || 1
  const slot = Math.floor(Date.now() / 1000 / (24 * 3600 * rotationDays))
  const currentID = ids[slot % ids.length]
  return userNames([currentID])
}

async function fetchUsers() {
  const res = await userApi.list(1, 500)
  users.value = (res as any).data?.list || []
}

async function fetchData() {
  loading.value = true
  try {
    const res = await onCallApi.list()
    rows.value = (res as any).data || []
  } finally {
    loading.value = false
  }
}

function openAdd() {
  isEdit.value = false
  editId.value = 0
  form.value = createDefaultForm()
  dialogVisible.value = true
}

function openEdit(row: any) {
  isEdit.value = true
  editId.value = row.id
  form.value = {
    name: row.name || '',
    description: row.description || '',
    timezone: row.timezone || 'Asia/Shanghai',
    user_ids: parseJSON(row.users_json),
    rotation_days: Number(row.rotation_days || 1),
    notify_channels: parseJSON(row.notify_channels_json, ['in_app']),
    escalation_minutes: Number(row.escalation_minutes || 0),
    enabled: Number(row.enabled ?? 1),
  }
  dialogVisible.value = true
}

async function submit() {
  if (!form.value.name.trim()) {
    ElMessage.warning('请填写值班名称')
    return
  }
  if (!form.value.user_ids.length) {
    ElMessage.warning('请至少选择一位值班成员')
    return
  }
  submitting.value = true
  try {
    const payload = {
      ...form.value,
      name: form.value.name.trim(),
      rotation_days: Number(form.value.rotation_days || 1),
      escalation_minutes: Number(form.value.escalation_minutes || 0),
    }
    if (isEdit.value) {
      await onCallApi.update(editId.value, payload)
    } else {
      await onCallApi.create(payload)
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
    await ElMessageBox.confirm(`确定删除值班表「${row.name}」？`, '提示', { type: 'warning' })
    await onCallApi.delete(row.id)
    ElMessage.success('删除成功')
    await fetchData()
  } catch {}
}

onMounted(async () => {
  await Promise.all([fetchUsers(), fetchData()])
})
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50">
    <div class="bg-white border-b border-gray-200 px-6 py-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold text-gray-900">OnCall 值班</h1>
        <p class="text-sm text-gray-500 mt-1">配置值班轮转、升级时间和通知渠道，用于告警升级和职责归属。</p>
      </div>
      <div class="flex items-center gap-3">
        <el-button plain @click="fetchData">刷新</el-button>
        <el-button type="primary" @click="openAdd">新增值班表</el-button>
      </div>
    </div>

    <div class="flex-1 overflow-auto p-6">
      <el-card shadow="never" class="border-gray-200">
        <el-table :data="rows" v-loading="loading" stripe border class="w-full">
          <el-table-column prop="name" label="值班表" min-width="180" show-overflow-tooltip />
          <el-table-column label="当前值班" min-width="160" show-overflow-tooltip>
            <template #default="{ row }">{{ currentOnCall(row) }}</template>
          </el-table-column>
          <el-table-column label="轮转周期" width="100" align="center">
            <template #default="{ row }">{{ row.rotation_days }} 天</template>
          </el-table-column>
          <el-table-column label="升级" width="110" align="center">
            <template #default="{ row }">{{ Number(row.escalation_minutes || 0) }} 分钟</template>
          </el-table-column>
          <el-table-column label="通知渠道" min-width="220" show-overflow-tooltip>
            <template #default="{ row }">{{ parseJSON(row.notify_channels_json).join(' / ') || '—' }}</template>
          </el-table-column>
          <el-table-column label="成员" min-width="260" show-overflow-tooltip>
            <template #default="{ row }">{{ userNames(parseJSON(row.users_json)) }}</template>
          </el-table-column>
          <el-table-column label="启用" width="90" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="Number(row.enabled) === 1 ? 'success' : 'info'">
                {{ Number(row.enabled) === 1 ? '启用' : '停用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="140" fixed="right" align="center">
            <template #default="{ row }">
              <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
              <el-button link type="danger" @click="removeRow(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑 OnCall 值班表' : '新增 OnCall 值班表'" width="640px" destroy-on-close align-center>
      <el-form label-width="110px" class="pr-6">
        <el-form-item label="值班表名称" required>
          <el-input v-model="form.name" placeholder="例如：平台运维值班" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="说明适用业务线、值班规则等" />
        </el-form-item>
        <el-form-item label="时区">
          <el-input v-model="form.timezone" placeholder="Asia/Shanghai" />
        </el-form-item>
        <el-form-item label="值班成员" required>
          <el-select v-model="form.user_ids" multiple filterable clearable class="w-full">
            <el-option v-for="item in userOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="轮转周期">
          <div class="flex items-center gap-2">
            <el-input-number v-model="form.rotation_days" :min="1" :max="90" />
            <span class="text-xs text-gray-500">单位：天</span>
          </div>
        </el-form-item>
        <el-form-item label="升级时间">
          <div class="flex items-center gap-2">
            <el-input-number v-model="form.escalation_minutes" :min="0" :step="5" />
            <span class="text-xs text-gray-500">单位：分钟，0 表示不升级</span>
          </div>
        </el-form-item>
        <el-form-item label="通知渠道">
          <el-select v-model="form.notify_channels" multiple clearable filterable class="w-full">
            <el-option v-for="item in channelOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="form.enabled" :active-value="1" :inactive-value="0" />
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
