<script setup lang="ts">
defineOptions({ name: 'NotificationConsole' })
import { computed, ref, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { notificationApi } from '../api'

const loading = ref(false)
const events = ref<any[]>([])
const configLoading = ref(false)
const configSaving = ref(false)

const configForm = ref<any>({
  default_channels: ['in_app'],
  max_retries: 3,
  retry_interval_seconds: 60,
  retry_scan_interval_seconds: 60,
  message_pusher: {
    enabled: false,
    server: '',
    username: '',
    token: '',
    timeout_seconds: 10,
  },
  channel_mapping: {
    wecom: 'corp',
    dingtalk: 'ding',
    lark: 'lark',
    webhook: 'custom',
  },
})

const channelOptions = [
  { label: '站内通知', value: 'in_app' },
  { label: '企业微信', value: 'wecom' },
  { label: '钉钉', value: 'dingtalk' },
  { label: '飞书', value: 'lark' },
  { label: 'Webhook', value: 'webhook' },
]

const mappingChannels = [
  { key: 'wecom', label: '企业微信', placeholder: 'corp 或 corp_app' },
  { key: 'dingtalk', label: '钉钉', placeholder: 'ding' },
  { key: 'lark', label: '飞书', placeholder: 'lark 或 lark_app' },
  { key: 'webhook', label: 'Webhook', placeholder: 'custom 或自定义通道名' },
]

const autoRefreshEnabled = ref(false)
const autoRefreshIntervalSeconds = 15
const lastRefreshAt = ref('')
let autoRefreshTimer: number | null = null
const savingHint = ref('')

async function loadEvents() {
  if (loading.value) return
  loading.value = true
  try {
    const res: any = await notificationApi.events()
    events.value = res.data || []
    lastRefreshAt.value = new Date().toLocaleString()
  } finally {
    loading.value = false
  }
}

async function loadConfig() {
  configLoading.value = true
  try {
    const res: any = await notificationApi.getConfig()
    configForm.value = {
      default_channels: res.data?.default_channels || ['in_app'],
      max_retries: res.data?.max_retries ?? 3,
      retry_interval_seconds: res.data?.retry_interval_seconds ?? 60,
      retry_scan_interval_seconds: res.data?.retry_scan_interval_seconds ?? 60,
      message_pusher: {
        enabled: res.data?.message_pusher?.enabled || false,
        server: res.data?.message_pusher?.server || '',
        username: res.data?.message_pusher?.username || '',
        token: res.data?.message_pusher?.token || '',
        timeout_seconds: res.data?.message_pusher?.timeout_seconds ?? 10,
      },
      channel_mapping: {
        wecom: res.data?.channel_mapping?.wecom || 'corp',
        dingtalk: res.data?.channel_mapping?.dingtalk || 'ding',
        lark: res.data?.channel_mapping?.lark || 'lark',
        webhook: res.data?.channel_mapping?.webhook || 'custom',
      },
    }
  } finally {
    configLoading.value = false
  }
}

async function saveConfig() {
  configSaving.value = true
  try {
    await notificationApi.updateConfig(configForm.value)
    ElMessage.success('通知配置已保存')
    savingHint.value = `已保存于 ${new Date().toLocaleTimeString()}`
    await loadConfig()
  } finally {
    configSaving.value = false
  }
}

async function retryEvent(row: any) {
  await notificationApi.retryEvent(row.id)
  ElMessage.success('已触发重试')
  loadEvents()
}

onMounted(async () => {
  await Promise.all([loadConfig(), loadEvents(), loadTemplates()])
})
onUnmounted(() => stopAutoRefresh())

watch(autoRefreshEnabled, (enabled) => {
  if (enabled) startAutoRefresh()
  else stopAutoRefresh()
})

function startAutoRefresh() {
  if (autoRefreshTimer) return
  autoRefreshTimer = window.setInterval(() => loadEvents(), autoRefreshIntervalSeconds * 1000)
}

function stopAutoRefresh() {
  if (!autoRefreshTimer) return
  clearInterval(autoRefreshTimer)
  autoRefreshTimer = null
}

function formatPayload(payload: string) {
  if (!payload) return ''
  try { return JSON.stringify(JSON.parse(payload), null, 2) }
  catch { return payload }
}

const configSummary = computed(() => {
  const cfg = configForm.value
  const channelLabels = (cfg?.default_channels || []).map((v: string) => channelOptions.find(o => o.value === v)?.label || v)
  return {
    channels: channelLabels.join(' / ') || '未设置',
    retry: `${cfg?.max_retries ?? 0} 次 · ${cfg?.retry_interval_seconds ?? 0}s · 扫描 ${cfg?.retry_scan_interval_seconds ?? 0}s`,
    mpEnabled: !!cfg?.message_pusher?.enabled,
  }
})

const templates = ref<any[]>([])
const templatesLoading = ref(false)
const editingTemplate = ref<any>(null)
const templateDialogVisible = ref(false)
const templateSaving = ref(false)
const previewResult = ref<{ title: string; content: string } | null>(null)

const eventTypeLabels: Record<string, string> = {
  alert_firing: '告警触发',
  alert_resolved: '告警恢复',
  pipeline_succeeded: '流水线成功',
  pipeline_failed: '流水线失败',
  approval_pending: '审批待处理',
  approval_approved: '审批已通过',
  approval_rejected: '审批已拒绝',
  notification_test: '通知测试',
}

async function loadTemplates() {
  templatesLoading.value = true
  try {
    const res: any = await notificationApi.listTemplates()
    templates.value = res.data || []
  } finally {
    templatesLoading.value = false
  }
}

function openEditTemplate(row: any) {
  editingTemplate.value = { ...row }
  previewResult.value = null
  templateDialogVisible.value = true
}

async function saveTemplate() {
  if (!editingTemplate.value) return
  templateSaving.value = true
  try {
    await notificationApi.updateTemplate(editingTemplate.value.id, {
      title: editingTemplate.value.title,
      content: editingTemplate.value.content,
    })
    ElMessage.success('模板已保存')
    templateDialogVisible.value = false
    loadTemplates()
  } finally {
    templateSaving.value = false
  }
}

async function previewTemplate() {
  if (!editingTemplate.value) return
  try {
    const sampleVars: Record<string, any> = {}
    const varsStr = editingTemplate.value.variables || ''
    varsStr.split(',').map((v: string) => v.trim()).filter(Boolean).forEach((v: string) => {
      sampleVars[v] = `[${v}]`
    })
    const res: any = await notificationApi.previewTemplate({
      title: editingTemplate.value.title,
      content: editingTemplate.value.content,
      variables: sampleVars,
    })
    previewResult.value = res.data
  } catch {
    ElMessage.error('预览失败')
  }
}
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50">
    <div class="bg-white border-b border-gray-200">
      <div class="px-6 py-5 flex flex-col gap-4">
        <div class="flex flex-col md:flex-row md:items-start justify-between gap-4">
          <div>
            <h1 class="text-xl font-bold text-gray-900">通知配置中心</h1>
            <p class="text-sm text-gray-500 mt-1">管理 Message Pusher 网关连接、通道映射与重试策略。用户个人通知偏好请在「个人设置」中配置。</p>
          </div>
          <div class="flex items-center gap-2">
            <div class="text-xs text-gray-500 mr-2 hidden md:block">{{ savingHint || ' ' }}</div>
            <el-button type="primary" :loading="configSaving" @click="saveConfig">保存配置</el-button>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <span class="text-xs font-medium px-2.5 py-1 rounded-full" :class="configSummary.mpEnabled ? 'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200' : 'bg-gray-50 text-gray-600 ring-1 ring-gray-200'">
            Message Pusher {{ configSummary.mpEnabled ? '已启用' : '未启用' }}
          </span>
          <span class="text-xs text-gray-500 ml-1">默认渠道：<span class="font-mono text-gray-700">{{ configSummary.channels }}</span></span>
          <span class="text-xs text-gray-500 ml-1">重试：<span class="font-mono text-gray-700">{{ configSummary.retry }}</span></span>
        </div>
      </div>
    </div>

    <div class="flex-1 overflow-auto p-6">
      <div class="grid grid-cols-1 xl:grid-cols-12 gap-6">

        <el-card shadow="never" class="border-gray-200 xl:col-span-7">
          <div class="mb-4">
            <h2 class="text-lg font-semibold text-gray-900">全局配置</h2>
            <div class="text-xs text-gray-500 mt-1">程序级别配置，对所有用户生效。外部通知统一通过 Message Pusher 网关投递。</div>
          </div>

          <el-form :model="configForm" label-width="140px" class="max-w-4xl pr-2" v-loading="configLoading">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <el-form-item label="默认渠道">
                <el-select v-model="configForm.default_channels" multiple clearable class="w-full">
                  <el-option v-for="item in channelOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="最大重试">
                <el-input-number v-model="configForm.max_retries" :min="0" :max="20" class="w-full max-w-[220px]" />
              </el-form-item>
              <el-form-item label="重试间隔(秒)">
                <el-input-number v-model="configForm.retry_interval_seconds" :min="1" :max="3600" class="w-full max-w-[220px]" />
              </el-form-item>
              <el-form-item label="扫描间隔(秒)">
                <el-input-number v-model="configForm.retry_scan_interval_seconds" :min="1" :max="3600" class="w-full max-w-[220px]" />
              </el-form-item>
            </div>

            <el-divider border-style="dashed" class="!my-6" />

            <div class="flex items-center justify-between mb-3">
              <div class="text-sm font-semibold text-gray-800">Message Pusher 网关</div>
              <el-switch v-model="configForm.message_pusher.enabled" />
            </div>
            <div class="text-xs text-gray-500 mb-4">
              外部通知（企业微信、钉钉、飞书、Webhook 等）统一通过
              <a href="https://github.com/songquanpeng/message-pusher" target="_blank" class="text-indigo-600 hover:underline cursor-pointer">Message Pusher</a>
              网关投递。请先在 Message Pusher 后台配置好各通道。
            </div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <el-form-item label="服务地址" class="md:col-span-2">
                <el-input v-model="configForm.message_pusher.server" placeholder="http://message-pusher:3000" class="max-w-md" />
              </el-form-item>
              <el-form-item label="推送账号">
                <el-input v-model="configForm.message_pusher.username" placeholder="Message Pusher 用户名" />
              </el-form-item>
              <el-form-item label="推送 Token">
                <el-input v-model="configForm.message_pusher.token" type="password" show-password placeholder="Message Pusher 鉴权 Token" />
              </el-form-item>
              <el-form-item label="超时(秒)">
                <el-input-number v-model="configForm.message_pusher.timeout_seconds" :min="1" :max="120" class="w-full max-w-[220px]" />
              </el-form-item>
            </div>

            <el-divider border-style="dashed" class="!my-6" />

            <div class="text-sm font-semibold text-gray-800 mb-2">通道映射</div>
            <div class="text-xs text-gray-500 mb-4">
              将 BigOps 通知通道名称映射为 Message Pusher 的 channel 名称。
              BigOps 投递时会使用映射后的 channel 值调用 Message Pusher。
            </div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <el-form-item v-for="mc in mappingChannels" :key="mc.key" :label="mc.label">
                <el-input v-model="configForm.channel_mapping[mc.key]" :placeholder="mc.placeholder" />
              </el-form-item>
            </div>
          </el-form>
        </el-card>

        <el-card shadow="never" class="border-gray-200 xl:col-span-5">
          <div class="flex justify-between items-center mb-3">
            <div>
              <h2 class="text-lg font-semibold text-gray-900">事件观测</h2>
              <div class="text-xs text-gray-500 mt-1">用于排查投递状态与失败原因。</div>
            </div>
            <div class="flex items-center gap-3">
              <div class="flex items-center gap-2">
                <el-switch v-model="autoRefreshEnabled" active-text="自动" inactive-text="手动" />
                <span class="text-xs text-gray-500 hidden sm:inline">每 {{ autoRefreshIntervalSeconds }} 秒</span>
              </div>
              <el-button type="primary" plain :loading="loading" @click="loadEvents">刷新</el-button>
            </div>
          </div>
          <div class="text-xs text-gray-500 mb-3 text-right">最后刷新：{{ lastRefreshAt || '未刷新' }}</div>

          <el-table :data="events" v-loading="loading" stripe border class="w-full">
            <el-table-column prop="id" label="ID" width="70" align="center" />
            <el-table-column prop="event_type" label="事件类型" width="160" />
            <el-table-column prop="title" label="标题" min-width="220" show-overflow-tooltip />
            <el-table-column label="状态" width="180">
              <template #default="{ row }">
                <div class="flex flex-col gap-1">
                  <span class="font-semibold text-gray-800">{{ row.status }}</span>
                  <span class="text-xs text-gray-500">{{ row.status_summary || '—' }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="170" align="center" />
            <el-table-column type="expand">
              <template #default="{ row }">
                <div class="p-4 bg-gray-50 flex flex-col gap-4">
                  <div class="flex items-center gap-2 text-sm">
                    <span class="font-semibold text-gray-800">业务维度：</span>
                    <span class="text-gray-600">{{ row.biz_type || '—' }}/{{ row.biz_id || '—' }}</span>
                  </div>
                  <div v-if="row.payload" class="flex flex-col gap-2">
                    <span class="font-semibold text-gray-800 text-sm">Payload</span>
                    <pre class="bg-gray-100 p-3 rounded-lg border border-gray-200 text-xs text-gray-700 whitespace-pre-wrap font-mono">{{ formatPayload(row.payload) }}</pre>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="投递" min-width="320">
              <template #default="{ row }">
                <div class="flex flex-col gap-2 py-2">
                  <div v-for="delivery in row.deliveries || []" :key="delivery.id" class="border border-gray-200 rounded-md p-2 bg-white shadow-sm">
                    <div class="flex items-center gap-3 mb-2">
                      <span class="font-semibold text-gray-800 text-sm">{{ delivery.channel }}</span>
                      <span
                        class="px-2 py-0.5 rounded-full text-xs font-medium"
                        :class="{
                          'bg-red-100 text-red-700': delivery.status === 'failed' || delivery.status === 'dead',
                          'bg-blue-100 text-blue-700': delivery.status === 'pending',
                          'bg-green-100 text-green-700': delivery.status !== 'failed' && delivery.status !== 'dead' && delivery.status !== 'pending'
                        }"
                      >
                        {{ delivery.status_summary || delivery.status }}
                      </span>
                      <span class="text-xs text-gray-500">retry {{ delivery.retry_count }}</span>
                    </div>
                    <div class="grid gap-1 text-xs text-gray-600">
                      <div class="flex gap-2">
                        <span class="text-gray-400 w-16 shrink-0">响应：</span>
                        <span class="font-mono truncate" :title="delivery.response">{{ delivery.response || '—' }}</span>
                      </div>
                      <div class="flex gap-2">
                        <span class="text-gray-400 w-16 shrink-0">下次重试：</span>
                        <span>{{ delivery.next_retry_at || '—' }}</span>
                      </div>
                      <div class="flex gap-2">
                        <span class="text-gray-400 w-16 shrink-0">发送时间：</span>
                        <span>{{ delivery.sent_at || delivery.last_attempt_at || '—' }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" fixed="right" align="center">
              <template #default="{ row }">
                <el-button link type="primary" :disabled="row.can_retry === false" @click="retryEvent(row)">重试</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </div>

      <el-card shadow="never" class="border-gray-200 mt-6">
        <div class="flex justify-between items-center mb-4">
          <div>
            <h2 class="text-lg font-semibold text-gray-900">通知模板</h2>
            <div class="text-xs text-gray-500 mt-1">管理各类事件的 Markdown 通知模板。外部通知投递时会使用模板渲染内容，支持 Go template 语法。</div>
          </div>
          <el-button plain :loading="templatesLoading" @click="loadTemplates">刷新</el-button>
        </div>

        <el-table :data="templates" v-loading="templatesLoading" stripe border class="w-full">
          <el-table-column label="事件类型" width="160">
            <template #default="{ row }">
              <div>
                <div class="font-medium text-gray-800">{{ eventTypeLabels[row.event_type] || row.event_type }}</div>
                <div class="text-xs text-gray-500 font-mono">{{ row.event_type }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="title" label="标题模板" min-width="250" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="font-mono text-sm text-gray-700">{{ row.title }}</span>
            </template>
          </el-table-column>
          <el-table-column label="可用变量" min-width="200" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="text-xs text-gray-500">{{ row.variables || '—' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="updated_at" label="更新时间" width="170" align="center" />
          <el-table-column label="操作" width="100" fixed="right" align="center">
            <template #default="{ row }">
              <el-button link type="primary" @click="openEditTemplate(row)">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-dialog v-model="templateDialogVisible" title="编辑通知模板" width="700px" destroy-on-close align-center>
        <div v-if="editingTemplate" class="space-y-4">
          <div>
            <div class="text-sm font-medium text-gray-700 mb-1">事件类型</div>
            <div class="text-sm text-gray-500 font-mono bg-gray-50 px-3 py-2 rounded">{{ eventTypeLabels[editingTemplate.event_type] || editingTemplate.event_type }}（{{ editingTemplate.event_type }}）</div>
          </div>
          <div>
            <div class="text-sm font-medium text-gray-700 mb-1">可用变量</div>
            <div class="text-xs text-gray-500 bg-gray-50 px-3 py-2 rounded font-mono">{{ editingTemplate.variables || '无' }}</div>
          </div>
          <div>
            <div class="text-sm font-medium text-gray-700 mb-1">标题模板</div>
            <el-input v-model="editingTemplate.title" placeholder="支持 Go template 语法，如 {{.rule_name}}" />
          </div>
          <div>
            <div class="text-sm font-medium text-gray-700 mb-1">内容模板（Markdown）</div>
            <el-input v-model="editingTemplate.content" type="textarea" :rows="10" placeholder="支持 Markdown + Go template 语法" class="font-mono" />
          </div>

          <div v-if="previewResult" class="border border-gray-200 rounded-lg p-4 bg-gray-50">
            <div class="text-sm font-semibold text-gray-700 mb-2">渲染预览（变量使用占位值）</div>
            <div class="text-sm text-gray-800 font-medium mb-1">{{ previewResult.title }}</div>
            <pre class="text-xs text-gray-700 whitespace-pre-wrap font-mono bg-white p-3 rounded border border-gray-200">{{ previewResult.content }}</pre>
          </div>
        </div>
        <template #footer>
          <div class="flex justify-between">
            <el-button @click="previewTemplate">预览</el-button>
            <div class="flex gap-2">
              <el-button @click="templateDialogVisible = false">取消</el-button>
              <el-button type="primary" :loading="templateSaving" @click="saveTemplate">保存</el-button>
            </div>
          </div>
        </template>
      </el-dialog>
    </div>
  </div>
</template>

<style scoped>
</style>
