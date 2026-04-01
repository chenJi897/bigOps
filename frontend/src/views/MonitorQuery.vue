<script setup lang="ts">
defineOptions({ name: 'MonitorQuery' })

import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { monitorApi } from '../api'

const loading = ref(false)
const datasources = ref<any[]>([])
const mode = ref<'instant' | 'range'>('instant')
const form = ref({
  datasource_id: 0,
  query: 'up',
  time: '',
  start: '',
  end: '',
  step: '60s',
})
const result = ref<any>(null)

const instantResultRows = computed(() => {
  const payload = result.value?.data?.result
  if (!Array.isArray(payload)) return []
  return payload.map((item: any) => ({
    metric: JSON.stringify(item.metric || {}),
    value: Array.isArray(item.value) ? item.value[1] : '-',
    timestamp: Array.isArray(item.value) ? item.value[0] : '-',
  }))
})

async function loadDatasources() {
  const res = await monitorApi.datasources()
  datasources.value = (res as any).data || []
  if (!form.value.datasource_id && datasources.value.length > 0) {
    form.value.datasource_id = datasources.value[0].id
  }
}

async function executeQuery() {
  if (!form.value.datasource_id || !form.value.query.trim()) {
    ElMessage.warning('请选择数据源并填写 PromQL')
    return
  }
  loading.value = true
  try {
    if (mode.value === 'instant') {
      const res = await monitorApi.query({
        datasource_id: form.value.datasource_id,
        query: form.value.query.trim(),
        time: form.value.time || undefined,
      })
      result.value = (res as any).data
    } else {
      const res = await monitorApi.queryRange({
        datasource_id: form.value.datasource_id,
        query: form.value.query.trim(),
        start: form.value.start || undefined,
        end: form.value.end || undefined,
        step: form.value.step || '60s',
      })
      result.value = (res as any).data
    }
  } finally {
    loading.value = false
  }
}

onMounted(loadDatasources)
</script>

<template>
  <div class="p-4 md:p-6 min-h-full flex flex-col gap-4 bg-slate-50">
    <el-card shadow="never" class="border-0 shadow-sm rounded-2xl">
      <template #header>
        <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
          <div>
            <div class="text-xl font-bold text-slate-800">PromQL 查询台</div>
            <div class="mt-1 text-sm text-slate-500">对接 Prometheus 数据源执行即时查询或范围查询。</div>
          </div>
          <el-button type="primary" :loading="loading" @click="executeQuery" class="w-full sm:w-auto">
            <el-icon class="mr-1"><VideoPlay /></el-icon> 执行查询
          </el-button>
        </div>
      </template>

      <el-form label-width="100px" label-position="left" class="max-w-4xl mt-2">
        <el-form-item label="数据源" required>
          <el-select v-model="form.datasource_id" placeholder="选择数据源" class="w-72">
            <el-option v-for="item in datasources" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="模式">
          <el-radio-group v-model="mode">
            <el-radio-button label="instant">即时查询</el-radio-button>
            <el-radio-button label="range">范围查询</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="PromQL" required>
          <el-input 
            v-model="form.query" 
            type="textarea" 
            :rows="5" 
            placeholder="例如：up" 
            class="font-mono text-sm shadow-inner"
            input-style="background-color: #f8fafc;"
          />
        </el-form-item>
        
        <div class="p-4 bg-slate-50 rounded-xl border border-slate-100">
          <el-form-item v-if="mode === 'instant'" label="时间点" class="!mb-0">
            <el-input v-model="form.time" placeholder="可选，RFC3339 时间，例如 2026-03-26T15:00:00+08:00" class="w-full sm:w-96" />
          </el-form-item>
          <template v-else>
            <el-form-item label="开始时间">
              <el-input v-model="form.start" placeholder="RFC3339，例如 2026-03-26T12:00:00+08:00" class="w-full sm:w-96" />
            </el-form-item>
            <el-form-item label="结束时间">
              <el-input v-model="form.end" placeholder="RFC3339，例如 2026-03-26T15:00:00+08:00" class="w-full sm:w-96" />
            </el-form-item>
            <el-form-item label="步长" class="!mb-0">
              <el-input v-model="form.step" placeholder="60s / 1m / 5m" class="w-48" />
            </el-form-item>
          </template>
        </div>
      </el-form>
    </el-card>

    <el-card shadow="never" class="border-0 shadow-sm rounded-2xl flex-1 flex flex-col">
      <template #header>
        <span class="font-medium text-slate-800">查询结果</span>
      </template>
      <div v-loading="loading" class="h-full">
        <el-table v-if="mode === 'instant' && instantResultRows.length" :data="instantResultRows" stripe border class="w-full">
          <el-table-column prop="metric" label="指标标签" min-width="360" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="font-mono text-xs text-indigo-600 bg-indigo-50 px-2 py-1 rounded break-all whitespace-normal">{{ row.metric }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="value" label="值" width="180" align="center">
            <template #default="{ row }">
              <strong class="text-slate-800">{{ row.value }}</strong>
            </template>
          </el-table-column>
          <el-table-column prop="timestamp" label="时间戳" width="180" align="center">
            <template #default="{ row }">
              <span class="text-slate-500 font-mono text-xs">{{ row.timestamp }}</span>
            </template>
          </el-table-column>
        </el-table>
        <el-input
          v-else
          type="textarea"
          :model-value="result ? JSON.stringify(result, null, 2) : ''"
          :rows="18"
          readonly
          placeholder="执行查询后在此展示原始 JSON 结果..."
          class="font-mono text-sm shadow-inner"
          input-style="background-color: #1e1e1e; color: #d4d4d4; padding: 16px;"
        />
      </div>
    </el-card>
  </div>
</template>

<style scoped>
:deep(.el-card__body) {
  flex: 1;
}
</style>
