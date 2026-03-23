import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface TagView {
  path: string
  title: string
  name?: string
  closable: boolean
}

export const useTagsViewStore = defineStore('tagsView', () => {
  const visitedViews = ref<TagView[]>([
    { path: '/dashboard', title: '仪表盘', name: 'Dashboard', closable: false },
  ])
  const activeView = ref('/dashboard')

  function addView(view: TagView) {
    activeView.value = view.path
    if (visitedViews.value.some(v => v.path === view.path)) return
    visitedViews.value.push(view)
  }

  function removeView(path: string) {
    const idx = visitedViews.value.findIndex(v => v.path === path)
    if (idx === -1) return activeView.value
    visitedViews.value.splice(idx, 1)
    // 如果关的是当前页，跳到右边或左边
    if (path === activeView.value) {
      const next = visitedViews.value[idx] || visitedViews.value[idx - 1]
      activeView.value = next?.path || '/dashboard'
    }
    return activeView.value
  }

  function closeOthers(path: string) {
    visitedViews.value = visitedViews.value.filter(v => !v.closable || v.path === path)
    activeView.value = path
  }

  function closeAll() {
    visitedViews.value = visitedViews.value.filter(v => !v.closable)
    activeView.value = '/dashboard'
    return activeView.value
  }

  return { visitedViews, activeView, addView, removeView, closeOthers, closeAll }
})
