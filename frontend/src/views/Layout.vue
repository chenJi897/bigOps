<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { authApi, notificationApi } from '../api'
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
const notificationVisible = ref(false)
const notificationLoading = ref(false)
const notifications = ref<any[]>([])
const unreadCount = ref(0)
let notificationTimer: number | undefined

// 修改密码
const pwdVisible = ref(false)
const pwdForm = ref({ old_password: '', new_password: '', confirm_password: '' })

const activeMenu = computed(() => (route.meta?.activeMenu as string) || route.path)

function filterSidebarMenus(items: any[]): any[] {
  const result: any[] = []
  for (const item of items || []) {
    if (item.type === 3 || item.visible === 0 || item.component === 'TicketDetail') continue
    const children = item.children?.length ? filterSidebarMenus(item.children) : []
    if (children.length > 0) {
      result.push({ ...item, children })
      continue
    }
    if (item.path) {
      result.push({ ...item, children: [] })
    }
  }
  return result
}

// 从 permissionStore 获取菜单树，并过滤掉不应该出现在侧边栏的路由入口。
const menuTree = computed(() => filterSidebarMenus(permissionStore.menus))

// 标签页：路由变化时自动添加
watch(() => route.path, (path) => {
  if (path === '/login' || path === '/404') return
  const title = (route.meta?.title as string) || route.name as string || path
  const componentName = (route.meta?.componentName as string) || ''
  tagsStore.addView({ path, title, name: route.name as string, componentName, closable: path !== '/dashboard' })
}, { immediate: true })

// keep-alive 缓存列表：所有已打开标签的组件名（匹配 Vue 组件 name）
const cachedViews = computed(() =>
  tagsStore.visitedViews.map(v => v.componentName).filter(Boolean) as string[]
)

// 右键菜单
const ctxMenuVisible = ref(false)
const ctxMenuX = ref(0)
const ctxMenuY = ref(0)
const ctxMenuPath = ref('')

function onTagContextMenu(path: string, e: MouseEvent) {
  e.preventDefault()
  ctxMenuPath.value = path
  ctxMenuX.value = e.clientX
  ctxMenuY.value = e.clientY
  ctxMenuVisible.value = true
  document.addEventListener('click', closeCtxMenu, { once: true })
}

function closeCtxMenu() {
  ctxMenuVisible.value = false
}

function ctxAction(action: string) {
  const path = ctxMenuPath.value
  ctxMenuVisible.value = false
  let next: string | undefined
  switch (action) {
    case 'closeCurrent':
      next = tagsStore.removeView(path)
      if (next !== route.path) router.push(next)
      break
    case 'closeOthers':
      tagsStore.closeOthers(path)
      if (path !== route.path) router.push(path)
      break
    case 'closeRight':
      tagsStore.closeRight(path)
      if (!tagsStore.visitedViews.some(v => v.path === route.path)) router.push(path)
      break
    case 'closeLeft':
      tagsStore.closeLeft(path)
      if (!tagsStore.visitedViews.some(v => v.path === route.path)) router.push(path)
      break
    case 'closeAll':
      next = tagsStore.closeAll()
      if (next !== route.path) router.push(next!)
      break
  }
}

function handleTabRemove(path: string) {
  const next = tagsStore.removeView(path)
  if (next !== route.path) router.push(next)
}

onMounted(async () => {
  if (!userStore.userInfo) {
    try {
      await userStore.fetchUserInfo()
    } catch {
      router.push('/login')
    }
  }
  fetchUnreadCount()
  notificationTimer = window.setInterval(() => {
    fetchUnreadCount()
  }, 30000)
})

onBeforeUnmount(() => {
  if (notificationTimer) {
    window.clearInterval(notificationTimer)
  }
})

async function handleLogout() {
  try {
    await ElMessageBox.confirm('确定退出登录？', '提示', { type: 'warning' })
    await userStore.logout()
    permissionStore.reset()
    tagsStore.reset()
    resetRouter()
    router.push('/login')
  } catch {}
}

async function fetchUnreadCount() {
  try {
    const res: any = await notificationApi.unreadCount()
    unreadCount.value = res.data?.count || 0
  } catch {}
}

async function openNotifications() {
  notificationVisible.value = true
  notificationLoading.value = true
  try {
    const res: any = await notificationApi.inApp()
    notifications.value = res.data || []
  } finally {
    notificationLoading.value = false
    fetchUnreadCount()
  }
}

