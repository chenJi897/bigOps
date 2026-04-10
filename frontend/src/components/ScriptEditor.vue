<template>
  <div class="script-editor-root">
    <div class="flex items-center justify-between mb-2">
      <div class="flex items-center gap-2">
        <span class="text-sm font-medium text-slate-700">脚本内容</span>
        <el-tag size="small" type="info" effect="plain">{{ langLabel }}</el-tag>
      </div>
      <div class="flex items-center gap-1">
        <el-dropdown trigger="click" @command="insertSnippet" v-if="snippetList.length">
          <el-button size="small" plain>插入片段 <el-icon class="ml-1"><ArrowDown /></el-icon></el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-for="s in snippetList" :key="s.key" :command="s.key">{{ s.label }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button size="small" plain @click="restoreDefault" v-if="defaultTemplate">恢复默认模板</el-button>
      </div>
    </div>
    <div ref="editorRef" class="editor-wrap"></div>
    <div v-if="warnings.length" class="mt-2">
      <div v-for="(w, i) in warnings" :key="i" class="text-xs" :class="w.level === 'error' ? 'text-red-500' : 'text-amber-500'">
        {{ w.level === 'error' ? '❌' : '⚠️' }} {{ w.message }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount, computed } from 'vue'
import { EditorState } from '@codemirror/state'
import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter } from '@codemirror/view'
import { defaultKeymap, indentWithTab } from '@codemirror/commands'
import { oneDark } from '@codemirror/theme-one-dark'
import { StreamLanguage } from '@codemirror/language'
import { shell } from '@codemirror/legacy-modes/mode/shell'
import { python } from '@codemirror/legacy-modes/mode/python'

type Warning = { level: 'error' | 'warn'; message: string }

const props = defineProps<{
  modelValue: string
  language: string
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'validate', warnings: Warning[]): void
}>()

const editorRef = ref<HTMLElement | null>(null)
let view: EditorView | null = null
let isUpdating = false

const langLabel = computed(() => {
  const m: Record<string, string> = { bash: 'Bash', sh: 'Shell', python: 'Python', powershell: 'PowerShell' }
  return m[props.language] || props.language || 'Script'
})

const defaultTemplates: Record<string, string> = {
  bash: `#!/usr/bin/env bash
set -euo pipefail

echo "[INFO] start at: $(date '+%F %T')"

# TODO: 在这里编写你的运维逻辑
# 示例：
# systemctl status nginx

echo "[INFO] done at: $(date '+%F %T')"
`,
  sh: `#!/bin/sh
set -eu

echo "[INFO] start at: $(date '+%F %T')"

# TODO: 编写你的脚本

echo "[INFO] done at: $(date '+%F %T')"
`,
  python: `#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import sys
import datetime

print(f"[INFO] start at: {datetime.datetime.now()}")

# TODO: 在这里编写你的 Python 运维逻辑

print(f"[INFO] done at: {datetime.datetime.now()}")
sys.exit(0)
`,
}

const bashSnippets: Record<string, { label: string; code: string }> = {
  if_else: { label: 'if / elif / else', code: `if [ "$VAR" = "value" ]; then\n  echo "matched"\nelif [ "$VAR" = "other" ]; then\n  echo "other"\nelse\n  echo "default"\nfi\n` },
  for_loop: { label: 'for 循环', code: `for item in a b c; do\n  echo "processing: $item"\ndone\n` },
  while_loop: { label: 'while 循环', code: `while read -r line; do\n  echo "$line"\ndone < /path/to/file\n` },
  function_def: { label: '函数定义', code: `function my_func() {\n  local arg="$1"\n  echo "arg: $arg"\n  return 0\n}\n\nmy_func "hello"\n` },
  error_handler: { label: '错误处理', code: `trap 'echo "[ERROR] 脚本执行失败，退出码: $?" >&2' ERR\n` },
  check_command: { label: '命令存在检查', code: `if ! command -v curl &>/dev/null; then\n  echo "[ERROR] curl 未安装" >&2\n  exit 1\nfi\n` },
}

