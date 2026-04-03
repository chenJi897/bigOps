<script setup lang="ts">
defineOptions({ name: 'Assets' })
import { ref, onMounted } from 'vue'
import { Plus, Edit, Delete, Search } from '@element-plus/icons-vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { assetApi, serviceTreeApi, userApi } from '../api'

const route = useRoute()
const loading = ref(false)
const tableData = ref<any[]>([])
const total = ref(0)
const query = ref({ page: 1, size: 20, status: '', source: '', service_tree_id: undefined as number | undefined, recursive: false, keyword: '', owner_id: undefined as number | undefined })

// 服务树数据（供筛选和表单选择）
const serviceTreeOptions = ref<any[]>([])
// 用户列表（负责人选择用）
const allUsers = ref<any[]>([])

// 详情抽屉
const drawerVisible = ref(false)
const currentAsset = ref<any>(null)
const activeTab = ref('info')
const changes = ref<any[]>([])
const changesTotal = ref(0)
const changesPage = ref(1)
const changesLoading = ref(false)

// 表单
const dialogVisible = ref(false)
const dialogTitle = ref('新增资产')
const isCreate = ref(true)
const editId = ref(0)

function createDefaultForm() {
  return {
    hostname: '',
    ip: '',
    inner_ip: '',
    os: '',
    os_version: '',
    cpu_cores: 0,
    memory_mb: 0,
    disk_gb: 0,
    status: 'online',
    asset_type: 'server',
    source: 'manual',
    service_tree_id: undefined as number | undefined,
    idc: '',
    sn: '',
    remark: '',
    owner_ids: [] as number[],
  }
}

const form = ref({
  hostname: '', ip: '', inner_ip: '', os: '', os_version: '',
  cpu_cores: 0, memory_mb: 0, disk_gb: 0, status: 'online',
  asset_type: 'server', source: 'manual', service_tree_id: undefined as number | undefined,
  idc: '', sn: '', remark: '', owner_ids: [] as number[],
})

async function fetchServiceTree() {
  try {
    const res: any = await serviceTreeApi.tree()
    serviceTreeOptions.value = res.data || []
  } catch {}
}

async function fetchData() {
  loading.value = true
  try {
    const res: any = await assetApi.list(query.value)
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  query.value.page = 1
  fetchData()
}

function handlePageChange(p: number) {
  query.value.page = p
  fetchData()
}

function handleAdd() {
  isCreate.value = true
  dialogTitle.value = '新增资产'
  form.value = createDefaultForm()
  dialogVisible.value = true
}

function handleEdit(row: any) {
  isCreate.value = false
  dialogTitle.value = '编辑资产'
  editId.value = row.id
  form.value = {
    hostname: row.hostname, ip: row.ip, inner_ip: row.inner_ip || '', os: row.os || '', os_version: row.os_version || '',
    cpu_cores: row.cpu_cores || 0, memory_mb: row.memory_mb || 0, disk_gb: row.disk_gb || 0,
    status: row.status, asset_type: row.asset_type, source: row.source,
    service_tree_id: row.service_tree_id || undefined,
    idc: row.idc || '', sn: row.sn || '', remark: row.remark || '',
    owner_ids: row.owner_ids ? (typeof row.owner_ids === 'string' ? JSON.parse(row.owner_ids) : row.owner_ids) : [],
  }
  dialogVisible.value = true
}

async function submitForm() {
  if (!form.value.hostname || !form.value.ip) { ElMessage.warning('主机名和 IP 必填'); return }
  const payload = {
    ...form.value,
    service_tree_id: form.value.service_tree_id || 0,
    owner_ids: JSON.stringify(form.value.owner_ids || []),
  }
  try {
    if (isCreate.value) {
      await assetApi.create(payload)
      ElMessage.success('创建成功')
    } else {
      await assetApi.update(editId.value, payload)
      ElMessage.success('更新成功')
    }
    dialogVisible.value = false
    fetchData()
  } catch {}
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(`确定删除 "${row.hostname}" 吗？`, '提示', { type: 'warning' })
    await assetApi.delete(row.id)
    ElMessage.success('删除成功')
    fetchData()
  } catch {}
}

