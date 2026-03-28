<script setup lang="ts">
defineOptions({ name: 'Dashboard' })

import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { dashboardApi, monitorApi, statsApi } from '../api'
import { usePermissionStore } from '../stores/permission'
import { useUserStore } from '../stores/user'

const router = useRouter()
const permissionStore = usePermissionStore()
const userStore = useUserStore()

const loading = ref(true)

const personal = ref({
  my_pending_tickets: 0,
  my_created_tickets: 0,
  my_assets: 0,
  my_alerts: 0,
  my_task_executions: 0,
  my_pipeline_runs: 0,
})

const platform = ref({
  asset_total: 0,
  asset_online: 0,
  asset_offline: 0,
  cloud_account_total: 0,
  cloud_account_failed: 0,
  service_tree_total: 0,
  user_total: 0,
  department_total: 0,
  ticket_open: 0,
  ticket_total: 0,
})

const distribution = ref<{ status_dist: any[]; source_dist: any[]; top_services: any[] }>({
  status_dist: [],
  source_dist: [],
  top_services: [],
})

const monitorSummary = ref({
  agent_total: 0,
  agent_online: 0,
  agent_offline: 0,
  alert_firing_total: 0,
  last_collected_at: '',
})

function flattenMenus(items: any[]): any[] {
  const result: any[] = []
  for (const item of items || []) {
    result.push(item)
    if (Array.isArray(item.children) && item.children.length > 0) {
      result.push(...flattenMenus(item.children))
    }
  }
  return result
}

const flatMenus = computed(() => flattenMenus(permissionStore.menus))
const visiblePaths = computed(() => new Set(flatMenus.value.map(item => item.path).filter(Boolean)))

function hasPath(path: string) {
  return visiblePaths.value.has(path)
}

const canViewCMDB = computed(() => hasPath('/cmdb/assets') || hasPath('/cmdb/cloud-accounts') || hasPath('/cmdb/service-tree'))
const canViewAssets = computed(() => hasPath('/cmdb/assets'))
const canViewCloudAccounts = computed(() => hasPath('/cmdb/cloud-accounts'))
const canViewServiceTree = computed(() => hasPath('/cmdb/service-tree'))
const canViewTicket = computed(() => hasPath('/ticket/todo') || hasPath('/ticket/applied') || hasPath('/ticket/create'))
const canLaunchTicket = computed(() => hasPath('/ticket/create'))
const canViewTask = computed(() => hasPath('/task/list'))
const canViewCICD = computed(() => hasPath('/cicd/projects') || hasPath('/cicd/pipelines') || hasPath('/cicd/runs'))
const canViewMonitor = computed(() => hasPath('/monitor/dashboard') || hasPath('/monitor/alert-rules') || hasPath('/monitor/alerts'))
const canViewSystem = computed(() => hasPath('/system/users') || hasPath('/system/roles') || hasPath('/system/menus'))

const availableModuleCount = computed(() => {
  return [canViewCMDB.value, canViewTicket.value, canViewTask.value, canViewCICD.value, canViewMonitor.value, canViewSystem.value].filter(Boolean).length
})

const displayName = computed(() => userStore.userInfo?.real_name || userStore.userInfo?.username || '用户')
const welcomeTitle = computed(() => {
  return availableModuleCount.value >= 4 ? `${displayName.value}，欢迎回到平台总览` : `${displayName.value}，这里是你的工作台`
})
const welcomeSubtitle = computed(() => {
  return availableModuleCount.value >= 4
    ? '先处理你自己的待办，再快速扫一眼平台态势。'
    : '首页优先展示你能操作、也最需要处理的内容。'
})

const quickActions = computed(() => {
  const items = [
    { key: 'launch-ticket', title: '发起工单', subtitle: '提交新的请求或变更', path: '/ticket/create', visible: canLaunchTicket.value },
    { key: 'todo-ticket', title: '我的待办', subtitle: '进入待处理工单列表', path: '/ticket/todo', visible: hasPath('/ticket/todo') },
    { key: 'my-apply', title: '我的申请', subtitle: '查看自己发起的工单', path: '/ticket/applied', visible: hasPath('/ticket/applied') },
    { key: 'alert-events', title: '告警事件', subtitle: '处理触发中的告警', path: '/monitor/alerts', visible: hasPath('/monitor/alerts') },
    { key: 'task-list', title: '任务中心', subtitle: '查看执行与任务状态', path: '/task/list', visible: hasPath('/task/list') },
    { key: 'pipeline-runs', title: '运行记录', subtitle: '跟踪最近发布与回滚', path: '/cicd/runs', visible: hasPath('/cicd/runs') },
  ]
  return items.filter(item => item.visible)
})

