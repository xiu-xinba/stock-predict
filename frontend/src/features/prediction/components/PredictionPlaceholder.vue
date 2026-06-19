<script setup lang="ts">
/** 预测服务迁移占位组件，在基金/股票详情页中展示迁移提示 */

defineProps<{
  code: string
  type: 'fund' | 'stock'
}>()
</script>

<template>
  <section class="card card-accent-top">
    <div class="card-header">
      <div class="card-title-wrap">
        <h2 class="card-title">AI 预测</h2>
      </div>
    </div>
    <div class="card-body">
      <div class="prediction-placeholder">
        <div class="prediction-content">
          <svg class="prediction-icon" viewBox="0 0 24 24" aria-hidden="true">
            <path
              fill="currentColor"
              d="M12 2L9.19 8.63 2 9.24l5.46 4.73L5.82 21 12 17.27 18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2z"
              opacity="0.15"
            />
            <path
              fill="currentColor"
              d="M9.5 2a.5.5 0 0 1 .5.5v2a.5.5 0 0 1-1 0v-2a.5.5 0 0 1 .5-.5zm5 0a.5.5 0 0 1 .5.5v2a.5.5 0 0 1-1 0v-2a.5.5 0 0 1 .5-.5zM12 5a7 7 0 1 1 0 14 7 7 0 0 1 0-14zm0 1a6 6 0 1 0 0 12 6 6 0 0 0 0-12zm-1 3v4l3.5 2.1.5-.87-3-1.78V9h-1z"
            />
          </svg>
          <div>
            <h3>预测服务已迁移</h3>
            <p>当前行情项目不再提供预测结果。旧预测 API 已废弃并返回 <code>410 Gone</code>。</p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.prediction-placeholder {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-4);
  padding: var(--sp-4);
  border: 1px dashed var(--color-brand-muted);
  border-radius: var(--radius-md);
  background: var(--color-brand-soft);
  animation: dash-pulse 3s ease-in-out infinite;
}

@keyframes dash-pulse {
  0%,
  100% {
    border-color: var(--color-brand-muted);
  }
  50% {
    border-color: var(--color-brand);
  }
}

.prediction-content {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
}

.prediction-icon {
  width: 24px;
  height: 24px;
  color: var(--color-brand);
  flex-shrink: 0;
}

h3 {
  margin: 0 0 var(--sp-1);
  color: var(--color-text-primary);
  font-size: var(--fs-base);
}

p {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  line-height: var(--lh-relaxed);
}

code {
  color: var(--color-brand);
  font-family: var(--font-mono);
  font-weight: var(--fw-semibold);
}

@media (max-width: 768px) {
  .prediction-placeholder {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
