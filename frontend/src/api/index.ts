import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => {
    const data = response.data
    if (data.code !== 0) {
      ElMessage.error(data.message || '请求失败')
      if (data.code === 401) {
        localStorage.removeItem('token')
        router.push('/login')
      }
      return Promise.reject(new Error(data.message))
    }
    return data
  },
  (error) => {
    ElMessage.error(error.message || '网络错误')
    return Promise.reject(error)
  }
)

// 认证
export const authApi = {
  login: (username: string, password: string) => api.post('/auth/login', { username, password }),
  register: (username: string, password: string, email: string) => api.post('/auth/register', { username, password, email }),
  logout: () => api.post('/auth/logout'),
  getInfo: () => api.get('/auth/info'),
  changePassword: (old_password: string, new_password: string) => api.post('/auth/password', { old_password, new_password }),
}

// 用户管理
export const userApi = {
  list: (page = 1, size = 20, keyword = '') => api.get('/users', { params: { page, size, ...(keyword ? { keyword } : {}) } }),
  update: (id: number, data: any) => api.post(`/users/${id}`, data),
  updateStatus: (id: number, status: number) => api.post(`/users/${id}/status`, { status }),
  delete: (id: number) => api.post(`/users/${id}/delete`),
  getRoles: (id: number) => api.get(`/users/${id}/roles`),
  setRoles: (id: number, role_ids: number[]) => api.post(`/users/${id}/roles`, { role_ids }),
  setDepartment: (id: number, department_id: number) => api.post(`/users/${id}/department`, { department_id }),
}

// 角色管理
export const roleApi = {
  list: (page = 1, size = 20) => api.get('/roles', { params: { page, size } }),
  getById: (id: number) => api.get(`/roles/${id}`),
  create: (data: any) => api.post('/roles', data),
  update: (id: number, data: any) => api.post(`/roles/${id}`, data),
  delete: (id: number) => api.post(`/roles/${id}/delete`),
  setMenus: (id: number, menu_ids: number[]) => api.post(`/roles/${id}/menus`, { menu_ids }),
}

// 菜单管理
export const menuApi = {
  tree: () => api.get('/menus'),
  userMenus: () => api.get('/menus/user'),
  create: (data: any) => api.post('/menus', data),
  update: (id: number, data: any) => api.post(`/menus/${id}`, data),
  delete: (id: number) => api.post(`/menus/${id}/delete`),
}

// 审计日志
export const auditLogApi = {
  list: (params: { page?: number; size?: number; username?: string; action?: string; resource?: string }) =>
    api.get('/audit-logs', { params }),
}

// 服务树
export const serviceTreeApi = {
  tree: () => api.get('/service-trees'),
  getById: (id: number) => api.get(`/service-trees/${id}`),
  create: (data: any) => api.post('/service-trees', data),
  update: (id: number, data: any) => api.post(`/service-trees/${id}`, data),
  delete: (id: number) => api.post(`/service-trees/${id}/delete`),
  move: (id: number, parent_id: number, sort: number) => api.post(`/service-trees/${id}/move`, { parent_id, sort }),
  assetCounts: () => api.get('/service-trees/asset-counts'),
}

// 云账号
export const cloudAccountApi = {
  list: (page = 1, size = 20) => api.get('/cloud-accounts', { params: { page, size } }),
  getById: (id: number) => api.get(`/cloud-accounts/${id}`),
  create: (data: any) => api.post('/cloud-accounts', data),
  update: (id: number, data: any) => api.post(`/cloud-accounts/${id}`, data),
  updateKeys: (id: number, access_key: string, secret_key: string) => api.post(`/cloud-accounts/${id}/keys`, { access_key, secret_key }),
  delete: (id: number) => api.post(`/cloud-accounts/${id}/delete`),
  sync: (id: number) => api.post(`/cloud-accounts/${id}/sync`),
  syncConfig: (id: number, sync_enabled: boolean, sync_interval: number) =>
    api.post(`/cloud-accounts/${id}/sync-config`, { sync_enabled, sync_interval }),
  syncTasks: (id: number, page = 1, size = 10) =>
    api.get(`/cloud-accounts/${id}/sync-tasks`, { params: { page, size } }),
}

// 同步日志
export const syncTaskApi = {
  list: (params: { page?: number; size?: number; status?: string; trigger_type?: string; cloud_account_id?: number }) =>
    api.get('/sync-tasks', { params }),
}