const workbenchCards = computed(() => {
  const items = [
    { key: 'pending', title: '我的待办', value: personal.value.my_pending_tickets, hint: '等待我处理的工单', tone: 'warning', path: '/ticket/todo', visible: hasPath('/ticket/todo') },
    { key: 'created', title: '我的申请', value: personal.value.my_created_tickets, hint: '我发起的工单', tone: 'primary', path: '/ticket/applied', visible: hasPath('/ticket/applied') },
    { key: 'assets', title: '我负责的资产', value: personal.value.my_assets, hint: '归属到我名下的主机', tone: 'success', path: '/cmdb/assets', visible: canViewAssets.value },
    { key: 'alerts', title: '我的相关告警', value: personal.value.my_alerts, hint: '归属到我的告警事件', tone: 'danger', path: '/monitor/alerts', visible: canViewMonitor.value },
    { key: 'tasks', title: '我的任务执行', value: personal.value.my_task_executions, hint: '我发起的任务执行记录', tone: 'info', path: '/task/list', visible: canViewTask.value },
    { key: 'runs', title: '我的流水线', value: personal.value.my_pipeline_runs, hint: '我触发过的流水线运行', tone: 'purple', path: '/cicd/runs', visible: canViewCICD.value },
  ]
  return items.filter(item => item.visible)
})

const overviewCards = computed(() => {
  const items = [
    { key: 'asset', title: '主机资产', value: platform.value.asset_total, meta: `在线 ${platform.value.asset_online} / 离线 ${platform.value.asset_offline}`, tone: 'primary', path: '/cmdb/assets', visible: canViewAssets.value },
    { key: 'cloud', title: '云账号', value: platform.value.cloud_account_total, meta: platform.value.cloud_account_failed > 0 ? `${platform.value.cloud_account_failed} 个同步异常` : '同步状态正常', tone: 'success', path: '/cmdb/cloud-accounts', visible: canViewCloudAccounts.value },
    { key: 'tree', title: '服务树', value: platform.value.service_tree_total, meta: '服务与资源归属结构', tone: 'gold', path: '/cmdb/service-tree', visible: canViewServiceTree.value },
    { key: 'alert', title: '触发中告警', value: monitorSummary.value.alert_firing_total, meta: `在线 ${monitorSummary.value.agent_online} / 离线 ${monitorSummary.value.agent_offline}`, tone: 'danger', path: '/monitor/alerts', visible: canViewMonitor.value },
    { key: 'ticket', title: '打开中的工单', value: platform.value.ticket_open, meta: `工单总数 ${platform.value.ticket_total}`, tone: 'warning', path: '/ticket/todo', visible: canViewTicket.value },
    { key: 'task', title: '在线 Agent', value: monitorSummary.value.agent_online, meta: monitorSummary.value.last_collected_at ? `最近采样 ${monitorSummary.value.last_collected_at}` : '等待采样', tone: 'info', path: '/monitor/dashboard', visible: canViewTask.value || canViewMonitor.value },
    { key: 'user', title: '平台用户', value: platform.value.user_total, meta: `部门 ${platform.value.department_total}`, tone: 'ink', path: '/system/users', visible: canViewSystem.value },
    { key: 'run', title: '流水线运行', value: personal.value.my_pipeline_runs, meta: '快速进入运行记录', tone: 'purple', path: '/cicd/runs', visible: canViewCICD.value },
  ]
  return items.filter(item => item.visible)
})

function maxServiceCount() {
  if (!distribution.value.top_services?.length) return 1
  return Math.max(...distribution.value.top_services.map((t: any) => t.count), 1)
}

function sourceLabel(s: string) {
  const map: Record<string, string> = { manual: '手工录入', aliyun: '阿里云', tencent: '腾讯云', aws: 'AWS' }
  return map[s] || s
}

function cardToneClass(tone: string) {
  return `tone-${tone}`
}