async function openDrawer(row: any) {
  currentAsset.value = row
  activeTab.value = 'info'
  drawerVisible.value = true
  fetchChanges(row.id)
}

async function fetchChanges(assetId: number) {
  changesLoading.value = true
  try {
    const res: any = await assetApi.changes(assetId, changesPage.value, 20)
    changes.value = res.data?.list || []
    changesTotal.value = res.data?.total || 0
  } finally {
    changesLoading.value = false
  }
}

function handleChangesPage(p: number) {
  changesPage.value = p
  if (currentAsset.value) fetchChanges(currentAsset.value.id)
}

function isExpiringSoon(dateStr: string) {
  if (!dateStr) return false
  const d = new Date(dateStr)
  const diff = d.getTime() - Date.now()
  return diff > 0 && diff < 30 * 24 * 3600 * 1000 // 30天内到期标红
}

const fieldLabels: Record<string, string> = {
  hostname: '主机名', ip: 'IP', inner_ip: '内网IP', os: '系统', os_version: '系统版本',
  status: '状态', asset_type: '类型', idc: '机房', sn: '序列号', remark: '备注',
  cpu_cores: 'CPU核数', memory_mb: '内存(MB)', disk_gb: '磁盘(GB)',
  service_tree_id: '服务树', owner_ids: '负责人',
}
function fieldLabel(field: string) { return fieldLabels[field] || field }
function formatChangeValue(field: string, value: string) {
  if (field === 'owner_ids') {
    if (!value || value === '[]') return '无'
    try {
      const ids = JSON.parse(value) as number[]
      return ids.map(id => {
        const u = allUsers.value.find((u: any) => u.id === id)
        return u ? (u.real_name || u.username) : `用户${id}`
      }).join('、')
    } catch { return value }
  }
  return value || '-'
}

onMounted(() => {
  // 从 URL query 读取筛选参数（首页/服务树页跳转过来）
  if (route.query.service_tree_id) {
    query.value.service_tree_id = Number(route.query.service_tree_id)
    query.value.recursive = route.query.recursive === 'true'
  }
  if (route.query.status) {
    query.value.status = route.query.status as string
  }
  if (route.query.source) {
    query.value.source = route.query.source as string
  }
  fetchData()
  fetchServiceTree()
  userApi.list(1, 200).then((res: any) => { allUsers.value = res.data?.list || [] }).catch(() => {})
})
</script>

