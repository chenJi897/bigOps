<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { serviceTreeApi, assetApi } from '../api'

const router = useRouter()
const loading = ref(false)
const treeData = ref<any[]>([])
const currentNode = ref<any>(null)
const assetCounts = ref<Record<number, number>>({})

// 节点下的资产
const assetLoading = ref(false)
const assetList = ref<any[]>([])
const assetTotal = ref(0)
const assetPage = ref(1)

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

async function fetchAssetCounts() {
  try {
    const res: any = await serviceTreeApi.assetCounts()
    assetCounts.value = res.data || {}
  } catch {}
}

async function fetchNodeAssets(nodeId: number, page = 1) {
  assetLoading.value = true
  assetPage.value = page
  try {
    const res: any = await assetApi.list({ service_tree_id: nodeId, recursive: true, page, size: 10 } as any)
    assetList.value = res.data?.list || []
    assetTotal.value = res.data?.total || 0
  } finally {
    assetLoading.value = false
  }
}

function handleNodeClick(data: any) {
  currentNode.value = data
  fetchNodeAssets(data.id)
}

function getNodeCount(node: any): number {
  let count = assetCounts.value[node.id] || 0
  if (node.children && node.children.length) {
    for (const child of node.children) {
      count += getNodeCount(child)
    }
  }
  return count
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
    fetchAssetCounts()
    if (currentNode.value?.id === node.id) {
      currentNode.value = null
      assetList.value = []
      assetTotal.value = 0
    }
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

function goToAssets(nodeId: number) {
  router.push({ path: '/cmdb/assets', query: { service_tree_id: String(nodeId), recursive: 'true' } })
}

onMounted(() => {
  fetchTree()
  fetchAssetCounts()
})
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
            :expand-on-click-node="false"
            @node-click="handleNodeClick"
            v-loading="loading"
          >
            <template #default="{ data }">
              <div class="tree-node">
                <span>
                  {{ data.name }}
                  <el-badge v-if="getNodeCount(data) > 0" :value="getNodeCount(data)" type="info" class="node-badge" />
                </span>
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

      <!-- 右侧：节点详情 + 资产列表 -->
      <el-col :span="16">
        <template v-if="currentNode">
          <!-- 节点详情卡片 -->
          <el-card shadow="never" style="margin-bottom: 16px;">
            <template #header>
              <div style="display: flex; justify-content: space-between; align-items: center;">
                <span>{{ currentNode.name }}</span>
                <el-button size="small" @click="goToAssets(currentNode.id)">
                  <el-icon><Right /></el-icon> 在资产页查看
                </el-button>
              </div>
            </template>
            <el-descriptions :column="3" border size="small">
              <el-descriptions-item label="ID">{{ currentNode.id }}</el-descriptions-item>
              <el-descriptions-item label="编码">{{ currentNode.code || '-' }}</el-descriptions-item>
              <el-descriptions-item label="层级">{{ currentNode.level }}</el-descriptions-item>
              <el-descriptions-item label="排序">{{ currentNode.sort }}</el-descriptions-item>
              <el-descriptions-item label="描述" :span="2">{{ currentNode.description || '-' }}</el-descriptions-item>
            </el-descriptions>
          </el-card>

          <!-- 资产列表 -->
          <el-card shadow="never">
            <template #header><span>关联资产（含子节点） · {{ assetTotal }} 台</span></template>
            <el-table :data="assetList" v-loading="assetLoading" stripe size="small">
              <el-table-column prop="hostname" label="主机名" min-width="130" show-overflow-tooltip />
              <el-table-column prop="ip" label="IP" width="130" />
              <el-table-column label="配置" width="100">
                <template #default="{ row }">{{ row.cpu_cores }}C/{{ (row.memory_mb / 1024).toFixed(0) }}G</template>
              </el-table-column>
              <el-table-column prop="status" label="状态" width="70">
                <template #default="{ row }">
                  <el-tag :type="row.status === 'online' ? 'success' : 'danger'" size="small">{{ row.status }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="source" label="来源" width="70" />
              <el-table-column prop="service_tree_name" label="所属节点" width="120">
                <template #default="{ row }">
                  <el-tooltip v-if="row.service_tree_path" :content="row.service_tree_path" placement="top">
                    <span style="cursor: default;">{{ row.service_tree_name }}</span>
                  </el-tooltip>
                  <span v-else>-</span>
                </template>
              </el-table-column>
              <el-table-column prop="idc" label="区域" width="100" show-overflow-tooltip />
            </el-table>
            <el-pagination
              v-if="assetTotal > 10"
              style="margin-top: 12px; justify-content: flex-end;"
              background layout="total, prev, pager, next"
              :total="assetTotal" :page-size="10" :current-page="assetPage"
              @current-change="(p: number) => fetchNodeAssets(currentNode.id, p)"
              small
            />
            <el-empty v-if="!assetLoading && assetList.length === 0" description="该节点下暂无资产" />
          </el-card>
        </template>

        <el-card v-else shadow="never">
          <el-empty description="请选择左侧服务树节点" />
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
.node-badge { margin-left: 6px; }
.node-badge :deep(.el-badge__content) { font-size: 10px; }
</style>
