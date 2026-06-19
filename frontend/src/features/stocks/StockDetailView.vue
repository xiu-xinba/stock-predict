<script setup lang="ts">
/** 股票详情页面，双栏 bento grid 布局展示走势图、盘口、资金流向、财务指标、股东信息等。
 * 视觉层级：走势图/盘口=main，资金流向/公司概况=secondary，财务/股东=auxiliary。
 * 错落进入动效：同行同时进入，不同行按层级递增延迟。
 */
import { computed, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useStockDetailStore } from '@/features/stocks/store/stockDetail'
import DetailPageLayout from '@/shared/components/DetailPageLayout.vue'
import StockHeader from '@/features/stocks/components/StockHeader.vue'
import MinuteChart from '@/features/stocks/components/MinuteChart.vue'
import OrderBook from '@/features/stocks/components/OrderBook.vue'
import StockCapitalFlow from '@/features/stocks/components/StockCapitalFlow.vue'
import StockFinancials from '@/features/stocks/components/StockFinancials.vue'
import StockShareholders from '@/features/stocks/components/StockShareholders.vue'
import StockResearch from '@/features/stocks/components/StockResearch.vue'
import StockDividends from '@/features/stocks/components/StockDividends.vue'
import StockMargin from '@/features/stocks/components/StockMargin.vue'
import StockShareholderTrend from '@/features/stocks/components/StockShareholderTrend.vue'
import StockRestricted from '@/features/stocks/components/StockRestricted.vue'
import CompanyInfo from '@/features/stocks/components/CompanyInfo.vue'
import { PredictionPlaceholder } from '@/features/prediction'
import type { AppError } from '@/shared/types/errors'

defineOptions({ name: 'StockDetailView' })

const route = useRoute()
const store = useStockDetailStore()

const stockCode = computed(() => {
  const raw = route.params.stockCode
  const code = Array.isArray(raw) ? raw[0] : raw
  return code || ''
})

function loadDetail() {
  const code = stockCode.value
  if (!code || !/^\d{6}$/.test(code)) {
    store.error = {
      code: 0,
      message: '股票代码格式无效，应为6位数字',
      retryable: false,
      type: 'business',
    } as AppError
    return
  }
  store.fetchDetail(code)
  store.startMinutePolling(code)
}

watch(stockCode, loadDetail, { immediate: true })
onUnmounted(() => store.stopMinutePolling())
</script>

<template>
  <DetailPageLayout
    :loading="store.loading"
    :error="store.error"
    :code="stockCode"
    :has-content="!!store.detail"
    :skeleton-count="7"
    wide
    @retry="loadDetail"
  >
    <template #header>
      <StockHeader
        v-if="store.detail"
        :basic="store.detail.basic"
        :quote="store.detail.quote"
        :financials="store.detail.financials"
      />
    </template>
    <template v-if="store.detail">
      <!-- Bento Grid: 主图 + 盘口 (L1 main tier) -->
      <div class="bento-row bento-main fade-slide-up" style="--delay: 2">
        <div class="bento-cell bento-chart">
          <MinuteChart
            :stock-code="stockCode"
            :quote="store.detail.quote"
            :kline="store.detail.kline"
            :minute-data="store.minuteData"
          />
        </div>
        <div class="bento-cell bento-orderbook">
          <OrderBook :quote="store.detail.quote" />
        </div>
      </div>

      <!-- Bento Grid: 资金流向 + 公司概况 (L2 secondary tier) -->
      <div class="bento-row bento-secondary fade-slide-up" style="--delay: 3">
        <div class="bento-cell bento-flow">
          <StockCapitalFlow :capital-flow="store.detail.capital_flow" />
        </div>
        <div class="bento-cell bento-company">
          <CompanyInfo
            :basic="store.detail.basic"
            :quote="store.detail.quote"
            :financials="store.detail.financials"
          />
        </div>
      </div>

      <!-- Bento Grid: 财务指标 + 股东信息 (L3 auxiliary tier) -->
      <div class="bento-row bento-tertiary fade-slide-up" style="--delay: 4">
        <div class="bento-cell bento-financials">
          <StockFinancials :financials="store.detail.financials" />
        </div>
        <div class="bento-cell bento-shareholders">
          <StockShareholders :shareholders="store.detail.shareholders" />
        </div>
      </div>

      <!-- Bento Grid: 研报评级 + 分红送配 + 限售解禁 (L3 auxiliary tier, 表格类) -->
      <div
        v-if="store.detail.research || store.detail.dividends || store.detail.restricted"
        class="bento-row bento-quaternary fade-slide-up"
        style="--delay: 5"
      >
        <div v-if="store.detail.research" class="bento-cell">
          <StockResearch :research="store.detail.research" />
        </div>
        <div v-if="store.detail.dividends" class="bento-cell">
          <StockDividends :dividends="store.detail.dividends" />
        </div>
        <div v-if="store.detail.restricted" class="bento-cell">
          <StockRestricted :restricted="store.detail.restricted" />
        </div>
      </div>

      <!-- Bento Grid: 融资融券 + 股东人数变化 (L3 auxiliary tier, 图表类) -->
      <div
        v-if="store.detail.margin || store.detail.shareholder_trend"
        class="bento-row bento-quinary fade-slide-up"
        style="--delay: 6"
      >
        <div v-if="store.detail.margin" class="bento-cell">
          <StockMargin :margin="store.detail.margin" />
        </div>
        <div v-if="store.detail.shareholder_trend" class="bento-cell">
          <StockShareholderTrend :shareholder-trend="store.detail.shareholder_trend" />
        </div>
      </div>

      <div class="prediction-section fade-slide-up" style="--delay: 7">
        <PredictionPlaceholder :code="stockCode" type="stock" />
      </div>
    </template>
  </DetailPageLayout>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 80ms);
}

.bento-row {
  display: grid;
  gap: var(--sp-4);
}

.bento-main {
  grid-template-columns: 1fr 340px;
}

.bento-secondary {
  grid-template-columns: 1fr 340px;
}

.bento-tertiary {
  grid-template-columns: 1fr 1fr;
}

.bento-quaternary {
  grid-template-columns: repeat(3, 1fr);
}

.bento-quinary {
  grid-template-columns: 1fr 1fr;
}

.bento-cell {
  min-width: 0;
}

.bento-cell > :deep(*) {
  height: 100%;
}

.prediction-section {
  margin-top: var(--sp-3);
}

@media (max-width: 1024px) {
  .bento-main,
  .bento-secondary {
    grid-template-columns: 1fr;
  }

  .bento-tertiary,
  .bento-quinary {
    grid-template-columns: 1fr;
  }

  .bento-quaternary {
    grid-template-columns: 1fr;
  }
}
</style>