<template>
  <div class="p-4 sm:p-6 lg:p-8 space-y-4 sm:space-y-6">
    <el-card shadow="never" class="border-0 shadow-sm ring-1 ring-gray-200/50 rounded-xl">
      <template #header>
        <div class="flex justify-between items-center">
          <div class="flex items-center gap-3">
            <h2 class="text-xl font-semibold text-gray-800 tracking-tight">主机资产</h2>
            <el-badge v-if="total > 0" :value="total" :max="999" class="ml-2" type="primary" />
          </div>
          <el-button v-permission="'asset:create'" type="primary" @click="handleAdd" class="shadow-sm !rounded-md">
            <template #icon><el-icon><Plus /></el-icon></template>
            新增资产
          </el-button>
        </div>
      </template>

      <div class="flex flex-col space-y-5">
        <div class="bg-slate-50/50 p-4 rounded-xl border border-slate-100 flex flex-wrap gap-4 items-center">
          <el-form :inline="true" @submit.prevent="handleSearch" class="flex flex-wrap gap-3 w-full items-center" style="margin-bottom: 0;">
            <el-input v-model="query.keyword" placeholder="搜索主机名或 IP" clearable class="w-48 sm:w-64 !rounded-md" @keyup.enter="handleSearch">
              <template #prefix><el-icon class="text-slate-400"><Search /></el-icon></template>
            </el-input>
            
            <el-select v-model="query.status" placeholder="状态" clearable class="w-24">
              <el-option label="在线" value="online" />
              <el-option label="离线" value="offline" />
            </el-select>
            
            <el-select v-model="query.source" placeholder="来源" clearable class="w-28">
              <el-option label="手动录入" value="manual" />
              <el-option label="阿里云" value="aliyun" />
              <el-option label="腾讯云" value="tencent" />
            </el-select>
            
            <el-tree-select
              v-model="query.service_tree_id"
              :data="serviceTreeOptions"
              :props="{ children: 'children', label: 'name', value: 'id' }"
              placeholder="所属服务"
              clearable
              check-strictly
              class="w-40 sm:w-48"
            />
            
            <el-select v-model="query.owner_id" placeholder="负责人" clearable class="w-32">
              <el-option v-for="u in allUsers" :key="u.id" :label="u.real_name || u.username" :value="u.id" />
            </el-select>
            
            <div class="flex items-center gap-2 ml-auto">
              <el-button type="primary" @click="handleSearch" class="!rounded-md">搜索</el-button>
            </div>
          </el-form>
        </div>

        <el-table :data="tableData" v-loading="loading" stripe :border="false" class="w-full shadow-sm rounded-xl overflow-hidden border border-slate-100" @row-click="openDrawer" style="cursor: pointer;">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="hostname" label="主机名" min-width="160" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="font-semibold text-slate-800 group-hover:text-indigo-600 transition-colors">{{ row.hostname }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="ip" label="公网 IP" width="130">
            <template #default="{ row }">
              <span class="font-mono text-sm text-slate-600 bg-slate-50 px-1.5 py-0.5 rounded border border-slate-200/60">{{ row.ip || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="inner_ip" label="内网 IP" width="130">
            <template #default="{ row }">
              <span class="font-mono text-sm text-slate-500 bg-slate-50 px-1.5 py-0.5 rounded border border-slate-200/60">{{ row.inner_ip || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="os" label="操作系统" width="150" show-overflow-tooltip>
             <template #default="{ row }">
              <div class="flex items-center gap-1.5">
                <el-icon class="text-slate-400"><Monitor /></el-icon>
                <span class="text-sm text-slate-600">{{ row.os || '-' }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="配置" width="120">
            <template #default="{ row }">
              <span class="text-sm font-medium text-slate-600 bg-slate-50 px-2 py-0.5 rounded border border-slate-100">{{ row.cpu_cores }}C / {{ (row.memory_mb / 1024).toFixed(0) }}G</span>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="90">
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
          <el-table-column label="负责人" min-width="120">
            <template #default="{ row }">
              <div v-if="row.owner_names?.length" class="flex flex-wrap gap-1">
                <span v-for="name in row.owner_names" :key="name" class="text-xs text-indigo-600 bg-indigo-50 px-1.5 py-0.5 rounded">{{ name }}</span>
              </div>
              <span v-else class="text-slate-400 text-sm">-</span>
            </template>
          </el-table-column>
          <el-table-column prop="service_tree_name" label="所属服务" width="160" show-overflow-tooltip>
            <template #default="{ row }">
              <el-tooltip v-if="row.service_tree_path" :content="row.service_tree_path" placement="top" :show-after="300">
                <span class="text-sm text-slate-700 font-medium cursor-help flex items-center gap-1 hover:text-indigo-600 transition-colors">
                  <el-icon class="text-indigo-400"><Folder /></el-icon>
                  <span class="truncate">{{ row.service_tree_name }}</span>
                </span>
              </el-tooltip>
              <span v-else-if="row.service_tree_name" class="text-sm text-slate-700 font-medium flex items-center gap-1">
                <el-icon class="text-indigo-400"><Folder /></el-icon>
                <span class="truncate">{{ row.service_tree_name }}</span>
              </span>
              <span v-else class="text-slate-400 text-sm">-</span>
            </template>
          </el-table-column>
          <el-table-column prop="idc" label="机房/区域" width="120" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="text-sm text-slate-600 flex items-center gap-1"><el-icon class="text-slate-400"><Location /></el-icon> {{ row.idc || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <div class="flex items-center gap-1">
                <el-tooltip content="编辑" placement="top" :show-after="300">
                  <el-button v-permission="'asset:edit'" link type="primary" @click.stop="handleEdit(row)" class="!p-1.5 hover:bg-indigo-50 rounded">
                    <el-icon class="text-base"><Edit /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top" :show-after="300">
                  <el-button v-permission="'asset:delete'" link type="danger" @click.stop="handleDelete(row)" class="!p-1.5 hover:bg-red-50 rounded">
                    <el-icon class="text-base"><Delete /></el-icon>
                  </el-button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <div class="flex justify-end pt-4 pb-2">
          <el-pagination
            v-if="total > 0"
            background
            layout="total, sizes, prev, pager, next, jumper"
            :page-sizes="[10, 20, 50, 100]"
            :total="total"
            v-model:page-size="query.size"
            :current-page="query.page"
            @size-change="(s: number) => { query.size = s; handleSearch() }"
            @current-change="handlePageChange"
          />
        </div>
      </div>
    </el-card>

    <!-- 详情抽屉 -->
    <el-drawer v-model="drawerVisible" :title="currentAsset?.hostname || '资产详情'" size="50%" class="custom-drawer">
      <div class="px-6 pb-6">
        <el-tabs v-model="activeTab" class="w-full">
          <el-tab-pane label="基本信息" name="info">
            <el-descriptions v-if="currentAsset" :column="2" border class="mt-4 shadow-sm rounded-lg overflow-hidden">
              <el-descriptions-item label="ID">{{ currentAsset.id }}</el-descriptions-item>
              <el-descriptions-item label="主机名">{{ currentAsset.hostname }}</el-descriptions-item>
              <el-descriptions-item label="公网 IP"><span class="font-mono">{{ currentAsset.ip || '-' }}</span></el-descriptions-item>
              <el-descriptions-item label="内网 IP"><span class="font-mono">{{ currentAsset.inner_ip || '-' }}</span></el-descriptions-item>
              <el-descriptions-item label="操作系统">{{ currentAsset.os || '-' }}</el-descriptions-item>
              <el-descriptions-item label="系统类型">{{ currentAsset.os_version || '-' }}</el-descriptions-item>
              <el-descriptions-item label="CPU">{{ currentAsset.cpu_cores }} 核</el-descriptions-item>
              <el-descriptions-item label="内存">{{ currentAsset.memory_mb }} MB</el-descriptions-item>
              <el-descriptions-item label="磁盘">{{ currentAsset.disk_gb }} GB</el-descriptions-item>
              <el-descriptions-item label="状态">
                <el-tag :type="currentAsset.status === 'online' ? 'success' : 'danger'" size="small" effect="light" class="rounded">{{ currentAsset.status }}</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="来源">{{ currentAsset.source }}</el-descriptions-item>
              <el-descriptions-item label="所属服务">{{ currentAsset.service_tree_path || currentAsset.service_tree_name || '-' }}</el-descriptions-item>
              <el-descriptions-item label="机房/区域">{{ currentAsset.idc || '-' }}</el-descriptions-item>
              <el-descriptions-item label="SN">{{ currentAsset.sn || '-' }}</el-descriptions-item>
              <el-descriptions-item label="云实例 ID">{{ currentAsset.cloud_instance_id || '-' }}</el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ currentAsset.created_at || '-' }}</el-descriptions-item>
              <el-descriptions-item label="到期时间">
                <span v-if="currentAsset.expired_at" :class="{ 'text-red-500 font-medium': isExpiringSoon(currentAsset.expired_at) }">{{ currentAsset.expired_at }}</span>
                <span v-else class="text-gray-400">-</span>
              </el-descriptions-item>
              <el-descriptions-item label="更新时间">{{ currentAsset.updated_at || '-' }}</el-descriptions-item>
            </el-descriptions>
          </el-tab-pane>

          <el-tab-pane label="变更历史" name="changes">
            <div class="mt-4">
              <el-table :data="changes" v-loading="changesLoading" stripe :border="false" class="w-full shadow-sm rounded-lg overflow-hidden border border-gray-100">
                <el-table-column prop="field" label="字段" width="120">
                  <template #default="{ row }"><span class="font-medium text-gray-700">{{ fieldLabel(row.field) }}</span></template>
                </el-table-column>
                <el-table-column label="旧值" min-width="150" show-overflow-tooltip>
                  <template #default="{ row }"><span class="text-red-600 line-through text-sm">{{ formatChangeValue(row.field, row.old_value) }}</span></template>
                </el-table-column>
                <el-table-column label="新值" min-width="150" show-overflow-tooltip>
                  <template #default="{ row }"><span class="text-green-600 text-sm">{{ formatChangeValue(row.field, row.new_value) }}</span></template>
                </el-table-column>
                <el-table-column prop="change_type" label="类型" width="80">
                  <template #default="{ row }"><el-tag size="small" type="info" effect="light">{{ row.change_type }}</el-tag></template>
                </el-table-column>
                <el-table-column prop="operator_name" label="操作人" width="100">
                   <template #default="{ row }"><span class="text-gray-700 text-sm">{{ row.operator_name || '-' }}</span></template>
                </el-table-column>
                <el-table-column prop="created_at" label="变更时间" width="170">
                   <template #default="{ row }"><span class="text-gray-500 text-sm">{{ row.created_at }}</span></template>
                </el-table-column>
              </el-table>
              <div class="flex justify-end pt-4 pb-2">
                <el-pagination
                  v-if="changesTotal > 0"
                  background
                  layout="total, prev, pager, next"
                  :total="changesTotal"
                  :page-size="20"
                  :current-page="changesPage"
                  @current-change="handleChangesPage"
                />
              </div>
              <el-empty v-if="!changesLoading && changes.length === 0" description="暂无变更记录" />
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-drawer>

    <!-- 新增/编辑 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="640px" class="rounded-xl overflow-hidden">
      <div class="pt-4 px-2">
        <el-form :model="form" label-width="85px" label-position="right" class="space-y-4">
          <el-row :gutter="24">
            <el-col :span="12"><el-form-item label="主机名"><el-input v-model="form.hostname" placeholder="如: web-server-01" /></el-form-item></el-col>
            <el-col :span="12"><el-form-item label="公网 IP"><el-input v-model="form.ip" placeholder="0.0.0.0" /></el-form-item></el-col>
            <el-col :span="12"><el-form-item label="内网 IP"><el-input v-model="form.inner_ip" placeholder="10.x.x.x" /></el-form-item></el-col>
            <el-col :span="12"><el-form-item label="操作系统"><el-input v-model="form.os" placeholder="如: Ubuntu 22.04" /></el-form-item></el-col>
            <el-col :span="8"><el-form-item label="CPU(核)"><el-input-number v-model="form.cpu_cores" :min="0" class="!w-full" controls-position="right" /></el-form-item></el-col>
            <el-col :span="8"><el-form-item label="内存(MB)"><el-input-number v-model="form.memory_mb" :min="0" class="!w-full" controls-position="right" /></el-form-item></el-col>
            <el-col :span="8"><el-form-item label="磁盘(GB)"><el-input-number v-model="form.disk_gb" :min="0" class="!w-full" controls-position="right" /></el-form-item></el-col>
            <el-col :span="12"><el-form-item label="机房区域"><el-input v-model="form.idc" placeholder="如: cn-hangzhou-a" /></el-form-item></el-col>
            <el-col :span="12">
              <el-form-item label="所属服务">
                <el-tree-select
                  v-model="form.service_tree_id"
                  :data="serviceTreeOptions"
                  :props="{ children: 'children', label: 'name', value: 'id' }"
                  placeholder="选择所属节点"
                  clearable
                  check-strictly
                  class="w-full"
                />
              </el-form-item>
            </el-col>
            <el-col :span="12"><el-form-item label="设备 SN"><el-input v-model="form.sn" placeholder="设备序列号" /></el-form-item></el-col>
            <el-col :span="24">
              <el-form-item label="负责人">
                <el-select v-model="form.owner_ids" multiple placeholder="选择负责人" class="w-full">
                  <el-option v-for="u in allUsers" :key="u.id" :label="u.real_name || u.username" :value="u.id" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="备注">
                <el-input v-model="form.remark" type="textarea" :rows="3" placeholder="其他补充信息..." />
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3 pt-4 border-t border-gray-100">
          <el-button @click="dialogVisible = false" class="px-6">取消</el-button>
          <el-button type="primary" @click="submitForm" class="px-6">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
:deep(.el-table__row) {
  @apply hover:bg-indigo-50/50 transition-colors duration-200;
}
:deep(.el-table th.el-table__cell) {
  @apply bg-gray-50/80 text-gray-600 font-medium;
}
:deep(.el-descriptions__label) {
  @apply bg-gray-50/50 w-32;
}
:deep(.custom-drawer .el-drawer__header) {
  @apply mb-0 border-b border-gray-100 pb-4 font-semibold text-gray-800;
}
</style>
