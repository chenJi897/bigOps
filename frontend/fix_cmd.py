import re

with open("src/views/Layout.vue", "r", encoding="utf-8") as f:
    content = f.read()

# Remove unused command palette variables
content = re.sub(r'const searchQuery = ref\(\'\'\)\nconst searchInputRef = ref<any>\(null\)\nconst selectedIndex = ref\(0\)\n\nwatch\(searchQuery, \(\) => \{\n  selectedIndex.value = 0\n\}\)\n', '', content)
content = re.sub(r'const filteredMenus = computed\(\(\) => \{.*?\n\}\)\n', '', content, flags=re.DOTALL)

# In handleCmdK we had:
#     searchQuery.value = ''
#     selectedIndex.value = 0
#     setTimeout(() => {
#       searchInputRef.value?.focus()
#     }, 100)
# We need to replace that with just opening the dialog.
handleCmdK_new = """function handleCmdK(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    cmdPaletteVisible.value = true
  }
}"""
content = re.sub(r'function handleCmdK\(e: KeyboardEvent\) \{.*?\n\}', handleCmdK_new, content, flags=re.DOTALL)

with open("src/views/Layout.vue", "w", encoding="utf-8") as f:
    f.write(content)
