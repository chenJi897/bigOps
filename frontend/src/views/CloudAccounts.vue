<script setup lang="ts">
defineOptions({ name: 'CloudAccounts' })
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { cloudAccountApi, serviceTreeApi, userApi } from '../api'

const loading = ref(false)
const tableData = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)

// 服务树数据
const serviceTreeData = ref<any[]>([])
const serviceTreeMap = ref<Record<number, string>>({})

// 表单
const dialogVisible = ref(false)
const dialogTitle = ref('新增云账号')
const editId = ref(0)
const form = ref<any>({ name: '', provider: 'aliyun', access_key: '', secret_key: '', region: '', service_tree_id: 0, owner_ids: [] as number[], sync_enabled: false, sync_interval: 0 })
const isCreate = ref(true)

// 用户列表（负责人选择用）
const allUsers = ref<any[]>([])

// 更新密钥
const keysDialogVisible = ref(false)
const keysForm = ref({ access_key: '', secret_key: '' })
const keysId = ref(0)

// 同步记录抽屉
const syncLogDrawer = ref(false)
const syncLogAccountId = ref(0)
const syncLogAccountName = ref('')
const syncLogs = ref<any[]>([])
const syncLogTotal = ref(0)
const syncLogPage = ref(1)
const syncLogLoading = ref(false)

const providerOptions = [
  { label: '阿里云', value: 'aliyun' },
  { label: '腾讯云', value: 'tencent' },
  { label: 'AWS', value: 'aws' },
]

const intervalOptions = [
  { label: '每 10 分钟', value: 10 },
  { label: '每 30 分钟', value: 30 },
  { label: '每小时', value: 60 },
  { label: '每天', value: 1440 },
]

// 构建服务树名称映射（含完整路径）
function buildTreeMap(nodes: any[], parentPath = '') {
  for (const node of nodes) {
    const fullPath = parentPath ? `${parentPath} / ${node.name}` : node.name
    serviceTreeMap.value[node.id] = fullPath
    if (node.children?.length) {
      buildTreeMap(node.children, fullPath)
    }
  }
}

async function fetchServiceTree() {
  try {
    const res: any = await serviceTreeApi.tree()
    serviceTreeData.value = res.data || []
    serviceTreeMap.value = {}
    buildTreeMap(serviceTreeData.value)
  } catch {}
}

