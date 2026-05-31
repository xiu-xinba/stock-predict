<template>
  <div class="predict-view">
    <transition name="fade" mode="out-in">
      <div v-if="store.loading" key="loading" class="state-loading">
        <div class="skeleton-top card skeleton-pulse"></div>
        <div class="skeleton-grid">
          <div class="skeleton-block card skeleton-pulse"></div>
          <div class="skeleton-block card skeleton-pulse"></div>
        </div>
      </div>

      <PredictionCard v-else-if="store.prediction" key="result" />

      <div v-else-if="store.error" key="error" class="state-panel error card">
        <div class="state-icon">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2 1 21h22L12 2zm1 15h-2v-2h2v2zm0-4h-2V8h2v5z"/></svg>
        </div>
        <div>
          <h2>预测请求失败</h2>
          <p>{{ store.error }}</p>
          <button class="retry-btn" type="button" @click="loadPrediction">重试</button>
        </div>
      </div>

      <div v-else key="empty" class="state-panel empty card">
        <div class="state-icon">
          <svg viewBox="0 0 1024 1024" aria-hidden="true"><path fill="currentColor" d="m795.904 750.72 124.992 124.928a32 32 0 0 1-45.248 45.248L750.656 795.904a416 416 0 1 1 45.248-45.248zM480 832a352 352 0 1 0 0-704 352 352 0 0 0 0 704"/></svg>
        </div>
        <div>
          <h2>搜索基金开始预测</h2>
          <p>点击右上角搜索图标，输入基金名称或代码</p>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { usePredictionStore } from '@/stores/prediction'
import { useFundCodeRoute } from '@/composables/useFundCodeRoute'
import PredictionCard from '@/components/PredictionCard.vue'

defineOptions({ name: 'PredictView' })

const store = usePredictionStore()
const { fundCode } = useFundCodeRoute()

function loadPrediction() {
  if (fundCode.value) store.predict(fundCode.value)
}

onMounted(loadPrediction)
watch(fundCode, loadPrediction)
</script>

<style scoped>
.predict-view {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}

.state-loading {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}

.skeleton-top,
.skeleton-block {
  position: relative;
  overflow: hidden;
}

.skeleton-top {
  height: 116px;
}

.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--sp-4);
}

.skeleton-block {
  height: 240px;
}

.state-panel {
  display: flex;
  align-items: center;
  gap: var(--sp-4);
  min-height: 160px;
  padding: var(--sp-6);
}

.state-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
}

.state-icon svg {
  width: 24px;
  height: 24px;
}

.state-panel.error .state-icon {
  color: var(--color-warning);
  background: var(--color-warning-bg);
}

.state-panel h2 {
  margin: 0 0 var(--sp-1);
  color: var(--color-text-primary);
  font-size: var(--fs-base);
}

.state-panel p {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
}

.retry-btn {
  margin-top: var(--sp-2);
  padding: var(--sp-2) var(--sp-4);
  border-radius: var(--radius-md);
  background: var(--color-brand);
  color: var(--color-brand-contrast);
  font-size: var(--fs-sm);
  cursor: pointer;
  border: none;
  transition: background var(--transition-fast);
}

.retry-btn:hover {
  background: var(--color-brand-hover);
}

@media (max-width: 768px) {
  .skeleton-grid {
    grid-template-columns: 1fr;
  }

  .state-panel {
    align-items: flex-start;
    padding: var(--sp-4);
  }
}

@media (prefers-reduced-motion: reduce) {
  .fade-enter-active,
  .fade-leave-active,
  .skeleton-top::after,
  .skeleton-block::after {
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
  }
}
</style>
