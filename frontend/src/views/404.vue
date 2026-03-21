<template>
  <div class="not-found">
    <div class="code">404</div>
    <div class="title">页面不存在</div>
    <p class="desc">你访问的页面已被删除或不存在，请检查路径是否正确</p>
    <el-button type="primary" size="large" @click="$router.push('/dashboard')">
      <el-icon style="margin-right:6px"><House /></el-icon>返回首页
    </el-button>
  </div>
</template>

<style scoped>
.not-found {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 80vh;
  gap: 16px;
}
.code {
  font-size: 100px;
  font-weight: 900;
  line-height: 1;
  background: linear-gradient(135deg, #304156, #409eff);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  user-select: none;
}
.title {
  font-size: 22px;
  font-weight: 600;
  color: #303133;
}
.desc {
  font-size: 14px;
  color: #909399;
  margin: 0 0 8px;
}
</style>
