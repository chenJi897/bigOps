<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { cloudAccountApi } from '../api'

const loading = ref(false)
const tableData = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)

// 表单
const dialogVisible = ref(false)
const dialogTitle = ref('新增云账号')
const editId = ref(0)
const form = ref({ name: '', provider: 'aliyun', access_key: '', secret_key: '', region: '' })
const isCreate = ref(true)

// 更新密钥
const keysDialogVisible = ref(false)
const keysForm = ref({ access_key: '', secret_key: '' })
const keysId = ref(0)

const providerOptions = [
  { label: '阿里云', value: 'aliyun' },
  { label: '腾讯云', value: 'tencent' },
  { label: 'AWS', value: 'aws' },
]

async function fetchData() {
  loading.value = true
  try {
    const res: any = await cloudAccountApi.list(page.value, size.value)
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

function handleAdd() {
  isCreate.value = true
  dialogTitle.value = '新增云账号'
  form.value = { name: '', provider: 'aliyun', access_key: '', secret_key: '', region: '' }
  dialogVisible.value = true
}

function handleEdit(row: any) {
  isCreate.value = false
  dialogTitle.value = '编辑云账号'
  editId.value = row.id
  form.value = { name: row.name, provider: row.provider, access_key: '', secret_key: '', region: row.region || '' }
  dialogVisible.value = true
}

async function submitForm() {
  if (!form.value.name) { ElMessage.warning('请输入名称'); return }
  try {
    if (isCreate.value) {
      if (!form.value.access_key || !form.value.secret_key) { ElMessage.warning('请输入 AK/SK'); return }
      await cloudAccountApi.create(form.value)
      ElMessage.success('创建成功')
    } else {
      await cloudAccountApi.update(editId.value, { name: form.value.name, region: form.value.region, status: 1 })
      ElMessage.success('更新成功')
    }
    dialogVisible.value = false
    fetchData()
  } catch {}
}

function handleUpdateKeys(row: any) {
  keysId.value = row.id
  keysForm.value = { access_key: '', secret_key: '' }
  keysDialogVisible.value = true
}

async function submitKeys() {
  if (!keysForm.value.access_key || !keysForm.value.secret_key) { ElMessage.warning('请输入 AK/SK'); return }
  try {
    await cloudAccountApi.updateKeys(keysId.value, keysForm.value.access_key, keysForm.value.secret_key)
    ElMessage.success('密钥更新成功')
    keysDialogVisible.value = false
  } catch {}
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(`确定删除 "${row.name}" 吗？`, '提示', { type: 'warning' })
    await cloudAccountApi.delete(row.id)
    ElMessage.success('删除成功')
    fetchData()
  } catch {}
}

async function handleSync(row: any) {
  try {
    ElMessage.info('开始同步...')
    const res: any = await cloudAccountApi.sync(row.id)
    ElMessage.success(res.message || '同步完成')
    fetchData()
  } catch {}
}

function providerLabel(val: string) {
  return providerOptions.find(o => o.value === val)?.label || val
}

function handlePageChange(p: number) {
  page.value = p
  fetchData()
}

onMounted(fetchData)
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>云账号管理</span>
          <el-button type="primary" @click="handleAdd"><el-icon><Plus /></el-icon> 新增</el-button>
        </div>
      </template>

      <el-table :data="tableData" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" min-width="150" />
        <el-table-column prop="provider" label="云厂商" width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ providerLabel(row.provider) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="region" label="Region" min-width="150" show-overflow-tooltip />
        <el-table-column prop="last_sync_status" label="同步状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.last_sync_status === 'success' ? 'success' : row.last_sync_status === 'failed' ? 'danger' : row.last_sync_status === 'syncing' ? 'warning' : 'info'" size="small">
              {{ row.last_sync_status || '未同步' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_sync_at" label="最后同步" width="180" />
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button link size="small" @click="handleSync(row)"><el-icon><Refresh /></el-icon> 同步</el-button>
            <el-button link size="small" @click="handleEdit(row)"><el-icon><Edit /></el-icon> 编辑</el-button>
            <el-button link size="small" @click="handleUpdateKeys(row)"><el-icon><Key /></el-icon> 密钥</el-button>
            <el-button link size="small" type="danger" @click="handleDelete(row)"><el-icon><Delete /></el-icon> 删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-if="total > 0"
        style="margin-top: 16px; justify-content: flex-end;"
        background layout="total, prev, pager, next"
        :total="total" :page-size="size" :current-page="page"
        @current-change="handlePageChange"
      />
    </el-card>

    <!-- 新增/编辑 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="云厂商" v-if="isCreate">
          <el-select v-model="form.provider" style="width: 100%;">
            <el-option v-for="o in providerOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="AccessKey" v-if="isCreate"><el-input v-model="form.access_key" /></el-form-item>
        <el-form-item label="SecretKey" v-if="isCreate"><el-input v-model="form.secret_key" type="password" show-password /></el-form-item>
        <el-form-item label="Region"><el-input v-model="form.region" placeholder="cn-hangzhou,cn-beijing" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

    <!-- 更新密钥 -->
    <el-dialog v-model="keysDialogVisible" title="更新密钥" width="450px">
      <el-form :model="keysForm" label-width="80px">
        <el-form-item label="AccessKey"><el-input v-model="keysForm.access_key" /></el-form-item>
        <el-form-item label="SecretKey"><el-input v-model="keysForm.secret_key" type="password" show-password /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="keysDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitKeys">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
</style>
