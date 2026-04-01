<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { notificationApi } from '../api'
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
const notificationFilter = ref<'unread' | 'all'>('unread')
let notificationTimer: number | undefined

const activeMenu = computed(() => {
  if (route.name === 'TicketDetail') {
    const from = route.query.from
    if (from === 'todo') return '/ticket/todo'
    if (from === 'applied') return '/ticket/applied'
  }
  return (route.meta?.activeMenu as string) || route.path
})

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

const tagsBarRef = ref<HTMLElement | null>(null)

function handleTagsScroll(e: WheelEvent) {
  if (tagsBarRef.value) {
    tagsBarRef.value.scrollLeft += e.deltaY > 0 ? 50 : -50
  }
}

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


// Command Palette
const cmdPaletteVisible = ref(false)
const searchQuery = ref('')
const searchInputRef = ref<any>(null)
const selectedIndex = ref(0)

watch(searchQuery, () => {
  selectedIndex.value = 0
})

const flatMenus = computed(() => {
  const result: any[] = []
  function flatten(menus: any[]) {
    for (const menu of menus) {
      if (menu.path && menu.title) {
        result.push(menu)
      }
      if (menu.children && menu.children.length) {
        flatten(menu.children)
      }
    }
  }
  flatten(menuTree.value)
  return result
})

const filteredMenus = computed(() => {
  if (!searchQuery.value) return flatMenus.value.slice(0, 10)
  return flatMenus.value.filter(m => m.title.toLowerCase().includes(searchQuery.value.toLowerCase())).slice(0, 10)
})

function handleCmdK(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    cmdPaletteVisible.value = true
    searchQuery.value = ''
    selectedIndex.value = 0
    setTimeout(() => {
      searchInputRef.value?.focus()
    }, 100)
  }
}

function handleSelectCommand(menu: any) {
  cmdPaletteVisible.value = false
  router.push(menu.path)
}

function handleCommandKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    if (filteredMenus.value.length > 0) {
      selectedIndex.value = (selectedIndex.value + 1) % filteredMenus.value.length
    }
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    if (filteredMenus.value.length > 0) {
      selectedIndex.value = (selectedIndex.value - 1 + filteredMenus.value.length) % filteredMenus.value.length
    }
  } else if (e.key === 'Enter' && filteredMenus.value.length > 0) {
    e.preventDefault()
    handleSelectCommand(filteredMenus.value[selectedIndex.value])
  }
}

