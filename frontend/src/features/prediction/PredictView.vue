<script setup lang="ts">
/** 预测服务迁移提示页面，告知用户预测功能已迁移至独立服务 */
defineOptions({ name: 'PredictView' })
</script>

<template>
  <main class="predict-view">
    <section
      class="migration-card card card-container fade-slide-up"
      aria-labelledby="predict-title"
    >
      <div class="migration-icon" aria-hidden="true">
        <svg viewBox="0 0 24 24">
          <path
            fill="none"
            stroke="currentColor"
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="1.8"
            d="M7 17 17 7m0 0h-7m7 0v7M5 5v14h14"
          />
        </svg>
      </div>

      <div class="migration-content">
        <p class="eyebrow">Prediction Service</p>
        <h1 id="predict-title">预测服务已迁移</h1>
        <p class="description">
          预测模型和推理能力已迁移至独立服务。当前行情项目不再提供预测结果，也不会展示模拟指标或准确率。
        </p>

        <div class="contract-note" role="note">
          <span class="status-code">410 Gone</span>
          <span>已保留旧 API 路径用于兼容调用；这些端点会明确返回资源已迁移。</span>
        </div>
      </div>
    </section>
  </main>
</template>

<style scoped>
.predict-view {
  display: flex;
  min-height: 100%;
  padding: var(--sp-5);
}

.migration-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: start;
  gap: var(--sp-4);
  width: min(760px, 100%);
  margin: clamp(var(--sp-4), 8vh, var(--sp-10)) auto auto;
  padding: clamp(var(--sp-5), 5vw, var(--sp-8));
}

.migration-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border: 1px solid var(--color-brand-muted);
  border-radius: var(--radius-md);
  background: var(--color-brand-soft);
  color: var(--color-brand);
}

.migration-icon svg {
  width: 28px;
  height: 28px;
}

.migration-content {
  min-width: 0;
}

.eyebrow {
  margin: 0 0 var(--sp-1);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

h1 {
  margin: 0;
  color: var(--color-text-primary);
  font-size: var(--fs-2xl);
  line-height: var(--lh-tight);
}

.description {
  max-width: 62ch;
  margin: var(--sp-3) 0 0;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  line-height: var(--lh-relaxed);
}

.contract-note {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  margin-top: var(--sp-5);
  padding: var(--sp-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  line-height: var(--lh-relaxed);
}

.status-code {
  flex-shrink: 0;
  padding: var(--sp-1) var(--sp-2);
  border-radius: var(--radius-sm);
  background: var(--color-brand-soft);
  color: var(--color-brand);
  font-family: var(--font-mono);
  font-weight: var(--fw-semibold);
}

@media (max-width: 768px) {
  .predict-view {
    padding: var(--sp-3);
  }

  .migration-card {
    grid-template-columns: 1fr;
    margin-top: var(--sp-3);
  }

  .contract-note {
    align-items: flex-start;
    flex-direction: column;
  }

  h1 {
    font-size: var(--fs-xl);
  }
}
</style>
