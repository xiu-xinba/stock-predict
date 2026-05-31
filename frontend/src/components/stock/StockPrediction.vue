<script setup lang="ts">
import { useRouter } from 'vue-router'
import CollapsibleCard from '@/components/CollapsibleCard.vue'

defineOptions({ name: 'StockPrediction' })

const props = defineProps<{ stockCode: string }>()
const router = useRouter()

function openPredictionWorkspace() {
  router.push(`/predict?stockCode=${props.stockCode}`)
}
</script>

<template>
  <CollapsibleCard title="AI 预测" class="card-accent-top" body-max-height="260px">
    <div class="prediction-placeholder">
      <div>
        <h3>预测模型已拆分为独立项目</h3>
        <p>当前主项目保留入口和展示位置，后续将通过独立预测服务接入。</p>
      </div>
      <button type="button" class="placeholder-action" @click="openPredictionWorkspace">
        查看入口
      </button>
    </div>
  </CollapsibleCard>
</template>

<style scoped>
.prediction-placeholder {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-4);
  padding: var(--sp-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-page);
}

h3 {
  margin: 0 0 var(--sp-1);
  color: var(--color-text-primary);
  font-size: var(--fs-base);
}

p {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  line-height: var(--lh-relaxed);
}

.placeholder-action {
  flex: 0 0 auto;
  min-height: 36px;
  padding: 0 var(--sp-4);
  border: 1px solid var(--color-brand-muted);
  border-radius: var(--radius-md);
  background: var(--color-brand-soft);
  color: var(--color-brand);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  cursor: pointer;
}

@media (max-width: 768px) {
  .prediction-placeholder {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
