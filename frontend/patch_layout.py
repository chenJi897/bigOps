import re

with open("src/views/Layout.vue", "r", encoding="utf-8") as f:
    content = f.read()

# 1. Imports
# We need `Search` icon. It should be globally registered if other icons are, but let's assume it's available or we can use `Search` from element-plus/icons-vue if not imported. Wait, component :is="menu.icon" is used so maybe Search isn't registered globally. We can just use `Search` if it's auto-imported, or an svg. Let's just use `<Search />`.

script_addition = """
// Command Palette
const cmdPaletteVisible = ref(false)
const searchQuery = ref('')
const searchInputRef = ref<any>(null)

const flatMenus = computed(() => {
  const result: any[] = []
  function flatten(menus: any[]) {
    for (const menu of menus) {
      if (menu.path && menu.title) {
        result.push(menu)
      }
      if (menu.children && menu.children.length) {
        flatten(menu.children)
      }
    }
  }
  flatten(menuTree.value)
  return result
})

const filteredMenus = computed(() => {
  if (!searchQuery.value) return flatMenus.value.slice(0, 10)
  return flatMenus.value.filter(m => m.title.toLowerCase().includes(searchQuery.value.toLowerCase())).slice(0, 10)
})

function handleCmdK(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    cmdPaletteVisible.value = true
    searchQuery.value = ''
    setTimeout(() => {
      searchInputRef.value?.focus()
    }, 100)
  }
}

function handleSelectCommand(menu: any) {
  cmdPaletteVisible.value = false
  router.push(menu.path)
}

function handleCommandKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && filteredMenus.value.length > 0) {
    handleSelectCommand(filteredMenus.value[0])
  }
}
"""

content = content.replace("onMounted(async () => {", script_addition + "\nonMounted(async () => {\n  window.addEventListener('keydown', handleCmdK)")
content = content.replace("onBeforeUnmount(() => {", "onBeforeUnmount(() => {\n  window.removeEventListener('keydown', handleCmdK)")

# Add tooltip/button to header if we want to visually show Cmd+K, or just keep it hidden shortcut. The user didn't ask for a button, just the shortcut.

template_addition = """
    <!-- Command Palette Dialog -->
    <el-dialog
      v-model="cmdPaletteVisible"
      :show-close="false"
      class="cmd-palette-dialog"
      width="600px"
      align-center
    >
      <div class="flex flex-col rounded-xl overflow-hidden bg-white shadow-2xl" @keydown="handleCommandKeydown">
        <div class="p-4 border-b border-gray-100 flex items-center gap-3">
          <el-icon class="text-xl text-gray-400"><Search /></el-icon>
          <input
            ref="searchInputRef"
            v-model="searchQuery"
            class="flex-1 bg-transparent border-none outline-none text-lg text-gray-700 placeholder-gray-400"
            placeholder="Search commands or jump to..."
            @keyup.enter="handleCommandKeydown"
          />
          <div class="flex items-center gap-1 text-xs text-gray-400 font-mono bg-gray-100 px-2 py-1 rounded">
            <span>ESC</span>
          </div>
        </div>
        <div class="max-h-[60vh] overflow-y-auto p-2">
          <div
            v-for="(menu, index) in filteredMenus"
            :key="menu.path"
            class="flex items-center justify-between p-3 rounded-lg cursor-pointer transition-colors duration-150 group hover:bg-indigo-50"
            :class="{ 'bg-indigo-50': index === 0 && searchQuery }"
            @click="handleSelectCommand(menu)"
          >
            <div class="flex items-center gap-3">
              <el-icon class="text-gray-400 group-hover:text-indigo-500"><component :is="menu.icon || 'Document'" /></el-icon>
              <span class="text-gray-700 font-medium group-hover:text-indigo-700">{{ menu.title }}</span>
            </div>
            <span class="text-xs text-gray-400 font-mono group-hover:text-indigo-400">{{ menu.path }}</span>
          </div>
          <div v-if="filteredMenus.length === 0" class="p-8 text-center text-gray-400">
            No commands found.
          </div>
        </div>
      </div>
    </el-dialog>
  </el-container>
</template>
"""

content = content.replace("  </el-container>\n</template>", template_addition)

style_addition = """
<style>
.cmd-palette-dialog {
  background: transparent !important;
  box-shadow: none !important;
  padding: 0 !important;
}
.cmd-palette-dialog .el-dialog__header {
  display: none !important;
}
.cmd-palette-dialog .el-dialog__body {
  padding: 0 !important;
  background: transparent !important;
}
</style>

<style scoped>
"""

content = content.replace("<style scoped>", style_addition)

with open("src/views/Layout.vue", "w", encoding="utf-8") as f:
    f.write(content)
