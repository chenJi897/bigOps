<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { authApi } from '../api'
import { useUserStore } from '../stores/user'
import { usePermissionStore } from '../stores/permission'
import { resetRouter } from '../router'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const permissionStore = usePermissionStore()
const isCollapse = ref(false)

// 修改密码
const pwdVisible = ref(false)
const pwdForm = ref({ old_password: '', new_password: '', confirm_password: '' })

const activeMenu = computed(() => route.path)

// 从 permissionStore 获取菜单树
const menuTree = computed(() => permissionStore.menus)

onMounted(async () => {
  if (!userStore.userInfo) {
    try {
      await userStore.fetchUserInfo()
    } catch {
      router.push('/login')
    }
  }
})

async function handleLogout() {
  try {
    await ElMessageBox.confirm('确定退出登录？', '提示', { type: 'warning' })
    await userStore.logout()
    permissionStore.reset()
    resetRouter()
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
    userStore.clearToken()
    permissionStore.reset()
    resetRouter()
    router.push('/login')
  } catch {}
}
</script>

<template>
  <el-container class="layout">
    <el-aside :width="isCollapse ? '64px' : '200px'" class="aside">
      <div class="logo">{{ isCollapse ? 'B' : 'BigOps' }}</div>
      <el-scrollbar>
        <el-menu
          :default-active="activeMenu"
          router
          :collapse="isCollapse"
          background-color="#304156"
          text-color="#bfcbd9"
          active-text-color="#409eff"
        >
          <!-- 仪表盘（固定） -->
          <el-menu-item index="/dashboard">
            <el-icon><Odometer /></el-icon>
            <template #title>仪表盘</template>
          </el-menu-item>

          <!-- 动态菜单（来自后端权限配置） -->
          <template v-for="menu in menuTree" :key="menu.id">
            <!-- 有子菜单的目录 -->
            <el-sub-menu v-if="menu.children?.length && menu.type !== 3" :index="menu.path || String(menu.id)">
              <template #title>
                <el-icon><component :is="menu.icon || 'Folder'" /></el-icon>
                <span>{{ menu.title }}</span>
              </template>
              <template v-for="child in menu.children" :key="child.id">
                <el-menu-item v-if="child.type !== 3 && child.path" :index="child.path">
                  <el-icon><component :is="child.icon || 'Document'" /></el-icon>
                  <template #title>{{ child.title }}</template>
                </el-menu-item>
              </template>
            </el-sub-menu>
            <!-- 没有子菜单的页面 -->
            <el-menu-item v-else-if="menu.type !== 3 && menu.path" :index="menu.path">
              <el-icon><component :is="menu.icon || 'Document'" /></el-icon>
              <template #title>{{ menu.title }}</template>
            </el-menu-item>
          </template>
        </el-menu>
      </el-scrollbar>
    </el-aside>

    <el-container>
      <el-header class="header">
        <div class="header-left">
          <el-icon class="collapse-btn" @click="isCollapse = !isCollapse">
            <Fold v-if="!isCollapse" /><Expand v-else />
          </el-icon>
          <!-- 面包屑 -->
          <el-breadcrumb separator="/" class="breadcrumb">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-for="item in route.matched.filter(r => r.meta?.title)" :key="item.path">
              {{ item.meta.title }}
            </el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <el-dropdown trigger="click">
            <span class="user-drop">
              <el-icon><User /></el-icon>
              {{ userStore.userInfo?.username }}
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
.header-left { display: flex; align-items: center; gap: 12px; }
.collapse-btn { font-size: 20px; cursor: pointer; }
.breadcrumb { margin-left: 4px; }
.user-drop { display: flex; align-items: center; gap: 4px; cursor: pointer; font-size: 14px; color: #606266; }
.main { background: #f0f2f5; }
.el-menu { border-right: none; }
</style>
