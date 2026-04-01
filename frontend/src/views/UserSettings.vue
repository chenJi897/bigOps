<script setup lang="ts">
defineOptions({ name: 'UserSettings' })

import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { notificationApi, userApi, authApi } from '../api'
import { useUserStore } from '../stores/user'
import { resetRouter } from '../router'

const router = useRouter()
const userStore = useUserStore()
const activeTab = ref('profile')

// --- 1. 个人资料 Profile ---
const profileSaving = ref(false)
const profileForm = ref({
  real_name: '',
  email: '',
  phone: ''
})

async function saveProfile() {
  const userId = userStore.userInfo?.id
  if (!userId) return

  profileSaving.value = true
  try {
    await userApi.update(userId, profileForm.value)
    ElMessage.success('个人资料已更新')
    await userStore.fetchUserInfo()
  } finally {
    profileSaving.value = false
  }
}

// --- 2. 通知偏好 Notifications ---
const notifLoading = ref(false)
const notifSaving = ref(false)
const notifForm = ref({
  enabled: 1,
  enabled_channels: ['in_app', 'wecom', 'dingtalk', 'lark'] as string[],
  subscribed_biz_types: ['alert_event', 'ticket', 'cicd_pipeline', 'task_execution', 'notification'] as string[],
  // channel_targets: key=通道类型（dingtalk/wecom/lark/webhook），value=Message Pusher 通道名
  channel_targets: {} as Record<string, string>,
})

const channelOptions = [
  { label: '站内通知', value: 'in_app' },
  { label: '企业微信', value: 'wecom' },
  { label: '钉钉', value: 'dingtalk' },
  { label: '飞书', value: 'lark' },
  { label: 'Webhook', value: 'webhook' },
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
  notifLoading.value = true
  try {
    const res: any = await notificationApi.getPreference()
    const rawTargets = res.data?.channel_targets
    let targets: Record<string, string> = {}
    if (typeof rawTargets === 'string') {
      targets = rawTargets ? JSON.parse(rawTargets) : {}
    } else if (rawTargets && typeof rawTargets === 'object') {
      targets = rawTargets
    }
    notifForm.value = {
      enabled: Number(res.data?.enabled ?? 1),
      enabled_channels: res.data?.enabled_channels ? JSON.parse(res.data.enabled_channels) : ['in_app', 'wecom', 'dingtalk', 'lark'],
      subscribed_biz_types: res.data?.subscribed_biz_types ? JSON.parse(res.data.subscribed_biz_types) : ['alert_event', 'ticket', 'cicd_pipeline', 'task_execution', 'notification'],
      channel_targets: targets,
    }
  } finally {
    notifLoading.value = false
  }
}

async function savePreference() {
  notifSaving.value = true
  try {
    await notificationApi.updatePreference(notifForm.value)
    ElMessage.success('通知设置已保存')
    await loadPreference()
  } finally {
    notifSaving.value = false
  }
}

// --- 3. 安全设置 Security ---
const pwdForm = ref({ old_password: '', new_password: '', confirm_password: '' })
const pwdSaving = ref(false)

async function submitPwd() {
  const { old_password, new_password, confirm_password } = pwdForm.value
  if (!old_password || !new_password) {
    ElMessage.warning('请填写完整密码信息')
    return
  }
  if (new_password !== confirm_password) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }
  if (new_password.length < 8) {
    ElMessage.warning('新密码长度不能少于 8 位')
    return
  }

  pwdSaving.value = true
  try {
    await authApi.changePassword(old_password, new_password)
    ElMessage.success('密码修改成功，请使用新密码重新登录')
    userStore.clearToken()
    resetRouter()
    router.push('/login')
  } catch {
    // 错误由拦截器处理
  } finally {
    pwdSaving.value = false
  }
}

