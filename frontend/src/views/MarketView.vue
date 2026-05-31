<template>
  <div class="market-page">
    <div class="tab-bar">
      <div class="tab-group">
        <button
          :class="['tab-btn', { active: activeTab === 'fund' }]"
          type="button"
          @click="activeTab = 'fund'"
        >基金排行</button>
        <button
          :class="['tab-btn', { active: activeTab === 'stock' }]"
          type="button"
          @click="activeTab = 'stock'"
        >股票排行</button>
      </div>
    </div>

    <div v-if="store.loading && store.indices.length === 0 && activeTab === 'fund'" class="skeleton-grid">
      <div v-for="i in 3" :key="i" class="skeleton-panel card card-accent-top">
        <div class="sk-header skeleton-pulse"></div>
        <div class="sk-value skeleton-pulse"></div>
        <div class="sk-row skeleton-pulse"></div>
        <div class="sk-chart skeleton-pulse"></div>
      </div>
    </div>

    <ErrorState
      v-else-if="store.error && store.indices.length === 0 && activeTab === 'fund'"
      :message="store.error"
      retry-label="重新加载"
      @retry="store.fetchMarketData(true)"
    />

    <template v-if="activeTab === 'fund'">
      <section v-if="store.topGainers.length > 0 || store.topLosers.length > 0" class="ranking-row">
        <FundRanking title="领涨基金" type="gainers" :items="store.topGainers" />
        <FundRanking title="领跌基金" type="losers" :items="store.topLosers" />
      </section>
    </template>

    <template v-if="activeTab === 'stock'">
      <div v-if="stockRankingLoading" class="skeleton-grid">
        <div v-for="i in 2" :key="i" class="skeleton-panel card card-accent-top">
          <div class="sk-header skeleton-pulse"></div>
          <div class="sk-value skeleton-pulse"></div>
          <div class="sk-row skeleton-pulse"></div>
          <div class="sk-chart skeleton-pulse"></div>
        </div>
      </div>

      <ErrorState
        v-else-if="stockRankingError"
        :message="stockRankingError"
        retry-label="重新加载"
        @retry="fetchStockRankings"
      />

      <section v-else-if="stockGainers.length > 0 || stockLosers.length > 0" class="ranking-row">
        <StockRanking title="领涨股票" type="gainers" :items="stockGainers" />
        <StockRanking title="领跌股票" type="losers" :items="stockLosers" />
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMarketStore } from '@/stores/market'
import { useStaggerEntry } from '@/composables/useStaggerEntry'
import { fetchStockRanking } from '@/api/stock'
import ErrorState from '@/components/ErrorState.vue'
import FundRanking from '@/components/market/FundRanking.vue'
import StockRanking from '@/components/market/StockRanking.vue'
import type { StockRankingItem } from '@/types/stock'

defineOptions({ name: 'MarketView' })

useStaggerEntry('.rank-panel', { staggerMs: 80, translateY: 12 })

const store = useMarketStore()
const route = useRoute()
const router = useRouter()

const activeTab = ref<'fund' | 'stock'>((route.query.tab === 'stock' ? 'stock' : 'fund') as 'fund' | 'stock')
const stockGainers = ref<StockRankingItem[]>([])
const stockLosers = ref<StockRankingItem[]>([])
const stockRankingLoading = ref(false)
const stockRankingError = ref<string | null>(null)

async function fetchStockRankings() {
  stockRankingLoading.value = true
  stockRankingError.value = null
  try {
    const [gainersRes, losersRes] = await Promise.all([
      fetchStockRanking('gainers'),
      fetchStockRanking('losers'),
    ])
    if (gainersRes.code === 0 && gainersRes.data) {
      stockGainers.value = gainersRes.data
    }
    if (losersRes.code === 0 && losersRes.data) {
      stockLosers.value = losersRes.data
    }
  } catch {
    stockRankingError.value = '获取股票排行失败，请稍后重试'
  } finally {
    stockRankingLoading.value = false
  }
}

watch(activeTab, (tab) => {
  router.replace({ query: { ...route.query, tab } })
  if (tab === 'stock' && stockGainers.value.length === 0 && stockLosers.value.length === 0) {
    fetchStockRankings()
  }
})

function handleVisibility() {
  if (document.hidden) {
    store.stopRefresh()
  } else {
    store.startRefresh()
  }
}

onMounted(() => {
  store.startRefresh()
  document.addEventListener('visibilitychange', handleVisibility)
})
onUnmounted(() => {
  store.stopRefresh()
  document.removeEventListener('visibilitychange', handleVisibility)
})
</script>

<style scoped>
.market-page {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}

.tab-bar {
  display: flex;
  align-items: center;
}

.ranking-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--sp-5);
}

.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--sp-5);
}

.skeleton-panel {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  padding: var(--sp-4);
  position: relative;
  overflow: hidden;
}

.skeleton-panel::before {
  display: none;
}

.sk-header,
.sk-value,
.sk-row,
.sk-chart {
  border-radius: var(--radius-sm);
  background: var(--color-bg-hover);
}

.sk-header { width: 60%; height: 18px; }
.sk-value { width: 45%; height: 28px; }
.sk-row { width: 80%; height: 14px; }
.sk-chart { width: 100%; height: 54px; }

@media (max-width: 1024px) {
  .skeleton-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .ranking-row {
    grid-template-columns: 1fr;
  }
}

@media (prefers-reduced-motion: reduce) {
  .skeleton-panel::after,
  .sk-header::after,
  .sk-value::after,
  .sk-row::after,
  .sk-chart::after {
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
  }
}
</style>
