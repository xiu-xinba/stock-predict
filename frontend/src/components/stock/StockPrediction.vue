<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { usePredictionStore } from '@/stores/prediction'
import CollapsibleCard from '@/components/CollapsibleCard.vue'
import PredictionDisplay from '@/components/PredictionDisplay.vue'

defineOptions({ name: 'StockPrediction' })

const props = defineProps<{ stockCode: string }>()
const router = useRouter()
const store = usePredictionStore()

onMounted(() => {
  if (props.stockCode) store.predictStockAction(props.stockCode)
})

watch(() => props.stockCode, (code) => {
  if (code) store.predictStockAction(code)
})

function goToFullPredict() {
  router.push(`/predict?stockCode=${props.stockCode}`)
}
</script>

<template>
  <CollapsibleCard title="AI 预测" class="card-accent-top" body-max-height="400px">
    <PredictionDisplay
      :prediction="store.stockPrediction"
      :loading="store.stockLoading"
      :error="store.stockError"
      @view-full="goToFullPredict"
    />
  </CollapsibleCard>
</template>
