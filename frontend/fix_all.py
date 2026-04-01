import re

with open("src/views/Layout.vue", "r", encoding="utf-8") as f:
    content = f.read()

# Make sure imports are there
imports = """
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { CommandDialog, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbSeparator, BreadcrumbPage } from '@/components/ui/breadcrumb'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
"""

if "import { Button }" not in content:
    content = content.replace("import { resetRouter } from '../router'\n", "import { resetRouter } from '../router'\n" + imports)

# Remove Element Plus imports that are no longer used here if any, wait, keep it safe
content = content.replace("import { authApi, notificationApi }", "import { notificationApi }")

# Remove unused variables if they exist
content = re.sub(r'// 修改密码\nconst pwdVisible = ref\(false\)\nconst pwdForm = ref\(\{ old_password: \'\', new_password: \'\', confirm_password: \'\' \}\)\n', '', content)
content = re.sub(r'const activeMenu = computed\(\(\) => \{.*?\n\}\)\n', '', content, flags=re.DOTALL)

# Fix command palette
content = re.sub(r'const searchQuery = ref\(\'\'\)\nconst searchInputRef = ref<any>\(null\)\nconst selectedIndex = ref\(0\)\n\nwatch\(searchQuery, \(\) => \{\n  selectedIndex.value = 0\n\}\)\n', '', content)
content = re.sub(r'const filteredMenus = computed\(\(\) => \{.*?\n\}\)\n', '', content, flags=re.DOTALL)

handleCmdK_new = """function handleCmdK(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    cmdPaletteVisible.value = true
  }
}"""
content = re.sub(r'function handleCmdK\(e: KeyboardEvent\) \{.*?\n\}', handleCmdK_new, content, flags=re.DOTALL)
content = re.sub(r'function handleCommandKeydown\(e: KeyboardEvent\) \{.*?\n\}\n', '', content, flags=re.DOTALL)
content = content.replace('function openNotificationConfig() {\n  notificationVisible.value = false\n  router.push(\'/notification/console\')\n}\n', '')

with open("src/views/Layout.vue", "w", encoding="utf-8") as f:
    f.write(content)

