import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

// 请求拦截器：自动附加 token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器：处理业务错误和 401
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

// 认证相关 API
export const authApi = {
  login(username: string, password: string) {
    return api.post('/auth/login', { username, password })
  },
  register(username: string, password: string, email: string) {
    return api.post('/auth/register', { username, password, email })
  },
  logout() {
    return api.post('/auth/logout')
  },
  getInfo() {
    return api.get('/auth/info')
  },
  changePassword(old_password: string, new_password: string) {
    return api.post('/auth/password', { old_password, new_password })
  },
}

export default api
