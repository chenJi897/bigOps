<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { roleApi, menuApi } from '../api'

const roles = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)

// 新增/编辑
const formVisible = ref(false)
const isEdit = ref(false)
const form = ref({ id: 0, name: '', display_name: '', description: '', sort: 0, status: 1 })

// 菜单权限
const menuVisible = ref(false)
const menuRoleId = ref(0)
const menuTree = ref<any[]>([])
const selectedMenus = ref<number[]>([])
const treeRef = ref<any>(null)

async function loadRoles() {
  loading.value = true
  try {
    const res: any = await roleApi.list(page.value, 100)
    roles.value = res.data.list || []
    total.value = res.data.total
  } catch {} finally { loading.value = false }
}

function openCreate() {
  isEdit.value = false
  form.value = { id: 0, name: '', display_name: '', description: '', sort: 0, status: 1 }
  formVisible.value = true
}

function openEdit(row: any) {
  isEdit.value = true
  form.value = { ...row }
  formVisible.value = true
}

async function submitForm() {
  if (!form.value.name || !form.value.display_name) { ElMessage.warning('请填写完整'); return }
  if (isEdit.value) {
    await roleApi.update(form.value.id, form.value)
  } else {
    await roleApi.create(form.value)
  }
  ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
  formVisible.value = false
  loadRoles()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确定删除角色 ${row.display_name}？`, '提示', { type: 'warning' })
  await roleApi.delete(row.id)
  ElMessage.success('删除成功')
  loadRoles()
}

async function openMenuDialog(row: any) {
  menuRoleId.value = row.id
  const [treeRes, roleRes]: any = await Promise.all([
    menuApi.tree(),
    roleApi.getById(row.id),
  ])
  menuTree.value = treeRes.data || []
  const menuIds = (roleRes.data.menus || []).map((m: any) => m.id)
  // 只勾选叶子节点，避免父节点被全选
  selectedMenus.value = filterLeafIds(menuTree.value, menuIds)
  menuVisible.value = true
}

function filterLeafIds(tree: any[], ids: number[]): number[] {
  const leafIds: number[] = []
  for (const node of tree) {
    if (node.children && node.children.length > 0) {
      leafIds.push(...filterLeafIds(node.children, ids))
    } else if (ids.includes(node.id)) {
      leafIds.push(node.id)
    }
  }
  return leafIds
}

async function submitMenus() {
  const checked = treeRef.value.getCheckedKeys()
  const half = treeRef.value.getHalfCheckedKeys()
  await roleApi.setMenus(menuRoleId.value, [...checked, ...half])
  ElMessage.success('菜单权限设置成功')
  menuVisible.value = false
}

onMounted(loadRoles)
</script>

<template>
  <div>
    <el-card>
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center">
          <span>角色管理</span>
          <el-button type="primary" size="small" @click="openCreate">新增角色</el-button>
        </div>
      </template>
      <el-table :data="roles" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="标识" width="120" />
        <el-table-column prop="display_name" label="名称" width="150" />
        <el-table-column prop="description" label="描述" />
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="220">
          <template #default="{ row }">
            <el-button size="small" @click="openMenuDialog(row)">菜单</el-button>
            <el-button size="small" type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)" :disabled="row.name === 'admin'">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增/编辑角色 -->
    <el-dialog v-model="formVisible" :title="isEdit ? '编辑角色' : '新增角色'" width="450px">
      <el-form label-width="70px">
        <el-form-item label="标识"><el-input v-model="form.name" :disabled="isEdit" placeholder="英文标识如 viewer" /></el-form-item>
        <el-form-item label="名称"><el-input v-model="form.display_name" placeholder="显示名称" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

    <!-- 菜单权限 -->
    <el-dialog v-model="menuVisible" title="菜单权限" width="400px">
      <el-tree ref="treeRef" :data="menuTree" show-checkbox node-key="id" :default-checked-keys="selectedMenus"
        :props="{ label: 'title', children: 'children' }" default-expand-all />
      <template #footer>
        <el-button @click="menuVisible = false">取消</el-button>
        <el-button type="primary" @click="submitMenus">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>
