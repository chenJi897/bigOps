import type { Directive } from 'vue'
import { usePermissionStore } from '../stores/permission'

/**
 * v-permission 按钮级权限指令
 * 用法: v-permission="'user:create'" 或 v-permission="['user:create', 'user:update']"
 * 如果用户没有对应权限，元素会被移除
 */
export const permission: Directive = {
  mounted(el, binding) {
    const permissionStore = usePermissionStore()
    const value = binding.value

    if (!value) return

    const perms = Array.isArray(value) ? value : [value]
    const hasPermission = perms.some((p: string) => permissionStore.hasPermission(p))

    if (!hasPermission) {
      el.parentNode?.removeChild(el)
    }
  },
}
