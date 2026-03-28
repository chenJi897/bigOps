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
  <div class="page">
    <el-card shadow="never">
      <template #header><span>发起工单</span></template>

      <!-- Step 1: 选择模板 -->
      <div v-if="step === 1">
        <p style="margin-bottom: 16px; color: #606266;">请选择工单模板：</p>
        <el-row :gutter="16">
          <el-col :span="6" v-for="template in allRequestTemplates" :key="'template-' + template.id">
            <el-card shadow="hover" class="type-card request-card" @click="selectRequestTemplate(template)" style="cursor: pointer; margin-bottom: 16px;">
              <div style="display: flex; align-items: center; gap: 12px;">
                <el-icon size="24" color="#16a34a"><component :is="template.icon || 'DocumentAdd'" /></el-icon>
                <div>
                  <div style="font-weight: 600;">{{ template.name }}</div>
                  <div style="font-size: 12px; color: #909399;">{{ template.description || template.category }}</div>
                </div>
              </div>
            </el-card>
          </el-col>
        </el-row>
        <el-empty v-if="allRequestTemplates.length === 0" description="暂无工单模板，请先创建" />
      </div>

      <!-- Step 2: 填写表单 -->
      <div v-if="step === 2">
        <el-button link @click="step = 1" style="margin-bottom: 16px;"><el-icon><ArrowLeft /></el-icon> 返回模板列表</el-button>
        <el-tag style="margin-left: 8px;">{{ selectedTemplate?.name }}</el-tag>
        <el-tag v-if="selectedTemplate" type="success" style="margin-left: 8px;">工单模板</el-tag>

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
          <template v-for="field in requestSchemaFields" :key="field.key">
            <el-form-item :label="field.label">
              <el-input
                v-if="field.type === 'text'"
                v-model="requestFormData[field.key]"
                :placeholder="field.placeholder || `请输入${field.label}`"
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
                style="width: 100%;"
              />
              <el-select
                v-else-if="field.type === 'select'"
                v-model="requestFormData[field.key]"
                clearable
                :placeholder="field.placeholder || `请选择${field.label}`"
                style="width: 100%;"
              >
                <el-option v-for="option in field.options || []" :key="String(option.value)" :label="option.label" :value="option.value" />
              </el-select>
              <el-switch
                v-else-if="field.type === 'switch'"
                v-model="requestFormData[field.key]"
              />
              <el-input
                v-else
                v-model="requestFormData[field.key]"
                :placeholder="field.placeholder || `请输入${field.label}`"
              />
            </el-form-item>
          </template>
          <el-form-item label="关联资源">
            <el-select v-model="form.resource_type" placeholder="选择资源类型" clearable style="width: 160px; margin-right: 8px;" @change="handleResourceTypeChange">
              <el-option v-for="o in resourceTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
            <!-- 资产选择器 -->
            <div v-if="form.resource_type === 'asset'" class="resource-selector">
              <div
                class="asset-selection-panel"
                :class="{ 'is-empty': !selectedAssets.length }"
                @click="openAssetDialog"
              >
                <div class="asset-selection-head">
                  <div class="asset-selection-meta">
                    <span class="asset-selection-label">主机资产</span>
                    <span class="asset-selection-count">
                      {{ selectedAssets.length ? `已选 ${selectedAssets.length} 台` : '未选择' }}
                    </span>
                  </div>
                  <span class="asset-selection-action">
                    {{ selectedAssets.length ? '点击重新选择' : '点击选择主机资产' }}
                  </span>
                </div>

                <div v-if="selectedAssets.length" class="asset-selection-tags">
                  <span
                    v-for="asset in selectedAssets"
                    :key="asset.id"
                    class="asset-selection-tag"
                    :title="`${asset.hostname} (${asset.ip})`"
                  >
                    {{ asset.hostname }}({{ asset.ip }})
                  </span>
                </div>
                <div v-else class="asset-selection-empty-text">
                  资源类型切换后会自动弹出选择窗口，后续可在这里继续调整。
                </div>
              </div>
            </div>
            <!-- 云账号 -->
            <el-select v-if="form.resource_type === 'cloud_account'" v-model="form.resource_id" placeholder="选择云账号" style="flex: 1;">
              <el-option v-for="a in cloudAccountOptions" :key="a.id" :label="a.label" :value="a.id" />
            </el-select>
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

    <el-dialog v-model="assetDialogVisible" title="资产管理" width="1040px" top="6vh" class="asset-picker-dialog">
      <div class="asset-picker">
        <aside class="asset-picker-sidebar">
          <div class="picker-tabs">
            <span class="picker-tab active">资产树</span>
          </div>
          <div class="picker-tree-toolbar">
            <el-button link :type="!selectedServiceTreeNode ? 'primary' : 'default'" @click="resetAssetTreeFilter">全部资产</el-button>
          </div>
          <el-scrollbar class="picker-tree-scroll">
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
                <div class="picker-tree-node">
                  <span class="picker-tree-label">{{ data.name }}</span>
                  <span class="picker-tree-count">{{ getNodeAssetCount(data) }}</span>
                </div>
              </template>
            </el-tree>
          </el-scrollbar>
        </aside>

        <section class="asset-picker-main">
          <div class="picker-toolbar">
            <el-input
              v-model="assetDialogQuery.keyword"
              placeholder="搜索主机名 / IP"
              clearable
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
            height="420"
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

          <div class="picker-footer">
            <div class="picker-selection">
              <span class="picker-selection-label">已选资产</span>
              <span v-if="Object.keys(draftSelectedAssets).length" class="picker-selection-value">
                {{ Object.keys(draftSelectedAssets).length }} 台
              </span>
              <span v-else class="picker-selection-empty">未选择</span>
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
        <el-button @click="assetDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmSelectedAsset">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
