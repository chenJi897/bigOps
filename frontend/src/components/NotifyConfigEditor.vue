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
    if (!forms.value[ct]) {
      forms.value = { ...forms.value, [ct]: { webhook_url: '', secret: '' } }
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
  <div class="notify-config-editor space-y-2">
    <!-- 站内通知：强制 -->
    <div class="flex items-center gap-2 py-1.5 px-3 bg-gray-50 rounded-lg">
      <el-checkbox :model-value="true" disabled />
      <span class="text-sm text-gray-600">站内通知</span>
      <span class="text-xs text-gray-400">默认开启，不可取消</span>
    </div>

    <!-- 外部渠道 -->
    <div v-for="ct in enabledTypes" :key="ct" class="border border-gray-200 rounded-lg overflow-hidden">
      <div
        class="flex items-center gap-2 py-2 px-3 cursor-pointer hover:bg-gray-50 transition-colors"
        @click="toggleChannel(ct, !checkedList.includes(ct))"
      >
        <el-checkbox
          :model-value="checkedList.includes(ct)"
          @update:model-value="(val: boolean) => toggleChannel(ct, val)"
          @click.stop
        />
        <span class="text-sm font-medium text-gray-700">{{ channelLabels[ct] || ct }}</span>
      </div>
      <div v-if="checkedList.includes(ct)" class="px-3 pb-3 pt-1 bg-gray-50/50 border-t border-gray-100">
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
            class="w-40"
            @input="onInput"
          />
          <el-button
            size="small"
            plain
            :loading="testing[ct]"
            @click.stop="testWebhook(ct)"
          >
            测试
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>
