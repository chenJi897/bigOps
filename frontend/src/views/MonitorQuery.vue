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
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div class="page-head">
          <div>
            <div class="page-title">PromQL 查询台</div>
            <div class="page-subtitle">对接 Prometheus 数据源执行即时查询或范围查询。</div>
          </div>
          <el-button type="primary" :loading="loading" @click="executeQuery">执行查询</el-button>
        </div>
      </template>

      <el-form label-width="100px">
        <el-form-item label="数据源">
          <el-select v-model="form.datasource_id" placeholder="选择数据源" style="width: 280px">
            <el-option v-for="item in datasources" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="模式">
          <el-radio-group v-model="mode">
            <el-radio-button label="instant">即时查询</el-radio-button>
            <el-radio-button label="range">范围查询</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="PromQL">
          <el-input v-model="form.query" type="textarea" :rows="4" placeholder="例如：up" />
        </el-form-item>
        <el-form-item v-if="mode === 'instant'" label="时间点">
          <el-input v-model="form.time" placeholder="可选，RFC3339 时间，例如 2026-03-26T15:00:00+08:00" />
        </el-form-item>
        <template v-else>
          <el-form-item label="开始时间">
            <el-input v-model="form.start" placeholder="RFC3339，例如 2026-03-26T12:00:00+08:00" />
          </el-form-item>
          <el-form-item label="结束时间">
            <el-input v-model="form.end" placeholder="RFC3339，例如 2026-03-26T15:00:00+08:00" />
          </el-form-item>
          <el-form-item label="步长">
            <el-input v-model="form.step" placeholder="60s / 1m / 5m" style="width: 200px" />
          </el-form-item>
        </template>
      </el-form>
    </el-card>

    <el-card shadow="never" class="result-card">
      <template #header><span>查询结果</span></template>
      <el-table v-if="mode === 'instant' && instantResultRows.length" :data="instantResultRows" stripe border>
        <el-table-column prop="metric" label="指标标签" min-width="360" show-overflow-tooltip />
        <el-table-column prop="value" label="值" width="180" />
        <el-table-column prop="timestamp" label="时间戳" width="180" />
      </el-table>
      <el-input
        v-else
        type="textarea"
        :model-value="result ? JSON.stringify(result, null, 2) : ''"
        :rows="18"
        readonly
        placeholder="执行查询后展示原始结果"
      />
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 20px; }
.page-head { display: flex; justify-content: space-between; align-items: center; gap: 16px; }
.page-title { font-size: 18px; font-weight: 700; color: #1f2937; }
.page-subtitle { margin-top: 4px; color: #6b7280; }
.result-card { margin-top: 16px; }
</style>
