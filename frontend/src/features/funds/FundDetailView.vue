<script setup lang="ts">
/** 基金详情页面，组合展示基金头部、业绩、经理、组合、风险等子组件 */
import { onMounted, watch } from 'vue'
import { useFundDetailStore } from '@/features/funds/store/fundDetail'
import { useFundCodeRoute } from '@/features/funds/composables/useFundCodeRoute'
import DetailPageLayout from '@/shared/components/DetailPageLayout.vue'
import FundHeader from '@/features/funds/components/FundHeader.vue'
import FundPerformance from '@/features/funds/components/FundPerformance.vue'
import FundManager from '@/features/funds/components/FundManager.vue'
import FundPortfolio from '@/features/funds/components/FundPortfolio.vue'
import FundRisk from '@/features/funds/components/FundRisk.vue'
import { PredictionPlaceholder } from '@/features/prediction'

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
      <PredictionPlaceholder :code="fundCode" type="fund" />
      <FundManager :manager="store.detail.manager" />
      <FundPortfolio :portfolio="store.detail.portfolio" />
      <FundRisk :risk="store.detail.risk" :risk-level="store.detail.basic.risk_level" />
    </template>
  </DetailPageLayout>
</template>
