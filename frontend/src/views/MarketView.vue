<template>
  <div class="market-page">
    <!-- 页面头部：标题 + 刷新状态 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">市场行情</h1>
        <span v-if="store.lastRefresh" class="refresh-time">
          <svg class="refresh-icon" viewBox="0 0 24 24" width="12" height="12"><path fill="currentColor" d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/></svg>
          {{ store.lastRefresh }}
        </span>
      </div>
      <button
        :class="['refresh-btn', { spinning: store.loading }]"
        @click="store.fetchMarketData(true)"
        :disabled="store.loading"
        title="刷新数据"
      >
        <svg viewBox="0 0 24 24" width="16" height="16"><path fill="currentColor" d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/></svg>
      </button>
    </div>

    <!-- 加载骨架屏 -->
    <div v-if="store.loading && store.indices.length === 0" class="skeleton-grid">
      <div v-for="i in 3" :key="i" class="skeleton-panel">
        <div class="sk-header"></div>
        <div class="sk-value"></div>
        <div class="sk-row"></div>
        <div class="sk-chart"></div>
      </div>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="store.error && store.indices.length === 0" class="error-state">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" width="48" height="48"><path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>
      </div>
      <p class="error-text">{{ store.error }}</p>
      <button class="retry-btn" @click="store.fetchMarketData(true)">重新加载</button>
    </div>

    <!-- 主内容区 -->
    <template v-else>
      <!-- 市场指数三栏 -->
      <div class="indices-row">
        <MarketSidebar label="A股" market="cn" :indices="cnIndices" />
        <MarketSidebar label="港股" market="hk" :indices="hkIndices" />
        <MarketSidebar label="美股" market="us" :indices="usIndices" />
      </div>

      <!-- 基金涨跌排行 -->
      <div v-if="store.topGainers.length > 0 || store.topLosers.length > 0" class="ranking-row">
        <FundRanking title="领涨基金" type="gainers" :items="store.topGainers" />
        <FundRanking title="领跌基金" type="losers" :items="store.topLosers" />
      </div>
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
  min-height: calc(100vh - 56px);
  padding-bottom: 100px;
}

/* 页面头部 */
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sp-4) 0 var(--sp-2);
}
.header-left {
  display: flex;
  align-items: baseline;
  gap: var(--sp-3);
}
.page-title {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-extrabold);
  color: var(--color-text-primary);
  letter-spacing: var(--ls-tight);
  margin: 0;
  line-height: var(--lh-tight);
}
.refresh-time {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: var(--fs-xs);
  color: var(--color-text-disabled);
  font-variant-numeric: tabular-nums;
}
.refresh-icon {
  opacity: 0.6;
}
.refresh-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-bg-card);
  color: var(--color-text-regular);
  cursor: pointer;
  transition: all var(--transition-fast);
}
.refresh-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-brand);
  border-color: var(--color-brand);
}
.refresh-btn:active {
  transform: scale(0.92);
}
.refresh-btn.spinning svg {
  animation: spin 0.8s linear infinite;
}
.refresh-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* 指数三栏 */
.indices-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--sp-3);
  align-items: start;
}

/* 基金排行双栏 */
.ranking-row {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--sp-3);
}

/* 骨架屏 */
.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--sp-3);
}
.skeleton-panel {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  padding: var(--sp-4);
  border: 1px solid var(--color-border-light);
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
}
.sk-header,
.sk-value,
.sk-row,
.sk-chart {
  border-radius: var(--radius-sm);
  background: linear-gradient(90deg, var(--color-bg-hover) 25%, var(--color-border-light) 50%, var(--color-bg-hover) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
}
.sk-header { height: 20px; width: 60%; }
.sk-value { height: 28px; width: 45%; }
.sk-row { height: 14px; width: 80%; }
.sk-chart { height: 40px; width: 100%; margin-top: var(--sp-1); }
@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* 错误状态 */
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--sp-12) var(--sp-4);
  text-align: center;
}
.error-icon {
  color: var(--color-text-disabled);
  margin-bottom: var(--sp-4);
}
.error-text {
  font-size: var(--fs-base);
  color: var(--color-text-secondary);
  margin: 0 0 var(--sp-4);
}
.retry-btn {
  padding: var(--sp-2) var(--sp-5);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-brand);
  background: var(--color-brand);
  color: #fff;
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  cursor: pointer;
  transition: all var(--transition-fast);
}
.retry-btn:hover {
  background: var(--color-brand);
  opacity: 0.9;
  box-shadow: 0 2px 8px rgba(51, 102, 255, 0.3);
}
.retry-btn:active {
  transform: scale(0.96);
}

/* 响应式 */
@media (max-width: 768px) {
  .indices-row {
    grid-template-columns: 1fr;
  }
  .ranking-row {
    grid-template-columns: 1fr;
  }
  .skeleton-grid {
    grid-template-columns: 1fr;
  }
  .page-title {
    font-size: var(--fs-xl);
  }
}

@media (max-width: 480px) {
  .market-page {
    gap: var(--sp-3);
  }
  .page-header {
    padding: var(--sp-3) 0 var(--sp-1);
  }
}
</style>
