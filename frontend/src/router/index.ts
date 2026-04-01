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
  'DashboardWorkbench': () => import('../views/dashboard/Workbench.vue'),
  'DashboardOverview': () => import('../views/dashboard/Overview.vue'),
  'MonitorDashboard': () => import('../views/MonitorDashboard.vue'),
  'AlertRules': () => import('../views/AlertRules.vue'),
  'AgentDetail': () => import('../views/AgentDetail.vue'),
  'AlertEvents': () => import('../views/AlertEvents.vue'),
  'AlertSilences': () => import('../views/AlertSilences.vue'),
  'MonitorDatasources': () => import('../views/MonitorDatasources.vue'),
  'MonitorQuery': () => import('../views/MonitorQuery.vue'),
  'OnCallSchedules': () => import('../views/OnCallSchedules.vue'),
  'Users': () => import('../views/Users.vue'),
  'Roles': () => import('../views/Roles.vue'),
  'Menus': () => import('../views/Menus.vue'),
  'AuditLogs': () => import('../views/AuditLogs.vue'),
  'ServiceTree': () => import('../views/ServiceTree.vue'),
  'CloudAccounts': () => import('../views/CloudAccounts.vue'),
  'Assets': () => import('../views/Assets.vue'),
  'CicdProjects': () => import('../views/CicdProjects.vue'),
  'CicdPipelines': () => import('../views/CicdPipelines.vue'),
  'CicdRuns': () => import('../views/CicdRuns.vue'),
  'Departments': () => import('../views/Departments.vue'),
  'TicketTypes': () => import('../views/TicketTypes.vue'),
  'TicketList': () => import('../views/TicketList.vue'),
  'TicketCreate': () => import('../views/TicketCreate.vue'),
  'TicketDetail': () => import('../views/TicketDetail.vue'),
  'ApprovalInbox': () => import('../views/ApprovalInbox.vue'),
  'RequestTemplates': () => import('../views/RequestTemplates.vue'),
  'ApprovalPolicies': () => import('../views/ApprovalPolicies.vue'),
  'NotificationConsole': () => import('../views/NotificationConsole.vue'),
  'UserSettings': () => import('../views/UserSettings.vue'),
  'TaskList': () => import('../views/TaskList.vue'),
  'TaskCreate': () => import('../views/TaskCreate.vue'),
  'TaskExecution': () => import('../views/TaskExecution.vue'),
  'AgentList': () => import('../views/AgentList.vue'),
}

