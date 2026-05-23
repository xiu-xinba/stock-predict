<template>
  <div class="market-page">
    <section class="page-head">
      <div>
        <p class="page-kicker">Market</p>
        <h1 class="page-title">市场行情</h1>
        <p class="page-desc">A 股、港股、美股指数与基金涨跌排行</p>
      </div>
      <div class="head-actions">
        <span v-if="store.lastRefresh" class="refresh-time">{{ store.lastRefresh }}</span>
        <span v-else class="refresh-time">等待同步</span>
        <button
          :class="['icon-btn', { spinning: store.loading }]"
          type="button"
          @click="store.fetchMarketData(true)"
          :disabled="store.loading"
          title="刷新数据"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/></svg>
        </button>
      </div>
    </section>

    <div v-if="store.loading && store.indices.length === 0" class="skeleton-grid">
      <div v-for="i in 3" :key="i" class="skeleton-panel">
        <div class="sk-header"></div>
        <div class="sk-value"></div>
        <div class="sk-row"></div>
        <div class="sk-chart"></div>
      </div>
    </div>

    <div v-else-if="store.error && store.indices.length === 0" class="error-state">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>
      </div>
      <p class="error-text">{{ store.error }}</p>
      <button class="retry-btn" type="button" @click="store.fetchMarketData(true)">重新加载</button>
    </div>

    <template v-else>
      <section class="indices-row">
        <MarketSidebar label="A股" market="cn" :indices="cnIndices" />
        <MarketSidebar label="港股" market="hk" :indices="hkIndices" />
        <MarketSidebar label="美股" market="us" :indices="usIndices" />
      </section>

      <section v-if="store.topGainers.length > 0 || store.topLosers.length > 0" class="ranking-row">
        <FundRanking title="领涨基金" type="gainers" :items="store.topGainers" />
        <FundRanking title="领跌基金" type="losers" :items="store.topLosers" />
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useMarketStore } from '@/stores/market'
import MarketSidebar from '@/components/market/MarketSidebar.vue'
import FundRanking from '@/components/market/FundRanking.vue'

const store = useMarketStore()

const cnIndices = computed(() => store.indices.filter(i => i.market === 'cn'))
const hkIndices = computed(() => store.indices.filter(i => i.market === 'hk'))
const usIndices = computed(() => store.indices.filter(i => i.market === 'us'))

onMounted(() => store.startRefresh())
onUnmounted(() => store.stopRefresh())
</script>

<style scoped>
.market-page {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}

.page-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--sp-4);
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

.head-actions {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
}

.refresh-time {
  padding-left: var(--sp-2);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: background-color var(--transition-fast), border-color var(--transition-fast), color var(--transition-fast);
}

.icon-btn svg {
  width: 16px;
  height: 16px;
}

.icon-btn:hover {
  color: var(--color-brand);
  background: var(--color-brand-soft);
  border-color: var(--color-brand-muted);
}

.icon-btn:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.icon-btn.spinning svg {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.indices-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--sp-4);
  align-items: start;
}

.ranking-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--sp-4);
}

.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--sp-4);
}

.skeleton-panel {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  padding: var(--sp-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
}

.sk-header,
.sk-value,
.sk-row,
.sk-chart {
  position: relative;
  overflow: hidden;
  border-radius: var(--radius-sm);
  background: var(--color-bg-hover);
}

.sk-header::after,
.sk-value::after,
.sk-row::after,
.sk-chart::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, transparent 25%, var(--color-border-light) 50%, transparent 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
}

.sk-header { width: 60%; height: 18px; }
.sk-value { width: 45%; height: 28px; }
.sk-row { width: 80%; height: 14px; }
.sk-chart { width: 100%; height: 54px; }

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  padding: var(--sp-8);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
  text-align: center;
}

.error-icon {
  width: 44px;
  height: 44px;
  color: var(--color-warning);
}

.error-text {
  margin: var(--sp-4) 0;
  color: var(--color-text-secondary);
  font-size: var(--fs-base);
}

.retry-btn {
  min-height: 34px;
  padding: 0 var(--sp-4);
  border: 1px solid var(--color-brand);
  border-radius: var(--radius-md);
  background: var(--color-brand);
  color: var(--color-brand-contrast);
  cursor: pointer;
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

@media (max-width: 1100px) {
  .indices-row,
  .skeleton-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .page-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .head-actions {
    width: 100%;
  }

  .ranking-row {
    grid-template-columns: 1fr;
  }
}
</style>
