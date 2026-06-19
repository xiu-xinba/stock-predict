<template>
  <div class="skeleton-strip">
    <div v-for="i in rowCount" :key="i" class="skeleton-row">
      <div class="sk-cell sk-code skeleton-pulse"></div>
      <div class="sk-cell sk-name skeleton-pulse"></div>
      <div class="sk-cell sk-nav skeleton-pulse"></div>
      <div class="sk-cell sk-pct skeleton-pulse"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
/** 骨架屏表格组件，在数据加载完成前展示占位行，提升用户感知速度 */
interface Props {
  rowCount?: number
}

withDefaults(defineProps<Props>(), {
  rowCount: 5,
})
</script>

<style scoped>
.skeleton-strip {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--color-bg-card);
}

.skeleton-row {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr) 80px 72px;
  align-items: center;
  gap: var(--sp-3);
  min-height: 64px;
  padding: var(--sp-2) var(--sp-4);
  border-bottom: 1px solid var(--color-border-light);
}

.skeleton-row:last-child {
  border-bottom: 0;
}

.sk-cell {
  height: 16px;
  border-radius: var(--radius-sm);
}

.sk-code {
  width: 56px;
}
.sk-name {
  width: 60%;
}
.sk-nav {
  width: 64px;
}
.sk-pct {
  width: 52px;
}

@media (prefers-reduced-motion: reduce) {
  .skeleton-strip,
  .sk-cell {
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
  }
}
</style>
