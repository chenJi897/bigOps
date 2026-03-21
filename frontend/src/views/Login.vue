<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { authApi } from '../api'
import { useUserStore } from '../stores/user'

const router = useRouter()
const userStore = useUserStore()
const isLogin = ref(true) // true=登录模式, false=注册模式
const loading = ref(false)

const loginForm = ref({ username: '', password: '' })
const registerForm = ref({ username: '', password: '', email: '' })

async function handleLogin() {
  if (!loginForm.value.username || !loginForm.value.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    await userStore.login(loginForm.value.username, loginForm.value.password)
    ElMessage.success('登录成功')
    router.push('/')
  } catch {
    // 错误已由拦截器处理
  } finally {
    loading.value = false
  }
}

async function handleRegister() {
  const { username, password, email } = registerForm.value
  if (!username || !password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    await authApi.register(username, password, email)
    ElMessage.success('注册成功，请登录')
    isLogin.value = true
    loginForm.value.username = username
    loginForm.value.password = ''
  } catch {
    // 错误已由拦截器处理
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-bg">
    <!-- 装饰性几何图形 -->
    <div class="geo geo-1"></div>
    <div class="geo geo-2"></div>
    <div class="geo geo-3"></div>

    <div class="login-card">
      <div class="brand">
        <div class="brand-icon"><el-icon><Monitor /></el-icon></div>
        <h1>BigOps</h1>
        <p class="brand-sub">智能运维管理平台</p>
      </div>

      <div class="tab-switch">
        <span :class="{ active: isLogin }" @click="isLogin = true">登 录</span>
        <span :class="{ active: !isLogin }" @click="isLogin = false">注 册</span>
      </div>

      <!-- 登录表单 -->
      <el-form v-if="isLogin" @submit.prevent="handleLogin" class="form">
        <el-form-item>
          <el-input
            v-model="loginForm.username"
            placeholder="用户名"
            prefix-icon="User"
            size="large"
            class="custom-input"
          />
        </el-form-item>
        <el-form-item>
          <el-input
            v-model="loginForm.password"
            type="password"
            placeholder="密码"
            prefix-icon="Lock"
            size="large"
            show-password
            class="custom-input"
            @keyup.enter="handleLogin"
          />
        </el-form-item>
        <el-button
          type="primary"
          size="large"
          :loading="loading"
          class="submit-btn"
          @click="handleLogin"
        >
          登录
        </el-button>
      </el-form>

      <!-- 注册表单 -->
      <el-form v-else @submit.prevent="handleRegister" class="form">
        <el-form-item>
          <el-input
            v-model="registerForm.username"
            placeholder="用户名（至少3位）"
            prefix-icon="User"
            size="large"
            class="custom-input"
          />
        </el-form-item>
        <el-form-item>
          <el-input
            v-model="registerForm.password"
            type="password"
            placeholder="密码（至少8位，含大小写字母和数字）"
            prefix-icon="Lock"
            size="large"
            show-password
            class="custom-input"
          />
        </el-form-item>
        <el-form-item>
          <el-input
            v-model="registerForm.email"
            placeholder="邮箱（可选）"
            prefix-icon="Message"
            size="large"
            class="custom-input"
            @keyup.enter="handleRegister"
          />
        </el-form-item>
        <el-button
          type="primary"
          size="large"
          :loading="loading"
          class="submit-btn"
          @click="handleRegister"
        >
          注册账号
        </el-button>
      </el-form>

      <p class="footer-tip">安全、高效、智能的云原生运维平台</p>
    </div>
  </div>
</template>

<style scoped>
.login-bg {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0f172a;
  position: relative;
  overflow: hidden;
}

/* 装饰几何形状 */
.geo {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.35;
  pointer-events: none;
}
.geo-1 {
  width: 500px; height: 500px;
  background: radial-gradient(circle, #1d4ed8, transparent);
  top: -100px; left: -100px;
}
.geo-2 {
  width: 400px; height: 400px;
  background: radial-gradient(circle, #0ea5e9, transparent);
  bottom: -80px; right: -80px;
}
.geo-3 {
  width: 300px; height: 300px;
  background: radial-gradient(circle, #6366f1, transparent);
  top: 50%; left: 55%;
  transform: translate(-50%, -50%);
}

.login-card {
  position: relative;
  z-index: 1;
  width: 420px;
  padding: 40px 44px 36px;
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 20px;
  box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5);
}

.brand {
  text-align: center;
  margin-bottom: 28px;
}
.brand-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px; height: 52px;
  background: linear-gradient(135deg, #3b82f6, #6366f1);
  border-radius: 14px;
  font-size: 26px;
  color: #fff;
  margin-bottom: 12px;
  box-shadow: 0 8px 24px rgba(99, 102, 241, 0.4);
}
.brand h1 {
  margin: 0 0 4px;
  font-size: 26px;
  font-weight: 700;
  color: #f1f5f9;
  letter-spacing: 2px;
}
.brand-sub {
  margin: 0;
  font-size: 13px;
  color: #94a3b8;
}

.tab-switch {
  display: flex;
  gap: 0;
  background: rgba(255, 255, 255, 0.07);
  border-radius: 10px;
  padding: 4px;
  margin-bottom: 24px;
}
.tab-switch span {
  flex: 1;
  text-align: center;
  padding: 8px 0;
  border-radius: 7px;
  font-size: 14px;
  color: #94a3b8;
  cursor: pointer;
  transition: all 0.25s;
  font-weight: 500;
  letter-spacing: 2px;
}
.tab-switch span.active {
  background: linear-gradient(135deg, #3b82f6, #6366f1);
  color: #fff;
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
}

.form :deep(.el-form-item) {
  margin-bottom: 16px;
}
.custom-input :deep(.el-input__wrapper) {
  background: rgba(255, 255, 255, 0.08) !important;
  border: 1px solid rgba(255, 255, 255, 0.15) !important;
  box-shadow: none !important;
  border-radius: 10px;
  transition: border-color 0.2s;
}
.custom-input :deep(.el-input__wrapper:hover) {
  border-color: rgba(99, 102, 241, 0.6) !important;
}
.custom-input :deep(.el-input__wrapper.is-focus) {
  border-color: #6366f1 !important;
  box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2) !important;
}
.custom-input :deep(.el-input__inner) {
  color: #f1f5f9;
  background: transparent;
}
.custom-input :deep(.el-input__inner::placeholder) {
  color: #64748b;
}
.custom-input :deep(.el-input__prefix-inner .el-icon),
.custom-input :deep(.el-input__suffix-inner .el-icon) {
  color: #64748b;
}

.submit-btn {
  width: 100%;
  border-radius: 10px;
  background: linear-gradient(135deg, #3b82f6, #6366f1) !important;
  border: none !important;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 4px;
  height: 46px;
  margin-top: 4px;
  box-shadow: 0 6px 20px rgba(99, 102, 241, 0.4);
  transition: opacity 0.2s, transform 0.1s;
}
.submit-btn:hover {
  opacity: 0.92;
  transform: translateY(-1px);
}
.submit-btn:active {
  transform: translateY(0);
}

.footer-tip {
  text-align: center;
  margin: 20px 0 0;
  font-size: 12px;
  color: #475569;
}
</style>
