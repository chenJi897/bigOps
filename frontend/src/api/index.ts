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
  list: (page = 1, size = 20) => api.get('/users', { params: { page, size } }),
  updateStatus: (id: number, status: number) => api.post(`/users/${id}/status`, { status }),
  delete: (id: number) => api.post(`/users/${id}/delete`),
  getRoles: (id: number) => api.get(`/users/${id}/roles`),
  setRoles: (id: number, role_ids: number[]) => api.post(`/users/${id}/roles`, { role_ids }),
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

export default api
