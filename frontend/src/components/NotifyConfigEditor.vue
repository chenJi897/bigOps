<script setup lang="ts">
/**
 * NotifyConfigEditor — 通知配置通用组件
 * 站内通知强制开启不可取消。用户勾选外部渠道后展开 Webhook URL 输入框。
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

// 使用数组跟踪已勾选的渠道，避免 Vue 对 object key 的响应式问题
const checkedList = ref<string[]>([])
const forms = ref<Record<string, { webhook_url: string; secret: string }>>({})

onMounted(async () => {
  try {
    const res: any = await notificationApi.enabledChannelTypes()
    enabledTypes.value = res.data || ['lark', 'dingtalk', 'wecom', 'webhook']
  } catch {
    enabledTypes.value = ['lark', 'dingtalk', 'wecom', 'webhook']
  }
  syncFromModel()
})

// Only sync from model on initial load, not on every parent update
// (parent updates are caused by our own emitUpdate, creating a loop)
let initialSynced = false

watch(() => props.modelValue, () => {
  if (!initialSynced) syncFromModel()
}, { deep: true })

function syncFromModel() {
  const cfg = props.modelValue || {}
  const checked: string[] = []
  const f: Record<string, { webhook_url: string; secret: string }> = {}
  for (const t of enabledTypes.value) {
    f[t] = cfg[t] && cfg[t].webhook_url
      ? { ...cfg[t] }
      : (forms.value[t] || { webhook_url: '', secret: '' })
    if (cfg[t] && cfg[t].webhook_url) {
      checked.push(t)
    }
  }
  checkedList.value = checked
  forms.value = f
  initialSynced = true
}

function toggleChannel(ct: string, val: boolean) {
  if (val) {
    if (!checkedList.value.includes(ct)) {
      checkedList.value = [...checkedList.value, ct]
    }
  } else {
    checkedList.value = checkedList.value.filter(c => c !== ct)
    forms.value = { ...forms.value, [ct]: { webhook_url: '', secret: '' } }
  }
  emitUpdate()
}

function emitUpdate() {
  const result: Record<string, { webhook_url: string; secret: string }> = {}
  for (const t of checkedList.value) {
    if (forms.value[t]?.webhook_url) {
      result[t] = { ...forms.value[t] }
    }
  }
  emit('update:modelValue', result)
}

function onInput() {
  emitUpdate()
}

async function testWebhook(channelType: string) {
  const form = forms.value[channelType]
  if (!form?.webhook_url) {
    ElMessage.warning('请先填写 Webhook 地址')
    return
  }
  testing.value = { ...testing.value, [channelType]: true }
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
    testing.value = { ...testing.value, [channelType]: false }
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
      <label class="channel-label" @click.prevent="toggleChannel(ct, !checkedList.includes(ct))">
        <input
          type="checkbox"
          :checked="checkedList.includes(ct)"
          @click.stop
        />
        <span class="ml-1.5">{{ channelLabels[ct] || ct }}</span>
      </label>
      <div v-if="checkedList.includes(ct)" class="channel-form">
        <div class="flex items-center gap-2">
          <el-input
            v-model="forms[ct].webhook_url"
            placeholder="Webhook URL"
            size="small"
            class="flex-1"
            @input="onInput"
          />
          <el-input
            v-if="ct === 'dingtalk' || ct === 'lark'"
            v-model="forms[ct].secret"
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
.channel-label {
  display: inline-flex;
  align-items: center;
  cursor: pointer;
  font-size: 14px;
  color: #606266;
  user-select: none;
}
.channel-label input[type="checkbox"] {
  width: 14px;
  height: 14px;
  accent-color: #409eff;
}
</style>
