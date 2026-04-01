<script setup lang="ts">
defineOptions({ name: 'Roles' })
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
  <div class="p-4 md:p-6 min-h-full flex flex-col">
    <el-card shadow="never" class="border-0 shadow-sm flex-1 flex flex-col">
      <template #header>
        <div class="flex justify-between items-center">
          <span class="text-base font-medium text-gray-800">角色管理</span>
          <el-button type="primary" @click="openCreate">
            <el-icon class="mr-1"><Plus /></el-icon> 新增角色
          </el-button>
        </div>
      </template>

      <el-table :data="roles" v-loading="loading" stripe border class="w-full">
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column prop="name" label="标识" width="160">
          <template #default="{ row }">
            <span class="font-mono text-gray-600 bg-gray-100 px-2 py-1 rounded text-xs">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="display_name" label="名称" width="180">
          <template #default="{ row }">
            <span class="font-medium text-gray-800">{{ row.display_name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="text-gray-500">{{ row.description || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="100" align="center" />
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small" effect="plain" round>
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" min-width="240" align="center">
          <template #default="{ row }">
            <div class="flex items-center justify-center gap-1">
              <el-button link type="primary" @click="openMenuDialog(row)">菜单权限</el-button>
              <el-divider direction="vertical" />
              <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
              <el-divider direction="vertical" />
              <el-button link type="danger" @click="handleDelete(row)" :disabled="row.name === 'admin'">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增/编辑角色 -->
    <el-dialog v-model="formVisible" :title="isEdit ? '编辑角色' : '新增角色'" width="500px" destroy-on-close align-center>
      <el-form label-width="90px" class="pr-6">
        <el-form-item label="标识" required>
          <el-input v-model="form.name" :disabled="isEdit" placeholder="英文标识，如 viewer" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.display_name" placeholder="显示名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="角色描述" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" class="!w-32" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="formVisible = false">取消</el-button>
          <el-button type="primary" @click="submitForm">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 菜单权限 -->
    <el-dialog v-model="menuVisible" title="菜单权限" width="460px" destroy-on-close align-center>
      <div class="bg-gray-50 border border-gray-200 rounded-lg p-3 max-h-[60vh] overflow-y-auto">
        <el-tree
          ref="treeRef"
          :data="menuTree"
          show-checkbox
          node-key="id"
          :default-checked-keys="selectedMenus"
          :props="{ label: 'title', children: 'children' }"
          default-expand-all
          class="!bg-transparent"
        />
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="menuVisible = false">取消</el-button>
          <el-button type="primary" @click="submitMenus">确定</el-button>
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
