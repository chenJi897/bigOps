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
  <div class="p-6">
    <div class="mb-6 flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 tracking-tight">发起工单</h1>
        <p class="text-sm text-gray-500 mt-1">选择对应的服务目录和工单模板，快速提交您的服务请求</p>
      </div>
    </div>

    <div v-loading="loading" class="min-h-[400px]">
      <div v-if="groupedTemplates.length === 0 && !loading" class="flex flex-col items-center justify-center py-20 bg-white rounded-xl border border-dashed border-gray-200">
        <el-icon class="text-gray-300 text-6xl mb-4"><Document /></el-icon>
        <p class="text-gray-500">暂无可用的工单模板</p>
      </div>

      <div class="space-y-6">
        <div v-for="group in groupedTemplates" :key="group.type.id" class="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
          <div 
            class="flex items-center justify-between px-6 py-4 bg-gray-50/50 hover:bg-gray-50 cursor-pointer border-b border-gray-100 transition-colors"
            @click="group.type._collapsed = !group.type._collapsed"
          >
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-lg bg-indigo-50 flex items-center justify-center text-indigo-600">
                <el-icon><FolderOpened /></el-icon>
              </div>
              <span class="text-base font-semibold text-gray-800">{{ group.type.name }}</span>
              <span class="text-xs px-2 py-0.5 rounded-full bg-gray-200 text-gray-600">{{ group.templates.length || 1 }} 个模板</span>
            </div>
            <el-icon 
              class="text-gray-400 transition-transform duration-300 text-lg" 
              :class="{ '-rotate-90': group.type._collapsed }"
            >
              <ArrowDown />
            </el-icon>
          </div>
          
          <el-collapse-transition>
            <div v-show="!group.type._collapsed" class="p-6 bg-white">
              <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                <div 
                  v-if="group.templates.length === 0" 
                  class="group flex items-center justify-between p-4 bg-white border border-gray-200 rounded-xl hover:border-indigo-500 hover:shadow-md cursor-pointer transition-all duration-300"
                  @click="openCreateDirect(group.type.id)"
                >
                  <div class="flex items-center gap-3 overflow-hidden">
                    <div class="w-10 h-10 rounded-full bg-blue-50 flex items-center justify-center text-blue-500 group-hover:bg-indigo-50 group-hover:text-indigo-600 transition-colors flex-shrink-0">
                      <el-icon><Document /></el-icon>
                    </div>
                    <span class="text-sm font-medium text-gray-700 group-hover:text-indigo-600 truncate">{{ group.type.name }}</span>
                  </div>
                  <el-icon class="text-gray-300 group-hover:text-indigo-500 group-hover:translate-x-1 transition-all"><ArrowRight /></el-icon>
                </div>
                
                <div 
                  v-for="tpl in group.templates" 
                  :key="tpl.id" 
                  class="group flex items-center justify-between p-4 bg-white border border-gray-200 rounded-xl hover:border-indigo-500 hover:shadow-md cursor-pointer transition-all duration-300"
                  @click="openCreate(group.type.id, tpl.id)"
                >
                  <div class="flex items-center gap-3 overflow-hidden">
                    <div class="w-10 h-10 rounded-full bg-gray-50 flex items-center justify-center text-gray-500 group-hover:bg-indigo-50 group-hover:text-indigo-600 transition-colors flex-shrink-0">
                      <el-icon><DocumentAdd /></el-icon>
                    </div>
                    <div class="flex flex-col overflow-hidden">
                      <span class="text-sm font-medium text-gray-700 group-hover:text-indigo-600 truncate">{{ tpl.name }}</span>
                      <span class="text-xs text-gray-400 truncate mt-0.5">点击发起该类型申请</span>
                    </div>
                  </div>
                  <el-icon class="text-gray-300 group-hover:text-indigo-500 group-hover:translate-x-1 transition-all"><ArrowRight /></el-icon>
                </div>
              </div>
            </div>
          </el-collapse-transition>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 移除旧的样式，完全使用 Tailwind */
</style>
