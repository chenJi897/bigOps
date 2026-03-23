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

export default api
