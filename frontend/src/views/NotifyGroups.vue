<script setup lang="ts">
defineOptions({ name: 'NotifyGroups' })

import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { notifyGroupApi, notificationApi, userApi } from '../api'

const loading = ref(false)
const tableData = ref<any[]>([])
const page = ref(1)
const size = ref(20)
const total = ref(0)
const keyword = ref('')

const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(0)
const submitting = ref(false)
const testing = ref(false)

const users = ref<any[]>([])
const enabledTypes = ref<string[]>(['lark', 'dingtalk', 'wecom', 'webhook'])

const channelLabels: Record<string, string> = {
  lark: '飞书', dingtalk: '钉钉', wecom: '企业微信', webhook: '自定义 Webhook',
}

const form = ref<any>({
  name: '', description: '',
  webhooks: [] as { channel_type: string; label: string; webhook_url: string; secret: string }[],
  notify_user_ids: [] as number[],
  repeat_enabled: 0, repeat_interval_seconds: 300,
  send_resolved: 1,
  escalation_enabled: 0, escalation_minutes: 20,
  escalation_user_ids: [] as number[],
  escalation_webhooks: [] as { channel_type: string; label: string; webhook_url: string; secret: string }[],
  status: 1,
})

async function fetchData() {
  loading.value = true
  try {
    const res: any = await notifyGroupApi.list({ page: page.value, size: size.value, keyword: keyword.value })
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally { loading.value = false }
}

async function loadUsers() {
  try {
    const res: any = await userApi.list(1, 200)
    users.value = res.data?.list || []
  } catch {}
}

async function loadEnabledTypes() {
  try {
    const res: any = await notificationApi.enabledChannelTypes()
    enabledTypes.value = res.data || ['lark', 'dingtalk', 'wecom', 'webhook']
  } catch {}
}

onMounted(() => { fetchData(); loadUsers(); loadEnabledTypes() })

function resetForm() {
  form.value = {
    name: '', description: '',
    webhooks: [],
    notify_user_ids: [],
    repeat_enabled: 0, repeat_interval_seconds: 300,
    send_resolved: 1,
    escalation_enabled: 0, escalation_minutes: 20,
    escalation_user_ids: [],
    escalation_webhooks: [],
    status: 1,
  }
}

function openCreate() {
  isEdit.value = false; editId.value = 0; resetForm(); dialogVisible.value = true
}

function openEdit(row: any) {
  isEdit.value = true; editId.value = row.id
  form.value = {
    name: row.name || '',
    description: row.description || '',
    webhooks: safeParseJSON(row.webhooks_json, []),
    notify_user_ids: safeParseJSON(row.notify_user_ids, []),
    repeat_enabled: row.repeat_enabled || 0,
    repeat_interval_seconds: row.repeat_interval_seconds || 300,
    send_resolved: row.send_resolved ?? 1,
    escalation_enabled: row.escalation_enabled || 0,
    escalation_minutes: row.escalation_minutes || 20,
    escalation_user_ids: safeParseJSON(row.escalation_user_ids, []),
    escalation_webhooks: safeParseJSON(row.escalation_webhooks_json, []),
    status: row.status ?? 1,
  }
  dialogVisible.value = true
}

function safeParseJSON(val: any, fallback: any) {
  if (!val) return fallback
  if (typeof val !== 'string') return val
  try { return JSON.parse(val) } catch { return fallback }
}

function addWebhook(list: any[]) {
  list.push({ channel_type: 'lark', label: '', webhook_url: '', secret: '' })
}

function removeWebhook(list: any[], idx: number) { list.splice(idx, 1) }

async function testSingleWebhook(wh: any) {
  if (!wh.webhook_url) { ElMessage.warning('请先填写 Webhook 地址'); return }
  try {
    await notificationApi.testWebhook({ channel_type: wh.channel_type, webhook_url: wh.webhook_url, secret: wh.secret || '' })
    ElMessage.success('测试消息发送成功')
  } catch {}
}

async function submitForm() {
  if (!form.value.name?.trim()) { ElMessage.warning('请填写名称'); return }
  submitting.value = true
  try {
    const payload = {
      ...form.value,
      webhooks_json: JSON.stringify(form.value.webhooks || []),
      notify_user_ids: JSON.stringify(form.value.notify_user_ids || []),
      escalation_user_ids: JSON.stringify(form.value.escalation_user_ids || []),
      escalation_webhooks_json: JSON.stringify(form.value.escalation_webhooks || []),
    }
    if (isEdit.value) {
      await notifyGroupApi.update(editId.value, payload)
    } else {
      await notifyGroupApi.create(payload)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } finally { submitting.value = false }
}

async function deleteGroup(row: any) {
  await ElMessageBox.confirm(`确认删除发送组「${row.name}」？`, '确认')
  await notifyGroupApi.delete(row.id)
  ElMessage.success('删除成功')
  fetchData()
}

async function testGroup(row: any) {
  testing.value = true
  try {
    await notifyGroupApi.test(row.id)
    ElMessage.success('测试消息已发送到所有渠道')
  } catch {} finally { testing.value = false }
}

function formatWebhooks(json: string) {
  const arr = safeParseJSON(json, [])
  return arr.map((w: any) => `${channelLabels[w.channel_type] || w.channel_type}${w.label ? '(' + w.label + ')' : ''}`).join(' / ') || '未配置'
}

const userOptions = ref<{ value: number; label: string }[]>([])
import { watch } from 'vue'
watch(users, (list) => {
  userOptions.value = list.map((u: any) => ({ value: u.id, label: u.real_name || u.username }))
}, { immediate: true })
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50">
    <div class="bg-white border-b border-gray-200 px-6 py-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold text-gray-900">发送组管理</h1>
        <p class="text-sm text-gray-500 mt-1">将通知对象（Webhook 群 + 通知人 + 升级链）打包管理，告警规则可引用发送组。</p>
      </div>
      <div class="flex items-center gap-3">
        <el-input v-model="keyword" placeholder="搜索名称" clearable class="w-48" @keyup.enter="fetchData" />
        <el-button @click="fetchData">搜索</el-button>
        <el-button v-permission="'notify_group:create'" type="primary" @click="openCreate">新增发送组</el-button>
      </div>
    </div>

    <div class="flex-1 overflow-auto p-6">
      <el-table :data="tableData" v-loading="loading" stripe border class="w-full">
        <el-table-column prop="id" label="ID" width="70" align="center" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="description" label="描述" min-width="180" show-overflow-tooltip />
        <el-table-column label="Webhook 渠道" min-width="200">
          <template #default="{ row }">
            <span class="text-sm text-gray-600">{{ formatWebhooks(row.webhooks_json) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="重复发送" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.repeat_enabled" type="warning" size="small">{{ row.repeat_interval_seconds }}s</el-tag>
            <span v-else class="text-xs text-gray-400">关闭</span>
          </template>
        </el-table-column>
        <el-table-column label="升级策略" width="120" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.escalation_enabled" type="danger" size="small">{{ row.escalation_minutes }}分钟</el-tag>
            <span v-else class="text-xs text-gray-400">关闭</span>
          </template>
        </el-table-column>
        <el-table-column label="恢复通知" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.send_resolved ? 'success' : 'info'" size="small">{{ row.send_resolved ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-permission="'notify_group:edit'" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="success" :loading="testing" @click="testGroup(row)">测试</el-button>
            <el-button link type="danger" @click="deleteGroup(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="mt-4 flex justify-end">
        <el-pagination v-model:current-page="page" v-model:page-size="size" :total="total" layout="total, prev, pager, next" @current-change="fetchData" />
      </div>
    </div>

    <!-- 创建/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑发送组' : '新增发送组'" width="720px" destroy-on-close align-center>
      <el-form :model="form" label-width="110px" class="pr-4">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="例如：SRE值班组" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" placeholder="用途说明" />
        </el-form-item>

        <el-divider content-position="left">Webhook 渠道</el-divider>
        <div v-for="(wh, idx) in form.webhooks" :key="idx" class="mb-3 p-3 bg-gray-50 rounded-lg border border-gray-200">
          <div class="flex items-center gap-2 mb-2">
            <el-select v-model="wh.channel_type" class="w-28" size="small">
              <el-option v-for="t in enabledTypes" :key="t" :label="channelLabels[t] || t" :value="t" />
            </el-select>
            <el-input v-model="wh.label" placeholder="标签（如 SRE飞书群）" size="small" class="w-40" />
            <el-button size="small" type="danger" link @click="removeWebhook(form.webhooks, Number(idx))">删除</el-button>
          </div>
          <div class="flex items-center gap-2">
            <el-input v-model="wh.webhook_url" placeholder="Webhook URL" size="small" class="flex-1" />
            <el-input v-if="wh.channel_type === 'dingtalk' || wh.channel_type === 'lark'" v-model="wh.secret" placeholder="签名密钥" size="small" class="w-36" />
            <el-button size="small" plain @click="testSingleWebhook(wh)">测试</el-button>
          </div>
        </div>
        <el-button size="small" plain @click="addWebhook(form.webhooks)">+ 添加渠道</el-button>

        <el-divider content-position="left">站内通知对象</el-divider>
        <el-form-item label="通知人">
          <el-select v-model="form.notify_user_ids" multiple clearable filterable class="w-full" placeholder="选择用户">
            <el-option v-for="u in userOptions" :key="u.value" :label="u.label" :value="Number(u.value)" />
          </el-select>
        </el-form-item>

        <el-divider content-position="left">发送策略</el-divider>
        <div class="grid grid-cols-2 gap-4 mb-4 ml-28">
          <div class="flex items-center gap-2">
            <el-checkbox v-model="form.repeat_enabled" :true-value="1" :false-value="0">重复发送</el-checkbox>
            <el-input-number v-if="form.repeat_enabled" v-model="form.repeat_interval_seconds" :min="60" :max="3600" size="small" class="w-28" />
            <span v-if="form.repeat_enabled" class="text-xs text-gray-500">秒</span>
          </div>
          <div class="flex items-center gap-2">
            <el-checkbox v-model="form.send_resolved" :true-value="1" :false-value="0">发送恢复通知</el-checkbox>
          </div>
        </div>

        <el-divider content-position="left">升级策略</el-divider>
        <div class="ml-28 mb-4">
          <div class="flex items-center gap-2 mb-3">
            <el-checkbox v-model="form.escalation_enabled" :true-value="1" :false-value="0">启用升级</el-checkbox>
            <span v-if="form.escalation_enabled" class="text-sm text-gray-600">持续</span>
            <el-input-number v-if="form.escalation_enabled" v-model="form.escalation_minutes" :min="1" :max="1440" size="small" class="w-28" />
            <span v-if="form.escalation_enabled" class="text-sm text-gray-600">分钟未确认后升级</span>
          </div>
          <template v-if="form.escalation_enabled">
            <el-form-item label="升级通知人" label-width="90px">
              <el-select v-model="form.escalation_user_ids" multiple clearable filterable class="w-full" placeholder="选择用户">
                <el-option v-for="u in userOptions" :key="u.value" :label="u.label" :value="Number(u.value)" />
              </el-select>
            </el-form-item>
            <div class="text-sm text-gray-500 mb-2">升级 Webhook（可选）：</div>
            <div v-for="(wh, idx) in form.escalation_webhooks" :key="'esc-'+idx" class="mb-2 flex items-center gap-2">
              <el-select v-model="wh.channel_type" class="w-24" size="small">
                <el-option v-for="t in enabledTypes" :key="t" :label="channelLabels[t] || t" :value="t" />
              </el-select>
              <el-input v-model="wh.webhook_url" placeholder="Webhook URL" size="small" class="flex-1" />
              <el-button size="small" type="danger" link @click="removeWebhook(form.escalation_webhooks, Number(idx))">删除</el-button>
            </div>
            <el-button size="small" plain @click="addWebhook(form.escalation_webhooks)">+ 添加升级渠道</el-button>
          </template>
        </div>

        <el-form-item label="是否启用">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>