// 统计
export const statsApi = {
  summary: () => api.get('/stats/summary'),
  assetDistribution: () => api.get('/stats/asset-distribution'),
}

export const dashboardApi = {
  personal: () => api.get('/dashboard/personal'),
}

// 部门管理
export const departmentApi = {
  list: (page = 1, size = 20) => api.get('/departments', { params: { page, size } }),
  all: () => api.get('/departments/all'),
  getById: (id: number) => api.get(`/departments/${id}`),
  create: (data: any) => api.post('/departments', data),
  update: (id: number, data: any) => api.post(`/departments/${id}`, data),
  delete: (id: number) => api.post(`/departments/${id}/delete`),
}

// 资产管理
export const assetApi = {
  list: (params: { page?: number; size?: number; status?: string; source?: string; service_tree_id?: number; keyword?: string }) =>
    api.get('/assets', { params }),
  getById: (id: number) => api.get(`/assets/${id}`),
  create: (data: any) => api.post('/assets', data),
  update: (id: number, data: any) => api.post(`/assets/${id}`, data),
  delete: (id: number) => api.post(`/assets/${id}/delete`),
  changes: (id: number, page = 1, size = 20) => api.get(`/assets/${id}/changes`, { params: { page, size } }),
}

// 工单类型
export const ticketTypeApi = {
  list: (page = 1, size = 20) => api.get('/ticket-types', { params: { page, size } }),
  all: () => api.get('/ticket-types/all'),
  create: (data: any) => api.post('/ticket-types', data),
  update: (id: number, data: any) => api.post(`/ticket-types/${id}`, data),
  delete: (id: number) => api.post(`/ticket-types/${id}/delete`),
}

// 工单管理
export const ticketApi = {
  list: (params: any) => api.get('/tickets', { params }),
  getById: (id: number) => api.get(`/tickets/${id}`),
  approvalInstance: (id: number) => api.get(`/tickets/${id}/approval-instance`),
  create: (data: any) => api.post('/tickets', data),
  assign: (id: number, assignee_id: number) => api.post(`/tickets/${id}/assign`, { assignee_id }),
  process: (id: number, action: string, content: string) => api.post(`/tickets/${id}/process`, { action, content }),
  close: (id: number, resolution: string, note: string) => api.post(`/tickets/${id}/close`, { resolution, note }),
  reopen: (id: number, content: string) => api.post(`/tickets/${id}/reopen`, { content }),
  comment: (id: number, content: string) => api.post(`/tickets/${id}/comment`, { content }),
  transfer: (id: number, assignee_id: number, content: string) => api.post(`/tickets/${id}/transfer`, { assignee_id, content }),
  activities: (id: number, page = 1, size = 50) => api.get(`/tickets/${id}/activities`, { params: { page, size } }),
}

export const requestTemplateApi = {
  list: (enabled_only = false) => api.get('/request-templates', { params: { enabled_only: enabled_only ? 1 : 0 } }),
  getById: (id: number) => api.get(`/request-templates/${id}`),
  create: (data: any) => api.post('/request-templates', data),
  update: (id: number, data: any) => api.post(`/request-templates/${id}`, data),
  delete: (id: number) => api.post(`/request-templates/${id}/delete`),
}

export const approvalPolicyApi = {
  list: () => api.get('/approval-policies'),
  getById: (id: number) => api.get(`/approval-policies/${id}`),
  create: (data: any) => api.post('/approval-policies', data),
  update: (id: number, data: any) => api.post(`/approval-policies/${id}`, data),
  delete: (id: number) => api.post(`/approval-policies/${id}/delete`),
}

export const approvalApi = {
  pending: () => api.get('/approval-instances/pending'),
  approve: (id: number, comment = '') => api.post(`/approval-instances/${id}/approve`, { comment }),
  reject: (id: number, comment: string) => api.post(`/approval-instances/${id}/reject`, { comment }),
}

