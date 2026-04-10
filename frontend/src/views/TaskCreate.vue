<script setup lang="ts">
defineOptions({ name: 'TaskCreate' })
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { taskApi } from '../api'
import ScriptEditor from '../components/ScriptEditor.vue'

const router = useRouter()
const route = useRoute()
const editId = computed(() => Number(route.params.id) || 0)
const isEdit = computed(() => editId.value > 0)
const loading = ref(false)
const submitting = ref(false)
const editorWarnings = ref<any[]>([])

const form = ref({
  name: '',
  task_type: 'script',
  script_type: 'bash',
  script_content: '',
  timeout: 60,
  run_as_user: '',
  description: '',
})

const taskTypeOptions = [
  { label: '脚本执行', value: 'script' },
  { label: '文件分发', value: 'file_transfer' },
  { label: 'API 调用', value: 'api_call' },
]

const scriptTypeOptions = [
  { label: 'Bash', value: 'bash' },
  { label: 'Python', value: 'python' },
  { label: 'Shell (sh)', value: 'sh' },
  { label: 'PowerShell', value: 'powershell' },
]

const isScript = computed(() => form.value.task_type === 'script')

function onTaskTypeChange(newType: string) {
  if (newType !== 'script') {
    form.value.script_type = ''
    form.value.script_content = ''
  } else if (!form.value.script_type) {
    form.value.script_type = 'bash'
  }
}

function onScriptTypeChange() {
  if (form.value.script_content.trim()) {
    ElMessageBox.confirm('切换脚本语言会清空已有内容，是否继续？', '确认', { type: 'warning' })
      .then(() => { form.value.script_content = '' })
      .catch(() => {})
  }
}

function onEditorValidate(w: any[]) { editorWarnings.value = w }

async function loadTask() {
  if (!isEdit.value) return
  loading.value = true
  try {
    const res: any = await taskApi.getById(editId.value)
    const d = res.data
    let tt = d.task_type || 'script'
    if (['shell', 'bash', 'python', 'sh', 'powershell'].includes(tt)) tt = 'script'
    form.value = {
      name: d.name || '',
      task_type: tt,
      script_type: d.script_type || (tt === 'script' ? 'bash' : ''),
      script_content: d.script_content || '',
      timeout: d.timeout || 60,
      run_as_user: d.run_as_user || '',
      description: d.description || '',
    }
  } catch {
    ElMessage.error('加载任务失败')
  } finally { loading.value = false }
}

async function handleSubmit() {
  if (!form.value.name.trim()) {
    ElMessage.warning('请输入任务名称')
    return
  }
  if (isScript.value) {
    if (!form.value.script_content.trim()) {
      ElMessage.warning('脚本内容不能为空')
      return
    }
    if (editorWarnings.value.some(w => w.level === 'error')) {
      ElMessage.error('脚本存在阻断性错误，请先修复')
      return
    }
  }
  const payload: any = {
    ...form.value,
    script_type: isScript.value ? form.value.script_type : '',
    script_content: isScript.value ? form.value.script_content : '',
  }
  submitting.value = true
  try {
    if (isEdit.value) {
      await taskApi.update(editId.value, payload)
      ElMessage.success('更新成功')
    } else {
      await taskApi.create(payload)
      ElMessage.success('创建成功')
    }
    router.push('/task/list')
  } catch {
    // api interceptor already shows error
  } finally { submitting.value = false }
}

function goBack() { router.push('/task/list') }

onMounted(() => { loadTask() })
</script>

<template>
  <div class="p-4 md:p-6 min-h-full flex flex-col">
    <el-card shadow="never" class="border-0 shadow-sm flex-1 flex flex-col" v-loading="loading">
      <template #header>
        <div class="flex items-center gap-3">
          <el-button link @click="goBack" class="text-gray-500 hover:text-gray-700 -ml-2">
            <el-icon class="text-lg"><Back /></el-icon>
          </el-button>
          <span class="text-base font-medium text-gray-800">{{ isEdit ? '编辑任务' : '创建任务' }}</span>
        </div>
      </template>

      <div class="max-w-4xl">
        <el-form :model="form" label-width="120px" class="mt-4" label-position="right">
          <el-form-item label="任务名称" required>
            <el-input v-model="form.name" placeholder="请输入任务名称" maxlength="200" />
          </el-form-item>

          <el-form-item label="任务大类" required>
            <el-select v-model="form.task_type" class="w-56" @change="onTaskTypeChange">
              <el-option v-for="o in taskTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
          </el-form-item>

          <el-form-item label="脚本语言" v-if="isScript" required>
            <el-select v-model="form.script_type" class="w-56" @change="onScriptTypeChange">
              <el-option v-for="o in scriptTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
          </el-form-item>

          <el-form-item v-if="isScript" required class="script-editor-item">
            <ScriptEditor v-model="form.script_content" :language="form.script_type" @validate="onEditorValidate" />
          </el-form-item>

          <div v-if="!isScript" class="ml-[120px] mb-6 p-4 bg-slate-50 rounded-lg border border-slate-200 text-sm text-slate-500">
            {{ form.task_type === 'file_transfer' ? '文件分发' : 'API 调用' }} 类型的配置区域待扩展
          </div>

          <el-form-item label="超时时间">
            <div class="flex items-center gap-3">
              <el-input-number v-model="form.timeout" :min="5" :max="86400" :step="10" />
              <span class="text-gray-500 text-sm">秒</span>
            </div>
          </el-form-item>

          <el-form-item label="执行用户">
            <el-input v-model="form.run_as_user" placeholder="留空则使用 Agent 运行用户" class="w-56" />
          </el-form-item>

          <el-form-item label="描述">
            <el-input v-model="form.description" type="textarea" :rows="3" placeholder="任务描述（可选）" />
          </el-form-item>

          <el-form-item class="mt-8">
            <el-button type="primary" :loading="submitting" @click="handleSubmit" class="w-32">
              {{ isEdit ? '保存修改' : '创建任务' }}
            </el-button>
            <el-button @click="goBack" class="w-24">取消</el-button>
          </el-form-item>
        </el-form>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
:deep(.el-card__body) { flex: 1; }
.script-editor-item :deep(.el-form-item__content) { display: block; }
</style>
