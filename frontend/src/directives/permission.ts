import type { Directive } from 'vue'
import { usePermissionStore } from '../stores/permission'

/**
 * v-permission 按钮级权限指令
 * 用法: v-permission="'user:create'" 或 v-permission="['user:create', 'user:update']"
 * 如果用户没有对应权限，元素会被隐藏（display:none）
 * 支持响应式：权限数据加载后自动更新
 */
function checkPermission(el: HTMLElement, value: string | string[]) {
  if (!value) return

  const permissionStore = usePermissionStore()
  const perms = Array.isArray(value) ? value : [value]
  const hasPermission = perms.some((p: string) => permissionStore.hasPermission(p))

  el.style.display = hasPermission ? '' : 'none'
}

export const permission: Directive = {
  mounted(el, binding) {
    checkPermission(el, binding.value)
  },
  updated(el, binding) {
    checkPermission(el, binding.value)
  },
}
