<script setup lang="ts">
defineOptions({ name: 'TicketRequest' })
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ticketTypeApi, requestTemplateApi } from '../api'

const router = useRouter()
const loading = ref(false)
const allTypes = ref<any[]>([])
const allTemplates = ref<any[]>([])

const groupedTemplates = computed(() => {
  const groups: { type: any; templates: any[] }[] = []
  for (const t of allTypes.value) {
    const templates = allTemplates.value.filter((tpl: any) => tpl.type_id === t.id)
    // 有模板的类型显示模板卡片；没有模板的类型显示一个通用发起入口
    groups.push({ type: { ...t, _collapsed: false }, templates })
  }
  return groups
})

async function loadData() {
  loading.value = true
  try {
    const [typesRes, templatesRes] = await Promise.allSettled([
      ticketTypeApi.all(),
      requestTemplateApi.list(true),
    ])
    if (typesRes.status === 'fulfilled') allTypes.value = (typesRes.value as any).data || []
    if (templatesRes.status === 'fulfilled') allTemplates.value = (templatesRes.value as any).data || []
  } finally { loading.value = false }
}

function openCreate(typeId: number, templateId?: number) {
  let path = `/ticket/create?type_id=${typeId}`
  if (templateId) path += `&template_id=${templateId}`
  router.push(path)
}

function openCreateDirect(typeId: number) {
  router.push(`/ticket/create?type_id=${typeId}`)
}

onMounted(() => { loadData() })
</script>

<template>
  <div class="page">
    <el-card shadow="never" v-loading="loading">
      <template #header>
        <span>发起工单</span>
      </template>

      <div v-if="groupedTemplates.length === 0 && !loading" style="text-align: center; padding: 40px; color: #909399;">
        暂无可用的工单模板
      </div>

      <div v-for="group in groupedTemplates" :key="group.type.id" class="type-group">
        <div class="type-header" @click="group.type._collapsed = !group.type._collapsed">
          <span class="type-title">{{ group.type.name }}</span>
          <el-icon style="transition: transform 0.2s;" :style="{ transform: group.type._collapsed ? 'rotate(-90deg)' : '' }"><ArrowDown /></el-icon>
        </div>
        <div v-show="!group.type._collapsed" class="template-grid">
          <div v-if="group.templates.length === 0" class="template-card" @click="openCreateDirect(group.type.id)">
            <span class="card-name">{{ group.type.name }}</span>
            <el-icon class="card-arrow"><ArrowRight /></el-icon>
          </div>
          <div v-for="tpl in group.templates" :key="tpl.id" class="template-card" @click="openCreate(group.type.id, tpl.id)">
            <span class="card-name">{{ tpl.name }}</span>
            <el-icon class="card-arrow"><ArrowRight /></el-icon>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
.type-group { margin-bottom: 20px; }
.type-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: #f5f7fa;
  border-radius: 4px;
  cursor: pointer;
  user-select: none;
}
.type-title { font-size: 15px; font-weight: 500; color: #303133; }
.template-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
  padding: 16px 0;
}
.template-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}
.template-card:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.15);
}
.card-name { font-size: 14px; color: #303133; }
.card-arrow { color: #909399; font-size: 16px; }
.template-card:hover .card-arrow { color: #409eff; }
</style>
