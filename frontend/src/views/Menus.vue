<script setup lang="ts">
defineOptions({ name: 'Menus' })
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { menuApi } from '../api'

const menuTree = ref<any[]>([])
const loading = ref(false)

// 新增/编辑
const formVisible = ref(false)
const isEdit = ref(false)
const form = ref<any>({ parent_id: 0, name: '', title: '', icon: '', path: '', component: '', api_path: '', api_method: '', type: 1, sort: 0, visible: 1 })

const typeOptions = [
  { label: '目录', value: 1 },
  { label: '菜单', value: 2 },
  { label: '按钮/API', value: 3 },
]

// 可用页面组件列表
const componentOptions = [
  'TicketList', 'TicketCreate', 'TicketDetail', 'RequestTemplates',
  'Dashboard', 'DashboardWorkbench', 'DashboardOverview',
  'Users', 'Roles', 'Menus', 'AuditLogs', 'Departments',
  'ServiceTree', 'CloudAccounts', 'Assets',
  'TicketTypes',
  'ApprovalInbox', 'ApprovalPolicies', 'NotificationConsole', 'UserSettings',
  'TaskList', 'TaskCreate', 'TaskExecution', 'AgentList',
  'MonitorDashboard', 'AlertRules', 'AgentDetail', 'AlertEvents', 'AlertSilences', 'MonitorDatasources', 'MonitorQuery', 'OnCallSchedules',
  'CicdProjects', 'CicdPipelines', 'CicdRuns',
  'NotifyGroups',
]

async function loadMenus() {
  loading.value = true
  try {
    const res: any = await menuApi.tree()
    menuTree.value = res.data || []
  } catch {} finally { loading.value = false }
}

function openCreate(parentId = 0) {
  isEdit.value = false
  form.value = { parent_id: parentId, name: '', title: '', icon: '', path: '', component: '', api_path: '', api_method: '', type: parentId === 0 ? 1 : 2, sort: 0, visible: 1 }
  formVisible.value = true
}

function openEdit(row: any) {
  isEdit.value = true
  form.value = { ...row }
  formVisible.value = true
}

async function submitForm() {
  if (!form.value.name || !form.value.title) { ElMessage.warning('请填写标识和名称'); return }
  if (isEdit.value) {
    await menuApi.update(form.value.id, form.value)
  } else {
    await menuApi.create(form.value)
  }
  ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
  formVisible.value = false
  loadMenus()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确定删除菜单 ${row.title}？`, '提示', { type: 'warning' })
  await menuApi.delete(row.id)
  ElMessage.success('删除成功')
  loadMenus()
}

onMounted(loadMenus)
</script>

<template>
  <div class="p-4 md:p-6 min-h-full flex flex-col">
    <el-card shadow="never" class="border-0 shadow-sm flex-1 flex flex-col">
      <template #header>
        <div class="flex justify-between items-center">
          <span class="text-base font-medium text-gray-800">菜单管理</span>
          <el-button v-permission="'menu:create'" type="primary" @click="openCreate(0)">
            <el-icon class="mr-1"><Plus /></el-icon> 新增顶级菜单
          </el-button>
        </div>
      </template>
      <el-table :data="menuTree" v-loading="loading" row-key="id" default-expand-all :tree-props="{ children: 'children' }" class="w-full">
        <el-table-column prop="title" label="菜单名称" min-width="180">
          <template #default="{ row }">
            <span class="font-medium text-gray-800">{{ row.title }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="标识" width="160">
          <template #default="{ row }">
            <span class="font-mono text-gray-500 bg-gray-50 px-2 py-1 rounded text-xs">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.type === 1 ? 'info' : row.type === 2 ? 'success' : 'warning'" effect="plain" round>
              {{ row.type === 1 ? '目录' : row.type === 2 ? '菜单' : '按钮' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="icon" label="图标" width="100" align="center">
          <template #default="{ row }">
            <el-icon v-if="row.icon" class="text-gray-500 text-lg align-middle"><component :is="row.icon" /></el-icon>
            <span v-else class="text-gray-300">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="path" label="路由" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="text-gray-500">{{ row.path || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="component" label="组件" width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="text-gray-500">{{ row.component || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="80" align="center" />
        <el-table-column label="可见" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.visible === 1 ? 'success' : 'info'" effect="light">
              {{ row.visible === 1 ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" min-width="240" align="center">
          <template #default="{ row }">
            <div class="flex items-center justify-center gap-1">
              <el-button v-permission="'menu:create'" link type="success" @click="openCreate(row.id)" v-if="row.type !== 3">添加子项</el-button>
              <el-divider direction="vertical" v-if="row.type !== 3" />
              <el-button v-permission="'menu:edit'" link type="primary" @click="openEdit(row)">编辑</el-button>
              <el-divider direction="vertical" />
              <el-button v-permission="'menu:delete'" link type="danger" @click="handleDelete(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="formVisible" :title="isEdit ? '编辑菜单' : '新增菜单'" width="540px" destroy-on-close align-center>
      <el-form :model="form" label-width="100px" class="pr-6">
        <el-form-item label="类型">
          <el-radio-group v-model="form.type">
            <el-radio v-for="t in typeOptions" :key="t.value" :value="t.value">{{ t.label }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="标识" required>
          <el-input v-model="form.name" placeholder="英文标识" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.title" placeholder="显示名称" />
        </el-form-item>
        <el-form-item label="图标" v-if="form.type !== 3">
          <el-input v-model="form.icon" placeholder="Element Plus 图标名，如 User" />
        </el-form-item>
        <el-form-item label="路由" v-if="form.type !== 3">
          <el-input v-model="form.path" placeholder="/system/users" />
        </el-form-item>
        <el-form-item label="页面组件" v-if="form.type === 2">
          <el-select v-model="form.component" placeholder="选择对应的前端页面" clearable class="w-full">
            <el-option v-for="c in componentOptions" :key="c" :label="c" :value="c" />
          </el-select>
        </el-form-item>
        <el-form-item label="API路径" v-if="form.type === 2 || form.type === 3">
          <el-input v-model="form.api_path" placeholder="/api/v1/xxx" />
        </el-form-item>
        <el-form-item label="API方法" v-if="form.type === 2 || form.type === 3">
          <el-select v-model="form.api_method" placeholder="选择方法" clearable class="w-full">
            <el-option label="GET" value="GET" />
            <el-option label="POST" value="POST" />
            <el-option label="PUT" value="PUT" />
            <el-option label="DELETE" value="DELETE" />
          </el-select>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" class="!w-32" />
        </el-form-item>
        <el-form-item label="可见" v-if="form.type !== 3">
          <el-switch v-model="form.visible" :active-value="1" :inactive-value="0" inline-prompt active-text="是" inactive-text="否" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="formVisible = false">取消</el-button>
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
