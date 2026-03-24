<script setup lang="ts">
defineOptions({ name: 'NotificationConsole' })
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { notificationApi } from '../api'

const submitting = ref(false)
const loading = ref(false)
const events = ref<any[]>([])
const form = ref({
  title: 'BigOps 通知测试',
  content: '这是一条用于联调邮件 / Webhook / Message Pusher 的测试消息。',
  channels: ['in_app'],
})

const channelOptions = [
  { label: '站内通知', value: 'in_app' },
  { label: '邮件', value: 'email' },
  { label: 'Webhook', value: 'webhook' },
  { label: 'Message Pusher', value: 'message_pusher' },
]

const autoRefreshEnabled = ref(false)
const autoRefreshIntervalSeconds = 15
const lastRefreshAt = ref('')
let autoRefreshTimer: number | null = null

async function loadEvents() {
  if (loading.value) {
    return
  }
  loading.value = true
  try {
    const res: any = await notificationApi.events()
    events.value = res.data || []
    lastRefreshAt.value = new Date().toLocaleString()
  } finally {
    loading.value = false
  }
}

async function submit() {
  if (!form.value.title || !form.value.content) {
    ElMessage.warning('请填写标题和内容')
    return
  }
  if (!form.value.channels.length) {
    ElMessage.warning('请至少选择一个渠道')
    return
  }
  submitting.value = true
  try {
    await notificationApi.testSend({
      title: form.value.title,
      content: form.value.content,
      channels: form.value.channels,
    })
    ElMessage.success('测试消息已发送')
    loadEvents()
  } finally {
    submitting.value = false
  }
}

async function retryEvent(row: any) {
  await notificationApi.retryEvent(row.id)
  ElMessage.success('已触发重试')
  loadEvents()
}

onMounted(loadEvents)
onUnmounted(() => stopAutoRefresh())

watch(autoRefreshEnabled, (enabled) => {
  if (enabled) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
})

function startAutoRefresh() {
  if (autoRefreshTimer) return
  autoRefreshTimer = window.setInterval(() => {
    loadEvents()
  }, autoRefreshIntervalSeconds * 1000)
}

function stopAutoRefresh() {
  if (!autoRefreshTimer) return
  clearInterval(autoRefreshTimer)
  autoRefreshTimer = null
}

function formatPayload(payload: string) {
  if (!payload) return ''
  try {
    const parsed = JSON.parse(payload)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return payload
  }
}
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header><span>通知联调</span></template>

      <el-alert
        title="业务侧应统一调用通知服务，联调页只用于验证渠道配置。"
        type="info"
        show-icon
        :closable="false"
        style="margin-bottom: 16px;"
      />

      <el-form :model="form" label-width="100px" style="max-width: 760px;">
        <el-form-item label="标题">
          <el-input v-model="form.title" />
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="form.content" type="textarea" :rows="5" />
        </el-form-item>
        <el-form-item label="渠道">
          <el-select v-model="form.channels" multiple style="width: 100%;">
            <el-option v-for="item in channelOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="submitting" @click="submit">发送测试消息</el-button>
        </el-form-item>
      </el-form>

      <el-divider>最近通知事件</el-divider>

      <div class="table-toolbar">
        <div class="toolbar-left">
          <el-button type="primary" :loading="loading" @click="loadEvents">刷新事件</el-button>
          <el-switch
            v-model="autoRefreshEnabled"
            active-color="#409eff"
            active-text="自动刷新"
            inactive-text="手动刷新"
            style="margin-left: 16px;"
          />
          <span class="toolbar-interval">间隔 {{ autoRefreshIntervalSeconds }} 秒</span>
        </div>
        <div class="toolbar-right">
          <span class="toolbar-meta">最后刷新：{{ lastRefreshAt || '未刷新' }}</span>
        </div>
      </div>

      <el-table :data="events" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="event_type" label="事件类型" width="160" />
        <el-table-column prop="title" label="标题" min-width="220" show-overflow-tooltip />
        <el-table-column label="状态" width="180">
          <template #default="{ row }">
            <div class="event-status">
              <div class="event-status-code">{{ row.status }}</div>
              <div class="event-status-text">{{ row.status_summary || '—' }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170" />
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="expand-panel">
              <div class="expand-row">
                <span class="expand-label">业务维度：</span>
                <span>{{ row.biz_type || '—' }}/{{ row.biz_id || '—' }}</span>
              </div>
              <div class="expand-row" v-if="row.payload">
                <span class="expand-label">Payload</span>
                <pre class="payload-block">{{ formatPayload(row.payload) }}</pre>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="投递" min-width="320">
          <template #default="{ row }">
            <div class="delivery-list">
              <div v-for="delivery in row.deliveries || []" :key="delivery.id" class="delivery-card">
                <div class="delivery-card__header">
                  <span class="delivery-channel">{{ delivery.channel }}</span>
                  <span
                    class="delivery-status"
                    :class="{
                      'delivery-status--failed': delivery.status === 'failed' || delivery.status === 'dead',
                      'delivery-status--pending': delivery.status === 'pending',
                    }"
                  >
                    {{ delivery.status_summary || delivery.status }}
                  </span>
                  <span class="delivery-retry">retry {{ delivery.retry_count }}</span>
                </div>
                <div class="delivery-card__body">
                  <div class="delivery-line">
                    <span class="delivery-label">响应：</span>
                    <span class="delivery-value" :title="delivery.response">{{ delivery.response || '—' }}</span>
                  </div>
                  <div class="delivery-line">
                    <span class="delivery-label">下次重试：</span>
                    <span class="delivery-value">{{ delivery.next_retry_at || '—' }}</span>
                  </div>
                  <div class="delivery-line">
                    <span class="delivery-label">发送时间：</span>
                    <span class="delivery-value">
                      {{ delivery.sent_at || delivery.last_attempt_at || '—' }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :disabled="row.can_retry === false" @click="retryEvent(row)">重试</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
.table-toolbar {
  margin: 16px 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}
.toolbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.toolbar-interval,
.toolbar-meta {
  font-size: 12px;
  color: #6b7280;
}
.event-status {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.event-status-code {
  color: #1f2937;
  font-weight: 600;
}
.event-status-text {
  font-size: 12px;
  color: #6b7280;
}
.delivery-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.delivery-card {
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 10px 12px;
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}
.delivery-card__header {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 13px;
  color: #374151;
}
.delivery-channel {
  font-weight: 600;
}
.delivery-status {
  padding: 2px 6px;
  border-radius: 999px;
  font-size: 11px;
  background: #dcfce7;
  color: #15803d;
}
.delivery-status--failed {
  background: #fee2e2;
  color: #b91c1c;
}
.delivery-status--pending {
  background: #e0f2fe;
  color: #0369a1;
}
.delivery-retry {
  font-size: 12px;
  color: #6b7280;
}
.delivery-card__body {
  margin-top: 8px;
  display: grid;
  gap: 4px;
  font-size: 12px;
  color: #4b5563;
}
.delivery-line {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  align-items: center;
}
.delivery-label {
  color: #6b7280;
}
.delivery-value {
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  word-break: break-word;
  max-width: 260px;
}
.expand-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 6px 0;
}
.expand-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: flex-start;
  font-size: 13px;
  color: #374151;
}
.expand-label {
  font-weight: 600;
  color: #1f2937;
}
.payload-block {
  margin: 0;
  width: 100%;
  background: #f1f5f9;
  border-radius: 4px;
  border: 1px solid #e2e8f0;
  padding: 10px;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
