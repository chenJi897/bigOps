<script setup lang="ts">
/**
 * NotifyConfigEditor — 通知渠道配置组件
 * 站内通知后端强制发送，此处只配置外部渠道（单选）。
 */
import { ref, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { notificationApi } from '../api'

const props = defineProps<{
  modelValue: Record<string, { webhook_url: string; secret: string }> | null
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', val: Record<string, { webhook_url: string; secret: string }>): void
}>()

const channelOptions = [
  { label: '不启用外部通知', value: '' },
  { label: '企业微信机器人', value: 'wecom' },
  { label: '钉钉机器人', value: 'dingtalk' },
  { label: '飞书机器人', value: 'lark' },
  { label: '自定义 Webhook', value: 'webhook' },
]

const selectedChannel = ref('')
const webhookUrl = ref('')
const secret = ref('')
const testing = ref(false)

const needsSecret = (ch: string) => ch === 'dingtalk' || ch === 'lark'

onMounted(() => syncFromModel())

let initialSynced = false
watch(() => props.modelValue, () => {
  if (!initialSynced) syncFromModel()
}, { deep: true })

function syncFromModel() {
  const cfg = props.modelValue || {}
  const keys = Object.keys(cfg).filter(k => cfg[k]?.webhook_url)
  if (keys.length > 0) {
    selectedChannel.value = keys[0]
    webhookUrl.value = cfg[keys[0]].webhook_url || ''
    secret.value = cfg[keys[0]].secret || ''
  } else {
    selectedChannel.value = ''
    webhookUrl.value = ''
    secret.value = ''
  }
  initialSynced = true
}

function onChannelChange() {
  webhookUrl.value = ''
  secret.value = ''
  emitUpdate()
}

function emitUpdate() {
  if (!selectedChannel.value || !webhookUrl.value) {
    emit('update:modelValue', {})
    return
  }
  emit('update:modelValue', {
    [selectedChannel.value]: { webhook_url: webhookUrl.value, secret: secret.value },
  })
}

async function testWebhook() {
  if (!webhookUrl.value) {
    ElMessage.warning('请先填写 Webhook 地址')
    return
  }
  testing.value = true
  try {
    await notificationApi.testWebhook({
      channel_type: selectedChannel.value,
      webhook_url: webhookUrl.value,
      secret: secret.value || '',
    })
    ElMessage.success('测试消息发送成功')
  } catch {
    // error handled by interceptor
  } finally {
    testing.value = false
  }
}
</script>

<template>
  <div class="notify-config-editor w-full space-y-3">
    <el-select v-model="selectedChannel" class="w-full" placeholder="选择外部通知渠道（默认仅站内通知）" @change="onChannelChange">
      <el-option v-for="item in channelOptions" :key="item.value" :label="item.label" :value="item.value" />
    </el-select>

    <template v-if="selectedChannel">
      <div class="p-3 bg-gray-50 rounded-lg border border-gray-200 space-y-2">
        <div>
          <div class="text-xs text-gray-500 mb-1">Webhook 地址</div>
          <div class="flex items-center gap-2">
            <el-input
              v-model="webhookUrl"
              :placeholder="selectedChannel === 'wecom' ? 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...' : selectedChannel === 'dingtalk' ? 'https://oapi.dingtalk.com/robot/send?access_token=...' : selectedChannel === 'lark' ? 'https://open.feishu.cn/open-apis/bot/v2/hook/...' : 'https://your-webhook-url'"
              @input="emitUpdate"
            />
            <el-button plain :loading="testing" @click="testWebhook">测试</el-button>
          </div>
        </div>
        <div v-if="needsSecret(selectedChannel)">
          <div class="text-xs text-gray-500 mb-1">
            {{ selectedChannel === 'dingtalk' ? '加签密钥（机器人安全设置页面，加签一栏下面显示的SEC开头的字符串）' : '签名校验密钥（可选）' }}
          </div>
          <el-input v-model="secret" :placeholder="selectedChannel === 'dingtalk' ? 'SECxxxxxxxx' : '可选'" @input="emitUpdate" />
        </div>
      </div>
    </template>
  </div>
</template>