async function fetchData() {
  loading.value = true
  try {
    const res: any = await cloudAccountApi.list(page.value, size.value)
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

function handleAdd() {
  isCreate.value = true
  dialogTitle.value = '新增云账号'
  form.value = { name: '', provider: 'aliyun', access_key: '', secret_key: '', region: '', service_tree_id: 0, owner_ids: [], sync_enabled: false, sync_interval: 0 }
  dialogVisible.value = true
}

function handleEdit(row: any) {
  isCreate.value = false
  dialogTitle.value = '编辑云账号'
  editId.value = row.id
  form.value = {
    name: row.name, provider: row.provider, access_key: '', secret_key: '',
    region: row.region || '', service_tree_id: row.service_tree_id || 0,
    owner_ids: row.owner_ids ? (typeof row.owner_ids === 'string' ? JSON.parse(row.owner_ids) : row.owner_ids) : [],
    sync_enabled: row.sync_enabled || false, sync_interval: row.sync_interval || 0
  }
  dialogVisible.value = true
}

async function submitForm() {
  if (!form.value.name) { ElMessage.warning('请输入名称'); return }
  try {
    if (isCreate.value) {
      if (!form.value.access_key || !form.value.secret_key) { ElMessage.warning('请输入 AK/SK'); return }
      await cloudAccountApi.create({ ...form.value, owner_ids: JSON.stringify(form.value.owner_ids || []) })
      ElMessage.success('创建成功')
    } else {
      await cloudAccountApi.update(editId.value, {
        name: form.value.name, region: form.value.region, status: 1,
        service_tree_id: form.value.service_tree_id,
        owner_ids: JSON.stringify(form.value.owner_ids || [])
      })
      await cloudAccountApi.syncConfig(editId.value, form.value.sync_enabled, form.value.sync_interval)
      ElMessage.success('更新成功')
    }
    dialogVisible.value = false
    fetchData()
  } catch {}
}

function handleUpdateKeys(row: any) {
  keysId.value = row.id
  keysForm.value = { access_key: '', secret_key: '' }
  keysDialogVisible.value = true
}

async function submitKeys() {
  if (!keysForm.value.access_key || !keysForm.value.secret_key) { ElMessage.warning('请输入 AK/SK'); return }
  try {
    await cloudAccountApi.updateKeys(keysId.value, keysForm.value.access_key, keysForm.value.secret_key)
    ElMessage.success('密钥更新成功')
    keysDialogVisible.value = false
  } catch {}
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(`确定删除 "${row.name}" 吗？`, '提示', { type: 'warning' })
    await cloudAccountApi.delete(row.id)
    ElMessage.success('删除成功')
    fetchData()
  } catch {}
}

async function handleSync(row: any) {
  try {
    ElMessage.info('开始同步...')
    const res: any = await cloudAccountApi.sync(row.id)
    ElMessage.success(res.message || '同步完成')
    fetchData()
  } catch {}
}

function providerLabel(val: string) {
  return providerOptions.find(o => o.value === val)?.label || val
}

function serviceTreeLabel(id: number) {
  return serviceTreeMap.value[id] || ''
}

function handlePageChange(p: number) {
  page.value = p
  fetchData()
}

// 同步记录
async function openSyncLogs(row: any) {
  syncLogAccountId.value = row.id
  syncLogAccountName.value = row.name
  syncLogPage.value = 1
  syncLogDrawer.value = true
  fetchSyncLogs()
}

async function fetchSyncLogs() {
  syncLogLoading.value = true
  try {
    const res: any = await cloudAccountApi.syncTasks(syncLogAccountId.value, syncLogPage.value, 10)
    syncLogs.value = res.data?.list || []
    syncLogTotal.value = res.data?.total || 0
  } finally {
    syncLogLoading.value = false
  }
}

function handleSyncLogPageChange(p: number) {
  syncLogPage.value = p
  fetchSyncLogs()
}

function formatDuration(ms: number) {
  if (!ms) return '-'
  if (ms < 1000) return ms + 'ms'
  return (ms / 1000).toFixed(1) + 's'
}

onMounted(() => {
  fetchData()
  fetchServiceTree()
  userApi.list(1, 200).then((res: any) => { allUsers.value = res.data?.list || [] }).catch(() => {})
})
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>云账号管理</span>
          <el-button type="primary" @click="handleAdd"><el-icon><Plus /></el-icon> 新增</el-button>
        </div>
      </template>

      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column prop="provider" label="云厂商" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ providerLabel(row.provider) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="所属服务" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.service_tree_id">{{ serviceTreeLabel(row.service_tree_id) }}</span>
            <span v-else style="color: #999;">未指定</span>
          </template>
        </el-table-column>
        <el-table-column prop="region" label="Region" min-width="130" show-overflow-tooltip />
        <el-table-column label="负责人" min-width="120">
          <template #default="{ row }">
            <span v-if="row.owner_names?.length">{{ row.owner_names.join('、') }}</span>
            <span v-else style="color: #999;">-</span>
          </template>
        </el-table-column>
        <el-table-column label="定时同步" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.sync_enabled" type="success" size="small">{{ row.sync_interval }}分钟</el-tag>
            <el-tag v-else type="info" size="small">关闭</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_sync_status" label="同步状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.last_sync_status === 'success' ? 'success' : row.last_sync_status === 'failed' ? 'danger' : row.last_sync_status === 'syncing' ? 'warning' : 'info'" size="small">
              {{ row.last_sync_status || '未同步' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_sync_at" label="最后同步" width="170" />
        <el-table-column label="操作" min-width="340" fixed="right">
          <template #default="{ row }">
            <el-button link size="small" @click="handleSync(row)"><el-icon><Refresh /></el-icon> 同步</el-button>
            <el-button link size="small" @click="openSyncLogs(row)"><el-icon><Document /></el-icon> 记录</el-button>
            <el-button link size="small" @click="handleEdit(row)"><el-icon><Edit /></el-icon> 编辑</el-button>
            <el-button link size="small" @click="handleUpdateKeys(row)"><el-icon><Key /></el-icon> 密钥</el-button>
            <el-button link size="small" type="danger" @click="handleDelete(row)"><el-icon><Delete /></el-icon> 删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-if="total > 0"
        style="margin-top: 16px; justify-content: flex-end;"
        background layout="total, prev, pager, next"
        :total="total" :page-size="size" :current-page="page"
        @current-change="handlePageChange"
      />
    </el-card>

    <!-- 新增/编辑 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="云厂商" v-if="isCreate">
          <el-select v-model="form.provider" style="width: 100%;">
            <el-option v-for="o in providerOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="AccessKey" v-if="isCreate"><el-input v-model="form.access_key" /></el-form-item>
        <el-form-item label="SecretKey" v-if="isCreate"><el-input v-model="form.secret_key" type="password" show-password /></el-form-item>
        <el-form-item label="Region"><el-input v-model="form.region" placeholder="cn-hangzhou,cn-beijing" /></el-form-item>
        <el-form-item label="所属服务">
          <el-tree-select
            v-model="form.service_tree_id"
            :data="serviceTreeData"
            :props="{ label: 'name', value: 'id', children: 'children' }"
            placeholder="选择服务树节点（同步资产将归属此节点）"
            clearable check-strictly
            style="width: 100%;"
          />
        </el-form-item>
        <el-form-item label="负责人">
          <el-select v-model="form.owner_ids" multiple placeholder="选择负责人" style="width: 100%;">
            <el-option v-for="u in allUsers" :key="u.id" :label="u.real_name || u.username" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="定时同步" v-if="!isCreate">
          <el-switch v-model="form.sync_enabled" />
        </el-form-item>
        <el-form-item label="同步周期" v-if="!isCreate && form.sync_enabled">
          <el-select v-model="form.sync_interval" style="width: 100%;">
            <el-option v-for="o in intervalOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

    <!-- 更新密钥 -->
    <el-dialog v-model="keysDialogVisible" title="更新密钥" width="450px">
      <el-form :model="keysForm" label-width="80px">
        <el-form-item label="AccessKey"><el-input v-model="keysForm.access_key" /></el-form-item>
        <el-form-item label="SecretKey"><el-input v-model="keysForm.secret_key" type="password" show-password /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="keysDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitKeys">确定</el-button>
      </template>
    </el-dialog>

    <!-- 同步记录抽屉 -->
    <el-drawer v-model="syncLogDrawer" :title="'同步记录 - ' + syncLogAccountName" size="700px">
      <el-table :data="syncLogs" v-loading="syncLogLoading" stripe size="small">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="trigger_type" label="触发" width="70">
          <template #default="{ row }">
            <el-tag :type="row.trigger_type === 'manual' ? '' : 'success'" size="small">
              {{ row.trigger_type === 'manual' ? '手动' : '定时' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : row.status === 'failed' ? 'danger' : 'warning'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="结果" min-width="180">
          <template #default="{ row }">
            <span v-if="row.status === 'success'">
              新增 {{ row.created_count }} / 更新 {{ row.updated_count }} / 无变化 {{ row.unchanged_count }} / 总计 {{ row.total_count }}
            </span>
            <span v-else-if="row.status === 'failed'" style="color: #f56c6c;">{{ row.error_message }}</span>
            <span v-else>运行中...</span>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="80">
          <template #default="{ row }">{{ formatDuration(row.duration_ms) }}</template>
        </el-table-column>
        <el-table-column prop="operator_name" label="操作人" width="80" />
        <el-table-column prop="started_at" label="开始时间" width="170" />
      </el-table>
      <el-pagination
        v-if="syncLogTotal > 10"
        style="margin-top: 12px; justify-content: flex-end;"
        background layout="total, prev, pager, next"
        :total="syncLogTotal" :page-size="10" :current-page="syncLogPage"
        @current-change="handleSyncLogPageChange"
      />
      <el-empty v-if="!syncLogLoading && syncLogs.length === 0" description="暂无同步记录" />
    </el-drawer>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
</style>
