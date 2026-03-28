<script setup lang="ts">
defineOptions({ name: 'CicdProjects' })
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { cicdProjectApi } from '../api'

const projects = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const loading = ref(false)
const searchKeyword = ref('')
const statusFilter = ref('')

const formVisible = ref(false)
const formLoading = ref(false)
const isEdit = ref(false)
const editingId = ref<number | null>(null)
const form = ref({ name: '', code: '', repository: '', default_branch: 'main', description: '', active: 1 })

function buildListParams() {
  const params: Record<string, any> = { page: page.value, size: size.value }
  if (searchKeyword.value) params.keyword = searchKeyword.value
  if (statusFilter.value !== '') params.active = Number(statusFilter.value)
  return params
}

async function loadProjects() {
  loading.value = true
  try {
    const res: any = await cicdProjectApi.list(buildListParams())
    projects.value = res.data.list || []
    total.value = res.data.total || 0
  } catch {} finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  loadProjects()
}

function handleReset() {
  searchKeyword.value = ''
  statusFilter.value = ''
  page.value = 1
  loadProjects()
}

function openCreate() {
  isEdit.value = false
  editingId.value = null
  form.value = { name: '', code: '', repository: '', default_branch: 'main', description: '', active: 1 }
  formVisible.value = true
}

function openEdit(row: any) {
  isEdit.value = true
  editingId.value = row.id
  form.value = {
    name: row.name || '',
    code: row.code || '',
    repository: row.repository || '',
    default_branch: row.default_branch || 'main',
    description: row.description || '',
    active: row.active === 1 ? 1 : 0,
  }
  formVisible.value = true
}

async function submitForm() {
  if (!form.value.name || !form.value.repository) {
    ElMessage.warning('请填写项目名称和仓库地址')
    return
  }
  formLoading.value = true
  try {
    const payload = { ...form.value }
    if (isEdit.value && editingId.value) {
      await cicdProjectApi.update(editingId.value, payload)
      ElMessage.success('项目更新成功')
    } else {
      await cicdProjectApi.create(payload)
      ElMessage.success('项目创建成功')
    }
    formVisible.value = false
    loadProjects()
  } finally {
    formLoading.value = false
  }
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`确定删除项目 ${row.name}？`, '提示', { type: 'warning' })
  await cicdProjectApi.delete(row.id)
  ElMessage.success('项目已删除')
  loadProjects()
}

async function toggleStatus(row: any) {
  const nextEnabled = row.status !== 1
  await cicdProjectApi.toggleStatus(row.id, nextEnabled)
  ElMessage.success(nextEnabled ? '项目已启用' : '项目已禁用')
  loadProjects()
}

function handlePageChange(current: number) {
  page.value = current
  loadProjects()
}

function handleSizeChange(currentSize: number) {
  size.value = currentSize
  page.value = 1
  loadProjects()
}

onMounted(loadProjects)
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div class="header-actions">
          <span>CI/CD 项目</span>
          <el-button type="primary" @click="openCreate"><el-icon><Plus /></el-icon> 新增项目</el-button>
        </div>
      </template>

      <el-alert
        v-if="projects.length > 0"
        class="webhook-alert"
        title="Webhook 配置入口"
        type="info"
        effect="light"
        show-icon
        description="在流水线管理界面可以直接配置 Webhook 开关、密钥和触发地址，建议将生成的地址保存到仓库的推送/合并回调里。"
      >
        <template #description>
          <span>Webhook 触发地址以流水线编码为唯一标识，<router-link to="/cicd/pipelines" target="_self">立即查看流水线</router-link>，可复制地址与密钥示例到 Git 平台的 Webhook 配置中。</span>
        </template>
      </el-alert>

      <el-form label-width="0" :inline="true" class="filter-row" @submit.prevent="handleSearch">
        <el-form-item>
          <el-input
            v-model="searchKeyword"
            placeholder="项目/编码/仓库"
            clearable
            style="width: 240px"
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item>
          <el-select v-model="statusFilter" placeholder="状态" clearable style="width: 120px">
            <el-option label="全部" value="" />
            <el-option label="启用" value="1" />
            <el-option label="禁用" value="0" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="projects" v-loading="loading" stripe border style="margin-top: 16px">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="项目名称" min-width="180" />
        <el-table-column prop="code" label="项目编码" width="140" />
        <el-table-column prop="repository" label="仓库" min-width="220" show-overflow-tooltip />
        <el-table-column prop="default_branch" label="默认分支" width="120" />
        <el-table-column prop="owner_name" label="负责人" width="140" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 1 ? 'success' : 'info'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column prop="updated_at" label="更新时间" width="180" />
        <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip />
        <el-table-column label="操作" fixed="right" width="250">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" @click="toggleStatus(row)">{{ row.active === 1 ? '禁用' : '启用' }}</el-button>
            <el-button type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-if="total > 0"
        style="margin-top: 16px; text-align: right"
        :current-page="page"
        :page-size="size"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        show-size-picker
        show-quick-jumper
        @current-change="handlePageChange"
        @size-change="handleSizeChange"
      />
    </el-card>

    <el-dialog v-model="formVisible" :title="isEdit ? '编辑项目' : '新增项目'" width="520px">
      <el-form label-width="110px">
        <el-form-item label="项目名称">
          <el-input v-model="form.name" placeholder="填写项目名称" />
        </el-form-item>
        <el-form-item label="项目编码">
          <el-input v-model="form.code" placeholder="英文/数字组合，留空自动生成" />
        </el-form-item>
        <el-form-item label="仓库地址">
          <el-input v-model="form.repository" placeholder="https://git.example.com/repo.git" />
        </el-form-item>
        <el-form-item label="默认分支">
          <el-input v-model="form.default_branch" placeholder="main" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input type="textarea" v-model="form.description" placeholder="可选的描述" rows="3" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.active" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">取消</el-button>
        <el-button type="primary" :loading="formLoading" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page {
  padding: 20px;
}

.header-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.filter-row {
  margin-top: 8px;
  flex-wrap: wrap;
}

.webhook-alert {
  margin: 20px 0 0;
  border-radius: 8px;
}
</style>
