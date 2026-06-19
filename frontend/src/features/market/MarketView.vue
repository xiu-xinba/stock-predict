<template>
  <div class="market-page">
    <!-- 顶部工具栏 -->
    <section class="market-toolbar card-glass fade-slide-up" style="--delay: 0">
      <div class="market-status">
        <span v-if="store.loading" class="status-loading">行情加载中</span>
        <span v-else-if="store.error" class="status-error">{{ store.error }}</span>
        <span v-else class="status-ready"><span class="live-dot"></span>实时行情</span>
        <span v-if="marketTimestamp" class="status-meta">最后更新 {{ marketTimestamp }}</span>
        <span v-if="marketSource" class="status-source">{{ marketSource }}</span>
      </div>
      <button
        class="refresh-btn btn-ghost"
        data-test="market-refresh"
        type="button"
        aria-label="刷新行情数据"
        :disabled="store.loading || store.stockRankingLoading"
        @click="refreshAllData"
      >
        刷新
      </button>
    </section>

    <!-- 指数概览卡片条 -->
    <IndexCards
      class="fade-slide-up"
      style="--delay: 1"
      :indices="store.indices"
      :selected-index="selectedMinuteIndex"
      :minute-data="store.indexMinute"
      @select="onIndexSelect"
    />

    <!-- Bento Grid - 不对称布局 -->
    <div class="bento-grid fade-slide-up" style="--delay: 2">
      <!-- 左列：分时图 (占 5/8 宽度) -->
      <div class="bento-cell bento-chart card card-spotlight" @mousemove="handleSpotlight">
        <div class="chart-header">
          <h3 class="section-heading">{{ selectedMinuteName }} 分时走势</h3>
          <span class="chart-code">{{ selectedMinuteIndex }}</span>
        </div>
        <IndexMinuteChart
          :code="selectedMinuteIndex"
          :minute-data="store.indexMinute.get(selectedMinuteIndex) ?? []"
          :quote="selectedMinuteQuote!"
          :height="360"
        />
      </div>

      <!-- 右列：涨跌排行 (占 3/8 宽度) -->
      <div class="bento-cell bento-rankings">
        <div class="bento-rankings-inner">
          <div class="rankings-toolbar">
            <button
              :class="['tab-btn', 'tab-direction', { active: rankDirection === 'up' }]"
              type="button"
              aria-label="涨幅榜"
              @click="rankDirection = 'up'"
            >
              <span class="dir-dot up"></span>涨幅榜
            </button>
            <div class="tab-group">
              <button
                :class="['tab-btn', { active: activeTab === 'fund' }]"
                type="button"
                aria-label="切换到基金排行"
                @click="activeTab = 'fund'"
              >
                基金
              </button>
              <button
                :class="['tab-btn', { active: activeTab === 'stock' }]"
                type="button"
                aria-label="切换到股票排行"
                @click="activeTab = 'stock'"
              >
                股票
              </button>
            </div>
            <button
              :class="['tab-btn', 'tab-direction', { active: rankDirection === 'down' }]"
              type="button"
              aria-label="跌幅榜"
              @click="rankDirection = 'down'"
            >
              <span class="dir-dot down"></span>跌幅榜
            </button>
          </div>
          <div class="rankings-body">
            <Transition name="rank-switch">
              <FundRanking
                v-if="activeTab === 'fund'"
                :key="`fund-${rankDirection}`"
                :items="rankDirection === 'up' ? store.topGainers : store.topLosers"
                :type="rankDirection"
                :loading="store.loading"
                :error="store.error"
              />
              <StockRanking
                v-else
                :key="`stock-${rankDirection}`"
                :items="rankDirection === 'up' ? store.stockGainers : store.stockLosers"
                :type="rankDirection"
                :loading="store.stockRankingLoading"
                :error="store.stockRankingError"
              />
            </Transition>
          </div>
        </div>
      </div>

      <!-- 底部左：板块热力图 -->
      <div class="bento-cell bento-sector">
        <SectorHeat :sectors="store.sectors" :loading="store.loading" :error="store.error" />
      </div>

      <!-- 底部右：沪深港通资金 -->
      <div class="bento-cell bento-northbound">
        <HSGTFlowChart
          :data="store.hsgtData"
          :loading="store.hsgtLoading"
          :error="store.hsgtError"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
