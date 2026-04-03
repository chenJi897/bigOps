<script setup lang="ts">
defineOptions({ name: 'Users' })
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { userApi, roleApi, authApi, departmentApi } from '../api'

const users = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const loading = ref(false)
const searchKeyword = ref('')

// 所有部门（下拉选择用）
const allDepts = ref<any[]>([])
// 所有角色
const allRoles = ref<any[]>([])

// 新建用户
const createVisible = ref(false)
const createForm = ref({ username: '', password: '', email: '', department_id: 0, role_ids: [] as number[] })
const createLoading = ref(false)

// 编辑用户
const editVisible = ref(false)
const editId = ref(0)
const editForm = ref({ real_name: '', phone: '', email: '', department_id: 0 })

// 角色分配
const roleVisible = ref(false)
const roleUserId = ref(0)
const roleUserName = ref('')
const selectedRoles = ref<number[]>([])

async function loadUsers() {
  loading.value = true
  try {
    const res: any = await userApi.list(page.value, size.value, searchKeyword.value)
    users.value = res.data.list || []
    total.value = res.data.total
  } catch {} finally { loading.value = false }
}

async function loadDepts() {
  try {
    const res: any = await departmentApi.all()
    allDepts.value = res.data || []
  } catch {}
}

async function loadRoles() {
  try {
    const res: any = await roleApi.list(1, 100)
    allRoles.value = res.data.list || []
  } catch {}
}

function handleSearch() { page.value = 1; loadUsers() }
function handleReset() { searchKeyword.value = ''; page.value = 1; loadUsers() }

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

// --- 新建用户 ---
function openCreateDialog() {
  createForm.value = { username: '', password: '', email: '', department_id: 0, role_ids: [] }
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
    // 注册成功后，获取新用户并设置部门和角色
    const listRes: any = await userApi.list(1, 1, username)
    const newUser = listRes.data?.list?.[0]
    if (newUser) {
      if (createForm.value.department_id > 0) {
        await userApi.setDepartment(newUser.id, createForm.value.department_id)
      }
      if (createForm.value.role_ids.length > 0) {
        await userApi.setRoles(newUser.id, createForm.value.role_ids)
      }
    }
    ElMessage.success('用户创建成功')
    createVisible.value = false
    loadUsers()
  } catch {} finally {
    createLoading.value = false
  }
}

// --- 编辑用户 ---
function openEditDialog(row: any) {
  editId.value = row.id
  editForm.value = {
    real_name: row.real_name || '',
    phone: row.phone || '',
    email: row.email || '',
    department_id: row.department_id || 0,
  }
  editVisible.value = true
}

async function submitEdit() {
  try {
    await userApi.update(editId.value, editForm.value)
    ElMessage.success('更新成功')
    editVisible.value = false
    loadUsers()
  } catch {}
}

// --- 角色分配 ---
async function openRoleDialog(row: any) {
  roleUserId.value = row.id
  roleUserName.value = row.username
  const res: any = await userApi.getRoles(row.id)
  selectedRoles.value = (res.data || []).map((r: any) => r.id)
  roleVisible.value = true
}

async function submitRoles() {
  await userApi.setRoles(roleUserId.value, selectedRoles.value)
  ElMessage.success('角色分配成功')
  roleVisible.value = false
}

onMounted(() => {
  loadUsers()
  loadDepts()
  loadRoles()
})
</script>

