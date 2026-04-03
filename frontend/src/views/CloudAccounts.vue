<script setup lang="ts">
defineOptions({ name: 'CloudAccounts' })
import { ref, onMounted } from 'vue'
import { Plus, Refresh, Document, Edit, Key, Delete } from '@element-plus/icons-vue'
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
  <div class="p-4 sm:p-6 lg:p-8 space-y-4 sm:space-y-6">
    <el-card shadow="never" class="border-0 shadow-sm ring-1 ring-slate-200/60 rounded-xl bg-white">
      <template #header>
        <div class="flex justify-between items-center px-1">
          <div class="flex items-center gap-3">
            <h2 class="text-xl font-semibold text-slate-800 tracking-tight">云账号管理</h2>
            <el-badge v-if="total > 0" :value="total" class="ml-2" type="primary" />
          </div>
          <el-button v-permission="'cloud_account:create'" type="primary" @click="handleAdd" class="shadow-sm !rounded-md">
            <template #icon><el-icon><Plus /></el-icon></template>
            新增账号
          </el-button>
        </div>
      </template>

      <div class="flex flex-col space-y-5">
        <el-table :data="tableData" v-loading="loading" stripe :border="false" class="w-full shadow-sm rounded-xl overflow-hidden border border-slate-100 table-modern">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="name" label="账号名称" min-width="140">
            <template #default="{ row }">
              <span class="font-semibold text-slate-800">{{ row.name }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="provider" label="云厂商" width="120">
            <template #default="{ row }">
              <el-tag size="small" effect="plain" class="!rounded-md !border-slate-200 !text-slate-600 !bg-slate-50">{{ providerLabel(row.provider) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="所属服务" min-width="160" show-overflow-tooltip>
            <template #default="{ row }">
              <span v-if="row.service_tree_id" class="text-sm text-slate-700 font-medium">{{ serviceTreeLabel(row.service_tree_id) }}</span>
              <span v-else class="text-sm text-slate-400">-</span>
            </template>
          </el-table-column>
          <el-table-column prop="region" label="Region" min-width="130" show-overflow-tooltip>
            <template #default="{ row }"><span class="text-sm text-slate-600">{{ row.region || '-' }}</span></template>
          </el-table-column>
          <el-table-column label="负责人" min-width="120">
            <template #default="{ row }">
              <div v-if="row.owner_names?.length" class="flex flex-wrap gap-1">
                <span v-for="name in row.owner_names" :key="name" class="text-xs text-indigo-600 bg-indigo-50 px-1.5 py-0.5 rounded">{{ name }}</span>
              </div>
              <span v-else class="text-slate-400 text-sm">-</span>
            </template>
          </el-table-column>
          <el-table-column label="定时同步" width="110">
            <template #default="{ row }">
              <el-tag v-if="row.sync_enabled" type="success" size="small" effect="light" class="!rounded-md">{{ row.sync_interval }} 分钟</el-tag>
              <el-tag v-else type="info" size="small" effect="plain" class="!rounded-md !bg-slate-50 !border-slate-200">已关闭</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="last_sync_status" label="同步状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.last_sync_status === 'success' ? 'success' : row.last_sync_status === 'failed' ? 'danger' : row.last_sync_status === 'syncing' ? 'warning' : 'info'" size="small" effect="light" class="!rounded-md">
                {{ row.last_sync_status || '未同步' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="last_sync_at" label="最后同步" width="170">
            <template #default="{ row }"><span class="text-slate-500 text-sm">{{ row.last_sync_at || '-' }}</span></template>
          </el-table-column>
          <el-table-column label="操作" min-width="380" fixed="right">
            <template #default="{ row }">
              <div class="flex items-center gap-1">
                <el-button v-permission="'cloud_account:sync'" link type="primary" size="small" @click="handleSync(row)" class="!px-2 hover:bg-indigo-50 !rounded">
                  <el-icon class="mr-1"><Refresh /></el-icon>同步
                </el-button>
                <el-button link type="primary" size="small" @click="openSyncLogs(row)" class="!px-2 hover:bg-indigo-50 !rounded">
                  <el-icon class="mr-1"><Document /></el-icon>记录
                </el-button>
                <el-button v-permission="'cloud_account:edit'" link type="primary" size="small" @click="handleEdit(row)" class="!px-2 hover:bg-indigo-50 !rounded">
                  <el-icon class="mr-1"><Edit /></el-icon>编辑
                </el-button>
                <el-button link type="primary" size="small" @click="handleUpdateKeys(row)" class="!px-2 hover:bg-indigo-50 !rounded">
                  <el-icon class="mr-1"><Key /></el-icon>密钥
                </el-button>
                <el-button v-permission="'cloud_account:delete'" link type="danger" size="small" @click="handleDelete(row)" class="!px-2 hover:bg-red-50 !rounded">
                  <el-icon class="mr-1"><Delete /></el-icon>删除
                </el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <div class="flex justify-end pt-5 pb-2">
          <el-pagination
            v-if="total > 0"
            background
            layout="total, prev, pager, next"
            :total="total"
            :page-size="size"
            :current-page="page"
            @current-change="handlePageChange"
          />
        </div>
      </div>
    </el-card>

    <!-- 新增/编辑 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="580px" class="rounded-xl overflow-hidden custom-dialog">
      <div class="pt-4 px-2">
        <el-form :model="form" label-width="100px" label-position="right" class="space-y-4">
          <el-form-item label="名称"><el-input v-model="form.name" placeholder="标识该账号的名称" class="!rounded-md" /></el-form-item>
          <el-form-item label="云厂商" v-if="isCreate">
            <el-select v-model="form.provider" class="w-full !rounded-md">
              <el-option v-for="o in providerOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
          </el-form-item>
          <template v-if="isCreate">
            <el-form-item label="AccessKey"><el-input v-model="form.access_key" placeholder="API 访问密钥" class="!rounded-md" /></el-form-item>
            <el-form-item label="SecretKey"><el-input v-model="form.secret_key" type="password" show-password placeholder="API 密钥" class="!rounded-md" /></el-form-item>
          </template>
          <el-form-item label="Region"><el-input v-model="form.region" placeholder="如：cn-hangzhou,cn-beijing" class="!rounded-md" /></el-form-item>
          <el-form-item label="所属服务">
            <el-tree-select
              v-model="form.service_tree_id"
              :data="serviceTreeData"
              :props="{ label: 'name', value: 'id', children: 'children' }"
              placeholder="选择服务树节点（同步资产将归属此节点）"
              clearable check-strictly
              class="w-full !rounded-md"
            />
          </el-form-item>
          <el-form-item label="负责人">
            <el-select v-model="form.owner_ids" multiple placeholder="选择负责人" class="w-full !rounded-md">
              <el-option v-for="u in allUsers" :key="u.id" :label="u.real_name || u.username" :value="u.id" />
            </el-select>
          </el-form-item>
          <template v-if="!isCreate">
            <el-form-item label="定时同步">
              <el-switch v-model="form.sync_enabled" active-text="开启" inactive-text="关闭" />
            </el-form-item>
            <el-form-item label="同步周期" v-if="form.sync_enabled">
              <el-select v-model="form.sync_interval" class="w-full !rounded-md">
                <el-option v-for="o in intervalOptions" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </el-form-item>
          </template>
        </el-form>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3 pt-4 border-t border-slate-100">
          <el-button @click="dialogVisible = false" class="px-6 !rounded-md">取消</el-button>
          <el-button type="primary" @click="submitForm" class="px-6 !rounded-md">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 更新密钥 -->
    <el-dialog v-model="keysDialogVisible" title="更新密钥" width="480px" class="rounded-xl overflow-hidden custom-dialog">
      <div class="pt-4 px-2">
        <el-form :model="keysForm" label-width="100px" class="space-y-4">
          <el-form-item label="AccessKey"><el-input v-model="keysForm.access_key" placeholder="输入新 AccessKey" class="!rounded-md" /></el-form-item>
          <el-form-item label="SecretKey"><el-input v-model="keysForm.secret_key" type="password" show-password placeholder="输入新 SecretKey" class="!rounded-md" /></el-form-item>
        </el-form>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3 pt-4 border-t border-slate-100">
          <el-button @click="keysDialogVisible = false" class="px-6 !rounded-md">取消</el-button>
          <el-button type="primary" @click="submitKeys" class="px-6 !rounded-md">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 同步记录抽屉 -->
    <el-drawer v-model="syncLogDrawer" :title="`同步记录 - ${syncLogAccountName}`" size="700px" class="custom-drawer">
      <div class="px-6 pb-6 h-full flex flex-col">
        <el-table :data="syncLogs" v-loading="syncLogLoading" stripe :border="false" class="w-full shadow-sm rounded-xl overflow-hidden border border-slate-100 mt-2 flex-1 table-modern">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="trigger_type" label="触发" width="80">
            <template #default="{ row }">
              <el-tag :type="row.trigger_type === 'manual' ? 'info' : 'success'" size="small" effect="plain" class="!rounded-md">
                {{ row.trigger_type === 'manual' ? '手动' : '定时' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="row.status === 'success' ? 'success' : row.status === 'failed' ? 'danger' : 'warning'" size="small" effect="light" class="!rounded-md">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="结果" min-width="220">
            <template #default="{ row }">
              <div v-if="row.status === 'success'" class="text-xs text-slate-600 flex flex-wrap gap-2">
                <span class="bg-emerald-50 text-emerald-600 px-1.5 py-0.5 rounded border border-emerald-100">新增 {{ row.created_count }}</span>
                <span class="bg-blue-50 text-blue-600 px-1.5 py-0.5 rounded border border-blue-100">更新 {{ row.updated_count }}</span>
                <span class="bg-slate-50 text-slate-500 px-1.5 py-0.5 rounded border border-slate-200">不变 {{ row.unchanged_count }}</span>
              </div>
              <span v-else-if="row.status === 'failed'" class="text-sm text-red-500 font-medium">{{ row.error_message }}</span>
              <span v-else class="text-sm text-amber-600 flex items-center gap-1"><el-icon class="is-loading"><Refresh /></el-icon>运行中...</span>
            </template>
          </el-table-column>
          <el-table-column label="耗时" width="80">
            <template #default="{ row }"><span class="text-slate-500 text-sm font-mono bg-slate-50 px-1.5 py-0.5 rounded border border-slate-100">{{ formatDuration(row.duration_ms) }}</span></template>
          </el-table-column>
          <el-table-column prop="started_at" label="开始时间" width="160">
            <template #default="{ row }"><span class="text-slate-500 text-sm">{{ row.started_at }}</span></template>
          </el-table-column>
        </el-table>
        <div class="flex justify-end pt-5 pb-2">
          <el-pagination
            v-if="syncLogTotal > 0"
            background
            layout="total, prev, pager, next"
            :total="syncLogTotal" :page-size="10" :current-page="syncLogPage"
            @current-change="handleSyncLogPageChange"
          />
        </div>
        <el-empty v-if="!syncLogLoading && syncLogs.length === 0" description="暂无同步记录" :image-size="80" class="py-12 opacity-80" />
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
:deep(.el-table__row) {
  @apply hover:bg-indigo-50/50 transition-colors duration-200;
}
:deep(.table-modern th.el-table__cell) {
  @apply bg-slate-50/80 text-slate-600 font-medium border-b border-slate-200;
}
:deep(.custom-drawer .el-drawer__header) {
  @apply mb-0 border-b border-slate-100 pb-4 font-semibold text-slate-800;
}
:deep(.custom-dialog .el-dialog__header) {
  @apply border-b border-slate-100 pb-4 mb-0 mr-0;
}
:deep(.custom-dialog .el-dialog__title) {
  @apply font-semibold text-slate-800;
}
</style>
