import { defineStore } from 'pinia'
import { ref } from 'vue'
import { authApi } from '../api'

export interface UserInfo {
  id: number
  username: string
  email: string | null
  phone: string
  real_name: string
  avatar: string
  status: number
}

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const userInfo = ref<UserInfo | null>(null)

  function setToken(t: string) {
    token.value = t
    localStorage.setItem('token', t)
  }

  function clearToken() {
    token.value = ''
    userInfo.value = null
    localStorage.removeItem('token')
  }

  async function fetchUserInfo() {
    const res: any = await authApi.getInfo()
    userInfo.value = res.data
    return res.data
  }

  async function login(username: string, password: string) {
    const res: any = await authApi.login(username, password)
    setToken(res.data.token)
    userInfo.value = res.data.user
    return res.data
  }

  async function logout() {
    try { await authApi.logout() } catch {}
    clearToken()
  }

  return { token, userInfo, setToken, clearToken, fetchUserInfo, login, logout }
})
