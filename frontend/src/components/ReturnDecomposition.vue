<template>
  <div v-if="decomposition?.enabled" class="panel decomposition-panel card">
    <div class="panel-header">
      <span class="panel-mark"></span>
      <span>收益拆解</span>
    </div>
    <div class="panel-body">
      <div class="formula-line">基金收益 = 跟踪指数收益 + 跟踪误差</div>
      <div v-for="item in decompositionItems" :key="item.key" class="decomposition-row">
        <span>{{ item.label }}</span>
        <strong :class="item.cls">{{ item.value }}</strong>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ReturnDecomposition } from '@/types/predict'

defineOptions({ name: 'ReturnDecomposition' })

const props = defineProps<{
  decomposition: ReturnDecomposition | null
}>()

const decompositionItems = computed(() => {
  const d = props.decomposition
  return [
    { key: 'index', label: '跟踪指数收益', ...formatComponentPct(d?.index_return_pct) },
    { key: 'error', label: '跟踪误差', ...formatComponentPct(d?.tracking_error_pct) },
    { key: 'direct', label: '直接回归输出', ...formatComponentPct(d?.direct_fund_return_pct) },
  ]
})

function formatComponentPct(val: number | null | undefined): { value: string; cls: string } {
  if (val == null) return { value: '--%', cls: 'flat' }
  const sign = val >= 0 ? '+' : ''
  return { value: `${sign}${val.toFixed(4)}%`, cls: val > 0 ? 'up' : val < 0 ? 'down' : 'flat' }
}
</script>

<style scoped>
.panel {
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  min-height: 42px;
  padding: 0 var(--sp-4);
  border-bottom: 1px solid var(--color-border-light);
  color: var(--color-text-primary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}

.panel-mark {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-sm);
  background: var(--color-brand);
}

.panel-body {
  padding: var(--sp-3) var(--sp-4);
}

.decomposition-panel {
  grid-column: 1 / -1;
}

.formula-line {
  margin-bottom: var(--sp-2);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.decomposition-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--sp-3);
  min-height: 36px;
  border-bottom: 1px solid var(--color-border-light);
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
}

.decomposition-row:last-child {
  border-bottom: 0;
}

.decomposition-row strong {
  font-size: var(--fs-sm);
}

.decomposition-row strong.up { color: var(--color-up); }
.decomposition-row strong.down { color: var(--color-down); }
.decomposition-row strong.flat { color: var(--color-flat); }
</style>
