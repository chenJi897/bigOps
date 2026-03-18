<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { authApi } from '../api'

const router = useRouter()
const userInfo = ref<any>(null)
const isCollapse = ref(false)

// 修改密码
const pwdVisible = ref(false)
const pwdForm = ref({ old_password: '', new_password: '', confirm_password: '' })

onMounted(async () => {
  try {
    const res: any = await authApi.getInfo()
    userInfo.value = res.data
  } catch {
    router.push('/login')
  }
})

async function handleLogout() {
  try {
    await ElMessageBox.confirm('确定退出登录？', '提示', { type: 'warning' })
    await authApi.logout()
    localStorage.removeItem('token')
    router.push('/login')
  } catch {}
}

function openPwdDialog() {
  pwdForm.value = { old_password: '', new_password: '', confirm_password: '' }
  pwdVisible.value = true
}

async function submitPwd() {
  const { old_password, new_password, confirm_password } = pwdForm.value
  if (!old_password || !new_password) { ElMessage.warning('请填写完整'); return }
  if (new_password !== confirm_password) { ElMessage.warning('两次密码不一致'); return }
  try {
    await authApi.changePassword(old_password, new_password)
    ElMessage.success('密码修改成功，请重新登录')
    pwdVisible.value = false
    localStorage.removeItem('token')
    router.push('/login')
  } catch {}
}
</script>

<template>
  <el-container class="layout">
    <el-aside :width="isCollapse ? '64px' : '200px'" class="aside">
      <div class="logo">{{ isCollapse ? 'B' : 'BigOps' }}</div>
      <el-menu :default-active="$route.path" router :collapse="isCollapse" background-color="#304156" text-color="#bfcbd9" active-text-color="#409eff">
        <el-sub-menu index="/system">
          <template #title><el-icon><Setting /></el-icon><span>系统管理</span></template>
          <el-menu-item index="/system/users"><el-icon><User /></el-icon>用户管理</el-menu-item>
          <el-menu-item index="/system/roles"><el-icon><Key /></el-icon>角色管理</el-menu-item>
          <el-menu-item index="/system/menus"><el-icon><Menu /></el-icon>菜单管理</el-menu-item>
        </el-sub-menu>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="header">
        <el-icon class="collapse-btn" @click="isCollapse = !isCollapse"><Fold v-if="!isCollapse" /><Expand v-else /></el-icon>
        <div class="header-right">
          <el-dropdown trigger="click">
            <span class="user-drop">
              <el-icon><User /></el-icon>
              {{ userInfo?.username }}
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="openPwdDialog"><el-icon><Lock /></el-icon>修改密码</el-dropdown-item>
                <el-dropdown-item divided @click="handleLogout"><el-icon><SwitchButton /></el-icon>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      <el-main class="main"><router-view /></el-main>
    </el-container>

    <el-dialog v-model="pwdVisible" title="修改密码" width="400px">
      <el-form label-width="80px">
        <el-form-item label="原密码"><el-input v-model="pwdForm.old_password" type="password" show-password /></el-form-item>
        <el-form-item label="新密码"><el-input v-model="pwdForm.new_password" type="password" show-password /></el-form-item>
        <el-form-item label="确认密码"><el-input v-model="pwdForm.confirm_password" type="password" show-password @keyup.enter="submitPwd" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdVisible = false">取消</el-button>
        <el-button type="primary" @click="submitPwd">确定</el-button>
      </template>
    </el-dialog>
  </el-container>
</template>

<style scoped>
.layout { height: 100vh; }
.aside { background: #304156; transition: width 0.3s; overflow: hidden; }
.logo { height: 50px; line-height: 50px; text-align: center; color: #fff; font-size: 18px; font-weight: 600; background: #263445; }
.header { background: #fff; display: flex; align-items: center; justify-content: space-between; box-shadow: 0 1px 4px rgba(0,0,0,0.08); padding: 0 16px; }
.collapse-btn { font-size: 20px; cursor: pointer; }
.user-drop { display: flex; align-items: center; gap: 4px; cursor: pointer; font-size: 14px; color: #606266; }
.main { background: #f0f2f5; }
.el-menu { border-right: none; }
</style>