onMounted(async () => {
  window.addEventListener('keydown', handleCmdK)
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
  window.removeEventListener('keydown', handleCmdK)
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

async function loadNotifications() {
  notificationLoading.value = true
  try {
    const res: any = await notificationApi.inApp(notificationFilter.value === 'unread')
    notifications.value = res.data || []
  } finally {
    notificationLoading.value = false
  }
}

async function openNotifications() {
  notificationVisible.value = true
  await loadNotifications()
  fetchUnreadCount()
}

watch(notificationFilter, () => {
  if (notificationVisible.value) {
    loadNotifications()
  }
})

async function markNotificationRead(item: any) {
  if (item.read_at) return
  try {
    await notificationApi.markRead(item.id)
    item.read_at = new Date().toISOString()
    if (notificationFilter.value === 'unread') {
      notifications.value = notifications.value.filter(notification => notification.id !== item.id)
    }
    unreadCount.value = Math.max(0, unreadCount.value - 1)
  } catch {}
}

async function markAllNotificationsRead() {
  if (unreadCount.value === 0) {
    ElMessage.info('当前没有未读通知')
    return
  }
  try {
    await notificationApi.markAllRead()
    unreadCount.value = 0
    if (notificationFilter.value === 'unread') {
      notifications.value = []
    } else {
      const now = new Date().toISOString()
      notifications.value = notifications.value.map(item => ({ ...item, read_at: item.read_at || now }))
    }
    ElMessage.success('已全部标记为已读')
  } catch {}
}

async function clearReadNotifications() {
  const hasRead = notifications.value.some(item => item.read_at)
  if (!hasRead && notificationFilter.value === 'all') {
    ElMessage.info('当前没有已读通知')
    return
  }
  try {
    await notificationApi.clearRead()
    if (notificationFilter.value === 'all') {
      notifications.value = notifications.value.filter(item => !item.read_at)
    }
    ElMessage.success('已清空已读通知')
  } catch {}
}

function resolveNotificationTarget(item: any) {
  if (!item?.biz_type || !item?.biz_id) return ''
  switch (item.biz_type) {
    case 'ticket':
      return `/ticket/detail/${item.biz_id}`
    case 'task_execution':
    case 'execution':
      return `/task/execution/${item.biz_id}`
    case 'cicd_pipeline':
      return `/cicd/runs?pipeline_id=${item.biz_id}`
    case 'approval':
      return `/approval/inbox`
    case 'alert_event':
      return '/monitor/alert-rules'
    case 'notification':
      return '/notification/console'
    default:
      return ''
  }
}

async function handleNotificationClick(item: any) {
  await markNotificationRead(item)
  const target = resolveNotificationTarget(item)
  if (target) {
    router.push(target)
    notificationVisible.value = false
  }
}

function openMyNotificationSettings() {
  notificationVisible.value = false
  router.push('/user/settings')
}
</script>

<template>
  <el-container class="layout h-screen bg-gray-100">
    <el-aside :width="isCollapse ? '64px' : '240px'" class="aside transition-all duration-300 bg-[#304156] shadow-xl z-20">
      <div class="logo h-14 flex items-center justify-center text-white text-xl font-bold bg-[#2b3643] shadow-sm">{{ isCollapse ? 'B' : 'BigOps' }}</div>
      <el-scrollbar class="menu-scroll h-[calc(100vh-3.5rem)]">
        <el-menu
          :default-active="activeMenu"
          router
          :collapse="isCollapse"
          background-color="#304156"
          text-color="#bfcbd9"
          active-text-color="#409eff"
        >
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

    <el-container class="flex-col overflow-hidden">
      <el-header class="header h-14 bg-white flex items-center justify-between px-4 shadow-sm z-10">
        <div class="header-left flex items-center gap-4">
          <el-icon class="collapse-btn text-xl cursor-pointer text-gray-500 hover:text-indigo-600 transition-colors" @click="isCollapse = !isCollapse">
            <Fold v-if="!isCollapse" /><Expand v-else />
          </el-icon>
          <el-breadcrumb separator="/" class="breadcrumb">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-for="item in route.matched.filter(r => r.meta?.title)" :key="item.path">
              {{ item.meta.title }}
            </el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right flex items-center gap-3 pr-4">
          <el-badge :value="unreadCount" :hidden="unreadCount === 0" class="notice-badge mt-1">
            <el-tooltip content="消息通知" placement="bottom" :show-after="300">
              <el-button circle text @click="openNotifications" class="hover:bg-gray-100 transition-colors text-gray-600 hover:text-indigo-600">
                <el-icon><Bell /></el-icon>
              </el-button>
            </el-tooltip>
          </el-badge>
          
          <el-dropdown trigger="click" class="cursor-pointer ml-3">
            <span class="user-drop flex items-center gap-2 text-gray-700 hover:text-indigo-600 font-medium transition-colors p-1 rounded-md hover:bg-indigo-50">
              <el-icon><User /></el-icon>
              {{ userStore.userInfo?.username }}
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="router.push('/user/settings')"><el-icon><UserFilled /></el-icon>个人设置</el-dropdown-item>
                <el-dropdown-item divided @click="handleLogout"><el-icon><SwitchButton /></el-icon>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <!-- 标签页 -->
      <div 
        class="tags-bar bg-white border-t border-gray-100 shadow-sm px-4 py-2 flex items-center overflow-x-auto whitespace-nowrap scrollbar-hide"
        ref="tagsBarRef"
        @wheel.prevent="handleTagsScroll"
      >
        <div class="tags-scroll flex gap-2 w-max min-w-full">
          <div
            v-for="tag in tagsStore.visitedViews"
            :key="tag.path"
            class="tag-item flex-shrink-0 flex items-center gap-1.5 px-3 py-1.5 text-xs text-gray-600 border border-gray-200 rounded-md cursor-pointer transition-all duration-200 hover:text-indigo-600 hover:border-indigo-300 bg-white shadow-sm"
            :class="{ 'active !bg-indigo-50 !text-indigo-600 !border-indigo-500 font-medium': tag.path === route.path }"
            @click="router.push(tag.path)"
            @contextmenu="onTagContextMenu(tag.path, $event)"
          >
            <span v-if="tag.path === route.path" class="w-2 h-2 rounded-full bg-indigo-500 mr-1 shadow-sm"></span>
            <span class="tag-title tracking-wide truncate max-w-[150px]">{{ tag.title }}</span>
            <el-icon v-if="tag.closable" class="tag-close hover:bg-indigo-100 hover:text-indigo-700 p-0.5 rounded-full transition-colors" @click.stop="handleTabRemove(tag.path)"><Close /></el-icon>
          </div>
        </div>
      </div>

      <!-- 右键菜单 -->
      <teleport to="body">
        <div
          v-if="ctxMenuVisible"
          class="fixed z-[9999] bg-white rounded-lg shadow-xl border border-gray-100 py-1.5 min-w-[120px] text-sm text-gray-700 font-medium"
          :style="{ left: ctxMenuX + 'px', top: ctxMenuY + 'px' }"
        >
          <div class="px-4 py-2 hover:bg-indigo-50 hover:text-indigo-600 cursor-pointer transition-colors flex items-center gap-2" @click="ctxAction('closeCurrent')">关闭当前</div>
          <div class="px-4 py-2 hover:bg-indigo-50 hover:text-indigo-600 cursor-pointer transition-colors flex items-center gap-2" @click="ctxAction('closeOthers')">关闭其他</div>
          <div class="px-4 py-2 hover:bg-indigo-50 hover:text-indigo-600 cursor-pointer transition-colors flex items-center gap-2" @click="ctxAction('closeLeft')">关闭左侧</div>
          <div class="px-4 py-2 hover:bg-indigo-50 hover:text-indigo-600 cursor-pointer transition-colors flex items-center gap-2" @click="ctxAction('closeRight')">关闭右侧</div>
          <div class="h-px bg-gray-100 my-1"></div>
          <div class="px-4 py-2 hover:bg-red-50 hover:text-red-600 cursor-pointer transition-colors flex items-center gap-2" @click="ctxAction('closeAll')">关闭全部</div>
        </div>
      </teleport>

      <el-main class="main bg-slate-50 relative p-5 overflow-auto flex-1">
        <!-- SVG Dot Matrix Background -->
        <div class="absolute inset-0 pointer-events-none opacity-[0.03]" style="background-image: radial-gradient(circle at 1px 1px, #000 1px, transparent 0); background-size: 24px 24px;"></div>
        
        <div class="relative z-10 min-h-full">
          <router-view v-slot="{ Component }">
            <transition name="fade-transform" mode="out-in">
              <keep-alive :include="cachedViews">
                <component :is="Component" :key="route.path" class="bg-white rounded-lg shadow-sm p-6 min-h-full border border-gray-100" />
              </keep-alive>
            </transition>
          </router-view>
        </div>
      </el-main>
    </el-container>

    <el-drawer v-model="notificationVisible" size="420px" :with-header="false">
      <div class="flex flex-col h-full overflow-hidden">
        <!-- Drawer Custom Header -->
        <div class="flex items-center justify-between px-5 py-4 border-b border-gray-100 bg-white">
          <div class="text-lg font-bold text-gray-900">站内通知</div>
          <el-button link type="primary" class="!text-gray-500 hover:!text-indigo-600 transition-colors" @click="openMyNotificationSettings">
            <el-icon class="mr-1.5"><Setting /></el-icon>接收设置
          </el-button>
        </div>

        <div class="flex flex-col flex-1 overflow-hidden p-4 pt-2">
          <!-- Toolbar -->
          <div class="flex items-center justify-between pb-3 mb-2 px-1">
          <el-radio-group v-model="notificationFilter" size="small">
            <el-radio-button label="unread">未读</el-radio-button>
            <el-radio-button label="all">全部</el-radio-button>
          </el-radio-group>
          <div class="flex items-center gap-1">
            <el-button size="small" text @click="loadNotifications" class="!px-2">刷新</el-button>
            <el-button size="small" text @click="markAllNotificationsRead" class="!px-2">全部已读</el-button>
            <el-button size="small" text type="danger" @click="clearReadNotifications" class="!px-2">清空已读</el-button>
          </div>
        </div>
        <div class="flex-1 overflow-y-auto pr-1 space-y-3" v-loading="notificationLoading">
          <div
            v-for="item in notifications"
            :key="item.id"
            class="p-4 rounded-xl border transition-all duration-200 cursor-pointer relative z-10 hover:z-20"
            :class="!item.read_at ? 'bg-indigo-50/50 border-indigo-100 shadow-sm hover:shadow-md' : 'bg-gray-50 border-gray-100 opacity-70 hover:opacity-100'"
            @click.stop="handleNotificationClick(item)"
          >
            <div v-if="!item.read_at" class="absolute top-4 left-0 w-1 h-10 bg-indigo-500 rounded-r-md"></div>
            <div class="flex justify-between items-start mb-2">
              <span class="font-bold text-gray-800 flex-1 pr-3 break-all" :class="{ 'text-indigo-900': !item.read_at }">{{ item.title }}</span>
              <el-tag v-if="!item.read_at" size="small" type="danger" effect="light" class="rounded-md shrink-0">未读</el-tag>
              <el-tag v-else size="small" type="info" effect="plain" class="rounded-md shrink-0">已读</el-tag>
            </div>
            <div class="text-sm text-gray-600 mb-3 whitespace-pre-wrap break-words leading-relaxed">{{ item.content }}</div>
            <div class="flex items-center justify-between text-xs text-gray-400">
              <span class="flex items-center gap-1"><el-icon><Calendar /></el-icon>{{ item.created_at }}</span>
              <el-tag size="small" type="info" class="!bg-transparent !border-none !text-gray-400">{{ item.level }}</el-tag>
            </div>
          </div>
          <el-empty v-if="!notificationLoading && notifications.length === 0" description="暂无通知" :image-size="60" class="mt-12" />
        </div>
      </div>
      </div>
    </el-drawer>

    <!-- Command Palette Dialog -->
    <el-dialog
      v-model="cmdPaletteVisible"
      :show-close="false"
      class="cmd-palette-dialog"
      modal-class="cmd-backdrop"
      width="600px"
      align-center
    >
      <div class="flex flex-col rounded-xl overflow-hidden bg-white shadow-2xl ring-1 ring-black/5" @keydown="handleCommandKeydown">
        <div class="p-4 border-b border-gray-100 flex items-center gap-3">
          <el-icon class="text-xl text-gray-400"><Search /></el-icon>
          <input
            ref="searchInputRef"
            v-model="searchQuery"
            class="flex-1 bg-transparent border-none outline-none text-lg text-gray-700 placeholder-gray-400"
            placeholder="Search commands or jump to..."
            @keydown="handleCommandKeydown"
          />
          <div class="flex items-center gap-1 text-xs text-gray-400 font-mono bg-gray-100 px-2 py-1 rounded">
            <span>ESC</span>
          </div>
        </div>
        <div class="max-h-[60vh] overflow-y-auto p-2">
          <div
            v-for="(menu, index) in filteredMenus"
            :key="menu.path"
            class="flex items-center justify-between p-3 rounded-lg cursor-pointer transition-colors duration-150 group hover:bg-indigo-50/80"
            :class="{ 'bg-indigo-50 border-l-2 border-indigo-500': index === selectedIndex }"
            @click="handleSelectCommand(menu)"
            @mouseenter="selectedIndex = index"
          >
            <div class="flex items-center gap-3">
              <el-icon class="text-gray-400 group-hover:text-indigo-500" :class="{ 'text-indigo-500': index === selectedIndex }"><component :is="menu.icon || 'Document'" /></el-icon>
              <span class="text-gray-700 font-medium group-hover:text-indigo-700" :class="{ 'text-indigo-700': index === selectedIndex }">{{ menu.title }}</span>
            </div>
            <span class="text-xs text-gray-400 font-mono group-hover:text-indigo-400" :class="{ 'text-indigo-400': index === selectedIndex }">{{ menu.path }}</span>
          </div>
          <div v-if="filteredMenus.length === 0" class="p-8 text-center text-gray-400">
            No commands found.
          </div>
        </div>
      </div>
    </el-dialog>
  </el-container>
</template>



<style>
.cmd-backdrop {
  backdrop-filter: blur(4px);
  background-color: rgba(0, 0, 0, 0.4) !important;
}
.cmd-palette-dialog {
  background: transparent !important;
  box-shadow: none !important;
  padding: 0 !important;
}
.cmd-palette-dialog .el-dialog__header {
  display: none !important;
}
.cmd-palette-dialog .el-dialog__body {
  padding: 0 !important;
  background: transparent !important;
}
</style>

<style scoped>

/* Remove old layout CSS properties that conflict with tailwind */
.layout {
  width: 100vw;
}
.menu-scroll {
  border-right: none;
}
.el-menu {
  border-right: none;
}

/* Custom fade-transform transition */
.fade-transform-leave-active,
.fade-transform-enter-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
.fade-transform-enter-from {
  opacity: 0;
  transform: translateX(-15px);
}
.fade-transform-leave-to {
  opacity: 0;
  transform: translateX(15px);
}
</style>
