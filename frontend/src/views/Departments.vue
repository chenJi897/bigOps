<script setup lang="ts">
defineOptions({ name: 'Departments' })
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { departmentApi, userApi } from '../api'

const loading = ref(false)
const tableData = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const allUsers = ref<any[]>([])

const dialogVisible = ref(false)
const dialogTitle = ref('新增部门')
const isEdit = ref(false)
const editId = ref(0)
const form = ref({ name: '', code: '', description: '', manager_id: 0, sort: 0 })

async function fetchData() {
  loading.value = true
  try {
    const res: any = await departmentApi.list(page.value, size.value)
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

function handleAdd() {
  isEdit.value = false
  dialogTitle.value = '新增部门'
  form.value = { name: '', code: '', description: '', manager_id: 0, sort: 0 }
  dialogVisible.value = true
}

function handleEdit(row: any) {
  isEdit.value = true
  dialogTitle.value = '编辑部门'
  editId.value = row.id
  form.value = { name: row.name, code: row.code || '', description: row.description || '', manager_id: row.manager_id || 0, sort: row.sort || 0 }
  dialogVisible.value = true
}

async function submitForm() {
  if (!form.value.name) { ElMessage.warning('请输入部门名称'); return }
  try {
    if (isEdit.value) {
      await departmentApi.update(editId.value, form.value)
      ElMessage.success('更新成功')
    } else {
      await departmentApi.create(form.value)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchData()
  } catch {}
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(`确定删除部门 "${row.name}" 吗？`, '提示', { type: 'warning' })
    await departmentApi.delete(row.id)
    ElMessage.success('删除成功')
    fetchData()
  } catch {}
}

function handlePageChange(p: number) {
  page.value = p
  fetchData()
}

onMounted(() => {
  fetchData()
  userApi.list(1, 200).then((res: any) => { allUsers.value = res.data?.list || [] }).catch(() => {})
})
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>部门管理</span>
          <el-button type="primary" @click="handleAdd"><el-icon><Plus /></el-icon> 新增</el-button>
        </div>
      </template>

      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="部门名称" min-width="150" />
        <el-table-column prop="code" label="编码" width="120" />
        <el-table-column prop="manager_name" label="负责人" width="120">
          <template #default="{ row }">{{ row.manager_name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="user_count" label="人数" width="80" />
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column prop="sort" label="排序" width="80" />
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" min-width="160" fixed="right">
          <template #default="{ row }">
            <el-button link size="small" @click="handleEdit(row)"><el-icon><Edit /></el-icon> 编辑</el-button>
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

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="480px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称"><el-input v-model="form.name" placeholder="如：运维部" /></el-form-item>
        <el-form-item label="编码"><el-input v-model="form.code" placeholder="如：ops" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item>
        <el-form-item label="负责人">
          <el-select v-model="form.manager_id" placeholder="选择负责人" clearable style="width: 100%;">
            <el-option label="无" :value="0" />
            <el-option v-for="u in allUsers" :key="u.id" :label="u.real_name || u.username" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
</style>
