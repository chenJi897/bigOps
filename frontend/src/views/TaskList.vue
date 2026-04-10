<script setup lang="ts">
defineOptions({ name: 'TaskList' })
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { taskApi, assetApi } from '../api'

const router = useRouter()
const loading = ref(false)
const tableData = ref<any[]>([])
const total = ref(0)
const query = ref({ page: 1, size: 20, keyword: '', task_type: '' })

const taskTypeMap: Record<string, string> = {
  shell: 'Shell', python: 'Python', file_transfer: '文件分发',
}

// 执行对话框
const execDialogVisible = ref(false)
const execTaskId = ref(0)
const execTaskName = ref('')
const execLoading = ref(false)
const assetQuery = ref({ page: 1, size: 15, keyword: '' })
const assetTableData = ref<any[]>([])
const assetTotal = ref(0)
const assetLoading = ref(false)
const selectedHosts = ref<string[]>([])


// 执行历史对话框
const historyVisible = ref(false)
const historyLoading = ref(false)
const historyData = ref<any[]>([])
const historyTotal = ref(0)
const historyQuery = ref({ task_id: 0, page: 1, size: 10 })

const execStatusMap: Record<string, { label: string; type: string }> = {
  pending: { label: '等待中', type: 'info' },
  running: { label: '执行中', type: 'warning' },
  success: { label: '成功', type: 'success' },
  partial_fail: { label: '部分失败', type: 'warning' },
  failed: { label: '失败', type: 'danger' },
  canceled: { label: '已取消', type: 'info' },
}

