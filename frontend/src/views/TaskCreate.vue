<script setup lang="ts">
defineOptions({ name: 'TaskCreate' })
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { taskApi } from '../api'

const router = useRouter()
const route = useRoute()
const editId = computed(() => Number(route.params.id) || 0)
const isEdit = computed(() => editId.value > 0)
const loading = ref(false)
const submitting = ref(false)

const form = ref({
  name: '',
  task_type: 'shell',
  script_type: 'bash',
  script_content: '',
  timeout: 60,
  run_as_user: '',
  description: '',
})

const taskTypeOptions = [
  { label: 'Shell 脚本', value: 'shell' },
  { label: 'Python 脚本', value: 'python' },
  { label: '文件分发', value: 'file_transfer' },
]

const scriptTypeOptions = [
  { label: 'Bash', value: 'bash' },
  { label: 'Python', value: 'python' },
  { label: 'PowerShell', value: 'powershell' },
]

async function loadTask() {
  if (!isEdit.value) return
  loading.value = true
  try {
    const res: any = await taskApi.getById(editId.value)
    const d = res.data
    form.value = {
      name: d.name || '',
      task_type: d.task_type || 'shell',
      script_type: d.script_type || 'bash',
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
  if (!form.value.script_content.trim() && form.value.task_type !== 'file_transfer') {
    ElMessage.warning('请输入脚本内容')
    return
  }
  submitting.value = true
  try {
    if (isEdit.value) {
      await taskApi.update(editId.value, form.value)
      ElMessage.success('更新成功')
    } else {
      await taskApi.create(form.value)
      ElMessage.success('创建成功')
    }
    router.push('/task/list')
  } catch (e: any) {
    // api interceptor already shows error
  } finally { submitting.value = false }
}

function goBack() { router.push('/task/list') }

onMounted(() => { loadTask() })
</script>

<template>
  <div class="page">
    <el-card shadow="never" v-loading="loading">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>{{ isEdit ? '编辑任务' : '创建任务' }}</span>
          <el-button @click="goBack">返回</el-button>
        </div>
      </template>

      <el-form :model="form" label-width="100px" style="max-width: 700px;">
        <el-form-item label="任务名称" required>
          <el-input v-model="form.name" placeholder="请输入任务名称" maxlength="200" />
        </el-form-item>
        <el-form-item label="任务类型">
          <el-select v-model="form.task_type" style="width: 200px;">
            <el-option v-for="o in taskTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="脚本类型" v-if="form.task_type !== 'file_transfer'">
          <el-select v-model="form.script_type" style="width: 200px;">
            <el-option v-for="o in scriptTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="脚本内容" v-if="form.task_type !== 'file_transfer'" required>
          <el-input v-model="form.script_content" type="textarea" :rows="12" placeholder="请输入脚本内容" style="font-family: 'Courier New', Consolas, monospace;" />
        </el-form-item>
        <el-form-item label="超时时间">
          <el-input-number v-model="form.timeout" :min="1" :max="86400" :step="10" />
          <span style="margin-left: 8px; color: #909399;">秒</span>
        </el-form-item>
        <el-form-item label="执行用户">
          <el-input v-model="form.run_as_user" placeholder="留空则使用 Agent 运行用户" style="width: 200px;" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="任务描述（可选）" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="submitting" @click="handleSubmit">{{ isEdit ? '保存修改' : '创建任务' }}</el-button>
          <el-button @click="goBack">取消</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
</style>
