<script setup lang="ts">
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
  <div>
    <el-card>
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center">
          <span>菜单管理</span>
          <el-button type="primary" size="small" @click="openCreate(0)">新增顶级菜单</el-button>
        </div>
      </template>
      <el-table :data="menuTree" v-loading="loading" row-key="id" default-expand-all :tree-props="{ children: 'children' }">
        <el-table-column prop="title" label="菜单名称" width="180" />
        <el-table-column prop="name" label="标识" width="130" />
        <el-table-column label="类型" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.type === 1 ? '' : row.type === 2 ? 'success' : 'warning'">
              {{ row.type === 1 ? '目录' : row.type === 2 ? '菜单' : '按钮' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="icon" label="图标" width="80" />
        <el-table-column prop="path" label="路由" width="160" />
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column label="可见" width="70">
          <template #default="{ row }">{{ row.visible === 1 ? '是' : '否' }}</template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="220">
          <template #default="{ row }">
            <el-button size="small" type="success" @click="openCreate(row.id)" v-if="row.type !== 3">添加</el-button>
            <el-button size="small" type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="formVisible" :title="isEdit ? '编辑菜单' : '新增菜单'" width="500px">
      <el-form label-width="80px">
        <el-form-item label="类型">
          <el-radio-group v-model="form.type">
            <el-radio v-for="t in typeOptions" :key="t.value" :value="t.value">{{ t.label }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="标识"><el-input v-model="form.name" placeholder="英文标识" /></el-form-item>
        <el-form-item label="名称"><el-input v-model="form.title" placeholder="显示名称" /></el-form-item>
        <el-form-item label="图标" v-if="form.type !== 3"><el-input v-model="form.icon" placeholder="Element Plus 图标名" /></el-form-item>
        <el-form-item label="路由" v-if="form.type !== 3"><el-input v-model="form.path" placeholder="/system/xxx" /></el-form-item>
        <el-form-item label="API路径" v-if="form.type === 3"><el-input v-model="form.api_path" placeholder="/api/v1/xxx" /></el-form-item>
        <el-form-item label="API方法" v-if="form.type === 3">
          <el-select v-model="form.api_method" placeholder="选择方法">
            <el-option label="GET" value="GET" />
            <el-option label="POST" value="POST" />
          </el-select>
        </el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
        <el-form-item label="可见" v-if="form.type !== 3">
          <el-switch v-model="form.visible" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>
