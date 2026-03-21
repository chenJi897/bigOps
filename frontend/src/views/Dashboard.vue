<template>
  <div class="dashboard">
    <h2>欢迎使用 BigOps 运维平台</h2>
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>主机资产</template>
          <div class="stat-num">--</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>云账号</template>
          <div class="stat-num">--</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>用户数</template>
          <div class="stat-num">--</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>今日操作</template>
          <div class="stat-num">--</div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.dashboard { padding: 20px; }
.stat-num { font-size: 32px; font-weight: 600; color: #409eff; text-align: center; padding: 20px 0; }
</style>
