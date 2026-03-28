<script setup lang="ts">
defineOptions({ name: 'MyNotificationSettings' })

import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { notificationApi } from '../api'

const loading = ref(false)
const saving = ref(false)
const form = ref({
  enabled: 1,
  enabled_channels: ['in_app', 'email', 'message_pusher'] as string[],
  subscribed_biz_types: ['alert_event', 'ticket', 'cicd_pipeline', 'task_execution', 'notification'] as string[],
})

const channelOptions = [
  { label: '站内通知', value: 'in_app' },
  { label: '邮件', value: 'email' },
  { label: 'Message Pusher', value: 'message_pusher' },
]

const bizTypeOptions = [
  { label: '监控告警', value: 'alert_event' },
  { label: '工单通知', value: 'ticket' },
  { label: '审批通知', value: 'approval' },
  { label: 'CI/CD 通知', value: 'cicd_pipeline' },
  { label: '任务执行通知', value: 'task_execution' },
  { label: '系统通知', value: 'notification' },
]

async function loadPreference() {
  loading.value = true
  try {
    const res: any = await notificationApi.getPreference()
    form.value = {
      enabled: Number(res.data?.enabled ?? 1),
      enabled_channels: res.data?.enabled_channels ? JSON.parse(res.data.enabled_channels) : ['in_app', 'email', 'message_pusher'],
      subscribed_biz_types: res.data?.subscribed_biz_types ? JSON.parse(res.data.subscribed_biz_types) : ['alert_event', 'ticket', 'cicd_pipeline', 'task_execution', 'notification'],
    }
  } finally {
    loading.value = false
  }
}

async function savePreference() {
  saving.value = true
  try {
    await notificationApi.updatePreference(form.value)
    ElMessage.success('个人通知设置已保存')
    await loadPreference()
  } finally {
    saving.value = false
  }
}

onMounted(loadPreference)
</script>

<template>
  <div class="page">
    <el-card shadow="never" v-loading="loading">
      <template #header><span>我的通知设置</span></template>

      <el-alert
        title="这里控制你个人接收哪些业务通知，以及优先通过哪些个人渠道接收。"
        type="info"
        show-icon
        :closable="false"
        style="margin-bottom: 16px;"
      />

      <el-form :model="form" label-width="120px" style="max-width: 760px;">
        <el-form-item label="启用通知">
          <el-switch v-model="form.enabled" :active-value="1" :inactive-value="0" />
        </el-form-item>
        <el-form-item label="个人渠道">
          <el-select v-model="form.enabled_channels" multiple clearable filterable style="width: 100%;">
            <el-option v-for="item in channelOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="订阅事件">
          <el-select v-model="form.subscribed_biz_types" multiple clearable filterable style="width: 100%;">
            <el-option v-for="item in bizTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="savePreference">保存我的设置</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
</style>