onMounted(() => {
  // 初始化 Profile
  const u = userStore.userInfo
  if (u) {
    profileForm.value = {
      real_name: u.real_name || '',
      email: u.email || '',
      phone: u.phone || ''
    }
  }
  // 初始化 通知偏好
  loadPreference()
})
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- Header -->
    <div class="mb-5">
      <h2 class="text-xl font-bold text-gray-900">个人设置</h2>
      <p class="text-sm text-gray-500 mt-1">管理你的个人资料、联系方式、通知接收偏好及账号安全。</p>
    </div>

    <!-- Content -->
    <div class="flex-1 flex gap-6 overflow-hidden">
      <!-- Sidebar -->
      <div class="w-56 shrink-0 flex flex-col gap-1">
        <div 
          class="px-4 py-2.5 rounded-lg cursor-pointer transition-colors flex items-center gap-3 text-sm font-medium"
          :class="activeTab === 'profile' ? 'bg-indigo-50 text-indigo-700' : 'text-gray-700 hover:bg-gray-100'"
          @click="activeTab = 'profile'"
        >
          <el-icon class="text-lg"><User /></el-icon>
          个人资料
        </div>
        <div 
          class="px-4 py-2.5 rounded-lg cursor-pointer transition-colors flex items-center gap-3 text-sm font-medium"
          :class="activeTab === 'notifications' ? 'bg-indigo-50 text-indigo-700' : 'text-gray-700 hover:bg-gray-100'"
          @click="activeTab = 'notifications'"
        >
          <el-icon class="text-lg"><Bell /></el-icon>
          通知偏好
        </div>
        <div 
          class="px-4 py-2.5 rounded-lg cursor-pointer transition-colors flex items-center gap-3 text-sm font-medium"
          :class="activeTab === 'security' ? 'bg-indigo-50 text-indigo-700' : 'text-gray-700 hover:bg-gray-100'"
          @click="activeTab = 'security'"
        >
          <el-icon class="text-lg"><Lock /></el-icon>
          安全设置
        </div>
      </div>

      <!-- Main Area -->
      <div class="flex-1 bg-white border border-gray-200 rounded-xl shadow-sm p-8 overflow-y-auto">
        <!-- 1. 个人资料 -->
        <div v-show="activeTab === 'profile'" class="max-w-2xl animate-fade-in">
          <h3 class="text-lg font-semibold text-gray-800 mb-6">基本信息</h3>
          <el-form :model="profileForm" label-position="top">
            <el-form-item label="用户名">
              <el-input :model-value="userStore.userInfo?.username" disabled class="bg-gray-50" />
              <div class="text-xs text-gray-400 mt-1">系统登录账号，不可修改。</div>
            </el-form-item>
            <el-form-item label="真实姓名">
              <el-input v-model="profileForm.real_name" placeholder="请输入你的真实姓名" />
            </el-form-item>
            
            <el-divider border-style="dashed" class="!my-8" />
            
            <h3 class="text-lg font-semibold text-gray-800 mb-6">联系方式</h3>
            <div class="text-sm text-gray-500 mb-4 -mt-2">完善联系方式后，你可以通过邮件或短信等渠道接收系统重要通知。</div>
            <el-form-item label="邮箱地址">
              <el-input v-model="profileForm.email" placeholder="example@company.com">
                <template #prefix><el-icon><Message /></el-icon></template>
              </el-input>
            </el-form-item>
            <el-form-item label="手机号码">
              <el-input v-model="profileForm.phone" placeholder="用于接收紧急短信/电话告警">
                <template #prefix><el-icon><Iphone /></el-icon></template>
              </el-input>
            </el-form-item>
            
            <div class="mt-8">
              <el-button type="primary" :loading="profileSaving" @click="saveProfile">保存个人资料</el-button>
            </div>
          </el-form>
        </div>

        <!-- 2. 通知偏好 -->
        <div v-show="activeTab === 'notifications'" class="max-w-2xl animate-fade-in" v-loading="notifLoading">
          <h3 class="text-lg font-semibold text-gray-800 mb-6">消息接收规则</h3>
          <el-form :model="notifForm" label-position="top">
            <el-form-item>
              <template #label>
                <div class="flex items-center gap-2">
                  <span class="font-medium text-gray-700">允许接收任何通知</span>
                  <el-switch v-model="notifForm.enabled" :active-value="1" :inactive-value="0" />
                </div>
              </template>
              <div class="text-xs text-gray-500">
                {{ notifForm.enabled ? '已开启。关闭后，你将不会收到任何系统事件推送（包括紧急告警）。' : '已关闭。系统已暂停向你发送任何通知。' }}
              </div>
            </el-form-item>

            <el-form-item label="默认接收渠道" class="mt-6">
              <el-select v-model="notifForm.enabled_channels" multiple clearable class="w-full">
                <el-option v-for="item in channelOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
              <div class="text-xs text-gray-500 mt-1.5 leading-snug">
                你期望通过哪些方式接收通知。前提是你已经在“个人资料”中绑定了对应的联系方式（如邮箱）。
              </div>
            </el-form-item>

            <el-form-item label="通道接收目标（可选）" class="mt-6">
              <div class="text-xs text-gray-500 mb-3 leading-snug">
                仅影响站外通知（企业微信/钉钉/飞书/Webhook）。留空则使用「通知配置中心」的全局通道映射；填写后将按个人偏好发送到对应的 Message Pusher 通道。
              </div>
              <div v-for="item in channelOptions" :key="item.value" v-show="notifForm.enabled_channels.includes(item.value)" class="mb-4">
                <div class="text-sm text-gray-700 mb-1">{{ item.label }} 接收目标（Message Pusher 通道名）</div>
                <el-input
                  v-model="notifForm.channel_targets[item.value]"
                  clearable
                  placeholder="例如：dingtalk_robot_001（留空表示使用全局映射）"
                />
              </div>
            </el-form-item>
            
            <el-form-item label="关心的业务事件" class="mt-6">
              <el-select v-model="notifForm.subscribed_biz_types" multiple clearable class="w-full">
                <el-option v-for="item in bizTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
              <div class="text-xs text-gray-500 mt-1.5 leading-snug">
                选择你希望被推送到上述渠道的事件类型。未选中的事件仅会在系统中默默产生，不会主动打扰你。
              </div>
            </el-form-item>

            <el-divider border-style="dashed" class="!my-8" />

            <div class="text-sm text-gray-500 mb-4">
              外部通知（企业微信/钉钉/飞书/Webhook）统一通过 Message Pusher 网关投递。
              管理员负责网关连接；你可以在上方为每个通道指定 Message Pusher 通道名，未填写则使用「通知配置中心」的全局通道映射。
            </div>

            <div class="mt-8">
              <el-button type="primary" :loading="notifSaving" @click="savePreference">更新通知偏好</el-button>
            </div>
          </el-form>
        </div>

        <!-- 3. 安全设置 -->
        <div v-show="activeTab === 'security'" class="max-w-md animate-fade-in">
          <h3 class="text-lg font-semibold text-gray-800 mb-6">修改登录密码</h3>
          <el-form :model="pwdForm" label-position="top" @submit.prevent>
            <el-form-item label="当前密码">
              <el-input v-model="pwdForm.old_password" type="password" show-password placeholder="请输入当前使用的密码" />
            </el-form-item>
            <el-form-item label="新密码">
              <el-input v-model="pwdForm.new_password" type="password" show-password placeholder="至少 8 个字符" />
            </el-form-item>
            <el-form-item label="确认新密码">
              <el-input v-model="pwdForm.confirm_password" type="password" show-password placeholder="再次输入新密码" @keyup.enter="submitPwd" />
            </el-form-item>

            <div class="mt-8">
              <el-button type="primary" :loading="pwdSaving" @click="submitPwd">确认修改密码</el-button>
            </div>
          </el-form>
        </div>

      </div>
    </div>
  </div>
</template>

<style scoped>
.animate-fade-in {
  animation: fadeIn 0.3s ease-out;
}
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(5px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
