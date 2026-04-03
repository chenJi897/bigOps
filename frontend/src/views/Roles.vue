<script setup lang="ts">
defineOptions({ name: 'Roles' })
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { roleApi, menuApi } from '../api'

const roles = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)

// 新增
const createVisible = ref(false)
const createForm = ref({ name: '', display_name: '', description: '', sort: 0, status: 1 })

// 编辑抽屉（基本信息 + 菜单权限）
const drawerVisible = ref(false)
const drawerTab = ref('info')
const editId = ref(0)
const editForm = ref({ id: 0, name: '', display_name: '', description: '', sort: 0, status: 1 })
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
  createForm.value = { name: '', display_name: '', description: '', sort: 0, status: 1 }
  createVisible.value = true
}

async function submitCreate() {
  if (!createForm.value.name || !createForm.value.display_name) { ElMessage.warning('请填写完整'); return }
  await roleApi.create(createForm.value)
  ElMessage.success('创建成功')
  createVisible.value = false
  loadRoles()
}

async function openDrawer(row: any, tab = 'info') {
  editId.value = row.id
  editForm.value = { ...row }
  drawerTab.value = tab

  const [treeRes, roleRes]: any = await Promise.all([menuApi.tree(), roleApi.getById(row.id)])
  menuTree.value = treeRes.data || []
  const menuIds = (roleRes.data.menus || []).map((m: any) => m.id)
  selectedMenus.value = filterLeafIds(menuTree.value, menuIds)
  drawerVisible.value = true
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

async function submitDrawer() {
  if (!editForm.value.name || !editForm.value.display_name) { ElMessage.warning('请填写完整'); return }
  // 保存基本信息
  await roleApi.update(editForm.value.id, editForm.value)
  // 保存菜单权限
  const checked = treeRef.value?.getCheckedKeys() || []
  const half = treeRef.value?.getHalfCheckedKeys() || []
  await roleApi.setMenus(editId.value, [...checked, ...half])
  ElMessage.success('保存成功')
  drawerVisible.value = false
  loadRoles()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确定删除角色 ${row.display_name}？`, '提示', { type: 'warning' })
  await roleApi.delete(row.id)
  ElMessage.success('删除成功')
  loadRoles()
}

onMounted(loadRoles)
</script>

<template>
  <div class="p-4 md:p-6 min-h-full flex flex-col">
    <el-card shadow="never" class="border-0 shadow-sm flex-1 flex flex-col">
      <template #header>
        <div class="flex justify-between items-center">
          <span class="text-base font-medium text-gray-800">角色管理</span>
          <el-button v-permission="'role:create'" type="primary" @click="openCreate">
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
          <template #default="{ row }"><span class="text-gray-500">{{ row.description || '-' }}</span></template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="100" align="center" />
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small" effect="plain" round>{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="140" align="center">
          <template #default="{ row }">
            <div class="flex items-center justify-center gap-1">
              <el-button v-permission="'role:edit'" link type="primary" @click="openDrawer(row)">编辑</el-button>
              <el-divider direction="vertical" />
              <el-button v-permission="'role:delete'" link type="danger" @click="handleDelete(row)" :disabled="row.name === 'admin'">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增角色 -->
    <el-dialog v-model="createVisible" title="新增角色" width="500px" destroy-on-close align-center>
      <el-form label-width="90px" class="pr-6">
        <el-form-item label="标识" required>
          <el-input v-model="createForm.name" placeholder="英文标识，如 viewer" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="createForm.display_name" placeholder="显示名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="createForm.description" type="textarea" :rows="3" placeholder="角色描述" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="createForm.sort" :min="0" class="!w-32" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="createVisible = false">取消</el-button>
          <el-button type="primary" @click="submitCreate">创建</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 编辑角色抽屉（基本信息 + 菜单权限） -->
    <el-drawer v-model="drawerVisible" :title="`编辑角色 — ${editForm.display_name}`" size="520px" destroy-on-close>
      <el-tabs v-model="drawerTab" class="px-1">
        <el-tab-pane label="基本信息" name="info">
          <el-form label-position="top" class="mt-2">
            <el-form-item label="标识">
              <el-input v-model="editForm.name" disabled />
            </el-form-item>
            <el-form-item label="名称" required>
              <el-input v-model="editForm.display_name" placeholder="显示名称" />
            </el-form-item>
            <el-form-item label="描述">
              <el-input v-model="editForm.description" type="textarea" :rows="3" placeholder="角色描述" />
            </el-form-item>
            <div class="grid grid-cols-2 gap-4">
              <el-form-item label="排序">
                <el-input-number v-model="editForm.sort" :min="0" class="!w-full" />
              </el-form-item>
              <el-form-item label="状态">
                <el-switch v-model="editForm.status" :active-value="1" :inactive-value="0" active-text="启用" inactive-text="禁用" />
              </el-form-item>
            </div>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="菜单与权限" name="menus">
          <div class="mt-2 bg-gray-50 border border-gray-200 rounded-lg p-3 max-h-[65vh] overflow-y-auto">
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
          <div class="text-xs text-gray-400 mt-2">勾选菜单和按钮权限，保存后生效。按钮权限（type=3）控制页面内操作按钮的显示。</div>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" @click="submitDrawer">保存</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
:deep(.el-card__body) { flex: 1; display: flex; flex-direction: column; }
:deep(.el-table) { flex: 1; }
</style>
