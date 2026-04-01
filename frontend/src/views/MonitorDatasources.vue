<script setup lang="ts">
defineOptions({ name: 'MonitorDatasources' })

import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { monitorApi } from '../api'

const loading = ref(false)
const dialogVisible = ref(false)
const dialogLoading = ref(false)
const isEdit = ref(false)
const editId = ref(0)
const tableData = ref<any[]>([])
const form = ref({
  name: '',
  type: 'prometheus',
  base_url: '',
  access_type: 'proxy',
  auth_type: 'none',
  username: '',
  password: '',
  headers_json: '{}',
  status: 'active',
})

async function fetchData() {
  loading.value = true
  try {
    const res = await monitorApi.datasources()
    tableData.value = (res as any).data || []
  } finally {
    loading.value = false
  }
}

function openAdd() {
  isEdit.value = false
  editId.value = 0
  form.value = { name: '', type: 'prometheus', base_url: '', access_type: 'proxy', auth_type: 'none', username: '', password: '', headers_json: '{}', status: 'active' }
  dialogVisible.value = true
}

function openEdit(row: any) {
  isEdit.value = true
  editId.value = row.id
  form.value = {
    name: row.name || '',
    type: row.type || 'prometheus',
    base_url: row.base_url || '',
    access_type: row.access_type || 'proxy',
    auth_type: row.auth_type || 'none',
    username: row.username || '',
    password: row.password || '',
    headers_json: row.headers_json || '{}',
    status: row.status || 'active',
  }
  dialogVisible.value = true
}

async function submit() {
  dialogLoading.value = true
  try {
    if (isEdit.value) {
      await monitorApi.updateDatasource(editId.value, form.value)
    } else {
      await monitorApi.createDatasource(form.value)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } finally {
    dialogLoading.value = false
  }
}

async function removeRow(row: any) {
  await ElMessageBox.confirm(`确定删除数据源 ${row.name}？`, '提示', { type: 'warning' })
  await monitorApi.deleteDatasource(row.id)
  ElMessage.success('删除成功')
  fetchData()
}

async function healthCheck(row: any) {
  const res = await monitorApi.datasourceHealth(row.id)
  const data = (res as any).data || {}
  ElMessage[data.ok ? 'success' : 'warning'](`健康检查：${data.message || (data.ok ? 'ok' : 'failed')}`)
}

onMounted(fetchData)
</script>

<template>
  <div class="p-4 md:p-6 min-h-full flex flex-col bg-slate-50">
    <el-card shadow="never" class="border-0 shadow-sm rounded-2xl flex-1 flex flex-col">
      <template #header>
        <div class="flex justify-between items-center">
          <span class="text-xl font-bold text-slate-800">监控数据源</span>
          <el-button type="primary" @click="openAdd">
            <el-icon class="mr-1"><Plus /></el-icon> 新增数据源
          </el-button>
        </div>
      </template>
      <el-table :data="tableData" v-loading="loading" stripe border class="w-full">
        <el-table-column prop="name" label="名称" min-width="180">
          <template #default="{ row }">
            <span class="font-medium text-slate-800">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="type" label="类型" width="140" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="info" effect="plain" round>{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="base_url" label="地址" min-width="260" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="font-mono text-slate-600 text-sm bg-slate-50 px-2 py-1 rounded">{{ row.base_url }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="auth_type" label="认证" width="100" align="center" />
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small" effect="light">
              {{ row.status === 'active' ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right" align="center">
          <template #default="{ row }">
            <div class="flex items-center justify-center gap-1">
              <el-button link type="success" @click="healthCheck(row)">健康检查</el-button>
              <el-divider direction="vertical" />
              <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
              <el-divider direction="vertical" />
              <el-button link type="danger" @click="removeRow(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑数据源' : '新增数据源'" width="560px" destroy-on-close align-center>
      <el-form label-width="110px" class="pr-6">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="例如：Prometheus-Prod" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" class="w-full">
            <el-option label="Prometheus" value="prometheus" />
          </el-select>
        </el-form-item>
        <el-form-item label="地址" required>
          <el-input v-model="form.base_url" placeholder="http://prometheus:9090" />
        </el-form-item>
        <el-form-item label="访问方式">
          <el-select v-model="form.access_type" class="w-full">
            <el-option label="Proxy" value="proxy" />
          </el-select>
        </el-form-item>
        <el-form-item label="认证方式">
          <el-select v-model="form.auth_type" class="w-full">
            <el-option label="无" value="none" />
            <el-option label="Basic" value="basic" />
          </el-select>
        </el-form-item>
        <template v-if="form.auth_type === 'basic'">
          <el-form-item label="用户名">
            <el-input v-model="form.username" placeholder="Basic Auth 用户名" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="form.password" type="password" show-password placeholder="Basic Auth 密码" />
          </el-form-item>
        </template>
        <el-form-item label="请求头 JSON">
          <el-input v-model="form.headers_json" type="textarea" :rows="4" placeholder="{&quot;X-Custom-Header&quot;: &quot;value&quot;}" class="font-mono text-sm" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status" class="w-full">
            <el-option label="启用" value="active" />
            <el-option label="禁用" value="inactive" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="dialogLoading" @click="submit">保存</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
:deep(.el-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
}
:deep(.el-table) {
  flex: 1;
}
</style>

