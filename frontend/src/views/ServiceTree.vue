<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { serviceTreeApi } from '../api'

const loading = ref(false)
const treeData = ref<any[]>([])
const currentNode = ref<any>(null)

// 表单
const dialogVisible = ref(false)
const dialogTitle = ref('新增节点')
const form = ref({ name: '', code: '', parent_id: 0, sort: 0, description: '', owner_id: 0 })

async function fetchTree() {
  loading.value = true
  try {
    const res: any = await serviceTreeApi.tree()
    treeData.value = res.data || []
  } finally {
    loading.value = false
  }
}

function handleNodeClick(data: any) {
  currentNode.value = data
}

function handleAdd(parentId = 0) {
  dialogTitle.value = '新增节点'
  form.value = { name: '', code: '', parent_id: parentId, sort: 0, description: '', owner_id: 0 }
  dialogVisible.value = true
}

function handleEdit(node: any) {
  dialogTitle.value = '编辑节点'
  form.value = { name: node.name, code: node.code, parent_id: node.parent_id, sort: node.sort, description: node.description || '', owner_id: node.owner_id || 0 }
  currentNode.value = node
  dialogVisible.value = true
}

async function handleDelete(node: any) {
  try {
    await ElMessageBox.confirm(`确定删除 "${node.name}" 吗？`, '提示', { type: 'warning' })
    await serviceTreeApi.delete(node.id)
    ElMessage.success('删除成功')
    fetchTree()
    if (currentNode.value?.id === node.id) currentNode.value = null
  } catch {}
}

async function submitForm() {
  if (!form.value.name) { ElMessage.warning('请输入节点名称'); return }
  try {
    if (dialogTitle.value === '编辑节点' && currentNode.value) {
      await serviceTreeApi.update(currentNode.value.id, form.value)
      ElMessage.success('更新成功')
    } else {
      await serviceTreeApi.create(form.value)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchTree()
  } catch {}
}

onMounted(fetchTree)
</script>

<template>
  <div class="page">
    <el-row :gutter="16">
      <!-- 左侧树 -->
      <el-col :span="8">
        <el-card shadow="never">
          <template #header>
            <div style="display: flex; justify-content: space-between; align-items: center;">
              <span>服务树</span>
              <el-button type="primary" size="small" @click="handleAdd(0)">
                <el-icon><Plus /></el-icon> 新增根节点
              </el-button>
            </div>
          </template>
          <el-tree
            :data="treeData"
            node-key="id"
            :props="{ children: 'children', label: 'name' }"
            default-expand-all
            highlight-current
            @node-click="handleNodeClick"
            v-loading="loading"
          >
            <template #default="{ data }">
              <div class="tree-node">
                <span>{{ data.name }}</span>
                <span class="tree-actions">
                  <el-button link size="small" @click.stop="handleAdd(data.id)"><el-icon><Plus /></el-icon></el-button>
                  <el-button link size="small" @click.stop="handleEdit(data)"><el-icon><Edit /></el-icon></el-button>
                  <el-button link size="small" type="danger" @click.stop="handleDelete(data)"><el-icon><Delete /></el-icon></el-button>
                </span>
              </div>
            </template>
          </el-tree>
        </el-card>
      </el-col>

      <!-- 右侧详情 -->
      <el-col :span="16">
        <el-card shadow="never">
          <template #header><span>{{ currentNode ? currentNode.name : '节点详情' }}</span></template>
          <template v-if="currentNode">
            <el-descriptions :column="2" border>
              <el-descriptions-item label="ID">{{ currentNode.id }}</el-descriptions-item>
              <el-descriptions-item label="名称">{{ currentNode.name }}</el-descriptions-item>
              <el-descriptions-item label="编码">{{ currentNode.code }}</el-descriptions-item>
              <el-descriptions-item label="层级">{{ currentNode.level }}</el-descriptions-item>
              <el-descriptions-item label="排序">{{ currentNode.sort }}</el-descriptions-item>
              <el-descriptions-item label="描述">{{ currentNode.description || '-' }}</el-descriptions-item>
            </el-descriptions>
          </template>
          <template v-else>
            <el-empty description="请选择左侧服务树节点" />
          </template>
        </el-card>
      </el-col>
    </el-row>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称"><el-input v-model="form.name" placeholder="节点名称" /></el-form-item>
        <el-form-item label="编码"><el-input v-model="form.code" placeholder="唯一编码(可选)" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
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
.tree-node { display: flex; justify-content: space-between; align-items: center; width: 100%; padding-right: 8px; }
.tree-actions { display: none; }
.tree-node:hover .tree-actions { display: inline-flex; }
</style>