<template>
  <div class="p-4 md:p-6 min-h-full flex flex-col">
    <el-card shadow="never" class="border-0 shadow-sm flex-1 flex flex-col">
      <template #header>
        <div class="flex justify-between items-center">
          <span class="text-base font-medium text-gray-800">用户管理</span>
          <el-button v-permission="'user:create'" type="primary" @click="openCreateDialog">
            <el-icon class="mr-1"><Plus /></el-icon> 新增用户
          </el-button>
        </div>
      </template>

      <div class="flex flex-wrap items-center gap-3 mb-4">
        <el-input 
          v-model="searchKeyword" 
          placeholder="用户名 / 邮箱 / 姓名" 
          clearable 
          class="w-64" 
          @keyup.enter="handleSearch" 
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button type="primary" @click="handleSearch">搜索</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table :data="users" v-loading="loading" stripe border class="w-full">
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column prop="username" label="用户名" min-width="120">
          <template #default="{ row }">
            <span class="font-medium text-gray-800">{{ row.username }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="real_name" label="姓名" width="120" align="center">
          <template #default="{ row }">
            <span class="text-gray-600">{{ row.real_name || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="部门" width="140" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.department_name" size="small" type="info" effect="light">
              {{ row.department_name }}
            </el-tag>
            <span v-else class="text-gray-400">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="email" label="邮箱" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="text-gray-600">{{ row.email || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="phone" label="手机号" width="130" align="center">
          <template #default="{ row }">
            <span class="text-gray-600">{{ row.phone || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small" effect="plain" round>
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170" align="center" />
        <el-table-column label="操作" fixed="right" width="260" align="center">
          <template #default="{ row }">
            <div class="flex items-center justify-center gap-1">
              <el-button v-permission="'user:edit'" link type="primary" @click="openEditDialog(row)">编辑</el-button>
              <el-divider direction="vertical" />
              <el-button v-permission="'user:assign_role'" link type="primary" @click="openRoleDialog(row)">角色</el-button>
              <el-divider direction="vertical" />
              <el-button link :type="row.status === 1 ? 'warning' : 'success'" @click="toggleStatus(row)">
                {{ row.status === 1 ? '禁用' : '启用' }}
              </el-button>
              <el-divider direction="vertical" />
              <el-button v-permission="'user:delete'" link type="danger" @click="handleDelete(row)" :disabled="row.id === 1">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div class="mt-4 flex justify-end">
        <el-pagination
          background 
          layout="total, sizes, prev, pager, next"
          :total="total" 
          :page-size="size" 
          :current-page="page" 
          :page-sizes="[10, 20, 50]"
          @current-change="(p: number) => { page = p; loadUsers() }"
          @size-change="(s: number) => { size = s; page = 1; loadUsers() }"
        />
      </div>
    </el-card>

    <!-- 新建用户 -->
    <el-dialog v-model="createVisible" title="新增用户" width="500px" destroy-on-close align-center>
      <el-form :model="createForm" label-width="90px" class="pr-6">
        <el-form-item label="用户名" required>
          <el-input v-model="createForm.username" placeholder="至少 3 位" />
        </el-form-item>
        <el-form-item label="密码" required>
          <el-input v-model="createForm.password" type="password" show-password placeholder="至少 8 位，含大小写字母和数字" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="createForm.email" placeholder="可选" />
        </el-form-item>
        <el-form-item label="部门">
          <el-select v-model="createForm.department_id" placeholder="选择部门" clearable class="w-full">
            <el-option v-for="d in allDepts" :key="d.id" :label="d.name" :value="d.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="createForm.role_ids" multiple placeholder="选择角色" class="w-full">
            <el-option v-for="r in allRoles" :key="r.id" :label="r.display_name" :value="r.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="createVisible = false">取消</el-button>
          <el-button type="primary" :loading="createLoading" @click="submitCreate">创建</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 编辑用户 -->
    <el-dialog v-model="editVisible" title="编辑用户" width="500px" destroy-on-close align-center>
      <el-form :model="editForm" label-width="90px" class="pr-6">
        <el-form-item label="姓名">
          <el-input v-model="editForm.real_name" placeholder="真实姓名" />
        </el-form-item>
        <el-form-item label="手机号">
          <el-input v-model="editForm.phone" placeholder="手机号" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="editForm.email" placeholder="邮箱" />
        </el-form-item>
        <el-form-item label="部门">
          <el-select v-model="editForm.department_id" placeholder="选择部门" clearable class="w-full">
            <el-option label="无部门" :value="0" />
            <el-option v-for="d in allDepts" :key="d.id" :label="d.name" :value="d.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="editVisible = false">取消</el-button>
          <el-button type="primary" @click="submitEdit">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 角色分配 -->
    <el-dialog v-model="roleVisible" :title="'分配角色 - ' + roleUserName" width="440px" destroy-on-close align-center>
      <div class="bg-gray-50 p-4 rounded-lg border border-gray-100 max-h-[60vh] overflow-y-auto">
        <el-checkbox-group v-model="selectedRoles" class="flex flex-col gap-3">
          <el-checkbox v-for="r in allRoles" :key="r.id" :value="r.id" class="!mr-0">
            <span class="text-gray-700 font-medium">{{ r.display_name }}</span>
            <span class="text-gray-400 text-xs ml-2">({{ r.name }})</span>
          </el-checkbox>
        </el-checkbox-group>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="roleVisible = false">取消</el-button>
          <el-button type="primary" @click="submitRoles">确定</el-button>
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
