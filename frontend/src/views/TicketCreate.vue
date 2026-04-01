<script setup lang="ts">
defineOptions({ name: 'TicketCreate' })
import { computed, nextTick, onActivated, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ticketApi, ticketTypeApi, requestTemplateApi, assetApi, cloudAccountApi, serviceTreeApi, departmentApi, userApi } from '../api'
import { useViewStateStore } from '../stores/viewState'

const router = useRouter()
const viewStateStore = useViewStateStore()
const step = ref(1)
const allTypes = ref<any[]>([])
const allRequestTemplates = ref<any[]>([])
const selectedTemplate = ref<any>(null)

const form = ref<any>({
  title: '', type_id: 0, priority: 'medium', description: '',
  resource_type: '', resource_id: undefined, resource_ids: [] as number[], handle_dept_id: undefined, assignee_id: undefined,
  request_template_id: 0, ticket_kind: 'incident',
})

// 资源选择
const cloudAccountOptions = ref<any[]>([])
const serviceTreeOptions = ref<any[]>([])
const serviceTreeAssetCounts = ref<Record<number, number>>({})
const allDepts = ref<any[]>([])
const allUsers = ref<any[]>([])
const submitting = ref(false)

const assetDialogVisible = ref(false)
const assetDialogLoading = ref(false)
const assetDialogData = ref<any[]>([])
const assetDialogTotal = ref(0)
const assetTableRef = ref<any>(null)
const assetDialogQuery = ref({
  page: 1,
  size: 15,
  keyword: '',
  service_tree_id: undefined as number | undefined,
})
const selectedServiceTreeNode = ref<number | undefined>(undefined)
const selectedAssets = ref<any[]>([])
const draftSelectedAssets = ref<Record<number, any>>({})
const draftSelectedAssetIDs = ref<number[]>([])
const isSyncingAssetSelection = ref(false)
const seenTicketTypeVersion = ref(0)
const seenRequestTemplateVersion = ref(0)
const requestFormData = ref<Record<string, any>>({})

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
]

interface RequestFormFieldOption {
  label: string
  value: string | number | boolean
}

interface RequestFormField {
  key: string
  label: string
  type: 'text' | 'textarea' | 'number' | 'select' | 'switch'
  required?: boolean
  placeholder?: string
  rows?: number
  options?: RequestFormFieldOption[]
  default?: string | number | boolean
}

const categoryOptions = [
  { label: '发版申请', value: 'release' },
  { label: '权限申请', value: 'access' },
  { label: '数据库上线', value: 'db_release' },
  { label: '代码仓库', value: 'repo' },
  { label: '其他', value: 'other' },
]

const categoryLabelMap = categoryOptions.reduce<Record<string, string>>((acc, item) => {
  acc[item.value] = item.label
  return acc
}, {})

const requestSchemaFields = computed<RequestFormField[]>(() => {
  if (!selectedTemplate.value?.form_schema) return []
  try {
    const parsed = JSON.parse(selectedTemplate.value.form_schema)
    const fields = Array.isArray(parsed?.fields) ? parsed.fields : []
    return fields.filter((field: any) => field?.key && field?.label)
  } catch {
    return []
  }
})

const groupedTemplates = computed(() => {
  const groups: Record<string, any[]> = {}
  for (const tpl of allRequestTemplates.value) {
    const rawCat = tpl.category || 'other'
    const catName = categoryLabelMap[rawCat] || rawCat
    if (!groups[catName]) groups[catName] = []
    groups[catName].push(tpl)
  }
  return groups
})

async function loadTicketTypes() {
  const [typeRes, templateRes] = await Promise.all([
    ticketTypeApi.all(),
    requestTemplateApi.list(true),
  ])
  allTypes.value = (typeRes as any).data || []
  allRequestTemplates.value = (templateRes as any).data || []
}

