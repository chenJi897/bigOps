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
  <div class="p-4 md:p-6 min-h-full flex flex-col">
    <el-card shadow="never" class="border-0 shadow-sm flex-1 flex flex-col">
      <template #header>
        <div class="flex justify-between items-center">
          <span class="text-base font-medium text-gray-800">部门管理</span>
          <el-button v-permission="'dept:create'" type="primary" @click="handleAdd">
            <el-icon class="mr-1"><Plus /></el-icon> 新增部门
          </el-button>
        </div>
      </template>

      <el-table :data="tableData" v-loading="loading" stripe border class="w-full">
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column prop="name" label="部门名称" min-width="180">
          <template #default="{ row }">
            <span class="font-medium text-gray-800">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="code" label="编码" width="140" align="center">
          <template #default="{ row }">
            <span class="font-mono text-gray-500 bg-gray-50 px-2 py-1 rounded text-xs">{{ row.code || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="manager_name" label="负责人" width="140" align="center">
          <template #default="{ row }">
            <span class="text-gray-700" v-if="row.manager_name">{{ row.manager_name }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="user_count" label="人数" width="100" align="center">
          <template #default="{ row }">
            <el-tag type="info" size="small" effect="plain" round>{{ row.user_count }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="text-gray-500">{{ row.description || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="90" align="center" />
        <el-table-column prop="created_at" label="创建时间" width="170" align="center" />
        <el-table-column label="操作" width="180" fixed="right" align="center">
          <template #default="{ row }">
            <div class="flex items-center justify-center gap-1">
              <el-button v-permission="'dept:edit'" link type="primary" @click="handleEdit(row)">编辑</el-button>
              <el-divider direction="vertical" />
              <el-button v-permission="'dept:delete'" link type="danger" @click="handleDelete(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="total > 0" class="mt-4 flex justify-end">
        <el-pagination
          background 
          layout="total, prev, pager, next"
          :total="total" 
          :page-size="size" 
          :current-page="page"
          @current-change="handlePageChange"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="480px" destroy-on-close align-center>
      <el-form :model="form" label-width="90px" class="pr-6">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="如：运维部" />
        </el-form-item>
        <el-form-item label="编码">
          <el-input v-model="form.code" placeholder="如：ops" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="部门描述" />
        </el-form-item>
        <el-form-item label="负责人">
          <el-select v-model="form.manager_id" placeholder="选择负责人" clearable class="w-full">
            <el-option label="无" :value="0" />
            <el-option v-for="u in allUsers" :key="u.id" :label="u.real_name || u.username" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" class="!w-32" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitForm">确定</el-button>
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