async function fetchData() {
  loading.value = true
  try {
    const jobs: Promise<any>[] = [dashboardApi.personal()]
    const keys = ['personal']
    if (canViewCMDB.value || canViewTicket.value || canViewSystem.value) {
      jobs.push(statsApi.summary())
      keys.push('summary')
    }
    if (canViewCMDB.value) {
      jobs.push(statsApi.assetDistribution())
      keys.push('distribution')
    }
    if (canViewMonitor.value || canViewTask.value) {
      jobs.push(monitorApi.summary())
      keys.push('monitor')
    }

    const results = await Promise.allSettled(jobs)
    results.forEach((item, index) => {
      if (item.status !== 'fulfilled') return
      const key = keys[index]
      const data = (item.value as any).data
      if (key === 'personal') personal.value = data || personal.value
      if (key === 'summary') platform.value = data || platform.value
      if (key === 'distribution') distribution.value = data || distribution.value
      if (key === 'monitor') {
        monitorSummary.value = {
          agent_total: data?.agent_total || 0,
          agent_online: data?.agent_online || 0,
          agent_offline: data?.agent_offline || 0,
          alert_firing_total: data?.alert_firing_total || 0,
          last_collected_at: data?.last_collected_at || '',
        }
      }
    })
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)
</script>

<template>
  <div class="dashboard-shell" v-loading="loading">
    <section class="hero-panel">
      <div class="hero-copy">
        <div class="eyebrow">Dashboard</div>
        <h2>{{ welcomeTitle }}</h2>
        <p>{{ welcomeSubtitle }}</p>
      </div>
      <div class="hero-quick-actions">
        <button
          v-for="item in quickActions"
          :key="item.key"
          class="quick-chip"
          type="button"
          @click="router.push(item.path)"
        >
          <span class="quick-chip__title">{{ item.title }}</span>
          <span class="quick-chip__sub">{{ item.subtitle }}</span>
        </button>
      </div>
    </section>

    <section class="dashboard-section">
      <div class="section-heading">
        <div>
          <div class="section-title">个人工作台</div>
          <div class="section-subtitle">这里的数字都与你自己有关，先把手边的事情处理掉。</div>
        </div>
      </div>

      <div v-if="workbenchCards.length" class="workbench-grid">
        <button
          v-for="card in workbenchCards"
          :key="card.key"
          class="workbench-card"
          :class="cardToneClass(card.tone)"
          type="button"
          @click="router.push(card.path)"
        >
          <div class="card-topline">{{ card.title }}</div>
          <div class="card-value">{{ card.value }}</div>
          <div class="card-hint">{{ card.hint }}</div>
        </button>
      </div>
      <el-empty v-else description="当前没有可展示的个人工作台内容" :image-size="56" />
    </section>

    <section class="dashboard-section">
      <div class="section-heading">
        <div>
          <div class="section-title">平台总览</div>
          <div class="section-subtitle">只显示你当前有权限访问的模块概况。</div>
        </div>
      </div>

      <div v-if="overviewCards.length" class="overview-grid">
        <button
          v-for="card in overviewCards"
          :key="card.key"
          class="overview-card"
          :class="cardToneClass(card.tone)"
          type="button"
          @click="router.push(card.path)"
        >
          <div class="overview-card__title">{{ card.title }}</div>
          <div class="overview-card__value">{{ card.value }}</div>
          <div class="overview-card__meta">{{ card.meta }}</div>
        </button>
      </div>
      <el-empty v-else description="当前角色没有可展示的平台概况模块" :image-size="56" />
    </section>

    <section v-if="canViewCMDB" class="dashboard-section charts">
      <div class="section-heading">
        <div>
          <div class="section-title">资源分布</div>
          <div class="section-subtitle">保留资产来源和服务树排行，帮助快速定位资源聚集点。</div>
        </div>
      </div>

      <el-row :gutter="16">
        <el-col :xs="24" :lg="10">
          <el-card shadow="never" class="panel-card">
            <template #header><span>资产来源分布</span></template>
            <div v-if="distribution.source_dist?.length">
              <div
                v-for="item in distribution.source_dist"
                :key="item.label"
                class="dist-row clickable"
                @click="router.push('/cmdb/assets?source=' + item.label)"
              >
                <span class="dist-label">{{ sourceLabel(item.label) }}</span>
                <el-progress
                  :percentage="platform.asset_total ? Math.round((item.count / platform.asset_total) * 100) : 0"
                  :stroke-width="16"
                  :show-text="false"
                  style="flex: 1; margin: 0 12px;"
                />
                <span class="dist-count">{{ item.count }}</span>
              </div>
            </div>
            <el-empty v-else description="暂无数据" :image-size="56" />
          </el-card>
        </el-col>

        <el-col :xs="24" :lg="14">
          <el-card shadow="never" class="panel-card">
            <template #header><span>服务树资产 Top 10</span></template>
            <div v-if="distribution.top_services?.length">
              <div
                v-for="item in distribution.top_services"
                :key="item.id"
                class="dist-row clickable"
                @click="router.push('/cmdb/assets?service_tree_id=' + item.id)"
              >
                <span class="dist-label wide" :title="item.name">{{ item.name }}</span>
                <el-progress
                  :percentage="Math.round((item.count / maxServiceCount()) * 100)"
                  :stroke-width="16"
                  :show-text="false"
                  color="#16a34a"
                  style="flex: 1; margin: 0 12px;"
                />
                <span class="dist-count">{{ item.count }}</span>
              </div>
            </div>
            <el-empty v-else description="暂无数据" :image-size="56" />
          </el-card>
        </el-col>
      </el-row>
    </section>
  </div>
</template>

<style scoped>
.dashboard-shell {
  padding: 24px;
  min-height: 100%;
  background:
    radial-gradient(circle at top left, rgba(14, 165, 233, 0.10), transparent 28%),
    radial-gradient(circle at top right, rgba(249, 115, 22, 0.08), transparent 24%),
    linear-gradient(180deg, #f8fbff 0%, #f4f7fb 100%);
}

.hero-panel,
.panel-card,
.workbench-card,
.overview-card {
  border: 1px solid #e4ebf3;
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.06);
}

.hero-panel {
  display: grid;
  grid-template-columns: 1.1fr 1fr;
  gap: 20px;
  padding: 28px;
  border-radius: 24px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.92), rgba(244, 247, 251, 0.96)),
    #fff;
}