async function loadCreatePageOptions() {
  const [typesRes, templatesRes, deptsRes, usersRes, accountsRes, treeRes, treeCountRes] = await Promise.allSettled([
    ticketTypeApi.all(),
    requestTemplateApi.list(true),
    departmentApi.all(),
    userApi.list(1, 200),
    cloudAccountApi.list(1, 100),
    serviceTreeApi.tree(),
    serviceTreeApi.assetCounts(),
  ])
  if (typesRes.status === 'fulfilled') allTypes.value = (typesRes.value as any).data || []
  if (templatesRes.status === 'fulfilled') allRequestTemplates.value = (templatesRes.value as any).data || []
  if (deptsRes.status === 'fulfilled') allDepts.value = (deptsRes.value as any).data || []
  if (usersRes.status === 'fulfilled') allUsers.value = (usersRes.value as any).data?.list || []
  if (accountsRes.status === 'fulfilled') cloudAccountOptions.value = ((accountsRes.value as any).data?.list || []).map((a: any) => ({ id: a.id, label: a.name }))
  if (treeRes.status === 'fulfilled') serviceTreeOptions.value = (treeRes.value as any).data || []
  if (treeCountRes.status === 'fulfilled') serviceTreeAssetCounts.value = (treeCountRes.value as any).data || {}
}

function selectRequestTemplate(template: any) {
  selectedTemplate.value = template
  form.value.type_id = template.type_id || 0
  form.value.request_template_id = template.id
  form.value.ticket_kind = template.ticket_kind || 'request'
  const matchedType = allTypes.value.find((item: any) => item.id === template.type_id)
  form.value.priority = matchedType?.priority || 'medium'
  form.value.handle_dept_id = matchedType?.handle_dept_id || undefined
  requestFormData.value = buildDefaultRequestFormData(template.form_schema)
  step.value = 2
}

function buildDefaultRequestFormData(rawSchema?: string) {
  const result: Record<string, any> = {}
  if (!rawSchema) return result
  try {
    const parsed = JSON.parse(rawSchema)
    const fields = Array.isArray(parsed?.fields) ? parsed.fields : []
    for (const field of fields) {
      if (!field?.key) continue
      if (field.default !== undefined) {
        result[field.key] = field.default
        continue
      }
      switch (field.type) {
        case 'switch':
          result[field.key] = false
          break
        case 'number':
          result[field.key] = undefined
          break
        default:
          result[field.key] = ''
      }
    }
  } catch {}
  return result
}

function getNodeAssetCount(node: any): number {
  let count = serviceTreeAssetCounts.value[node.id] || 0
  if (node.children?.length) {
    for (const child of node.children) {
      count += getNodeAssetCount(child)
    }
  }
  return count
}

async function handleResourceTypeChange(value?: string) {
  form.value.resource_id = undefined
  form.value.resource_ids = []
  selectedAssets.value = []
  draftSelectedAssets.value = {}
  assetDialogVisible.value = false
  selectedServiceTreeNode.value = undefined
  assetDialogQuery.value = { page: 1, size: 15, keyword: '', service_tree_id: undefined }
  if (value === 'asset') {
    openAssetDialog()
  }
}

async function fetchAssetDialogData() {
  assetDialogLoading.value = true
  try {
    const params: any = {
      page: assetDialogQuery.value.page,
      size: assetDialogQuery.value.size,
      keyword: assetDialogQuery.value.keyword || '',
    }
    if (assetDialogQuery.value.service_tree_id) {
      params.service_tree_id = assetDialogQuery.value.service_tree_id
      params.recursive = true
    }
    const res: any = await assetApi.list(params)
    assetDialogData.value = res.data?.list || []
    assetDialogTotal.value = res.data?.total || 0
    await nextTick()
    syncAssetSelection()
  } finally {
    assetDialogLoading.value = false
  }
}

function openAssetDialog() {
  draftSelectedAssets.value = Object.fromEntries(selectedAssets.value.map(asset => [asset.id, { ...asset }]))
  draftSelectedAssetIDs.value = selectedAssets.value.map(asset => asset.id)
  assetDialogVisible.value = true
  fetchAssetDialogData()
}

function handleAssetDialogSearch() {
  assetDialogQuery.value.page = 1
  fetchAssetDialogData()
}

