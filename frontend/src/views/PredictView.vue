<template>
  <div class="predict-view">
    <section class="placeholder-panel">
      <div class="placeholder-icon" aria-hidden="true">
        <svg viewBox="0 0 24 24">
          <path fill="currentColor" d="M4 5.5A2.5 2.5 0 0 1 6.5 3h11A2.5 2.5 0 0 1 20 5.5v13a2.5 2.5 0 0 1-2.5 2.5h-11A2.5 2.5 0 0 1 4 18.5zm2.5-.5a.5.5 0 0 0-.5.5v13a.5.5 0 0 0 .5.5h11a.5.5 0 0 0 .5-.5v-13a.5.5 0 0 0-.5-.5zm2 3h7v2h-7zm0 4h7v2h-7zm0 4h4v2h-4z"/>
        </svg>
      </div>
      <div>
        <p class="eyebrow">Prediction Workspace</p>
        <h1>预测模型已拆分为独立项目</h1>
        <p class="copy">当前主项目保留入口和展示位置，后续将通过独立预测服务接入。</p>
        <p v-if="targetCode" class="code-chip">{{ targetLabel }} {{ targetCode }}</p>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useFundCodeRoute } from '@/composables/useFundCodeRoute'
import { computed } from 'vue'
import { useRoute } from 'vue-router'

defineOptions({ name: 'PredictView' })

const route = useRoute()
const { fundCode } = useFundCodeRoute()

const stockCode = computed(() => {
  const raw = route.query.stockCode
  const code = Array.isArray(raw) ? raw[0] : raw
  return code && /^\d{6}$/.test(code) ? code : ''
})

const targetCode = computed(() => fundCode.value || stockCode.value)
const targetLabel = computed(() => (fundCode.value ? '当前基金' : '当前股票'))
</script>

<style scoped>
.predict-view {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}

.placeholder-panel {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: var(--sp-4);
  align-items: center;
  min-height: 220px;
  padding: var(--sp-6);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
  box-shadow: var(--shadow-sm);
}

.placeholder-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-brand-soft);
  color: var(--color-brand);
}

.placeholder-icon svg {
  width: 28px;
  height: 28px;
}

.eyebrow {
  margin: 0 0 var(--sp-1);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  text-transform: uppercase;
}

h1 {
  margin: 0;
  color: var(--color-text-primary);
  font-size: var(--fs-2xl);
  line-height: var(--lh-tight);
}

.copy {
  margin: var(--sp-2) 0 0;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  line-height: var(--lh-relaxed);
}

.code-chip {
  display: inline-flex;
  margin: var(--sp-4) 0 0;
  padding: var(--sp-1) var(--sp-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-full);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  background: var(--color-bg-page);
}

@media (max-width: 768px) {
  .placeholder-panel {
    grid-template-columns: 1fr;
    padding: var(--sp-4);
  }

  h1 {
    font-size: var(--fs-xl);
  }
}
</style>
