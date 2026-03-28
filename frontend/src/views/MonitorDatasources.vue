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
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div class="page-head">
          <span class="page-title">监控数据源</span>
          <el-button type="primary" @click="openAdd">新增数据源</el-button>
        </div>
      </template>
      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="name" label="名称" min-width="180" />
        <el-table-column prop="type" label="类型" width="120" />
        <el-table-column prop="base_url" label="地址" min-width="260" show-overflow-tooltip />
        <el-table-column prop="auth_type" label="认证" width="100" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="healthCheck(row)">健康检查</el-button>
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="removeRow(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑数据源' : '新增数据源'" width="560px">
      <el-form label-width="100px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="类型"><el-select v-model="form.type"><el-option label="Prometheus" value="prometheus" /></el-select></el-form-item>
        <el-form-item label="地址"><el-input v-model="form.base_url" placeholder="http://prometheus:9090" /></el-form-item>
        <el-form-item label="访问方式"><el-select v-model="form.access_type"><el-option label="Proxy" value="proxy" /></el-select></el-form-item>
        <el-form-item label="认证方式"><el-select v-model="form.auth_type"><el-option label="无" value="none" /><el-option label="Basic" value="basic" /></el-select></el-form-item>
        <el-form-item label="用户名"><el-input v-model="form.username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password /></el-form-item>
        <el-form-item label="请求头 JSON"><el-input v-model="form.headers_json" type="textarea" :rows="4" /></el-form-item>
        <el-form-item label="状态"><el-select v-model="form.status"><el-option label="active" value="active" /><el-option label="inactive" value="inactive" /></el-select></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="dialogLoading" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
.page-head { display: flex; justify-content: space-between; align-items: center; }
.page-title { font-size: 18px; font-weight: 700; color: #1f2937; }
</style>