function handleAssetTreeClick(node: any) {
  selectedServiceTreeNode.value = node.id
  assetDialogQuery.value.service_tree_id = node.id
  assetDialogQuery.value.page = 1
  fetchAssetDialogData()
}

function resetAssetTreeFilter() {
  selectedServiceTreeNode.value = undefined
  assetDialogQuery.value.service_tree_id = undefined
  assetDialogQuery.value.page = 1
  fetchAssetDialogData()
}

function handleAssetSelectionChange(rows: any[]) {
  if (isSyncingAssetSelection.value) return
  const currentPageIDs = new Set(assetDialogData.value.map(item => item.id))
  const nextMap = { ...draftSelectedAssets.value }
  for (const id of currentPageIDs) {
    if (!rows.some(row => row.id === id)) {
      delete nextMap[id]
    }
  }
  for (const row of rows) {
    nextMap[row.id] = row
  }
  draftSelectedAssets.value = nextMap

  const rowMap = new Map(rows.map(row => [row.id, row]))
  const nextIDs = draftSelectedAssetIDs.value.filter(id => !(currentPageIDs.has(id) && !rowMap.has(id)))
  for (const row of assetDialogData.value) {
    if (rowMap.has(row.id) && !nextIDs.includes(row.id)) {
      nextIDs.push(row.id)
    }
  }
  draftSelectedAssetIDs.value = nextIDs
}

function handleAssetRowDoubleClick(row: any) {
  const nextMap = { ...draftSelectedAssets.value, [row.id]: row }
  draftSelectedAssets.value = nextMap
  if (!draftSelectedAssetIDs.value.includes(row.id)) {
    draftSelectedAssetIDs.value = [...draftSelectedAssetIDs.value, row.id]
  }
  syncAssetSelection()
  confirmSelectedAsset()
}

function confirmSelectedAsset() {
  const assets = draftSelectedAssetIDs.value
    .map(id => draftSelectedAssets.value[id])
    .filter(Boolean)
  if (assets.length === 0) {
    ElMessage.warning('请至少选择一台主机资产')
    return
  }
  selectedAssets.value = assets
  form.value.resource_ids = assets.map((asset: any) => asset.id)
  form.value.resource_id = assets[0]?.id || undefined
  assetDialogVisible.value = false
}

async function submitForm() {
  if (!form.value.title) { ElMessage.warning('请输入工单标题'); return }
  if (!form.value.type_id && !form.value.request_template_id) {
    ElMessage.warning('请选择工单模板')
    return
  }
  for (const field of requestSchemaFields.value) {
    if (!field.required) continue
    const value = requestFormData.value[field.key]
    if (value === '' || value === undefined || value === null) {
      ElMessage.warning(`请填写${field.label}`)
      return
    }
  }
  submitting.value = true
  try {
    const payload = {
      ...form.value,
      resource_id: form.value.resource_id || 0,
      handle_dept_id: form.value.handle_dept_id || 0,
      assignee_id: form.value.assignee_id || 0,
      request_template_id: form.value.request_template_id || 0,
      ticket_kind: form.value.ticket_kind || 'incident',
      extra_fields: selectedTemplate.value ? { request_form: requestFormData.value } : undefined,
    }
    const res: any = await ticketApi.create(payload)
    viewStateStore.markTicketListDirty()
    ElMessage.success('工单发起成功')
    router.push('/ticket/detail/' + res.data.id)
  } catch {} finally { submitting.value = false }
}

function syncAssetSelection() {
  if (!assetTableRef.value) return
  isSyncingAssetSelection.value = true
  assetTableRef.value.clearSelection()
  for (const row of assetDialogData.value) {
    if (draftSelectedAssets.value[row.id]) {
      assetTableRef.value.toggleRowSelection(row, true)
    }
  }
  isSyncingAssetSelection.value = false
}

onMounted(() => {
  loadCreatePageOptions()
  seenTicketTypeVersion.value = viewStateStore.ticketTypeVersion
  seenRequestTemplateVersion.value = viewStateStore.requestTemplateVersion
})

