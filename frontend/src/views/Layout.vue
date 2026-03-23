<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { authApi } from '../api'
import { useUserStore } from '../stores/user'
import { usePermissionStore } from '../stores/permission'
import { useTagsViewStore } from '../stores/tagsView'
import { resetRouter } from '../router'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const permissionStore = usePermissionStore()
const tagsStore = useTagsViewStore()
const isCollapse = ref(false)

// 修改密码
const pwdVisible = ref(false)
const pwdForm = ref({ old_password: '', new_password: '', confirm_password: '' })

const activeMenu = computed(() => route.path)

// 从 permissionStore 获取菜单树
const menuTree = computed(() => permissionStore.menus)

// 标签页：路由变化时自动添加
watch(() => route.path, (path) => {
  if (path === '/login' || path === '/404') return
  const title = (route.meta?.title as string) || route.name as string || path
  tagsStore.addView({ path, title, name: route.name as string, closable: path !== '/dashboard' })
}, { immediate: true })

function handleTabClick(tab: any) {
  const path = tab.props.name
  if (path !== route.path) router.push(path)
}

function handleTabRemove(path: string) {
  const next = tagsStore.removeView(path)
  if (next !== route.path) router.push(next)
}

function handleTabCommand(cmd: string) {
  if (cmd === 'closeOthers') {
    tagsStore.closeOthers(route.path)
  } else if (cmd === 'closeAll') {
    const next = tagsStore.closeAll()
    if (next !== route.path) router.push(next)
  }
}

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
          <el-menu-item index="/dashboard">
            <el-icon><Odometer /></el-icon>
            <template #title>仪表盘</template>
          </el-menu-item>

          <template v-for="menu in menuTree" :key="menu.id">
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

      <!-- 标签页 -->
      <div class="tags-bar">
        <el-tabs
          v-model="tagsStore.activeView"
          type="card"
          closable
          @tab-click="handleTabClick"
          @tab-remove="handleTabRemove"
        >
          <el-tab-pane
            v-for="tag in tagsStore.visitedViews"
            :key="tag.path"
            :label="tag.title"
            :name="tag.path"
            :closable="tag.closable"
          />
        </el-tabs>
        <el-dropdown trigger="click" @command="handleTabCommand" class="tags-action">
          <el-icon size="16"><ArrowDown /></el-icon>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="closeOthers">关闭其他</el-dropdown-item>
              <el-dropdown-item command="closeAll">关闭全部</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>

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
.header { background: #fff; display: flex; align-items: center; justify-content: space-between; box-shadow: 0 1px 4px rgba(0,0,0,0.08); padding: 0 16px; height: 50px; }
.header-left { display: flex; align-items: center; gap: 12px; }
.collapse-btn { font-size: 20px; cursor: pointer; }
.breadcrumb { margin-left: 4px; }
.user-drop { display: flex; align-items: center; gap: 4px; cursor: pointer; font-size: 14px; color: #606266; }
.main { background: #f0f2f5; }
.el-menu { border-right: none; }

/* 标签栏 */
.tags-bar {
  display: flex; align-items: center;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  padding: 0 8px;
  height: 34px;
}
.tags-bar :deep(.el-tabs) { flex: 1; }
.tags-bar :deep(.el-tabs__header) { margin: 0; border: none; }
.tags-bar :deep(.el-tabs__nav) { border: none; }
.tags-bar :deep(.el-tabs__item) {
  height: 28px; line-height: 28px;
  font-size: 12px;
  padding: 0 12px;
  border: 1px solid #d8dce5 !important;
  border-radius: 3px;
  margin: 0 3px;
  transition: all 0.2s;
}
.tags-bar :deep(.el-tabs__item.is-active) {
  background: #409eff; color: #fff;
  border-color: #409eff !important;
}
.tags-bar :deep(.el-tabs__item.is-active .is-icon-close) { color: #fff; }
.tags-action { cursor: pointer; margin-left: 8px; color: #606266; flex-shrink: 0; }
</style>
