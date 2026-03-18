<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { userApi, roleApi } from '../api'

const users = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const loading = ref(false)

// 角色分配
const roleVisible = ref(false)
const roleUserId = ref(0)
const allRoles = ref<any[]>([])
const selectedRoles = ref<number[]>([])

async function loadUsers() {
  loading.value = true
  try {
    const res: any = await userApi.list(page.value, size.value)
    users.value = res.data.list || []
    total.value = res.data.total
  } catch {} finally { loading.value = false }
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

onMounted(loadUsers)
</script>

<template>
  <div>
    <el-card>
      <template #header><span>用户管理</span></template>
      <el-table :data="users" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="username" label="用户名" width="120" />
        <el-table-column prop="email" label="邮箱" width="180" />
        <el-table-column prop="real_name" label="姓名" width="100" />
        <el-table-column prop="phone" label="手机号" width="130" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" fixed="right" width="220">
          <template #default="{ row }">
            <el-button size="small" @click="openRoleDialog(row)">角色</el-button>
            <el-button size="small" :type="row.status === 1 ? 'warning' : 'success'" @click="toggleStatus(row)">{{ row.status === 1 ? '禁用' : '启用' }}</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)" :disabled="row.id === 1">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination style="margin-top:16px;justify-content:flex-end" v-model:current-page="page" v-model:page-size="size"
        :total="total" layout="total, prev, pager, next" @current-change="loadUsers" />
    </el-card>

    <el-dialog v-model="roleVisible" title="分配角色" width="400px">
      <el-checkbox-group v-model="selectedRoles">
        <el-checkbox v-for="r in allRoles" :key="r.id" :value="r.id">{{ r.display_name }}</el-checkbox>
      </el-checkbox-group>
      <template #footer>
        <el-button @click="roleVisible = false">取消</el-button>
        <el-button type="primary" @click="submitRoles">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>
