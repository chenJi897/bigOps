import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useUserStore } from '../stores/user'
import { usePermissionStore } from '../stores/permission'

// 静态路由（不需要权限）
const constantRoutes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
  },
  {
    path: '/404',
    name: 'NotFound',
    component: () => import('../views/404.vue'),
  },
]

// 布局容器路由（动态子路由挂载于此）
export const layoutRoute: RouteRecordRaw = {
  path: '/',
  name: 'Layout',
  component: () => import('../views/Layout.vue'),
  meta: { requiresAuth: true },
  redirect: '/dashboard',
  children: [],
}

// 页面组件映射表：后端菜单 component 字段 → 前端组件
const viewModules: Record<string, () => Promise<any>> = {
  'Dashboard': () => import('../views/Dashboard.vue'),
  'Users': () => import('../views/Users.vue'),
  'Roles': () => import('../views/Roles.vue'),
  'Menus': () => import('../views/Menus.vue'),
  'AuditLogs': () => import('../views/AuditLogs.vue'),
  'ServiceTree': () => import('../views/ServiceTree.vue'),
  'CloudAccounts': () => import('../views/CloudAccounts.vue'),
  'Assets': () => import('../views/Assets.vue'),
  'Departments': () => import('../views/Departments.vue'),
}

// 系统管理静态路由（仪表盘始终可访问）
const dashboardRoute: RouteRecordRaw = {
  path: 'dashboard',
  name: 'Dashboard',
  component: viewModules['Dashboard'],
  meta: { title: '仪表盘', icon: 'Odometer' },
}

const router = createRouter({
  history: createWebHistory(),
  routes: [...constantRoutes, layoutRoute],
})

// 路由守卫：登录检查 + 动态路由加载
let routesAdded = false

router.beforeEach(async (to) => {
  const token = localStorage.getItem('token')

  if (to.path === '/login') {
    if (token) return '/'
    return true
  }

  if (!token) return '/login'

  // 如果动态路由还没加载
  if (!routesAdded) {
    const userStore = useUserStore()
    const permissionStore = usePermissionStore()

    try {
      if (!userStore.userInfo) await userStore.fetchUserInfo()

      // 仪表盘始终可访问
      router.addRoute('Layout', dashboardRoute)

      // 加载后端动态菜单路由
      const menus = await permissionStore.fetchMenus()
      const dynamicChildren = generateRoutes(menus)
      dynamicChildren.forEach(route => {
        if (route.name && route.name !== 'Dashboard' && !router.hasRoute(route.name)) {
          router.addRoute('Layout', route)
        }
      })

      // 兜底 404
      router.addRoute({ path: '/:pathMatch(.*)*', redirect: '/404' })
      routesAdded = true
      return to.fullPath // 重新导航
    } catch {
      localStorage.removeItem('token')
      return '/login'
    }
  }
})

/**
 * 将后端菜单树转换为 Vue Router 路由
 */
function generateRoutes(menus: any[]): RouteRecordRaw[] {
  const routes: RouteRecordRaw[] = []
  for (const menu of menus) {
    if (menu.type === 3) continue // 按钮权限，不生成路由
    if (!menu.path) continue

    // 子路由 path 不能以 / 开头，去掉前导斜杠
    const routePath = menu.path.startsWith('/') ? menu.path.slice(1) : menu.path

    const route: RouteRecordRaw = {
      path: routePath,
      name: menu.name,
      meta: { title: menu.title, icon: menu.icon },
      component: undefined,
      children: [],
    }

    if (menu.component && viewModules[menu.component]) {
      route.component = viewModules[menu.component]
    }

    if (menu.children?.length) {
      const childRoutes = generateRoutes(menu.children)
      if (route.component) {
        // 有自身组件的同时有子路由（目录+页面合一）
        routes.push(route)
        routes.push(...childRoutes)
      } else {
        // 纯目录节点，子路由直接展平
        routes.push(...childRoutes)
      }
    } else if (route.component) {
      routes.push(route)
    }
  }
  return routes
}

// 供登出时重置路由状态
export function resetRouter() {
  routesAdded = false
  layoutRoute.children = []
}

export default router
