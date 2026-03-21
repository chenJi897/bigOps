<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { userApi, roleApi, authApi } from '../api'

const users = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const loading = ref(false)
const searchKeyword = ref('')

// 角色分配
const roleVisible = ref(false)
const roleUserId = ref(0)
const allRoles = ref<any[]>([])
const selectedRoles = ref<number[]>([])

// 新增用户
const createVisible = ref(false)
const createForm = ref({ username: '', password: '', email: '' })
const createLoading = ref(false)

async function loadUsers() {
  loading.value = true
  try {
    const res: any = await userApi.list(page.value, size.value, searchKeyword.value)
    users.value = res.data.list || []
    total.value = res.data.total
  } catch {} finally { loading.value = false }
}

function handleSearch() {
  page.value = 1
  loadUsers()
}

function handleReset() {
  searchKeyword.value = ''
  page.value = 1
  loadUsers()
}

async function toggleStatus(row: any) {
  const newStatus = row.status === 1 ? 0 : 1
  await userApi.updateStatus(row.id, newStatus)
  ElMessage.success(newStatus === 1 ? '已启用' : '已禁用')
  loadUsers()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确定删除用户 ${row.username}？`, '提示', { type: 'warning' })
  await userApi.delete(row.id)
  ElMessage.success('删除成功')
  loadUsers()
}

async function openRoleDialog(row: any) {
  roleUserId.value = row.id
  const [rolesRes, userRolesRes]: any = await Promise.all([
    roleApi.list(1, 100),
    userApi.getRoles(row.id),
  ])
  allRoles.value = rolesRes.data.list || []
  selectedRoles.value = (userRolesRes.data || []).map((r: any) => r.id)
  roleVisible.value = true
}

async function submitRoles() {
  await userApi.setRoles(roleUserId.value, selectedRoles.value)
  ElMessage.success('角色分配成功')
  roleVisible.value = false
}

function openCreateDialog() {
  createForm.value = { username: '', password: '', email: '' }
  createVisible.value = true
}

async function submitCreate() {
  const { username, password, email } = createForm.value
  if (!username || !password) { ElMessage.warning('请填写用户名和密码'); return }
  if (password.length < 8) { ElMessage.warning('密码至少 8 位'); return }
  if (!/[A-Z]/.test(password)) { ElMessage.warning('密码必须包含大写字母'); return }
  if (!/[a-z]/.test(password)) { ElMessage.warning('密码必须包含小写字母'); return }
  if (!/[0-9]/.test(password)) { ElMessage.warning('密码必须包含数字'); return }
  createLoading.value = true
  try {
    await authApi.register(username, password, email)
    ElMessage.success('用户创建成功')
    createVisible.value = false
    loadUsers()
  } catch {} finally {
    createLoading.value = false
  }
}

onMounted(loadUsers)
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center">
          <span>用户管理</span>
          <el-button type="primary" @click="openCreateDialog"><el-icon><Plus /></el-icon> 新增用户</el-button>
        </div>
      </template>

      <!-- 搜索栏 -->
      <el-form :inline="true" @submit.prevent="handleSearch" style="margin-bottom:16px">
        <el-form-item>
          <el-input
            v-model="searchKeyword"
            placeholder="用户名 / 邮箱 / 姓名"
            clearable
            style="width:220px"
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="users" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="username" label="用户名" width="130" />
        <el-table-column prop="email" label="邮箱" min-width="180" show-overflow-tooltip />
        <el-table-column prop="real_name" label="姓名" width="100" />
        <el-table-column prop="phone" label="手机号" width="140" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" fixed="right" width="220">
          <template #default="{ row }">
            <el-button size="small" @click="openRoleDialog(row)">角色</el-button>
            <el-button
              size="small"
              :type="row.status === 1 ? 'warning' : 'success'"
              @click="toggleStatus(row)"
            >{{ row.status === 1 ? '禁用' : '启用' }}</el-button>
            <el-button
              size="small"
              type="danger"
              @click="handleDelete(row)"
              :disabled="row.id === 1"
            >删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        style="margin-top:16px;justify-content:flex-end"
        background
        layout="total, sizes, prev, pager, next"
        :total="total"
        :page-size="size"
        :current-page="page"
        :page-sizes="[10, 20, 50]"
        @current-change="(p: number) => { page = p; loadUsers() }"
        @size-change="(s: number) => { size = s; page = 1; loadUsers() }"
      />
    </el-card>

    <!-- 角色分配 -->
    <el-dialog v-model="roleVisible" title="分配角色" width="400px">
      <el-checkbox-group v-model="selectedRoles" style="display:flex;flex-direction:column;gap:8px">
        <el-checkbox v-for="r in allRoles" :key="r.id" :value="r.id">{{ r.display_name }}</el-checkbox>
      </el-checkbox-group>
      <template #footer>
        <el-button @click="roleVisible = false">取消</el-button>
        <el-button type="primary" @click="submitRoles">确定</el-button>
      </template>
    </el-dialog>

    <!-- 新增用户 -->
    <el-dialog v-model="createVisible" title="新增用户" width="420px">
      <el-form :model="createForm" label-width="80px">
        <el-form-item label="用户名">
          <el-input v-model="createForm.username" placeholder="至少 3 位" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="createForm.password" type="password" show-password placeholder="至少 8 位，含大小写字母和数字" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="createForm.email" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="createLoading" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
</style>