.type-card:hover { border-color: #409eff; }

.resource-selector {
  flex: 1;
  min-width: 0;
}

.asset-selection-panel {
  width: 100%;
  min-width: 0;
  border: 1px solid #d7dee7;
  border-radius: 12px;
  background: #fff;
  padding: 10px 12px;
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, background-color 0.2s ease;
}

.asset-selection-panel:hover {
  border-color: #a9c2de;
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.06);
}

.asset-selection-panel.is-empty {
  border-style: dashed;
  background: #fafcff;
}

.asset-selection-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.asset-selection-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.asset-selection-label {
  font-size: 13px;
  font-weight: 700;
  color: #233548;
}

.asset-selection-count {
  font-size: 12px;
  color: #66788a;
}

.asset-selection-action {
  flex-shrink: 0;
  font-size: 12px;
  color: #2563eb;
}

.asset-selection-tags {
  margin-top: 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  max-height: 92px;
  overflow-y: auto;
  padding-right: 4px;
}

.asset-selection-tag {
  max-width: 100%;
  padding: 7px 12px;
  border-radius: 10px;
  border: 1px solid #dde3ea;
  background: #f3f4f6;
  color: #3f4d5a;
  font-size: 12px;
  line-height: 1.2;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.asset-selection-empty-text {
  margin-top: 8px;
  font-size: 12px;
  color: #94a3b8;
}

.asset-picker {
  display: flex;
  min-height: 540px;
  background: #f5f7fb;
  border: 1px solid #e6edf5;
  border-radius: 16px;
  overflow: hidden;
}

.asset-picker-sidebar {
  width: 270px;
  background: linear-gradient(180deg, #fcfdff 0%, #f8fafc 100%);
  border-right: 1px solid #e5ebf2;
  display: flex;
  flex-direction: column;
}

.picker-tabs {
  display: flex;
  padding: 14px 16px 0;
  border-bottom: 1px solid #edf2f7;
}

.picker-tab {
  position: relative;
  padding: 10px 12px 12px;
  font-size: 14px;
  font-weight: 700;
  color: #243447;
}

.picker-tab.active::after {
  content: '';
  position: absolute;
  left: 12px;
  right: 12px;
  bottom: 0;
  height: 3px;
  border-radius: 999px;
  background: linear-gradient(90deg, #14b8a6 0%, #0ea5e9 100%);
}

.picker-tree-toolbar {
  padding: 12px 16px 4px;
}

.picker-tree-scroll {
  flex: 1;
  padding: 4px 10px 12px 12px;
}

.picker-tree-node {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding-right: 6px;
}

.picker-tree-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.picker-tree-count {
  min-width: 22px;
  height: 22px;
  padding: 0 6px;
  border-radius: 999px;
  background: #ecfdf5;
  color: #0f766e;
  font-size: 12px;
  line-height: 22px;
  text-align: center;
}

.asset-picker-main {
  flex: 1;
  padding: 18px;
  background: linear-gradient(180deg, #f9fbfd 0%, #f4f7fb 100%);
}

.picker-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}

.picker-toolbar .el-input {
  max-width: 320px;
}

.picker-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 14px;
}

.picker-selection {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.picker-selection-label {
  color: #6b7a89;
  font-size: 13px;
}

.picker-selection-value {
  color: #213547;
  font-weight: 600;
}

.picker-selection-empty {
  color: #98a2b3;
}
</style>
