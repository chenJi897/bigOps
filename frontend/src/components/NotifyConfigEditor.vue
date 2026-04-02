<script setup lang="ts">
/**
 * NotifyConfigEditor — 通知配置通用组件
 *
 * 用于告警规则/流水线/工单模板表单中，让用户选择通知渠道并填写 Webhook 地址。
 * 站内通知强制开启不可取消。
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

const channelLabels: Record<string, string> = {
  lark: '飞书',
  dingtalk: '钉钉',
  wecom: '企业微信',
  webhook: '自定义 Webhook',
}

const enabledTypes = ref<string[]>([])
const testing = ref<Record<string, boolean>>({})

// 内部状态：哪些渠道被勾选
const checkedChannels = ref<Record<string, boolean>>({})
const channelForms = ref<Record<string, { webhook_url: string; secret: string }>>({})

onMounted(async () => {
  try {
    const res: any = await notificationApi.enabledChannelTypes()
    enabledTypes.value = res.data || ['lark', 'dingtalk', 'wecom', 'webhook']
  } catch {
    enabledTypes.value = ['lark', 'dingtalk', 'wecom', 'webhook']
  }
  // 从 modelValue 初始化
  syncFromModel()
})

watch(() => props.modelValue, () => syncFromModel(), { deep: true })

function syncFromModel() {
  const cfg = props.modelValue || {}
  for (const t of enabledTypes.value) {
    if (cfg[t] && cfg[t].webhook_url) {
      checkedChannels.value[t] = true
      channelForms.value[t] = { ...cfg[t] }
    } else {
      checkedChannels.value[t] = false
      if (!channelForms.value[t]) {
        channelForms.value[t] = { webhook_url: '', secret: '' }
      }
    }
  }
}

function emitUpdate() {
  const result: Record<string, { webhook_url: string; secret: string }> = {}
  for (const t of enabledTypes.value) {
    if (checkedChannels.value[t] && channelForms.value[t]?.webhook_url) {
      result[t] = { ...channelForms.value[t] }
    }
  }
  emit('update:modelValue', result)
}

function onToggle(channelType: string) {
  if (!checkedChannels.value[channelType]) {
    // 取消勾选时清空
    channelForms.value[channelType] = { webhook_url: '', secret: '' }
  }
  emitUpdate()
}

function onInput() {
  emitUpdate()
}

async function testWebhook(channelType: string) {
  const form = channelForms.value[channelType]
  if (!form?.webhook_url) {
    ElMessage.warning('请先填写 Webhook 地址')
    return
  }
  testing.value[channelType] = true
  try {
    await notificationApi.testWebhook({
      channel_type: channelType,
      webhook_url: form.webhook_url,
      secret: form.secret || '',
    })
    ElMessage.success('测试消息发送成功')
  } catch {
    // error handled by interceptor
  } finally {
    testing.value[channelType] = false
  }
}
</script>

<template>
  <div class="notify-config-editor">
    <!-- 站内通知：强制 -->
    <div class="channel-row">
      <el-checkbox :model-value="true" disabled>站内通知</el-checkbox>
      <span class="text-xs text-gray-400 ml-2">默认开启，不可取消</span>
    </div>

    <!-- 外部渠道 -->
    <div v-for="ct in enabledTypes" :key="ct" class="channel-row">
      <el-checkbox v-model="checkedChannels[ct]" @change="onToggle(ct)">
        {{ channelLabels[ct] || ct }}
      </el-checkbox>
      <div v-if="checkedChannels[ct]" class="channel-form">
        <div class="flex items-center gap-2">
          <el-input
            v-model="channelForms[ct].webhook_url"
            placeholder="Webhook URL"
            size="small"
            class="flex-1"
            @input="onInput"
          />
          <el-input
            v-if="ct === 'dingtalk' || ct === 'lark'"
            v-model="channelForms[ct].secret"
            placeholder="签名密钥（可选）"
            size="small"
            class="w-44"
            @input="onInput"
          />
          <el-button
            size="small"
            plain
            :loading="testing[ct]"
            @click="testWebhook(ct)"
          >
            测试
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.channel-row {
  padding: 6px 0;
}
.channel-form {
  margin-top: 4px;
  margin-left: 24px;
}
</style>
