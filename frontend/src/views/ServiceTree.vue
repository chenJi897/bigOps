<script setup lang="ts">
defineOptions({ name: 'ServiceTree' })
import { ref, onMounted } from 'vue'
import { Plus, Edit, Delete, Right, Guide, Folder } from '@element-plus/icons-vue'
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
  <div class="p-4 sm:p-6 lg:p-8 space-y-4 sm:space-y-6">
    <div class="flex flex-col lg:flex-row gap-6">
      <!-- 左侧树 -->
      <div class="w-full lg:w-1/3 xl:w-1/4 flex-shrink-0">
        <el-card shadow="never" class="border-0 shadow-sm ring-1 ring-slate-200/60 rounded-xl h-full flex flex-col bg-white">
          <template #header>
            <div class="flex justify-between items-center px-1">
              <h2 class="text-lg font-semibold text-slate-800 tracking-tight flex items-center gap-2">
                <el-icon class="text-indigo-500"><Folder /></el-icon>
                服务树
              </h2>
              <el-button type="primary" @click="handleAdd(0)" class="shadow-sm !rounded-md" size="default">
                <template #icon><el-icon><Plus /></el-icon></template>
                新增根节点
              </el-button>
            </div>
          </template>
          <div class="overflow-y-auto max-h-[calc(100vh-220px)] pr-2 -mr-2 custom-scrollbar">
            <el-tree
              :data="treeData"
              node-key="id"
              :props="{ children: 'children', label: 'name' }"
              default-expand-all
              highlight-current
              :expand-on-click-node="false"
              @node-click="handleNodeClick"
              v-loading="loading"
              class="bg-transparent !text-slate-700"
            >
              <template #default="{ data }">
              <div class="flex justify-between items-center w-full pr-2 py-1.5 group hover:bg-indigo-50/50 rounded-md transition-all duration-200 cursor-pointer">
                <span class="flex items-center gap-2 text-slate-700 font-medium group-hover:text-indigo-600 transition-colors">
                  <el-icon class="text-slate-400 group-hover:text-indigo-400"><Folder /></el-icon>
                  {{ data.name }}
                  <el-badge v-if="getNodeCount(data) > 0" :value="getNodeCount(data)" type="primary" class="node-badge transform scale-90 ml-1 shadow-sm" />
                </span>
                <span class="opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
                  <el-tooltip content="添加子节点" placement="top" :show-after="300">
                    <el-button link type="primary" @click.stop="handleAdd(data.id)" class="!p-1 hover:bg-indigo-100 rounded text-indigo-500">
                      <el-icon class="text-sm"><Plus /></el-icon>
                    </el-button>
                  </el-tooltip>
                  <el-tooltip content="编辑" placement="top" :show-after="300">
                    <el-button link type="primary" @click.stop="handleEdit(data)" class="!p-1 hover:bg-indigo-100 rounded text-indigo-500">
                      <el-icon class="text-sm"><Edit /></el-icon>
                    </el-button>
                  </el-tooltip>
                  <el-tooltip content="删除" placement="top" :show-after="300">
                    <el-button link type="danger" @click.stop="handleDelete(data)" class="!p-1 hover:bg-red-100 rounded text-red-500">
                      <el-icon class="text-sm"><Delete /></el-icon>
                    </el-button>
                  </el-tooltip>
                </span>
              </div>
              </template>
            </el-tree>
          </div>
        </el-card>
      </div>

      <!-- 右侧：节点详情 + 资产列表 -->
      <div class="w-full lg:w-2/3 xl:w-3/4 flex flex-col gap-6">
        <template v-if="currentNode">
          <!-- 节点详情卡片 -->
          <el-card shadow="never" class="border-0 shadow-sm ring-1 ring-slate-200/60 rounded-xl bg-white">
            <template #header>
              <div class="flex justify-between items-center px-1">
                <h3 class="text-lg font-semibold text-slate-800 flex items-center gap-2">
                  <span class="w-1.5 h-4 bg-indigo-500 rounded-full"></span>
                  {{ currentNode.name }}
                </h3>
                <el-button @click="goToAssets(currentNode.id)" class="!rounded-md hover:text-indigo-600 hover:border-indigo-300 transition-colors">
                  <template #icon><el-icon><Right /></el-icon></template>
                  在资产页查看
                </el-button>
              </div>
            </template>
            <el-descriptions :column="3" border class="shadow-sm rounded-lg overflow-hidden descriptions-modern">
              <el-descriptions-item label="ID">{{ currentNode.id }}</el-descriptions-item>
              <el-descriptions-item label="编码"><span class="font-mono text-slate-600 bg-slate-50 px-2 py-0.5 rounded border border-slate-100">{{ currentNode.code || '-' }}</span></el-descriptions-item>
              <el-descriptions-item label="层级"><el-tag size="small" type="info" effect="plain" class="!rounded-md !border-slate-200 !text-slate-500 !bg-slate-50">{{ currentNode.level }}</el-tag></el-descriptions-item>
              <el-descriptions-item label="排序">{{ currentNode.sort }}</el-descriptions-item>
              <el-descriptions-item label="描述" :span="2"><span class="text-slate-600">{{ currentNode.description || '-' }}</span></el-descriptions-item>
            </el-descriptions>
          </el-card>

          <!-- 资产列表 -->
          <el-card shadow="never" class="border-0 shadow-sm ring-1 ring-slate-200/60 rounded-xl flex-1 bg-white flex flex-col">
            <template #header>
              <div class="flex items-center px-1">
                <span class="font-medium text-slate-700 flex items-center gap-2">
                  关联资产 <span class="text-slate-400 text-sm font-normal">（含子节点）</span>
                </span>
                <el-tag size="small" type="primary" effect="light" class="ml-3 !rounded-full !px-3 font-medium">{{ assetTotal }} 台</el-tag>
              </div>
            </template>
            <el-table :data="assetList" v-loading="assetLoading" stripe :border="false" class="w-full border border-slate-100 rounded-xl overflow-hidden shadow-sm table-modern">
              <el-table-column prop="hostname" label="主机名" min-width="150" show-overflow-tooltip>
                 <template #default="{ row }">
                   <span class="font-semibold text-slate-800 group-hover:text-indigo-600 transition-colors">{{ row.hostname }}</span>
                 </template>
              </el-table-column>
              <el-table-column prop="ip" label="IP" width="130">
                 <template #default="{ row }">
                   <span class="font-mono text-sm text-slate-600 bg-slate-50 px-1.5 py-0.5 rounded border border-slate-200/60">{{ row.ip || '-' }}</span>
                 </template>
              </el-table-column>
              <el-table-column label="配置" width="120">
                <template #default="{ row }">
                  <span class="text-sm font-medium text-slate-600 bg-slate-50 px-2 py-0.5 rounded border border-slate-100">{{ row.cpu_cores }}C / {{ (row.memory_mb / 1024).toFixed(0) }}G</span>
                </template>
              </el-table-column>
              <el-table-column prop="status" label="状态" width="100">
                <template #default="{ row }">
                  <div class="flex items-center gap-1.5 bg-slate-50 w-fit px-2 py-0.5 rounded-full border border-slate-100">
                    <span :class="['w-1.5 h-1.5 rounded-full', row.status === 'online' ? 'bg-emerald-500 shadow-[0_0_4px_rgba(16,185,129,0.4)]' : 'bg-red-500 shadow-[0_0_4px_rgba(239,68,68,0.4)]']"></span>
                    <span :class="row.status === 'online' ? 'text-emerald-700' : 'text-red-700'" class="text-xs font-medium">{{ row.status === 'online' ? '在线' : '离线' }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="source" label="来源" width="100">
                <template #default="{ row }">
                  <el-tag size="small" type="info" effect="plain" class="!rounded-md !border-slate-200 !text-slate-500 !bg-slate-50">{{ row.source }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="service_tree_name" label="所属节点" width="150" show-overflow-tooltip>
                <template #default="{ row }">
                  <el-tooltip v-if="row.service_tree_path" :content="row.service_tree_path" placement="top" :show-after="300">
                    <span class="text-sm text-slate-700 font-medium cursor-help flex items-center gap-1">
                      <el-icon class="text-indigo-400"><Folder /></el-icon>
                      <span class="truncate">{{ row.service_tree_name }}</span>
                    </span>
                  </el-tooltip>
                  <span v-else class="text-slate-400 text-sm">-</span>
                </template>
              </el-table-column>
              <el-table-column prop="idc" label="区域" width="120" show-overflow-tooltip>
                <template #default="{ row }"><span class="text-sm text-slate-600">{{ row.idc || '-' }}</span></template>
              </el-table-column>
            </el-table>
            
            <div class="flex justify-end pt-5 pb-2">
              <el-pagination
                v-if="assetTotal > 0"
                background 
                layout="total, prev, pager, next"
                :total="assetTotal" 
                :page-size="10" 
                :current-page="assetPage"
                @current-change="(p: number) => fetchNodeAssets(currentNode.id, p)"
              />
            </div>
            <el-empty v-if="!assetLoading && assetList.length === 0" description="该节点下暂无关联资产" :image-size="80" class="py-12 opacity-80" />
          </el-card>
        </template>

        <el-card v-else shadow="never" class="border-0 shadow-sm ring-1 ring-slate-200/60 rounded-xl h-full flex flex-col items-center justify-center min-h-[500px] bg-slate-50/30">
          <el-empty description="请选择左侧服务树节点以查看详情" :image-size="120" class="opacity-70">
            <template #image>
              <div class="w-32 h-32 bg-white rounded-full shadow-sm flex items-center justify-center border border-slate-100 mx-auto mb-4">
                <el-icon class="text-indigo-300 text-6xl"><Guide /></el-icon>
              </div>
            </template>
          </el-empty>
        </el-card>
      </div>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="540px" class="rounded-xl overflow-hidden custom-dialog">
      <div class="pt-4 px-2">
        <el-form :model="form" label-width="85px" label-position="right" class="space-y-4">
          <el-form-item label="名称"><el-input v-model="form.name" placeholder="请输入节点名称" class="!rounded-md" /></el-form-item>
          <el-form-item label="编码"><el-input v-model="form.code" placeholder="唯一编码 (可选)" class="!rounded-md" /></el-form-item>
          <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" class="!w-full !rounded-md" controls-position="right" /></el-form-item>
          <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="3" placeholder="添加备注描述..." class="!rounded-md" /></el-form-item>
        </el-form>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3 pt-4 border-t border-slate-100">
          <el-button @click="dialogVisible = false" class="px-6 !rounded-md">取消</el-button>
          <el-button type="primary" @click="submitForm" class="px-6 !rounded-md">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
:deep(.el-tree-node__content) {
  @apply h-auto py-0.5;
}
:deep(.el-tree-node.is-current > .el-tree-node__content) {
  @apply bg-indigo-50/50 rounded-md;
}
:deep(.el-tree-node.is-current > .el-tree-node__content .el-tree-node__label) {
  @apply font-semibold text-indigo-700;
}
:deep(.el-table__row) {
  @apply hover:bg-indigo-50/50 transition-colors duration-200;
}
:deep(.table-modern th.el-table__cell) {
  @apply bg-slate-50/80 text-slate-600 font-medium border-b border-slate-200;
}
:deep(.descriptions-modern .el-descriptions__label) {
  @apply bg-slate-50/80 w-28 text-slate-600 font-medium;
}
:deep(.custom-dialog .el-dialog__header) {
  @apply border-b border-slate-100 pb-4 mb-0 mr-0;
}
:deep(.custom-dialog .el-dialog__title) {
  @apply font-semibold text-slate-800;
}

/* Custom scrollbar for tree */
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #e2e8f0;
  border-radius: 4px;
}
.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: #cbd5e1;
}
</style>