// 系统管理静态路由（仪表盘始终可访问）
const dashboardRoute: RouteRecordRaw = {
  path: 'dashboard',
  name: 'Dashboard',
  redirect: '/dashboard/workbench',
  meta: { title: '仪表盘', icon: 'Odometer' },
  children: [
    {
      path: 'workbench',
      name: 'DashboardWorkbench',
      component: viewModules.DashboardWorkbench,
      meta: { title: '工作台', componentName: 'DashboardWorkbench', activeMenu: '/dashboard' },
    },
    {
      path: 'overview',
      name: 'DashboardOverview',
      component: viewModules.DashboardOverview,
      meta: { title: '概览', componentName: 'DashboardOverview', activeMenu: '/dashboard' },
    },
  ],
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

  if (!routesAdded) {
    const userStore = useUserStore()
    const permissionStore = usePermissionStore()

    try {
      if (!userStore.userInfo) {
        await userStore.fetchUserInfo()
      }

      if (!router.hasRoute('Dashboard')) {
        router.addRoute('Layout', dashboardRoute)
      }

      const menus = await permissionStore.fetchMenus()
      const dynamicChildren = generateRoutes(menus)
      dynamicChildren.forEach(route => {
        if (route.name && route.name !== 'Dashboard' && !router.hasRoute(route.name)) {
          router.addRoute('Layout', route)
          dynamicRouteNames.add(String(route.name))
        }
      })

      if (!router.hasRoute(dynamicFallbackRouteName)) {
        router.addRoute({ name: dynamicFallbackRouteName, path: '/:pathMatch(.*)*', redirect: '/404' })
      }

      routesAdded = true

      if (to.path === '/') {
        return '/dashboard'
      }
      return to.fullPath
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
    if (menu.type === 3) continue
    if (menu.visible === 0) continue
    if (!menu.path && !menu.children?.length) continue

    const routePath = menu.path ? normalizeRoutePath(menu) : ''
    const routeMeta: Record<string, any> = {
      title: menu.title,
      icon: menu.icon,
      componentName: menu.component || '',
      activeMenu: menu.path || '',
    }

    const ticketMode = menu.component === 'TicketList' ? resolveTicketModeFromPath(menu.path) : undefined
    if (ticketMode) {
      routeMeta.ticketMode = ticketMode
    }

    const route: RouteRecordRaw = {
      path: routePath,
      name: menu.name,
      meta: routeMeta,
      component: undefined,
      children: [],
    }

    if (menu.component && viewModules[menu.component]) {
      route.component = viewModules[menu.component]
    }

    if (menu.children?.length) {
      const childRoutes = generateRoutes(menu.children)
      if (route.component) {
        routes.push(route)
        routes.push(...childRoutes)
      } else {
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

  const todoRoute = nextRoutes.find(route =>
    route.meta?.componentName === 'TicketList' && route.meta?.ticketMode === 'todo',
  )
  const appliedRoute = nextRoutes.find(route =>
    route.meta?.componentName === 'TicketList' && route.meta?.ticketMode === 'applied',
  )
  const fallbackTicketRoute = nextRoutes.find(route =>
    route.meta?.componentName === 'TicketList' || route.name === 'TicketList',
  )

  if (!fallbackTicketRoute) {
    return nextRoutes
  }

  const defaultTicketActiveMenu = (todoRoute?.meta?.activeMenu as string)
    || (appliedRoute?.meta?.activeMenu as string)
    || (fallbackTicketRoute?.meta?.activeMenu as string)
    || '/ticket/todo'

  ensureHiddenRoute(nextRoutes, {
    path: 'ticket/create',
    name: 'TicketCreate',
    component: viewModules.TicketCreate,
    title: '发起工单',
    componentName: 'TicketCreate',
    activeMenu: '/ticket/create',
  })

  ensureHiddenRoute(nextRoutes, {
    path: 'ticket/detail/:id?',
    name: 'TicketDetail',
    component: viewModules.TicketDetail,
    title: '工单详情',
    componentName: 'TicketDetail',
    activeMenu: defaultTicketActiveMenu,
  })

  ensureHiddenRoute(nextRoutes, {
    path: 'approval/inbox',
    name: 'ApprovalInbox',
    component: viewModules.ApprovalInbox,
    title: '审批待办',
    componentName: 'ApprovalInbox',
    activeMenu: defaultTicketActiveMenu,
  })

  const templatesActiveMenu = resolveActiveMenuForComponent(nextRoutes, 'RequestTemplates', '/ticket/templates')

  ensureHiddenRoute(nextRoutes, {
    path: 'ticket/templates',
    name: 'RequestTemplates',
    component: viewModules.RequestTemplates,
    title: '工单模板',
    componentName: 'RequestTemplates',
    activeMenu: templatesActiveMenu,
  })

  ensureHiddenRoute(nextRoutes, {
    path: 'approval/policies',
    name: 'ApprovalPolicies',
    component: viewModules.ApprovalPolicies,
    title: '审批策略',
    componentName: 'ApprovalPolicies',
    activeMenu: templatesActiveMenu,
  })

  ensureHiddenRoute(nextRoutes, {
    path: 'notification/console',
    name: 'NotificationConsole',
    component: viewModules.NotificationConsole,
    title: '通知配置中心',
    componentName: 'NotificationConsole',
    activeMenu: '/notification/console',
  })

  ensureHiddenRoute(nextRoutes, {
    path: 'user/settings',
    name: 'UserSettings',
    component: viewModules.UserSettings,
    title: '个人设置',
    componentName: 'UserSettings',
    activeMenu: '/dashboard',
  })

  const taskListRoute = nextRoutes.find(route =>
    route.meta?.componentName === 'TaskList' || route.name === 'TaskList',
  )
  if (taskListRoute) {
    const taskActiveMenu = (taskListRoute.meta?.activeMenu as string) || `/${String(taskListRoute.path)}`
    const hasTaskCreate = nextRoutes.some(route =>
      route.meta?.componentName === 'TaskCreate' || route.name === 'TaskCreate',
    )
    const hasTaskExecution = nextRoutes.some(route =>
      route.meta?.componentName === 'TaskExecution' || route.name === 'TaskExecution',
    )

    if (!hasTaskCreate) {
      nextRoutes.push({
        path: 'task/create/:id?',
        name: 'TaskCreate',
        component: viewModules.TaskCreate,
        meta: {
          title: '创建任务',
          componentName: 'TaskCreate',
          activeMenu: taskActiveMenu,
        },
      })
    }

    if (!hasTaskExecution) {
      nextRoutes.push({
        path: 'task/execution/:id?',
        name: 'TaskExecution',
        component: viewModules.TaskExecution,
        meta: {
          title: '执行详情',
          componentName: 'TaskExecution',
          activeMenu: taskActiveMenu,
        },
      })
    }
  }

  ensureHiddenRoute(nextRoutes, {
    path: 'monitor/dashboard',
    name: 'MonitorDashboard',
    component: viewModules.MonitorDashboard,
    title: '监控仪表盘',
    componentName: 'MonitorDashboard',
    activeMenu: '/monitor/dashboard',
  })

  ensureHiddenRoute(nextRoutes, {
    path: 'monitor/alert-rules',
    name: 'AlertRules',
    component: viewModules.AlertRules,
    title: '告警规则',
    componentName: 'AlertRules',
    activeMenu: '/monitor/alert-rules',
  })

  ensureHiddenRoute(nextRoutes, {
    path: 'monitor/agents/:agentId',
    name: 'AgentDetail',
    component: viewModules.AgentDetail,
    title: 'Agent 详情',
    componentName: 'AgentDetail',
    activeMenu: '/monitor/dashboard',
  })

  ensureHiddenRoute(nextRoutes, {
    path: 'monitor/alerts',
    name: 'AlertEvents',
    component: viewModules.AlertEvents,
    title: '告警事件',
    componentName: 'AlertEvents',
    activeMenu: '/monitor/alert-rules',
  })

  ensureHiddenRoute(nextRoutes, {
    path: 'monitor/silences',
    name: 'AlertSilences',
    component: viewModules.AlertSilences,
    title: '告警静默',
    componentName: 'AlertSilences',
    activeMenu: '/monitor/alert-rules',
  })

  ensureHiddenRoute(nextRoutes, {
    path: 'monitor/datasources',
    name: 'MonitorDatasources',
    component: viewModules.MonitorDatasources,
    title: '监控数据源',
    componentName: 'MonitorDatasources',
    activeMenu: '/monitor/dashboard',
  })

  ensureHiddenRoute(nextRoutes, {
    path: 'monitor/query',
    name: 'MonitorQuery',
    component: viewModules.MonitorQuery,
    title: 'PromQL 查询台',
    componentName: 'MonitorQuery',
    activeMenu: '/monitor/dashboard',
  })

  ensureHiddenRoute(nextRoutes, {
    path: 'monitor/oncall',
    name: 'OnCallSchedules',
    component: viewModules.OnCallSchedules,
    title: 'OnCall 值班',
    componentName: 'OnCallSchedules',
    activeMenu: '/monitor/alert-rules',
  })

  return nextRoutes
}

type HiddenRouteConfig = {
  name: string
  componentName: string
  component: () => Promise<any>
  path: string
  title: string
  activeMenu: string
}

function ensureHiddenRoute(routes: RouteRecordRaw[], config: HiddenRouteConfig) {
  const existingRoute = routes.find(route =>
    route.meta?.componentName === config.componentName || route.name === config.name,
  )

  if (existingRoute) {
    existingRoute.meta = {
      ...existingRoute.meta,
      title: config.title,
      componentName: config.componentName,
      activeMenu: config.activeMenu,
    }
    existingRoute.component = config.component
    return
  }

  routes.push({
    path: config.path,
    name: config.name,
    component: config.component,
    meta: {
      title: config.title,
      componentName: config.componentName,
      activeMenu: config.activeMenu,
    },
  })
}

function resolveActiveMenuForComponent(routes: RouteRecordRaw[], componentName: string, fallback: string): string {
  const existingRoute = routes.find(route =>
    route.meta?.componentName === componentName || route.name === componentName,
  )
  if (!existingRoute) {
    return fallback
  }
  if (existingRoute.meta?.activeMenu) {
    return existingRoute.meta.activeMenu as string
  }
  return ensureLeadingSlash(existingRoute.path) || fallback
}

function ensureLeadingSlash(path?: string): string | undefined {
  if (!path) {
    return undefined
  }
  return path.startsWith('/') ? path : `/${path}`
}

function normalizeRoutePath(menu: any): string {
  const rawPath = menu.path.startsWith('/') ? menu.path.slice(1) : menu.path
  const normalizedPath = rawPath.replace(/\/+$/, '')

  if (menu.component === 'TicketDetail' && !normalizedPath.includes('/:id')) {
    return `${normalizedPath}/:id?`
  }
  if (menu.component === 'TaskExecution' && !normalizedPath.includes('/:id')) {
    return `${normalizedPath}/:id?`
  }
  if (menu.component === 'TaskCreate' && !normalizedPath.includes('/:id')) {
    return `${normalizedPath}/:id?`
  }

  return normalizedPath
}

function resolveTicketModeFromPath(path?: string): 'todo' | 'applied' | undefined {
  if (!path) return undefined
  if (path === '/ticket/todo') return 'todo'
  if (path === '/ticket/applied') return 'applied'
  return undefined
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