/** 行情总览页面，展示指数走势、排行、板块热力图、北向资金等 */
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMarketStore } from '@/features/market'
import { useStaggerEntry } from '@/shared/composables/useStaggerEntry'
import FundRanking from '@/features/market/components/FundRanking.vue'
import StockRanking from '@/features/market/components/StockRanking.vue'
import SectorHeat from '@/features/market/components/SectorHeat.vue'
import HSGTFlowChart from '@/features/market/components/HSGTFlowChart.vue'
import IndexMinuteChart from '@/features/market/components/IndexMinuteChart.vue'
import IndexCards from '@/features/market/components/IndexCards.vue'

defineOptions({ name: 'MarketView' })

useStaggerEntry('.rank-panel', { staggerMs: 80, translateY: 12 })

const store = useMarketStore()
const route = useRoute()
const router = useRouter()

const activeTab = ref<'fund' | 'stock'>(
  (route.query.tab === 'stock' ? 'stock' : 'fund') as 'fund' | 'stock',
)
const rankDirection = ref<'up' | 'down'>('up')

const selectedMinuteIndex = ref('000001')

const selectedMinuteName = computed(() => {
  const idx = store.indices.find((i) => i.code === selectedMinuteIndex.value)
  return idx?.name ?? '上证指数'
})

const selectedMinuteQuote = computed(() => {
  return store.indices.find((i) => i.code === selectedMinuteIndex.value) ?? null
})

function onIndexSelect(code: string) {
  selectedMinuteIndex.value = code
  if (!store.indexMinute.get(code)?.length) {
    store.fetchIndexMinuteData(code)
  }
}

/** 卡片聚光灯效果：追踪鼠标位置 */
function handleSpotlight(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  el.style.setProperty('--mouse-x', `${e.clientX - rect.left}px`)
  el.style.setProperty('--mouse-y', `${e.clientY - rect.top}px`)
}

const marketTimestamp = computed(() => {
  const idx = store.indices.find((i) => i.code === selectedMinuteIndex.value)
  return idx?.update_time || store.lastRefresh
})
const marketSource = computed(() => {
  const idx = store.indices.find((i) => i.code === selectedMinuteIndex.value)
  return idx?.data_source || ''
})

function loadAllData(force = false) {
  store.fetchMarketData(force)
  store.fetchSectorData()
  store.fetchHSGTData()
  store.fetchStockRankingData(5)
  // 加载主要指数分时数据
  for (const code of ['000001', '399001', '399006']) {
    store.fetchIndexMinuteData(code)
  }
}

function refreshAllData() {
  loadAllData(true)
}

watch(activeTab, (tab) => {
  router.replace({ query: { ...route.query, tab } })
})

function handleVisibility() {
  if (document.hidden) {
    store.stopRefresh()
  } else {
    store.startRefresh()
  }
}

onMounted(() => {
  loadAllData()
  store.startRefresh()
  store.startMinuteRefresh()
  document.addEventListener('visibilitychange', handleVisibility)
})

onUnmounted(() => {
  store.stopRefresh()
  store.stopMinuteRefresh()
  document.removeEventListener('visibilitychange', handleVisibility)
})
</script>

<style scoped>
.market-page {
  display: flex;
  flex-direction: column;
  gap: var(--sp-5);
  max-width: 1440px;
  margin: 0 auto;
}

/* ── Toolbar ── */
.market-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-3);
  padding: var(--sp-3) var(--sp-4);
  border-left: 3px solid var(--color-brand);
}

.market-status {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  min-width: 0;
  flex-wrap: wrap;
}

