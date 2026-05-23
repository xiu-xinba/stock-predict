<template>
  <div class="predict-view">
    <section class="page-head">
      <div>
        <p class="page-kicker">Prediction</p>
        <h1 class="page-title">模型预测</h1>
        <p class="page-desc">基金方向、波动区间与因子贡献</p>
      </div>
    </section>

    <FundSearch />

    <transition name="fade" mode="out-in">
      <div v-if="store.loading" key="loading" class="state-loading">
        <div class="skeleton-top"></div>
        <div class="skeleton-grid">
          <div class="skeleton-block"></div>
          <div class="skeleton-block"></div>
        </div>
      </div>

      <PredictionCard v-else-if="store.prediction" key="result" />

      <div v-else-if="store.error" key="error" class="state-panel error">
        <div class="state-icon">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2 1 21h22L12 2zm1 15h-2v-2h2v2zm0-4h-2V8h2v5z"/></svg>
        </div>
        <div>
          <h2>预测请求失败</h2>
          <p>{{ store.error }}</p>
        </div>
      </div>

      <div v-else key="empty" class="state-panel empty">
        <div class="state-icon">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 5a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v14H4V5zm2 0v12h12V5H6zm2 2h8v2H8V7zm0 4h5v2H8v-2z"/></svg>
        </div>
        <div>
          <h2>等待基金代码</h2>
          <p>预测结果将在此处生成。</p>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { usePredictionStore } from '@/stores/prediction'
import FundSearch from '@/components/FundSearch.vue'
import PredictionCard from '@/components/PredictionCard.vue'

const store = usePredictionStore()
const route = useRoute()

function loadPrediction() {
  const fundCode = route.params.fundCode as string
  if (fundCode && /^\d{6}$/.test(fundCode)) {
    store.predict(fundCode)
  }
}

onMounted(loadPrediction)
watch(() => route.params.fundCode, loadPrediction)
</script>

<style scoped>
.predict-view {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}

.page-head {
  padding: var(--sp-2) 0 var(--sp-1);
}

.page-kicker {
  margin: 0 0 var(--sp-1);
  color: var(--color-brand);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  line-height: var(--lh-tight);
}

.page-title {
  margin: 0;
  color: var(--color-text-primary);
  font-size: var(--fs-3xl);
  font-weight: var(--fw-extrabold);
  line-height: var(--lh-snug);
}

.page-desc {
  margin: var(--sp-1) 0 0;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
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
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
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

.skeleton-top::after,
.skeleton-block::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, transparent 25%, var(--color-border-light) 50%, transparent 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.state-panel {
  display: flex;
  align-items: center;
  gap: var(--sp-4);
  min-height: 160px;
  padding: var(--sp-6);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
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

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.16s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (max-width: 760px) {
  .skeleton-grid {
    grid-template-columns: 1fr;
  }

  .state-panel {
    align-items: flex-start;
    padding: var(--sp-4);
  }
}
</style>