.eyebrow {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: #0ea5e9;
}

.hero-copy h2 {
  margin: 12px 0 8px;
  font-size: 30px;
  line-height: 1.15;
  color: #0f172a;
}

.hero-copy p {
  margin: 0;
  color: #64748b;
  font-size: 14px;
}

.hero-quick-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.quick-chip {
  border: 0;
  border-radius: 18px;
  padding: 16px;
  text-align: left;
  cursor: pointer;
  background: #0f172a;
  color: #f8fafc;
  transition: transform 0.18s ease, box-shadow 0.18s ease, opacity 0.18s ease;
}

.quick-chip:hover {
  transform: translateY(-2px);
  box-shadow: 0 18px 32px rgba(15, 23, 42, 0.18);
}

.quick-chip__title {
  display: block;
  font-size: 15px;
  font-weight: 700;
}

.quick-chip__sub {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.5;
  color: rgba(248, 250, 252, 0.74);
}

.dashboard-section {
  margin-top: 24px;
}

.section-heading {
  display: flex;
  justify-content: space-between;
  align-items: end;
  gap: 16px;
  margin-bottom: 14px;
}

.section-title {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
}

.section-subtitle {
  margin-top: 6px;
  font-size: 13px;
  color: #64748b;
}

.workbench-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.overview-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.workbench-card,
.overview-card {
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.92);
  padding: 18px;
  text-align: left;
  cursor: pointer;
  transition: transform 0.18s ease, box-shadow 0.18s ease;
}

.workbench-card:hover,
.overview-card:hover {
  transform: translateY(-2px);
}

.card-topline,
.overview-card__title {
  font-size: 13px;
  font-weight: 700;
  color: #475569;
}

.card-value,
.overview-card__value {
  margin-top: 12px;
  font-size: 32px;
  line-height: 1;
  font-weight: 800;
  color: #0f172a;
}

.card-hint,
.overview-card__meta {
  margin-top: 10px;
  min-height: 36px;
  font-size: 12px;
  line-height: 1.5;
  color: #64748b;
}

.tone-primary { background: linear-gradient(180deg, #ffffff 0%, #eef6ff 100%); }
.tone-success { background: linear-gradient(180deg, #ffffff 0%, #edfdf4 100%); }
.tone-warning { background: linear-gradient(180deg, #ffffff 0%, #fff8eb 100%); }
.tone-danger { background: linear-gradient(180deg, #ffffff 0%, #fff1f2 100%); }
.tone-info { background: linear-gradient(180deg, #ffffff 0%, #f2f8ff 100%); }
.tone-gold { background: linear-gradient(180deg, #ffffff 0%, #fff6e8 100%); }
.tone-purple { background: linear-gradient(180deg, #ffffff 0%, #f6f0ff 100%); }
.tone-ink { background: linear-gradient(180deg, #ffffff 0%, #f2f5fa 100%); }

.panel-card {
  border-radius: 20px;
}

.dist-row {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.dist-row:last-child {
  margin-bottom: 0;
}

.dist-label {
  min-width: 88px;
  font-size: 13px;
  color: #475569;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dist-label.wide {
  min-width: 150px;
}

.dist-count {
  min-width: 40px;
  text-align: right;
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
}

.clickable {
  cursor: pointer;
}

@media (max-width: 1200px) {
  .overview-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 960px) {
  .hero-panel {
    grid-template-columns: 1fr;
  }

  .workbench-grid,
  .overview-grid,
  .hero-quick-actions {
    grid-template-columns: 1fr;
  }
}
</style>
