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
    if (execId) router.push('/task/execution/' + execId)
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
  router.push('/task/execution/' + row.id)
}

onMounted(() => { fetchData() })
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>任务管理</span>
          <el-button type="primary" @click="openCreate"><el-icon><Plus /></el-icon> 创建任务</el-button>
        </div>
      </template>

      <el-form :inline="true" @submit.prevent="handleSearch" style="margin-bottom: 12px;">
        <el-form-item>
          <el-select v-model="query.task_type" placeholder="任务类型" clearable style="width: 120px;">
            <el-option v-for="(v, k) in taskTypeMap" :key="k" :label="v" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-input v-model="query.keyword" placeholder="搜索任务名称" clearable style="width: 200px;" @keyup.enter="handleSearch" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="任务名称" min-width="180" show-overflow-tooltip />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">{{ taskTypeMap[row.task_type] || row.task_type }}</template>
        </el-table-column>
        <el-table-column prop="script_type" label="脚本类型" width="90" />
        <el-table-column label="超时(秒)" width="90" prop="timeout" />
        <el-table-column prop="creator_name" label="创建人" width="90">
          <template #default="{ row }">{{ row.creator_name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170" />
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openExecDialog(row)">执行</el-button>
            <el-button link type="primary" @click="openHistory(row)">历史</el-button>
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination v-if="total > 0" style="margin-top: 16px; justify-content: flex-end;" background layout="total, prev, pager, next" :total="total" :page-size="query.size" :current-page="query.page" @current-change="(p: number) => { query.page = p; fetchData() }" />
    </el-card>

    <!-- 执行对话框：选择目标主机 -->
    <el-dialog v-model="execDialogVisible" :title="'执行任务: ' + execTaskName" width="720px" destroy-on-close>
      <el-form :inline="true" @submit.prevent="fetchAssets" style="margin-bottom: 12px;">
        <el-form-item>
          <el-input v-model="assetQuery.keyword" placeholder="搜索主机名/IP" clearable style="width: 200px;" @keyup.enter="fetchAssets" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchAssets">搜索</el-button>
        </el-form-item>
      </el-form>
      <el-table :data="assetTableData" v-loading="assetLoading" stripe border @selection-change="handleAssetSelect" max-height="360">
        <el-table-column type="selection" width="40" />
        <el-table-column prop="hostname" label="主机名" min-width="150" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP" width="140" />
        <el-table-column prop="os" label="系统" width="100" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 'online' ? 'success' : 'danger'" size="small">{{ row.status === 'online' ? '在线' : '离线' }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination v-if="assetTotal > 0" style="margin-top: 12px; justify-content: flex-end;" background layout="total, prev, pager, next" :total="assetTotal" :page-size="assetQuery.size" :current-page="assetQuery.page" @current-change="(p: number) => { assetQuery.page = p; fetchAssets() }" />
      <template #footer>
        <span style="margin-right: 12px; color: #909399;">已选 {{ selectedHosts.length }} 台主机</span>
        <el-button @click="execDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="execLoading" @click="confirmExecute">确认执行</el-button>
      </template>
    </el-dialog>

    <!-- 执行历史对话框 -->
    <el-dialog v-model="historyVisible" title="执行历史" width="700px" destroy-on-close>
      <el-table :data="historyData" v-loading="historyLoading" stripe border @row-click="openExecution" style="cursor: pointer;">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="(execStatusMap[row.status]?.type as any) || ''" size="small">{{ execStatusMap[row.status]?.label || row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" width="110">
          <template #default="{ row }">{{ row.success_count }}/{{ row.total_count }} 成功</template>
        </el-table-column>
        <el-table-column prop="operator_name" label="执行人" width="90">
          <template #default="{ row }">{{ row.operator_name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170" />
      </el-table>
      <el-pagination v-if="historyTotal > 0" style="margin-top: 12px; justify-content: flex-end;" background layout="total, prev, pager, next" :total="historyTotal" :page-size="historyQuery.size" :current-page="historyQuery.page" @current-change="(p: number) => { historyQuery.page = p; fetchHistory() }" />
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
</style>