onActivated(() => {
  if (seenTicketTypeVersion.value !== viewStateStore.ticketTypeVersion) {
    seenTicketTypeVersion.value = viewStateStore.ticketTypeVersion
    loadTicketTypes().catch(() => {})
  }
  if (seenRequestTemplateVersion.value !== viewStateStore.requestTemplateVersion) {
    seenRequestTemplateVersion.value = viewStateStore.requestTemplateVersion
    loadTicketTypes().catch(() => {})
  }
})
</script>

<template>
  <div class="h-full flex flex-col">
    <div class="mb-6 flex items-start flex-col gap-2">
      <el-button v-if="step === 2" link @click="step = 1" class="!text-gray-500 hover:!text-indigo-600 transition-colors -ml-1">
        <el-icon class="mr-1"><ArrowLeft /></el-icon> 返回模板列表
      </el-button>
      <div class="flex items-center justify-between w-full">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 tracking-tight">发起工单</h1>
          <p class="text-sm text-gray-500 mt-1">按步骤选择服务模板并填写详细请求信息</p>
        </div>
      </div>
    </div>

    <!-- Step 1: 选择模板 -->
    <div v-if="step === 1" class="space-y-6">
      <div v-if="Object.keys(groupedTemplates).length > 0">
        <div v-for="(templates, category) in groupedTemplates" :key="category" class="bg-white rounded-xl shadow-sm border border-gray-100 p-6 mb-6">
          <h3 class="text-base font-medium text-gray-800 mb-4 flex items-center gap-2">
            <span class="w-1.5 h-4 bg-indigo-500 rounded-full"></span>
            {{ category }}
            <span class="text-xs font-normal text-gray-400 ml-2 bg-gray-100 px-2 py-0.5 rounded-full">{{ templates.length }}</span>
          </h3>
          
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            <div 
              v-for="template in templates" 
              :key="'template-' + template.id"
              class="group relative bg-white border border-gray-200 rounded-xl p-5 cursor-pointer hover:border-indigo-500 hover:shadow-md transition-all duration-300 flex flex-col gap-3"
              @click="selectRequestTemplate(template)"
            >
              <div class="flex items-center justify-between">
                <div class="w-12 h-12 rounded-xl bg-green-50 text-green-600 flex items-center justify-center group-hover:bg-indigo-50 group-hover:text-indigo-600 transition-colors">
                  <el-icon class="text-2xl"><component :is="template.icon || 'DocumentAdd'" /></el-icon>
                </div>
                <el-icon class="text-gray-300 opacity-0 group-hover:opacity-100 group-hover:translate-x-1 group-hover:text-indigo-500 transition-all text-xl"><ArrowRight /></el-icon>
              </div>
              <div>
                <h4 class="text-base font-semibold text-gray-800 group-hover:text-indigo-600 transition-colors">{{ template.name }}</h4>
                <p class="text-xs text-gray-500 mt-1 line-clamp-2">{{ template.description || '暂无描述信息' }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
      
      <div v-else class="flex flex-col items-center justify-center py-16 bg-white shadow-sm rounded-xl border border-dashed border-gray-200">
        <el-icon class="text-gray-300 text-5xl mb-3"><DocumentDelete /></el-icon>
        <p class="text-gray-500 text-sm">暂无工单模板，请联系管理员创建</p>
      </div>
    </div>

    <!-- Step 2: 填写表单 -->
    <div v-if="step === 2" class="space-y-6">
      <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-8 relative overflow-hidden">
        <!-- 装饰背景 -->
        <div class="absolute top-0 right-0 w-64 h-64 bg-indigo-50 rounded-bl-full -mr-16 -mt-16 opacity-50 z-0 pointer-events-none"></div>
        
        <div class="relative z-10">
          <div class="flex items-center gap-3 mb-8 pb-4 border-b border-gray-100">
            <div class="w-10 h-10 rounded-lg bg-indigo-100 text-indigo-600 flex items-center justify-center">
              <el-icon class="text-xl"><EditPen /></el-icon>
            </div>
            <div>
              <h2 class="text-lg font-bold text-gray-800">{{ selectedTemplate?.name }}</h2>
              <p class="text-xs text-gray-500 mt-0.5">请详细填写以下申请信息，带 * 为必填项</p>
            </div>
          </div>

          <el-form :model="form" label-width="110px" class="w-full" label-position="right" require-asterisk-position="right">
            <el-form-item label="工单标题" required class="font-medium">
              <el-input v-model="form.title" placeholder="请简明扼要地描述您的申请或问题" size="large" class="!rounded-md" />
            </el-form-item>
            
            <el-form-item label="优先级">
              <el-select v-model="form.priority" class="w-full" size="large">
                <template #prefix>
                  <el-icon class="text-gray-400"><WarnTriangleFilled /></el-icon>
                </template>
                <el-option v-for="o in priorityOptions" :key="o.value" :label="o.label" :value="o.value">
                  <span class="flex items-center justify-between w-full">
                    <span>{{ o.label }}</span>
                    <span class="w-2 h-2 rounded-full" :class="{'bg-red-500': o.value==='high', 'bg-orange-400': o.value==='medium', 'bg-green-500': o.value==='low'}"></span>
                  </span>
                </el-option>
              </el-select>
            </el-form-item>
            
            <el-form-item label="详细描述">
              <el-input v-model="form.description" type="textarea" :rows="4" placeholder="请详细描述问题背景、影响范围或期望达到的处理结果..." class="!rounded-md font-normal text-sm" />
            </el-form-item>
            
            <div v-if="requestSchemaFields.length > 0" class="my-8 pt-6 border-t border-gray-100">
              <h3 class="text-sm font-semibold text-gray-700 mb-6 flex items-center gap-2">
                <el-icon class="text-indigo-500"><Document /></el-icon> 附加表单信息
              </h3>
              
              <template v-for="field in requestSchemaFields" :key="field.key">
                <el-form-item :label="field.label" :required="field.required">
                  <el-input
                    v-if="field.type === 'text'"
                    v-model="requestFormData[field.key]"
                    :placeholder="field.placeholder || `请输入${field.label}`"
                    size="large"
                  />
                  <el-input
                    v-else-if="field.type === 'textarea'"
                    v-model="requestFormData[field.key]"
                    type="textarea"
                    :rows="field.rows || 3"
                    :placeholder="field.placeholder || `请输入${field.label}`"
                  />
                  <el-input-number
                    v-else-if="field.type === 'number'"
                    v-model="requestFormData[field.key]"
                    class="!w-full"
                    size="large"
                  />
                  <el-select
                    v-else-if="field.type === 'select'"
                    v-model="requestFormData[field.key]"
                    clearable
                    :placeholder="field.placeholder || `请选择${field.label}`"
                    class="w-full"
                    size="large"
                  >
                    <el-option v-for="option in field.options || []" :key="String(option.value)" :label="option.label" :value="option.value" />
                  </el-select>
                  <el-switch
                    v-else-if="field.type === 'switch'"
                    v-model="requestFormData[field.key]"
                    class="mt-1.5"
                  />
                  <el-input
                    v-else
                    v-model="requestFormData[field.key]"
                    :placeholder="field.placeholder || `请输入${field.label}`"
                    size="large"
                  />
                </el-form-item>
              </template>
            </div>

            <div class="my-8 pt-6 border-t border-gray-100">
              <h3 class="text-sm font-semibold text-gray-700 mb-6 flex items-center gap-2">
                <el-icon class="text-indigo-500"><Connection /></el-icon> 资源与派发
              </h3>
              
              <el-form-item label="关联资源">
                <div class="flex flex-col sm:flex-row gap-3 w-full">
                  <el-select v-model="form.resource_type" placeholder="选择资源类型" clearable class="w-full sm:w-40 flex-shrink-0" size="large" @change="handleResourceTypeChange">
                    <el-option v-for="o in resourceTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
                  </el-select>
                  
                  <!-- 资产选择器 -->
                  <div v-if="form.resource_type === 'asset'" class="w-full">
                    <div
                      class="flex flex-col border border-gray-200 rounded-lg p-3 cursor-pointer transition-all hover:border-indigo-400 group bg-gray-50/50 hover:bg-indigo-50/30"
                      :class="{ 'border-dashed border-gray-300 bg-gray-50': !selectedAssets.length }"
                      @click="openAssetDialog"
                    >
                      <div class="flex items-center justify-between mb-2">
                        <div class="flex items-center gap-2 text-sm">
                          <el-icon class="text-gray-400 group-hover:text-indigo-500"><Monitor /></el-icon>
                          <span class="font-medium text-gray-700">主机资产</span>
                          <span class="text-xs px-2 py-0.5 rounded-full bg-indigo-100 text-indigo-700 ml-2">
                            {{ selectedAssets.length ? `已选 ${selectedAssets.length} 台` : '未选择' }}
                          </span>
                        </div>
                        <span class="text-xs text-indigo-600 font-medium opacity-0 group-hover:opacity-100 transition-opacity">
                          {{ selectedAssets.length ? '重新选择' : '点击选择' }} <el-icon><ArrowRight /></el-icon>
                        </span>
                      </div>

                      <div v-if="selectedAssets.length" class="flex flex-wrap gap-2 mt-2">
                        <span
                          v-for="asset in selectedAssets"
                          :key="asset.id"
                          class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-white border border-gray-200 text-xs text-gray-600 shadow-sm"
                          :title="`${asset.hostname} (${asset.ip})`"
                        >
                          <el-icon class="text-green-500"><CircleCheckFilled /></el-icon>
                          {{ asset.hostname }} <span class="text-gray-400 ml-1">{{ asset.ip }}</span>
                        </span>
                      </div>
                      <div v-else class="text-xs text-gray-400 mt-1 pl-6">
                        点击此处打开资产选择窗口，选择关联的主机设备。
                      </div>
                    </div>
                  </div>
                  
                  <!-- 云账号 -->
                  <el-select v-if="form.resource_type === 'cloud_account'" v-model="form.resource_id" placeholder="请选择云账号" class="w-full" size="large">
                    <el-option v-for="a in cloudAccountOptions" :key="a.id" :label="a.label" :value="a.id" />
                  </el-select>
                </div>
              </el-form-item>
              
              <el-form-item label="处理部门">
                <el-select v-model="form.handle_dept_id" placeholder="可留空，若不选将根据路由规则指派" clearable class="w-full" size="large">
                  <template #prefix><el-icon class="text-gray-400"><OfficeBuilding /></el-icon></template>
                  <el-option v-for="d in allDepts" :key="d.id" :label="d.name" :value="d.id" />
                </el-select>
              </el-form-item>
              
              <el-form-item label="处理人">
                <el-select v-model="form.assignee_id" placeholder="可留空，自动分派给部门人员" clearable class="w-full" size="large">
                  <template #prefix><el-icon class="text-gray-400"><User /></el-icon></template>
                  <el-option v-for="u in allUsers" :key="u.id" :label="u.real_name || u.username" :value="u.id" />
                </el-select>
              </el-form-item>
            </div>

            <div class="mt-10 pt-6 border-t border-gray-100 flex items-center justify-end gap-4">
              <el-button @click="step = 1" size="large">取消</el-button>
              <el-button type="primary" :loading="submitting" @click="submitForm" size="large" class="!px-8 !rounded-md shadow-sm shadow-indigo-200">
                <el-icon class="mr-2"><Promotion /></el-icon> 提交工单申请
              </el-button>
            </div>
          </el-form>
        </div>
      </div>
    </div>

    <el-dialog v-model="assetDialogVisible" title="资产管理" width="1040px" top="6vh" class="!rounded-xl overflow-hidden" :body-style="{ padding: '0px' }">
      <div class="flex h-[600px] bg-white">
        <aside class="w-72 border-r border-slate-200 flex flex-col bg-slate-50">
          <div class="flex border-b border-slate-200 bg-white">
            <span class="flex-1 text-center py-3 text-sm cursor-pointer border-b-2 border-indigo-500 text-indigo-600 font-bold">资产树</span>
          </div>
          <div class="p-3 bg-white border-b border-slate-100">
            <el-button link :type="!selectedServiceTreeNode ? 'primary' : 'default'" @click="resetAssetTreeFilter">全部资产</el-button>
          </div>
          <el-scrollbar class="flex-1 p-2">
            <el-tree
              :data="serviceTreeOptions"
              node-key="id"
              :current-node-key="selectedServiceTreeNode"
              :props="{ children: 'children', label: 'name' }"
              default-expand-all
              highlight-current
              :expand-on-click-node="false"
              @node-click="handleAssetTreeClick"
            >
              <template #default="{ data }">
                <div class="w-full flex items-center justify-between gap-2 pr-1.5">
                  <span class="truncate">{{ data.name }}</span>
                  <span class="min-w-[22px] h-[22px] px-1.5 rounded-full bg-emerald-50 text-teal-700 text-xs leading-[22px] text-center">{{ getNodeAssetCount(data) }}</span>
                </div>
              </template>
            </el-tree>
          </el-scrollbar>
        </aside>

        <section class="flex-1 p-5 bg-gradient-to-b from-slate-50 to-slate-100 flex flex-col overflow-hidden">
          <div class="flex items-center gap-3 mb-4">
            <el-input
              v-model="assetDialogQuery.keyword"
              placeholder="搜索主机名 / IP"
              clearable
              class="max-w-xs"
              @keyup.enter="handleAssetDialogSearch"
            >
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
            <el-button type="primary" plain @click="handleAssetDialogSearch">搜索</el-button>
          </div>

          <el-table
            ref="assetTableRef"
            :data="assetDialogData"
            v-loading="assetDialogLoading"
            border
            class="flex-1 h-0 shadow-sm rounded-lg"
            row-key="id"
            @selection-change="handleAssetSelectionChange"
            @row-dblclick="handleAssetRowDoubleClick"
          >
            <el-table-column type="selection" width="54" reserve-selection />
            <el-table-column prop="hostname" label="名称" min-width="180" show-overflow-tooltip />
            <el-table-column prop="ip" label="地址" width="160" />
            <el-table-column prop="source" label="平台" width="110">
              <template #default="{ row }">
                {{ row.source || 'manual' }}
              </template>
            </el-table-column>
            <el-table-column prop="service_tree_name" label="归属服务" min-width="160" show-overflow-tooltip />
          </el-table>

          <div class="flex items-center justify-between gap-4 mt-4">
            <div class="flex items-center gap-2 min-w-0">
              <span class="text-slate-500 text-sm">已选资产</span>
              <span v-if="Object.keys(draftSelectedAssets).length" class="text-slate-800 font-semibold">
                {{ Object.keys(draftSelectedAssets).length }} 台
              </span>
              <span v-else class="text-slate-400">未选择</span>
            </div>
            <el-pagination
              v-if="assetDialogTotal > 0"
              background
              layout="sizes, prev, pager, next"
              :total="assetDialogTotal"
              :page-size="assetDialogQuery.size"
              :current-page="assetDialogQuery.page"
              :page-sizes="[15, 30, 50]"
              @size-change="(size: number) => { assetDialogQuery.size = size; assetDialogQuery.page = 1; fetchAssetDialogData() }"
              @current-change="(page: number) => { assetDialogQuery.page = page; fetchAssetDialogData() }"
            />
          </div>
        </section>
      </div>

      <template #footer>
        <div class="border-t border-slate-100 pt-3 flex justify-end gap-2 pr-2">
          <el-button @click="assetDialogVisible = false" class="!rounded-lg">取消</el-button>
          <el-button type="primary" @click="confirmSelectedAsset" class="!rounded-lg">确认</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
/* Scoped styles removed in favor of Tailwind */
</style>
