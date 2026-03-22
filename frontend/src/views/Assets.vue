<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { assetApi, serviceTreeApi } from '../api'

const route = useRoute()
const loading = ref(false)
const tableData = ref<any[]>([])
const total = ref(0)
const query = ref({ page: 1, size: 20, status: '', source: '', service_tree_id: undefined as number | undefined, recursive: false, keyword: '' })

// 服务树数据（供筛选和表单选择）
const serviceTreeOptions = ref<any[]>([])

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
const form = ref({
  hostname: '', ip: '', inner_ip: '', os: '', os_version: '',
  cpu_cores: 0, memory_mb: 0, disk_gb: 0, status: 'online',
  asset_type: 'server', source: 'manual', service_tree_id: 0,
  idc: '', sn: '', remark: '',
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
  form.value = { hostname: '', ip: '', inner_ip: '', os: '', os_version: '', cpu_cores: 0, memory_mb: 0, disk_gb: 0, status: 'online', asset_type: 'server', source: 'manual', service_tree_id: 0, idc: '', sn: '', remark: '' }
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
    service_tree_id: row.service_tree_id || 0,
    idc: row.idc || '', sn: row.sn || '', remark: row.remark || '',
  }
  dialogVisible.value = true
}

async function submitForm() {
  if (!form.value.hostname || !form.value.ip) { ElMessage.warning('主机名和 IP 必填'); return }
  try {
    if (isCreate.value) {
      await assetApi.create(form.value)
      ElMessage.success('创建成功')
    } else {
      await assetApi.update(editId.value, form.value)
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

onMounted(() => {
  // 从 URL query 读取筛选参数（服务树页跳转过来）
  if (route.query.service_tree_id) {
    query.value.service_tree_id = Number(route.query.service_tree_id)
    query.value.recursive = route.query.recursive === 'true'
  }
  fetchData()
  fetchServiceTree()
})
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>主机资产</span>
          <el-button type="primary" @click="handleAdd"><el-icon><Plus /></el-icon> 新增</el-button>
        </div>
      </template>

      <el-form :inline="true" @submit.prevent="handleSearch" style="margin-bottom: 16px;">
        <el-form-item>
          <el-input v-model="query.keyword" placeholder="主机名/IP" clearable style="width: 180px;" />
        </el-form-item>
        <el-form-item>
          <el-select v-model="query.status" placeholder="状态" clearable style="width: 100px;">
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="query.source" placeholder="来源" clearable style="width: 100px;">
            <el-option label="手动" value="manual" />
            <el-option label="阿里云" value="aliyun" />
            <el-option label="腾讯云" value="tencent" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-tree-select
            v-model="query.service_tree_id"
            :data="serviceTreeOptions"
            :props="{ children: 'children', label: 'name', value: 'id' }"
            placeholder="所属服务"
            clearable
            check-strictly
            style="width: 160px;"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="tableData" v-loading="loading" stripe @row-click="openDrawer" style="cursor: pointer;">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="hostname" label="主机名" min-width="140" show-overflow-tooltip />
        <el-table-column prop="ip" label="公网IP" width="130" />
        <el-table-column prop="inner_ip" label="内网IP" width="130" />
        <el-table-column prop="os" label="操作系统" width="150" show-overflow-tooltip />
        <el-table-column label="配置" width="120">
          <template #default="{ row }">{{ row.cpu_cores }}C / {{ (row.memory_mb / 1024).toFixed(0) }}G</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 'online' ? 'success' : 'danger'" size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="source" label="来源" width="80">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.source }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="service_tree_name" label="所属服务" width="120">
          <template #default="{ row }">
            <el-tooltip v-if="row.service_tree_path" :content="row.service_tree_path" placement="top">
              <span style="cursor: default;">{{ row.service_tree_name }}</span>
            </el-tooltip>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="idc" label="机房/区域" width="120" show-overflow-tooltip />
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link size="small" @click.stop="handleEdit(row)"><el-icon><Edit /></el-icon></el-button>
            <el-button link size="small" type="danger" @click.stop="handleDelete(row)"><el-icon><Delete /></el-icon></el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-if="total > 0"
        style="margin-top: 16px; justify-content: flex-end;"
        background layout="total, prev, pager, next"
        :total="total" :page-size="query.size" :current-page="query.page"
        @current-change="handlePageChange"
      />
    </el-card>

    <!-- 详情抽屉 -->
    <el-drawer v-model="drawerVisible" :title="currentAsset?.hostname" size="50%">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="基本信息" name="info">
          <el-descriptions v-if="currentAsset" :column="2" border>
            <el-descriptions-item label="ID">{{ currentAsset.id }}</el-descriptions-item>
            <el-descriptions-item label="主机名">{{ currentAsset.hostname }}</el-descriptions-item>
            <el-descriptions-item label="公网IP">{{ currentAsset.ip }}</el-descriptions-item>
            <el-descriptions-item label="内网IP">{{ currentAsset.inner_ip }}</el-descriptions-item>
            <el-descriptions-item label="操作系统">{{ currentAsset.os }}</el-descriptions-item>
            <el-descriptions-item label="系统类型">{{ currentAsset.os_version }}</el-descriptions-item>
            <el-descriptions-item label="CPU">{{ currentAsset.cpu_cores }} 核</el-descriptions-item>
            <el-descriptions-item label="内存">{{ currentAsset.memory_mb }} MB</el-descriptions-item>
            <el-descriptions-item label="磁盘">{{ currentAsset.disk_gb }} GB</el-descriptions-item>
            <el-descriptions-item label="状态">{{ currentAsset.status }}</el-descriptions-item>
            <el-descriptions-item label="来源">{{ currentAsset.source }}</el-descriptions-item>
            <el-descriptions-item label="所属服务">{{ currentAsset.service_tree_path || currentAsset.service_tree_name || '-' }}</el-descriptions-item>
            <el-descriptions-item label="机房/区域">{{ currentAsset.idc }}</el-descriptions-item>
            <el-descriptions-item label="SN">{{ currentAsset.sn }}</el-descriptions-item>
            <el-descriptions-item label="云实例ID">{{ currentAsset.cloud_instance_id }}</el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ currentAsset.created_at }}</el-descriptions-item>
            <el-descriptions-item label="更新时间">{{ currentAsset.updated_at }}</el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>

        <el-tab-pane label="变更历史" name="changes">
          <el-table :data="changes" v-loading="changesLoading" stripe>
            <el-table-column prop="field" label="字段" width="120" />
            <el-table-column prop="old_value" label="旧值" min-width="150" show-overflow-tooltip />
            <el-table-column prop="new_value" label="新值" min-width="150" show-overflow-tooltip />
            <el-table-column prop="change_type" label="类型" width="80" />
            <el-table-column prop="created_at" label="变更时间" width="180" />
          </el-table>
          <el-pagination
            v-if="changesTotal > 0"
            style="margin-top: 12px; justify-content: flex-end;"
            background layout="total, prev, pager, next"
            :total="changesTotal" :page-size="20" :current-page="changesPage"
            @current-change="handleChangesPage"
          />
          <el-empty v-if="!changesLoading && changes.length === 0" description="暂无变更记录" />
        </el-tab-pane>
      </el-tabs>
    </el-drawer>

    <!-- 新增/编辑 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="600px">
      <el-form :model="form" label-width="80px">
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="主机名"><el-input v-model="form.hostname" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="公网IP"><el-input v-model="form.ip" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="内网IP"><el-input v-model="form.inner_ip" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="操作系统"><el-input v-model="form.os" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="CPU(核)"><el-input-number v-model="form.cpu_cores" :min="0" style="width: 100%;" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="内存MB"><el-input-number v-model="form.memory_mb" :min="0" style="width: 100%;" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="磁盘GB"><el-input-number v-model="form.disk_gb" :min="0" style="width: 100%;" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="机房"><el-input v-model="form.idc" /></el-form-item></el-col>
          <el-col :span="12">
            <el-form-item label="所属服务">
              <el-tree-select
                v-model="form.service_tree_id"
                :data="serviceTreeOptions"
                :props="{ children: 'children', label: 'name', value: 'id' }"
                placeholder="选择服务树节点"
                clearable
                check-strictly
                style="width: 100%;"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12"><el-form-item label="SN"><el-input v-model="form.sn" /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" /></el-form-item></el-col>
        </el-row>
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
</style>
