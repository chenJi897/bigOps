import { ref, onMounted } from 'vue'

/**
 * 表格列宽拖拽持久化 composable
 * 用法：
 * const { savedWidths, onHeaderDragend } = useTableResize('Users')
 * <el-table border @header-dragend="onHeaderDragend">
 *   <el-table-column prop="name" :width="savedWidths.name || 150" />
 */
export function useTableResize(pageKey: string) {
  const STORAGE_KEY = `bigops_table_widths_${pageKey}`
  const savedWidths = ref<Record<string, number>>({})

  onMounted(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (raw) savedWidths.value = JSON.parse(raw)
    } catch {}
  })

  function onHeaderDragend(newWidth: number, _oldWidth: number, column: any) {
    if (column.property) {
      savedWidths.value[column.property] = Math.round(newWidth)
      localStorage.setItem(STORAGE_KEY, JSON.stringify(savedWidths.value))
    }
  }

  return { savedWidths, onHeaderDragend }
}