.status-loading,
.status-ready,
.status-error {
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

.status-loading {
  color: var(--color-brand);
}
.status-ready {
  color: var(--color-text-primary);
}
.status-error {
  color: var(--color-warning);
}

.status-meta,
.status-source {
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
}

.refresh-btn {
  min-height: 30px;
}
.refresh-btn:disabled {
  color: var(--color-text-disabled);
  cursor: not-allowed;
  opacity: 0.6;
}

/* ── Section Headings ── */
.section-heading {
  margin: 0;
  color: var(--color-text-primary);
  font-size: var(--fs-md);
  font-weight: var(--fw-bold);
  letter-spacing: var(--ls-tight);
}

/* ── Chart Header ── */
.chart-header {
  display: flex;
  align-items: baseline;
  gap: var(--sp-2);
  padding: var(--sp-3) var(--sp-4);
  border-bottom: 1px solid var(--color-border-light);
}

.chart-code {
  font-size: var(--fs-xs);
  font-family: var(--font-mono);
  color: var(--color-text-disabled);
  letter-spacing: var(--ls-wide);
}

/* ── Bento Grid ── */
.bento-grid {
  display: grid;
  grid-template-columns: 7fr 3fr;
  grid-template-rows: auto 1fr;
  gap: var(--sp-5);
  align-items: stretch;
}

.bento-cell {
  min-width: 0;
  min-height: 0;
}

.bento-chart {
  grid-column: 1;
  grid-row: 1;
}

.bento-rankings {
  grid-column: 2;
  grid-row: 1;
  display: flex;
  flex-direction: column;
}

.bento-sector {
  grid-column: 2;
  grid-row: 2;
  display: flex;
}

.bento-northbound {
  grid-column: 1;
  grid-row: 2;
  display: flex;
}

/* Rankings */
.bento-rankings-inner {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  flex: 1;
  min-height: 0;
}

.rankings-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-1);
  padding: var(--sp-2) var(--sp-3);
  background: var(--color-bg-elevated);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
}

.dir-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  margin-right: 4px;
  vertical-align: middle;
  transition: box-shadow var(--transition-fast);
}

.dir-dot.up {
  background: var(--color-up);
}

.dir-dot.down {
  background: var(--color-down);
}

.tab-direction {
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  padding: var(--sp-1) var(--sp-2_5);
  border-radius: var(--radius-md);
  color: var(--color-text-secondary);
  transition:
    color var(--transition-fast),
    background var(--transition-fast),
    box-shadow var(--transition-fast);
}

.tab-direction:hover {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}

.tab-direction.active {
  font-weight: var(--fw-bold);
  color: var(--color-brand-contrast);
  background: var(--color-brand);
  box-shadow: 0 2px 8px color-mix(in srgb, var(--color-brand) 30%, transparent);
}

.tab-direction.active .dir-dot.up {
  box-shadow: 0 0 6px var(--color-up);
  background: #fff;
}

.tab-direction.active .dir-dot.down {
  box-shadow: 0 0 6px var(--color-down);
  background: #fff;
}

.rankings-body {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  align-items: stretch;
  flex: 1;
  min-height: 0;
}

/* ── Ranking switch transition (cross-fade, no layout shift) ── */
.rankings-body {
  position: relative;
}

.rank-switch-enter-active {
  transition: opacity 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}

.rank-switch-leave-active {
  transition: opacity 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  position: absolute;
  inset: 0;
}

.rank-switch-enter-from {
  opacity: 0;
}

.rank-switch-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .rank-switch-enter-active,
  .rank-switch-leave-active {
    transition-duration: 0.01ms !important;
  }
}

/* ── Responsive ── */
@media (max-width: 1200px) {
  .bento-grid {
    grid-template-columns: 1fr;
  }
  .bento-chart,
  .bento-rankings,
  .bento-sector,
  .bento-northbound {
    grid-column: 1;
    grid-row: auto;
  }
}

@media (max-width: 768px) {
  .market-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }
  .market-status {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--sp-1);
  }
}

@media (max-width: 480px) {
  .market-page {
    gap: var(--sp-3);
  }
}
</style>
