<template>
  <div class="p-6">
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-800">任务模板</h1>
        <p class="text-gray-500 mt-1">管理可复用的任务模板，支持版本控制和变量定义</p>
      </div>
      <div class="flex gap-3">
        <el-button type="primary" @click="createTemplate">
          <el-icon><Plus /></el-icon>
          新建模板
        </el-button>
        <el-button @click="refreshList">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 搜索和筛选 -->
    <div class="bg-white rounded-2xl shadow-sm p-4 mb-6 flex gap-4">
      <el-input v-model="searchQuery" placeholder="搜索模板名称、标签..." class="w-80" clearable>
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
      
      <el-select v-model="categoryFilter" placeholder="分类" class="w-40">
        <el-option label="全部" value="" />
        <el-option label="运维" value="ops" />
        <el-option label="安全" value="security" />
        <el-option label="云管" value="cloud" />
        <el-option label="业务" value="business" />
      </el-select>

      <el-select v-model="statusFilter" placeholder="状态" class="w-40">
        <el-option label="全部" value="" />
        <el-option label="启用" value="1" />
        <el-option label="禁用" value="0" />
      </el-select>
    </div>

    <!-- 模板列表 -->
    <el-table :data="templates" stripe style="width: 100%" class="rounded-2xl overflow-hidden">
      <el-table-column prop="name" label="模板名称" width="280" />
      <el-table-column prop="version" label="版本" width="100" />
      <el-table-column prop="category" label="分类" width="110">
        <template #default="{ row }">
          <el-tag :type="getCategoryType(row.category)" size="small">{{ getCategoryName(row.category) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="task_type" label="类型" width="100" />
      <el-table-column prop="tags" label="标签">
        <template #default="{ row }">
          <div class="flex gap-1 flex-wrap">
            <el-tag v-for="tag in (row.tags || '').split(',').filter(Boolean)" 
                    :key="tag" size="small" type="info">{{ tag }}</el-tag>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" show-overflow-tooltip />
      <el-table-column prop="status" label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
            {{ row.status === 1 ? '启用' : '禁用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link @click="editTemplate(row)">编辑</el-button>
          <el-button type="success" link @click="executeTemplate(row)">立即执行</el-button>
          <el-button type="danger" link @click="deleteTemplate(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页 -->
    <div class="flex justify-between items-center mt-6">
      <div class="text-sm text-gray-500">
        共 {{ total }} 个模板
      </div>
      <el-pagination 
        v-model:current-page="currentPage"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next, jumper"
        @current-change="handlePageChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const searchQuery = ref('')
const categoryFilter = ref('')
const statusFilter = ref('')
const templates = ref<any[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(15)

const getCategoryType = (category: string) => {
  const map: any = { ops: 'success', security: 'warning', cloud: 'primary', business: 'info' }
  return map[category] || 'info'
}

const getCategoryName = (category: string) => {
  const map: any = { ops: '运维', security: '安全', cloud: '云管', business: '业务' }
  return map[category] || category
}

const loadTemplates = () => {
  // 模拟数据，后续替换为API调用
  templates.value = [
    {
      id: 1,
      name: "重启服务",
      version: "v1.2.0",
      category: "ops",
      task_type: "shell",
      tags: "restart,service,common",
      description: "优雅重启指定服务，支持健康检查",
      status: 1
    },
    {
      id: 2,
      name: "清理日志",
      version: "v2.0.1",
      category: "ops",
      task_type: "shell",
      tags: "log,cleanup",
      description: "清理超过30天的日志文件",
      status: 1
    },
    {
      id: 3,
      name: "安全基线检查",
      version: "v1.0.0",
      category: "security",
      task_type: "shell",
      tags: "security,baseline",
      description: "执行CIS安全基线检查",
      status: 1
    }
  ]
  total.value = templates.value.length
}

const createTemplate = () => {
  ElMessage.info('新建任务模板功能开发中...')
}

const editTemplate = (row: any) => {
  ElMessage.info(`编辑模板: ${row.name}`)
}

const executeTemplate = (row: any) => {
  ElMessage.success(`开始执行: ${row.name}`)
}

const deleteTemplate = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定删除模板 "${row.name}" 吗？`, '警告', {
      type: 'warning'
    })
    ElMessage.success('删除成功')
    loadTemplates()
  } catch {}
}

const refreshList = () => {
  loadTemplates()
  ElMessage.success('刷新成功')
}

const handlePageChange = () => {
  loadTemplates()
}

onMounted(() => {
  loadTemplates()
})
</script>