const pythonSnippets: Record<string, { label: string; code: string }> = {
  try_except: { label: 'try / except', code: `try:\n    pass  # TODO\nexcept Exception as e:\n    print(f"[ERROR] {e}", file=sys.stderr)\n    sys.exit(1)\n` },
  subprocess: { label: '执行系统命令', code: `import subprocess\n\nresult = subprocess.run(["ls", "-la"], capture_output=True, text=True)\nprint(result.stdout)\nif result.returncode != 0:\n    print(result.stderr, file=sys.stderr)\n` },
  file_ops: { label: '文件读写', code: `with open("/tmp/example.txt", "r") as f:\n    content = f.read()\nprint(content)\n` },
}

const snippetList = computed(() => {
  const map = props.language === 'python' ? pythonSnippets : bashSnippets
  return Object.entries(map).map(([key, val]) => ({ key, label: val.label }))
})

const defaultTemplate = computed(() => defaultTemplates[props.language] || '')

const warnings = ref<Warning[]>([])

const dangerousPatterns = [
  { pattern: /rm\s+-rf\s+\/(?:\s|$)/, message: '检测到危险命令 rm -rf /' },
  { pattern: /mkfs\b/, message: '检测到危险命令 mkfs（格式化磁盘）' },
  { pattern: /dd\s+if=.*of=\/dev\//, message: '检测到危险命令 dd 写入块设备' },
  { pattern: />\s*\/dev\/sda/, message: '检测到向 /dev/sda 写入' },
]

function validate(content: string): Warning[] {
  const result: Warning[] = []
  if (!content.trim()) {
    result.push({ level: 'error', message: '脚本内容不能为空' })
    return result
  }
  if ((props.language === 'bash' || props.language === 'sh') && !content.startsWith('#!')) {
    result.push({ level: 'warn', message: '建议添加 shebang 行（如 #!/usr/bin/env bash）' })
  }
  for (const dp of dangerousPatterns) {
    if (dp.pattern.test(content)) {
      result.push({ level: 'error', message: dp.message })
    }
  }
  return result
}

function getLanguageExtension() {
  if (props.language === 'python') return StreamLanguage.define(python)
  return StreamLanguage.define(shell)
}

function createEditor(content: string) {
  if (!editorRef.value) return
  if (view) { view.destroy(); view = null }

  const state = EditorState.create({
    doc: content,
    extensions: [
      lineNumbers(),
      highlightActiveLine(),
      highlightActiveLineGutter(),
      keymap.of([...defaultKeymap, indentWithTab]),
      getLanguageExtension(),
      oneDark,
      EditorView.updateListener.of((update) => {
        if (update.docChanged && !isUpdating) {
          const val = update.state.doc.toString()
          emit('update:modelValue', val)
          const w = validate(val)
          warnings.value = w
          emit('validate', w)
        }
      }),
      EditorView.theme({
        '&': { height: '320px', fontSize: '13px' },
        '.cm-scroller': { overflow: 'auto' },
      }),
    ],
  })
  view = new EditorView({ state, parent: editorRef.value })
}

function insertSnippet(key: string) {
  const map = props.language === 'python' ? pythonSnippets : bashSnippets
  const snippet = map[key]
  if (!snippet || !view) return
  const pos = view.state.selection.main.head
  view.dispatch({ changes: { from: pos, insert: snippet.code } })
}

function restoreDefault() {
  const tpl = defaultTemplate.value
  if (!tpl) return
  emit('update:modelValue', tpl)
  if (view) {
    isUpdating = true
    view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: tpl } })
    isUpdating = false
  }
}

watch(() => props.modelValue, (newVal) => {
  if (!view) return
  const current = view.state.doc.toString()
  if (current !== newVal) {
    isUpdating = true
    view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: newVal } })
    isUpdating = false
  }
})

watch(() => props.language, () => {
  createEditor(props.modelValue || '')
})

onMounted(() => {
  createEditor(props.modelValue || '')
  if (!props.modelValue && defaultTemplate.value) {
    emit('update:modelValue', defaultTemplate.value)
  }
})

onBeforeUnmount(() => {
  if (view) { view.destroy(); view = null }
})

defineExpose({ validate: () => validate(props.modelValue), insertSnippet, restoreDefault })
</script>

<style scoped>
.editor-wrap {
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  overflow: hidden;
}
.editor-wrap :deep(.cm-editor) { border-radius: 8px; }
.editor-wrap :deep(.cm-gutters) { border-radius: 8px 0 0 8px; }
</style>
