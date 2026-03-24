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
  'TicketTypes': () => import('../views/TicketTypes.vue'),
  'TicketList': () => import('../views/TicketList.vue'),
  'TicketCreate': () => import('../views/TicketCreate.vue'),
  'TicketDetail': () => import('../views/TicketDetail.vue'),
  'ApprovalInbox': () => import('../views/ApprovalInbox.vue'),
  'RequestTemplates': () => import('../views/RequestTemplates.vue'),
  'ApprovalPolicies': () => import('../views/ApprovalPolicies.vue'),
  'NotificationConsole': () => import('../views/NotificationConsole.vue'),
}

// 系统管理静态路由（仪表盘始终可访问）
const dashboardRoute: RouteRecordRaw = {
  path: 'dashboard',
  name: 'Dashboard',
  component: viewModules['Dashboard'],
  meta: { title: '仪表盘', icon: 'Odometer', componentName: 'Dashboard' },
}

const router = createRouter({
  history: createWebHistory(),
  routes: [...constantRoutes, layoutRoute],
})

// 路由守卫：登录检查 + 动态路由加载
let routesAdded = false
const dynamicRouteNames = new Set<string>()
const dynamicFallbackRouteName = 'DynamicFallback404'

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
      if (!router.hasRoute('Dashboard')) {
        router.addRoute('Layout', dashboardRoute)
      }

      // 加载后端动态菜单路由
      const menus = await permissionStore.fetchMenus()
      const dynamicChildren = generateRoutes(menus)
      dynamicChildren.forEach(route => {
        if (route.name && route.name !== 'Dashboard' && !router.hasRoute(route.name)) {
          router.addRoute('Layout', route)
          dynamicRouteNames.add(String(route.name))
        }
      })

      // 兜底 404
      if (!router.hasRoute(dynamicFallbackRouteName)) {
        router.addRoute({ name: dynamicFallbackRouteName, path: '/:pathMatch(.*)*', redirect: '/404' })
      }
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

    const routePath = normalizeRoutePath(menu)

    const route: RouteRecordRaw = {
      path: routePath,
      name: menu.name,
      meta: {
        title: menu.title,
        icon: menu.icon,
        componentName: menu.component || '',
        activeMenu: menu.component === 'TicketDetail' ? '/tickets' : menu.path,
      },
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
  return ensureCompanionRoutes(routes)
}

function ensureCompanionRoutes(routes: RouteRecordRaw[]): RouteRecordRaw[] {
  const nextRoutes = [...routes]
  const ticketListRoute = nextRoutes.find(route =>
    route.meta?.componentName === 'TicketList' || route.name === 'TicketList'
  )

  if (!ticketListRoute) {
    return nextRoutes
  }

  const ticketActiveMenu = (ticketListRoute.meta?.activeMenu as string) || `/${String(ticketListRoute.path)}`
  const hasTicketCreateRoute = nextRoutes.some(route =>
    route.meta?.componentName === 'TicketCreate' || route.name === 'TicketCreate'
  )
  const hasTicketDetailRoute = nextRoutes.some(route =>
    route.meta?.componentName === 'TicketDetail' || route.name === 'TicketDetail'
  )
  const hasApprovalInboxRoute = nextRoutes.some(route =>
    route.meta?.componentName === 'ApprovalInbox' || route.name === 'ApprovalInbox'
  )

  if (!hasTicketCreateRoute) {
    nextRoutes.push({
      path: 'ticket/create',
      name: 'TicketCreate',
      component: viewModules['TicketCreate'],
      meta: {
        title: '创建工单',
        componentName: 'TicketCreate',
        activeMenu: ticketActiveMenu,
      },
    })
  }

  if (!hasTicketDetailRoute) {
    nextRoutes.push({
      path: 'ticket/detail/:id?',
      name: 'TicketDetail',
      component: viewModules['TicketDetail'],
      meta: {
        title: '工单详情',
        componentName: 'TicketDetail',
        activeMenu: ticketActiveMenu,
      },
    })
  }

  if (!hasApprovalInboxRoute) {
    nextRoutes.push({
      path: 'approval/inbox',
      name: 'ApprovalInbox',
      component: viewModules['ApprovalInbox'],
      meta: {
        title: '审批待办',
        componentName: 'ApprovalInbox',
        activeMenu: ticketActiveMenu,
      },
    })
  }

  const ticketTypeRoute = nextRoutes.find(route =>
    route.meta?.componentName === 'TicketTypes' || route.name === 'TicketTypes'
  )
  if (ticketTypeRoute) {
    const configActiveMenu = (ticketTypeRoute.meta?.activeMenu as string) || `/${String(ticketTypeRoute.path)}`
    const hasRequestTemplatesRoute = nextRoutes.some(route =>
      route.meta?.componentName === 'RequestTemplates' || route.name === 'RequestTemplates'
    )
    const hasApprovalPoliciesRoute = nextRoutes.some(route =>
      route.meta?.componentName === 'ApprovalPolicies' || route.name === 'ApprovalPolicies'
    )
    const hasNotificationConsoleRoute = nextRoutes.some(route =>
      route.meta?.componentName === 'NotificationConsole' || route.name === 'NotificationConsole'
    )
    if (!hasRequestTemplatesRoute) {
      nextRoutes.push({
        path: 'request/templates',
        name: 'RequestTemplates',
        component: viewModules['RequestTemplates'],
        meta: {
          title: '请求模板',
          componentName: 'RequestTemplates',
          activeMenu: configActiveMenu,
        },
      })
    }
    if (!hasApprovalPoliciesRoute) {
      nextRoutes.push({
        path: 'approval/policies',
        name: 'ApprovalPolicies',
        component: viewModules['ApprovalPolicies'],
        meta: {
          title: '审批策略',
          componentName: 'ApprovalPolicies',
          activeMenu: configActiveMenu,
        },
      })
    }
    if (!hasNotificationConsoleRoute) {
      nextRoutes.push({
        path: 'notification/console',
        name: 'NotificationConsole',
        component: viewModules['NotificationConsole'],
        meta: {
          title: '通知联调',
          componentName: 'NotificationConsole',
          activeMenu: configActiveMenu,
        },
      })
    }
  }

  return nextRoutes
}

function normalizeRoutePath(menu: any): string {
  const rawPath = menu.path.startsWith('/') ? menu.path.slice(1) : menu.path
  const normalizedPath = rawPath.replace(/\/+$/, '')

  // 详情页菜单通常存的是固定路径，但页面实际需要带 ID 参数。
  if (menu.component === 'TicketDetail' && !normalizedPath.includes('/:id')) {
    return `${normalizedPath}/:id?`
  }

  return normalizedPath
}

// 供登出时重置路由状态
export function resetRouter() {
  routesAdded = false
  layoutRoute.children = []
  for (const name of dynamicRouteNames) {
    if (router.hasRoute(name)) {
      router.removeRoute(name)
    }
  }
  dynamicRouteNames.clear()
  if (router.hasRoute(dynamicFallbackRouteName)) {
    router.removeRoute(dynamicFallbackRouteName)
  }
}

export default router