export const notificationApi = {
  inApp: (unread_only = false) => api.get('/notifications/in-app', { params: { unread_only: unread_only ? 1 : 0 } }),
  unreadCount: () => api.get('/notifications/in-app/unread-count'),
  markRead: (id: number) => api.post(`/notifications/in-app/${id}/read`),
  markAllRead: () => api.post('/notifications/in-app/read-all'),
  clearRead: () => api.post('/notifications/in-app/clear-read'),
  getPreference: () => api.get('/notifications/preferences'),
  updatePreference: (data: any) => api.post('/notifications/preferences', data),
  getConfig: () => api.get('/notifications/config'),
  updateConfig: (data: any) => api.post('/notifications/config', data),
  testSend: (data: { title: string; content: string; channels: string[]; user_ids?: number[] }) => api.post('/notifications/test', data),
  events: () => api.get('/notifications/events'),
  retryEvent: (id: number) => api.post(`/notifications/events/${id}/retry`),
  listTemplates: () => api.get('/notifications/templates'),
  updateTemplate: (id: number, data: { title: string; content: string }) => api.post(`/notifications/templates/${id}`, data),
  previewTemplate: (data: { title: string; content: string; variables: Record<string, any> }) => api.post('/notifications/templates/preview', data),
  testWebhook: (data: { channel_type: string; webhook_url: string; secret?: string }) => api.post('/notifications/test-webhook', data),
  enabledChannelTypes: () => api.get('/notifications/enabled-channel-types'),
}

export const notifyGroupApi = {
  list: (params?: { page?: number; size?: number; keyword?: string }) => api.get('/notify-groups', { params }),
  all: () => api.get('/notify-groups/all'),
  getById: (id: number) => api.get(`/notify-groups/${id}`),
  create: (data: any) => api.post('/notify-groups', data),
  update: (id: number, data: any) => api.post(`/notify-groups/${id}`, data),
  delete: (id: number) => api.post(`/notify-groups/${id}/delete`),
  test: (id: number) => api.post(`/notify-groups/${id}/test`),
}

export const monitorApi = {
  summary: () => api.get('/monitor/summary'),
  agents: (params: { page?: number; size?: number; status?: string; keyword?: string }) => api.get('/monitor/agents', { params }),
  trends: (agentID: string, metric_type: string, minutes = 60, limit = 120) =>
    api.get(`/monitor/agents/${agentID}/trends`, { params: { metric_type, minutes, limit } }),
  aggregateServiceTrees: () => api.get('/monitor/aggregates/service-trees'),
  aggregateOwners: () => api.get('/monitor/aggregates/owners'),
  datasources: () => api.get('/monitor/datasources'),
  createDatasource: (data: any) => api.post('/monitor/datasources', data),
  updateDatasource: (id: number, data: any) => api.post(`/monitor/datasources/${id}`, data),
  deleteDatasource: (id: number) => api.post(`/monitor/datasources/${id}/delete`),
  datasourceHealth: (id: number) => api.get(`/monitor/datasources/${id}/health`),
  query: (data: { datasource_id: number; query: string; time?: string }) => api.post('/monitor/query', data),
  queryRange: (data: { datasource_id: number; query: string; start?: string; end?: string; step?: string }) => api.post('/monitor/query-range', data),
  goldenSignals: (minutes = 60) => api.get('/monitor/golden-signals', { params: { minutes } }),
  goldenSignalsDimensions: (minutes = 60, dimension: 'service' | 'interface' | 'instance' = 'service') =>
    api.get('/monitor/golden-signals/dimensions', { params: { minutes, dimension } }),
}

export const alertSilenceApi = {
  list: () => api.get('/alert-silences'),
  create: (data: any) => api.post('/alert-silences', data),
  update: (id: number, data: any) => api.post(`/alert-silences/${id}`, data),
  delete: (id: number) => api.post(`/alert-silences/${id}/delete`),
}

export const onCallApi = {
  list: () => api.get('/oncall-schedules'),
  create: (data: any) => api.post('/oncall-schedules', data),
  update: (id: number, data: any) => api.post(`/oncall-schedules/${id}`, data),
  delete: (id: number) => api.post(`/oncall-schedules/${id}/delete`),
}

export function buildWebhookUrl(pipelineCode = 'pipeline-code') {
  const code = encodeURIComponent((pipelineCode || 'pipeline-code').trim())
  const host = typeof window !== 'undefined' ? window.location.origin : ''
  return `${host}/api/v1/cicd/webhook/${code}`
}

// CI/CD
export const cicdProjectApi = {
  list: (params: { page?: number; size?: number; keyword?: string; active?: number }) =>
    api.get('/cicd/projects', { params }),
  create: (data: any) => api.post('/cicd/projects', data),
  update: (id: number, data: any) => api.post(`/cicd/projects/${id}`, data),
  delete: (id: number) => api.post(`/cicd/projects/${id}/delete`),
  toggleStatus: (id: number, enabled: boolean) => api.post(`/cicd/projects/${id}/status`, { enabled }),
}