async function markNotificationRead(item: any) {
  if (item.read_at) return
  try {
    await notificationApi.markRead(item.id)
    item.read_at = new Date().toISOString()
    fetchUnreadCount()
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
    tagsStore.reset()
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
          <el-badge :value="unreadCount" :hidden="unreadCount === 0" class="notice-badge">
            <el-button circle text @click="openNotifications">
              <el-icon><Bell /></el-icon>
            </el-button>
          </el-badge>
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
        <div class="tags-scroll">
          <div
            v-for="tag in tagsStore.visitedViews"
            :key="tag.path"
            class="tag-item"
            :class="{ active: tag.path === route.path }"
            @click="router.push(tag.path)"
            @contextmenu="onTagContextMenu(tag.path, $event)"
          >
            <span class="tag-dot" />
            <span class="tag-title">{{ tag.title }}</span>
            <el-icon v-if="tag.closable" class="tag-close" @click.stop="handleTabRemove(tag.path)"><Close /></el-icon>
          </div>
        </div>
      </div>

      <!-- 右键菜单 -->
      <teleport to="body">
        <div
          v-if="ctxMenuVisible"
          class="ctx-menu"
          :style="{ left: ctxMenuX + 'px', top: ctxMenuY + 'px' }"
        >
          <div class="ctx-item" @click="ctxAction('closeCurrent')">关闭当前</div>
          <div class="ctx-item" @click="ctxAction('closeOthers')">关闭其他</div>
          <div class="ctx-item" @click="ctxAction('closeLeft')">关闭左侧</div>
          <div class="ctx-item" @click="ctxAction('closeRight')">关闭右侧</div>
          <div class="ctx-item" @click="ctxAction('closeAll')">关闭全部</div>
        </div>
      </teleport>

      <el-main class="main">
        <router-view v-slot="{ Component }">
          <keep-alive :include="cachedViews">
            <component :is="Component" :key="route.path" />
          </keep-alive>
        </router-view>
      </el-main>
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

    <el-drawer v-model="notificationVisible" title="站内通知" size="420px">
      <div class="notification-list" v-loading="notificationLoading">
        <div
          v-for="item in notifications"
          :key="item.id"
          class="notification-item"
          :class="{ unread: !item.read_at }"
          @click="markNotificationRead(item)"
        >
          <div class="notification-title-row">
            <span class="notification-title">{{ item.title }}</span>
            <el-tag v-if="!item.read_at" size="small" type="danger">未读</el-tag>
          </div>
          <div class="notification-content">{{ item.content }}</div>
          <div class="notification-meta">
            <span>{{ item.created_at }}</span>
            <span>{{ item.level }}</span>
          </div>
        </div>
        <el-empty v-if="!notificationLoading && notifications.length === 0" description="暂无通知" />
      </div>
    </el-drawer>
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
.header-right { display: flex; align-items: center; gap: 12px; }
.user-drop { display: flex; align-items: center; gap: 4px; cursor: pointer; font-size: 14px; color: #606266; }
.main { background: #f0f2f5; }
.el-menu { border-right: none; }
.notice-badge :deep(.el-badge__content) { top: 6px; right: 6px; }

/* 标签栏 */
.tags-bar {
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  padding: 4px 8px;
}
.tags-scroll {
  display: flex; align-items: center;
  gap: 4px;
  overflow-x: auto;
  scrollbar-width: none;
}
.tags-scroll::-webkit-scrollbar { display: none; }

.tag-item {
  display: inline-flex; align-items: center; gap: 4px;
  height: 26px; padding: 0 10px;
  border: 1px solid #d8dce5; border-radius: 3px;
  font-size: 12px; color: #495060;
  cursor: pointer; white-space: nowrap;
  transition: all 0.2s;
  user-select: none;
}
.tag-item:hover { color: #409eff; border-color: #b3d8ff; }
.tag-item.active { background: #409eff; color: #fff; border-color: #409eff; }
.tag-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: #e4e7ed;
}
.tag-item.active .tag-dot { background: #fff; }
.tag-close { font-size: 12px; border-radius: 50%; transition: all 0.15s; }
.tag-close:hover { background: rgba(0,0,0,0.15); color: #fff; }
.tag-item.active .tag-close:hover { background: rgba(255,255,255,0.3); }

/* 右键菜单 */
.ctx-menu {
  position: fixed; z-index: 9999;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.12);
  padding: 4px 0;
  min-width: 120px;
}
.ctx-item {
  padding: 6px 16px;
  font-size: 13px; color: #606266;
  cursor: pointer;
}
.ctx-item:hover { background: #ecf5ff; color: #409eff; }

.notification-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.notification-item {
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 12px;
  background: #fff;
  cursor: pointer;
}

.notification-item.unread {
  border-color: #93c5fd;
  background: #f8fbff;
}

.notification-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.notification-title {
  font-weight: 700;
  color: #1f2937;
}

.notification-content {
  margin-top: 6px;
  color: #4b5563;
  line-height: 1.6;
  white-space: pre-wrap;
}

.notification-meta {
  margin-top: 8px;
  display: flex;
  justify-content: space-between;
  color: #9ca3af;
  font-size: 12px;
}
</style>
