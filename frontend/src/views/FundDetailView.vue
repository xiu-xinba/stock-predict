<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useFundDetailStore } from '@/stores/fundDetail'
import { useFundCodeRoute } from '@/composables/useFundCodeRoute'
import DetailPageLayout from '@/components/common/DetailPageLayout.vue'
import FundHeader from '@/components/fund/FundHeader.vue'
import FundPerformance from '@/components/fund/FundPerformance.vue'
import FundManager from '@/components/fund/FundManager.vue'
import FundPortfolio from '@/components/fund/FundPortfolio.vue'
import FundRisk from '@/components/fund/FundRisk.vue'
import FundPrediction from '@/components/fund/FundPrediction.vue'

defineOptions({ name: 'FundDetailView' })

const store = useFundDetailStore()
const { fundCode } = useFundCodeRoute()

function loadDetail() {
  if (fundCode.value) store.fetchDetail(fundCode.value)
}

onMounted(loadDetail)
watch(fundCode, loadDetail)
</script>

<template>
  <DetailPageLayout
    :loading="store.loading"
    :error="store.error"
    :code="fundCode"
    :has-content="!!store.detail"
    :skeleton-count="3"
    @retry="loadDetail"
  >
    <template #header>
      <FundHeader v-if="store.detail" :basic="store.detail.basic" :quote="store.detail.quote" />
    </template>
    <template v-if="store.detail">
      <FundPerformance :performance="store.detail.performance" />
      <FundPrediction :fund-code="store.detail.basic.fund_code" />
      <FundManager :manager="store.detail.manager" />
      <FundPortfolio :portfolio="store.detail.portfolio" />
      <FundRisk :risk="store.detail.risk" :risk-level="store.detail.basic.risk_level" />
    </template>
  </DetailPageLayout>
</template>
