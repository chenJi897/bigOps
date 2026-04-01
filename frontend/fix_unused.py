import re

with open("src/views/Layout.vue", "r", encoding="utf-8") as f:
    content = f.read()

content = content.replace("import { authApi, notificationApi } from '../api'", "import { notificationApi } from '../api'")

content = re.sub(r'// 修改密码\nconst pwdVisible = ref\(false\)\nconst pwdForm = ref\(\{ old_password: \'\', new_password: \'\', confirm_password: \'\' \}\)\n', '', content)

content = re.sub(r'const activeMenu = computed\(\(\) => \{.*?\n\}\)\n', '', content, flags=re.DOTALL)

content = re.sub(r'function handleCommandKeydown\(e: KeyboardEvent\) \{.*?\n\}\n', '', content, flags=re.DOTALL)

content = content.replace('function openNotificationConfig() {\n  notificationVisible.value = false\n  router.push(\'/notification/console\')\n}\n', '')

with open("src/views/Layout.vue", "w", encoding="utf-8") as f:
    f.write(content)