async function fetchData() {
  loading.value = true
  try {
    const params: any = { ...query.value }
    Object.keys(params).forEach(k => { if (params[k] === '' || params[k] === null) delete params[k] })
    const res: any = await taskApi.list(params)
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally { loading.value = false }
}

function handleSearch() { query.value.page = 1; fetchData() }
function handleReset() { query.value = { page: 1, size: 20, keyword: '', task_type: '' }; fetchData() }

function openCreate() { router.push('/task/create') }
function openExecutions() { router.push('/task/executions') }
function openEdit(row: any) { router.push('/task/create/' + row.id) }

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确定删除任务「${row.name}」？`, '确认')
  await taskApi.delete(row.id)
  ElMessage.success('删除成功')
  fetchData()
}

// 执行相关
function openExecDialog(row: any) {
  execTaskId.value = row.id
  execTaskName.value = row.name
  selectedHosts.value = []
  assetQuery.value = { page: 1, size: 15, keyword: '' }
  execDialogVisible.value = true
  fetchAssets()
}

async function fetchAssets() {
  assetLoading.value = true
  try {
    const res: any = await assetApi.list({ ...assetQuery.value, status: 'online' })
    assetTableData.value = res.data?.list || []
    assetTotal.value = res.data?.total || 0
  } finally { assetLoading.value = false }
}

function handleAssetSelect(selection: any[]) {
  selectedHosts.value = selection.map((a: any) => a.ip)
}

async function confirmExecute() {
  if (selectedHosts.value.length === 0) {
    ElMessage.warning('请选择目标主机')
    return
  }
  execLoading.value = true
  try {
    const res: any = await taskApi.execute(execTaskId.value, { host_ips: selectedHosts.value })
    ElMessage.success('任务已下发')
    execDialogVisible.value = false
    const execId = res.data?.id
    if (execId) {
      router.push({ path: '/task/executions', query: { new_exec: String(execId) } })
    }
  } catch (e: any) {
    ElMessage.error(e.message || '执行失败')
  } finally { execLoading.value = false }
}

// 执行历史
function openHistory(row: any) {
  historyQuery.value = { task_id: row.id, page: 1, size: 10 }
  historyVisible.value = true
  fetchHistory()
}

async function fetchHistory() {
  historyLoading.value = true
  try {
    const res: any = await taskApi.executions(historyQuery.value)
    historyData.value = res.data?.list || []
    historyTotal.value = res.data?.total || 0
  } finally { historyLoading.value = false }
}

function openExecution(row: any) {
  historyVisible.value = false
  router.push('/task/executions/' + row.id)
}

onMounted(() => { fetchData() })
</script>

<template>
  <div class="p-4 md:p-6 min-h-full flex flex-col">
    <el-card shadow="never" class="border-0 shadow-sm flex-1 flex flex-col">
      <template #header>
        <div class="flex justify-between items-center flex-wrap gap-2">
          <span class="text-base font-medium text-gray-800">任务管理</span>
          <div class="flex items-center gap-2">
            <el-button @click="openExecutions">
              <el-icon class="mr-1"><Document /></el-icon> 执行记录
            </el-button>
            <el-button v-permission="'task:create'" type="primary" @click="openCreate">
              <el-icon class="mr-1"><Plus /></el-icon> 创建任务
            </el-button>
          </div>
        </div>
      </template>

      <div class="flex flex-wrap items-center gap-3 mb-4">
        <el-select v-model="query.task_type" placeholder="任务类型" clearable class="w-32">
          <el-option v-for="(v, k) in taskTypeMap" :key="k" :label="v" :value="k" />
        </el-select>
        <el-input 
          v-model="query.keyword" 
          placeholder="搜索任务名称" 
          clearable 
          class="w-56" 
          @keyup.enter="handleSearch" 
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button type="primary" @click="handleSearch">搜索</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table :data="tableData" v-loading="loading" stripe border class="w-full">
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column prop="name" label="任务名称" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="font-medium text-gray-800">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="120" align="center">
          <template #default="{ row }">
            <el-tag :type="row.task_type === 'shell' ? 'primary' : row.task_type === 'python' ? 'warning' : 'info'" size="small">
              {{ taskTypeMap[row.task_type] || row.task_type }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="script_type" label="脚本类型" width="100" align="center" />
        <el-table-column label="超时(秒)" width="100" prop="timeout" align="center" />
        <el-table-column prop="creator_name" label="创建人" width="120" align="center">
          <template #default="{ row }">
            <span>{{ row.creator_name || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170" align="center" />
        <el-table-column label="操作" width="220" fixed="right" align="center">
          <template #default="{ row }">
            <div class="flex items-center justify-center gap-1">
              <el-button link type="primary" @click="openExecDialog(row)">执行</el-button>
              <el-divider direction="vertical" />
              <el-button link type="primary" @click="openHistory(row)">历史</el-button>
              <el-divider direction="vertical" />
              <el-button v-permission="'task:edit'" link type="primary" @click="openEdit(row)">编辑</el-button>
              <el-divider direction="vertical" />
              <el-button v-permission="'task:delete'" link type="danger" @click="handleDelete(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="total > 0" class="mt-4 flex justify-end">
        <el-pagination 
          background 
          layout="total, prev, pager, next" 
          :total="total" 
          :page-size="query.size" 
          :current-page="query.page" 
          @current-change="(p: number) => { query.page = p; fetchData() }" 
        />
      </div>
    </el-card>

    <!-- 执行对话框：选择目标主机 -->
    <el-dialog v-model="execDialogVisible" :title="'执行任务: ' + execTaskName" width="720px" destroy-on-close align-center>
      <div class="flex items-center gap-2 mb-4">
        <el-input 
          v-model="assetQuery.keyword" 
          placeholder="搜索主机名/IP" 
          clearable 
          class="w-64" 
          @keyup.enter="fetchAssets" 
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button type="primary" @click="fetchAssets">搜索</el-button>
      </div>
      
      <el-table :data="assetTableData" v-loading="assetLoading" stripe border @selection-change="handleAssetSelect" max-height="360" class="w-full border-gray-200 rounded">
        <el-table-column type="selection" width="50" align="center" />
        <el-table-column prop="hostname" label="主机名" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="font-medium text-gray-800">{{ row.hostname }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="IP" width="140" align="center">
          <template #default="{ row }">
            <span class="text-gray-600">{{ row.ip }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="os" label="系统" width="100" align="center" />
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'online' ? 'success' : 'danger'" size="small" effect="plain" round>
              {{ row.status === 'online' ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
      
      <div v-if="assetTotal > 0" class="mt-4 flex justify-end">
        <el-pagination 
          background 
          layout="total, prev, pager, next" 
          :total="assetTotal" 
          :page-size="assetQuery.size" 
          :current-page="assetQuery.page" 
          @current-change="(p: number) => { assetQuery.page = p; fetchAssets() }" 
        />
      </div>
      
      <template #footer>
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-500">已选 <strong class="text-indigo-600">{{ selectedHosts.length }}</strong> 台主机</span>
          <div class="flex gap-2">
            <el-button @click="execDialogVisible = false">取消</el-button>
            <el-button type="primary" :loading="execLoading" @click="confirmExecute">确认执行</el-button>
          </div>
        </div>
      </template>
    </el-dialog>

    <!-- 执行历史对话框 -->
    <el-dialog v-model="historyVisible" title="执行历史" width="760px" destroy-on-close align-center>
      <el-table :data="historyData" v-loading="historyLoading" stripe border @row-click="openExecution" class="w-full cursor-pointer hover:bg-gray-50 transition-colors">
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column label="状态" width="110" align="center">
          <template #default="{ row }">
            <el-tag :type="(execStatusMap[row.status]?.type as any) || ''" size="small" effect="light">
              {{ execStatusMap[row.status]?.label || row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" width="130" align="center">
          <template #default="{ row }">
            <span class="text-gray-700 font-medium">{{ row.success_count }}</span>
            <span class="text-gray-400 mx-1">/</span>
            <span class="text-gray-500">{{ row.total_count }}</span>
            <span class="text-xs text-gray-400 ml-1">成功</span>
          </template>
        </el-table-column>
        <el-table-column prop="operator_name" label="执行人" width="120" align="center">
          <template #default="{ row }">
            <span>{{ row.operator_name || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" min-width="170" align="center" />
      </el-table>
      
      <div v-if="historyTotal > 0" class="mt-4 flex justify-end">
        <el-pagination 
          background 
          layout="total, prev, pager, next" 
          :total="historyTotal" 
          :page-size="historyQuery.size" 
          :current-page="historyQuery.page" 
          @current-change="(p: number) => { historyQuery.page = p; fetchHistory() }" 
        />
      </div>
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
