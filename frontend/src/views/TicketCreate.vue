<script setup lang="ts">
defineOptions({ name: 'TicketCreate' })
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ticketApi, ticketTypeApi, assetApi, cloudAccountApi, serviceTreeApi, departmentApi, userApi } from '../api'

const router = useRouter()
const step = ref(1)
const allTypes = ref<any[]>([])
const selectedType = ref<any>(null)

const form = ref<any>({
  title: '', type_id: 0, priority: 'medium', description: '',
  resource_type: '', resource_id: 0, handle_dept_id: 0, assignee_id: 0,
})

// 资源搜索
const assetOptions = ref<any[]>([])
const cloudAccountOptions = ref<any[]>([])
const serviceTreeOptions = ref<any[]>([])
const allDepts = ref<any[]>([])
const allUsers = ref<any[]>([])
const submitting = ref(false)

const priorityOptions = [
  { label: '低', value: 'low' },
  { label: '中', value: 'medium' },
  { label: '高', value: 'high' },
  { label: '紧急', value: 'urgent' },
]

const resourceTypeOptions = [
  { label: '不关联资源', value: '' },
  { label: '主机资产', value: 'asset' },
  { label: '云账号', value: 'cloud_account' },
  { label: '服务树', value: 'service_tree' },
]

function selectType(t: any) {
  selectedType.value = t
  form.value.type_id = t.id
  form.value.priority = t.priority || 'medium'
  form.value.handle_dept_id = t.handle_dept_id || 0
  step.value = 2
}

async function searchAssets(keyword: string) {
  if (!keyword) return
  const res: any = await assetApi.list({ page: 1, size: 20, keyword })
  assetOptions.value = (res.data?.list || []).map((a: any) => ({ id: a.id, label: `${a.hostname} (${a.ip})` }))
}

async function submitForm() {
  if (!form.value.title) { ElMessage.warning('请输入工单标题'); return }
  submitting.value = true
  try {
    const res: any = await ticketApi.create(form.value)
    ElMessage.success('工单创建成功')
    router.push('/ticket/detail/' + res.data.id)
  } catch {} finally { submitting.value = false }
}

onMounted(async () => {
  const [typesRes, deptsRes, usersRes, accountsRes, treeRes] = await Promise.allSettled([
    ticketTypeApi.all(),
    departmentApi.all(),
    userApi.list(1, 200),
    cloudAccountApi.list(1, 100),
    serviceTreeApi.tree(),
  ])
  if (typesRes.status === 'fulfilled') allTypes.value = (typesRes.value as any).data || []
  if (deptsRes.status === 'fulfilled') allDepts.value = (deptsRes.value as any).data || []
  if (usersRes.status === 'fulfilled') allUsers.value = (usersRes.value as any).data?.list || []
  if (accountsRes.status === 'fulfilled') cloudAccountOptions.value = ((accountsRes.value as any).data?.list || []).map((a: any) => ({ id: a.id, label: a.name }))
  if (treeRes.status === 'fulfilled') serviceTreeOptions.value = (treeRes.value as any).data || []
})
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header><span>创建工单</span></template>

      <!-- Step 1: 选择类型 -->
      <div v-if="step === 1">
        <p style="margin-bottom: 16px; color: #606266;">请选择工单类型：</p>
        <el-row :gutter="16">
          <el-col :span="6" v-for="t in allTypes" :key="t.id">
            <el-card shadow="hover" class="type-card" @click="selectType(t)" style="cursor: pointer; margin-bottom: 16px;">
              <div style="display: flex; align-items: center; gap: 12px;">
                <el-icon size="24" color="#409eff"><component :is="t.icon || 'Tickets'" /></el-icon>
                <div>
                  <div style="font-weight: 600;">{{ t.name }}</div>
                  <div style="font-size: 12px; color: #909399;">{{ t.description || t.code }}</div>
                </div>
              </div>
            </el-card>
          </el-col>
        </el-row>
        <el-empty v-if="allTypes.length === 0" description="暂无工单类型，请先创建" />
      </div>

      <!-- Step 2: 填写表单 -->
      <div v-if="step === 2">
        <el-button link @click="step = 1" style="margin-bottom: 16px;"><el-icon><ArrowLeft /></el-icon> 返回选择类型</el-button>
        <el-tag style="margin-left: 8px;">{{ selectedType?.name }}</el-tag>

        <el-form :model="form" label-width="100px" style="max-width: 700px; margin-top: 16px;">
          <el-form-item label="标题"><el-input v-model="form.title" placeholder="简明描述问题" /></el-form-item>
          <el-form-item label="优先级">
            <el-select v-model="form.priority" style="width: 100%;">
              <el-option v-for="o in priorityOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="描述">
            <el-input v-model="form.description" type="textarea" :rows="4" placeholder="详细描述问题、影响范围、期望处理方式" />
          </el-form-item>
          <el-form-item label="关联资源">
            <el-select v-model="form.resource_type" placeholder="选择资源类型" clearable style="width: 160px; margin-right: 8px;" @change="form.resource_id = 0">
              <el-option v-for="o in resourceTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
            <!-- 资产搜索 -->
            <el-select v-if="form.resource_type === 'asset'" v-model="form.resource_id" filterable remote :remote-method="searchAssets" placeholder="搜索主机名/IP" style="flex: 1;">
              <el-option v-for="a in assetOptions" :key="a.id" :label="a.label" :value="a.id" />
            </el-select>
            <!-- 云账号 -->
            <el-select v-if="form.resource_type === 'cloud_account'" v-model="form.resource_id" placeholder="选择云账号" style="flex: 1;">
              <el-option v-for="a in cloudAccountOptions" :key="a.id" :label="a.label" :value="a.id" />
            </el-select>
            <!-- 服务树 -->
            <el-tree-select v-if="form.resource_type === 'service_tree'" v-model="form.resource_id" :data="serviceTreeOptions" :props="{ label: 'name', value: 'id', children: 'children' }" placeholder="选择节点" clearable check-strictly style="flex: 1;" />
          </el-form-item>
          <el-form-item label="处理部门">
            <el-select v-model="form.handle_dept_id" placeholder="选择部门" clearable style="width: 100%;">
              <el-option v-for="d in allDepts" :key="d.id" :label="d.name" :value="d.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="处理人">
            <el-select v-model="form.assignee_id" placeholder="可留空（自动分派）" clearable style="width: 100%;">
              <el-option v-for="u in allUsers" :key="u.id" :label="u.real_name || u.username" :value="u.id" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="submitting" @click="submitForm">提交工单</el-button>
          </el-form-item>
        </el-form>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
.type-card:hover { border-color: #409eff; }
</style>
