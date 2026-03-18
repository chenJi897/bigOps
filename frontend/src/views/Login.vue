<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { authApi } from '../api'

const router = useRouter()
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
    const res: any = await authApi.login(loginForm.value.username, loginForm.value.password)
    localStorage.setItem('token', res.data.token)
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
  <div class="login-container">
    <div class="login-card">
      <h2>BigOps 运维平台</h2>
      <p class="subtitle">{{ isLogin ? '登录' : '注册' }}</p>

      <!-- 登录表单 -->
      <el-form v-if="isLogin" @submit.prevent="handleLogin">
        <el-form-item>
          <el-input v-model="loginForm.username" placeholder="用户名" prefix-icon="User" size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="loginForm.password" type="password" placeholder="密码" prefix-icon="Lock" size="large"
            show-password @keyup.enter="handleLogin" />
        </el-form-item>
        <el-button type="primary" size="large" :loading="loading" style="width:100%" @click="handleLogin">
          登录
        </el-button>
      </el-form>

      <!-- 注册表单 -->
      <el-form v-else @submit.prevent="handleRegister">
        <el-form-item>
          <el-input v-model="registerForm.username" placeholder="用户名（至少3位）" prefix-icon="User" size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="registerForm.password" type="password" placeholder="密码（至少6位）" prefix-icon="Lock"
            size="large" show-password />
        </el-form-item>
        <el-form-item>
          <el-input v-model="registerForm.email" placeholder="邮箱（可选）" prefix-icon="Message" size="large"
            @keyup.enter="handleRegister" />
        </el-form-item>
        <el-button type="primary" size="large" :loading="loading" style="width:100%" @click="handleRegister">
          注册
        </el-button>
      </el-form>

      <div class="toggle">
        <span v-if="isLogin">没有账号？<el-link type="primary" @click="isLogin = false">去注册</el-link></span>
        <span v-else>已有账号？<el-link type="primary" @click="isLogin = true">去登录</el-link></span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
  width: 400px;
  padding: 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
}

.login-card h2 {
  text-align: center;
  margin-bottom: 4px;
  color: #303133;
}

.subtitle {
  text-align: center;
  color: #909399;
  margin-bottom: 30px;
  font-size: 14px;
}

.toggle {
  text-align: center;
  margin-top: 16px;
  font-size: 14px;
  color: #909399;
}
</style>
