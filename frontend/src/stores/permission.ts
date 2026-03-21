import { defineStore } from 'pinia'
import { ref } from 'vue'
import { menuApi } from '../api'
import type { RouteRecordRaw } from 'vue-router'

export interface MenuItem {
  id: number
  parent_id: number
  name: string
  title: string
  icon: string
  path: string
  component: string
  api_path: string
  api_method: string
  type: number  // 1=目录 2=菜单 3=按钮
  sort: number
  visible: number
  children?: MenuItem[]
}

export const usePermissionStore = defineStore('permission', () => {
  const menus = ref<MenuItem[]>([])
  const permissions = ref<string[]>([])  // 按钮权限标识列表
  const dynamicRoutes = ref<RouteRecordRaw[]>([])
  const isRoutesGenerated = ref(false)

  async function fetchMenus() {
    const res: any = await menuApi.userMenus()
    menus.value = res.data || []
    // 从菜单树中提取 type=3 的按钮权限
    permissions.value = extractPermissions(menus.value)
    return menus.value
  }

  function extractPermissions(items: MenuItem[]): string[] {
    const perms: string[] = []
    for (const item of items) {
      if (item.type === 3 && item.name) perms.push(item.name)
      if (item.children?.length) perms.push(...extractPermissions(item.children))
    }
    return perms
  }

  function hasPermission(perm: string): boolean {
    return permissions.value.includes(perm)
  }

  function reset() {
    menus.value = []
    permissions.value = []
    dynamicRoutes.value = []
    isRoutesGenerated.value = false
  }

  return { menus, permissions, dynamicRoutes, isRoutesGenerated, fetchMenus, hasPermission, reset }
})
