<template>
  <div class="not-found-page">
    <svg viewBox="0 0 120 120" class="not-found-svg" aria-hidden="true">
      <text x="50%" y="45%" text-anchor="middle" dominant-baseline="central" font-size="48" font-weight="800" fill="var(--color-brand)" opacity="0.18">404</text>
      <circle cx="60" cy="88" r="4" fill="var(--color-text-secondary)" opacity="0.4"/>
    </svg>
    <h1 class="not-found-title">页面未找到</h1>
    <p class="not-found-desc">您访问的页面不存在，请检查地址是否正确。</p>
    <router-link to="/watchlist" class="btn-primary">返回首页</router-link>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'NotFoundView' })
</script>

<style scoped>
.not-found-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  padding: var(--sp-8) var(--sp-4);
  text-align: center;
}

.not-found-svg {
  width: 120px;
  height: 120px;
  margin-bottom: var(--sp-6);
}

.not-found-title {
  margin: 0 0 var(--sp-2);
  color: var(--color-text-primary);
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
}

.not-found-desc {
  margin: 0 0 var(--sp-6);
  color: var(--color-text-secondary);
  font-size: var(--fs-base);
  line-height: var(--lh-relaxed);
  max-width: 320px;
}
</style>