export const cicdPipelineApi = {
  list: (params: { page?: number; size?: number; keyword?: string; project_id?: number; active?: number }) =>
    api.get('/cicd/pipelines', { params }),
  create: (data: any) => api.post('/cicd/pipelines', data),
  update: (id: number, data: any) => api.post(`/cicd/pipelines/${id}`, data),
  delete: (id: number) => api.post(`/cicd/pipelines/${id}/delete`),
  trigger: (id: number) => api.post(`/cicd/pipelines/${id}/trigger`),
  runs: (params: { page?: number; size?: number; project_id?: number; pipeline_id?: number; status?: string }) =>
    api.get('/cicd/runs', { params }),
  runDetail: (id: number) => api.get(`/cicd/runs/${id}`),
  retryRun: (id: number) => api.post(`/cicd/runs/${id}/retry`),
  rollbackRun: (id: number) => api.post(`/cicd/runs/${id}/rollback`),
}

export const alertRuleApi = {
  list: (params: { page?: number; size?: number; keyword?: string; metric_type?: string; severity?: string; enabled?: number }) =>
    api.get('/alert-rules', { params }),
  create: (data: any) => api.post('/alert-rules', data),
  update: (id: number, data: any) => api.post(`/alert-rules/${id}`, data),
  delete: (id: number) => api.post(`/alert-rules/${id}/delete`),
  evaluate: () => api.post('/alert-rules/evaluate'),
  events: (params: { page?: number; size?: number; status?: string; severity?: string; agent_id?: string; keyword?: string; rule_id?: number }) =>
    api.get('/alert-events', { params }),
  eventGroups: (params: { page?: number; size?: number; status?: string; severity?: string; agent_id?: string; keyword?: string; window_minutes?: number }) =>
    api.get('/alert-events/groups', { params }),
  getEvent: (id: number) => api.get(`/alert-events/${id}`),
  eventTimeline: (id: number) => api.get(`/alert-events/${id}/timeline`),
  eventRootCause: (id: number) => api.get(`/alert-events/${id}/root-cause`),
  eventContext: (id: number) => api.get(`/alert-events/${id}/context`),
  ackEvent: (id: number, note = '') => api.post(`/alert-events/${id}/ack`, { note }),
  resolveEvent: (id: number, note = '') => api.post(`/alert-events/${id}/resolve`, { note }),
}

// 任务管理
export const taskApi = {
  list: (params: { page?: number; size?: number; keyword?: string; task_type?: string }) =>
    api.get('/tasks', { params }),
  getById: (id: number) => api.get(`/tasks/${id}`),
  create: (data: any) => api.post('/tasks', data),
  update: (id: number, data: any) => api.post(`/tasks/${id}`, data),
  delete: (id: number) => api.post(`/tasks/${id}/delete`),
  execute: (id: number, data: { host_ips: string[] }) => api.post(`/tasks/${id}/execute`, data),
  executions: (params: { task_id?: number; page?: number; size?: number }) =>
    api.get('/task-executions', { params }),
  getExecution: (id: number) => api.get(`/task-executions/${id}`),
  cancelExecution: (id: number) => api.post(`/task-executions/${id}/cancel`),
  retryExecution: (id: number, scope: 'failed' | 'all' = 'failed', host_ips?: string[]) =>
    api.post(`/task-executions/${id}/retry`, host_ips?.length ? { host_ips } : {}, { params: { scope } }),
}

// Agent 管理
export const agentApi = {
  list: (params: { page?: number; size?: number; status?: string }) =>
    api.get('/agents', { params }),
}

export const inspectionApi = {
  templates: (params: { page?: number; size?: number }) => api.get('/inspection/templates', { params }),
  createTemplate: (data: any) => api.post('/inspection/templates', data),
  updateTemplate: (id: number, data: any) => api.post(`/inspection/templates/${id}`, data),
  plans: (params: { page?: number; size?: number }) => api.get('/inspection/plans', { params }),
  createPlan: (data: any) => api.post('/inspection/plans', data),
  updatePlan: (id: number, data: any) => api.post(`/inspection/plans/${id}`, data),
  runPlan: (id: number) => api.post(`/inspection/plans/${id}/run`),
  records: (params: { page?: number; size?: number }) => api.get('/inspection/records', { params }),
  recordReport: (id: number) => api.get(`/inspection/records/${id}/report`),
  recordReportExportUrl: (id: number, format: 'json' | 'csv' = 'json') =>
    `/api/v1/inspection/records/${id}/report/export?format=${format}`,
  templateTrend: (id: number) => api.get(`/inspection/templates/${id}/trend`),
}

export default api
