<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useStockDetailStore } from '@/stores/stockDetail'
import DetailPageLayout from '@/components/common/DetailPageLayout.vue'
import StockHeader from '@/components/stock/StockHeader.vue'
import StockQuote from '@/components/stock/StockQuote.vue'
import StockKline from '@/components/stock/StockKline.vue'
import StockCapitalFlow from '@/components/stock/StockCapitalFlow.vue'
import StockFinancials from '@/components/stock/StockFinancials.vue'
import StockShareholders from '@/components/stock/StockShareholders.vue'
import StockPrediction from '@/components/stock/StockPrediction.vue'
import type { AppError } from '@/types'

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
    store.error = { code: 0, message: '股票代码格式无效，应为6位数字', retryable: false, type: 'business' } as AppError
    return
  }
  store.fetchDetail(code)
}

onMounted(loadDetail)
watch(stockCode, loadDetail)
</script>

<template>
  <DetailPageLayout
    :loading="store.loading"
    :error="store.error"
    :code="stockCode"
    :has-content="!!store.detail"
    :skeleton-count="7"
    @retry="loadDetail"
  >
    <template #header>
      <StockHeader v-if="store.detail" :basic="store.detail.basic" :quote="store.detail.quote" />
    </template>
    <template v-if="store.detail">
      <StockQuote :quote="store.detail.quote" />
      <StockKline :kline="store.detail.kline" />
      <StockCapitalFlow :capital-flow="store.detail.capital_flow" />
      <StockFinancials :financials="store.detail.financials" />
      <StockShareholders :shareholders="store.detail.shareholders" />
      <StockPrediction :stock-code="store.detail.basic.stock_code" />
    </template>
  </DetailPageLayout>
</template>
