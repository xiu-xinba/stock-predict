<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { usePredictionStore } from '@/stores/prediction'
import CollapsibleCard from '@/components/CollapsibleCard.vue'
import PredictionDisplay from '@/components/PredictionDisplay.vue'

defineOptions({ name: 'FundPrediction' })

const props = defineProps<{ fundCode: string }>()
const router = useRouter()
const store = usePredictionStore()

onMounted(() => {
  if (props.fundCode) store.predict(props.fundCode)
})

watch(() => props.fundCode, (code) => {
  if (code) store.predict(code)
})

function goToFullPredict() {
  router.push(`/predict/${props.fundCode}`)
}
</script>

<template>
  <CollapsibleCard title="AI 预测" class="card-accent-top" body-max-height="400px">
    <PredictionDisplay
      :prediction="store.prediction"
      :loading="store.loading"
      :error="store.error"
      @view-full="goToFullPredict"
    />
  </CollapsibleCard>
</template>
