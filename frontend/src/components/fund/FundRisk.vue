<script setup lang="ts">
import { computed } from 'vue'
import { colorWithAlpha } from '@/utils/format'
import CollapsibleCard from '@/components/CollapsibleCard.vue'
import type { FundRiskMetrics } from '@/types'

defineOptions({ name: 'FundRisk' })

const props = defineProps<{
  risk: FundRiskMetrics
  riskLevel?: string
}>()

const riskItems = computed(() => [
  { label: '年化波动率', value: props.risk.volatility_1y, unit: '%', desc: '衡量收益率的波动程度' },
  { label: '最大回撤', value: props.risk.max_drawdown_1y, unit: '%', desc: '历史最大亏损幅度' },
  { label: '夏普比率', value: props.risk.sharpe_1y, unit: '', desc: '风险调整后的收益指标' },
  { label: 'Beta系数', value: props.risk.beta_1y, unit: '', desc: '相对市场的敏感度' },
])

const riskLevelMap: Record<string, { label: string; color: string }> = {
  '低风险': { label: '低', color: 'var(--color-risk-low)' },
  '中低风险': { label: '中低', color: 'var(--color-risk-medium-low)' },
  '中风险': { label: '中', color: 'var(--color-risk-medium)' },
  '中高风险': { label: '中高', color: 'var(--color-risk-medium-high)' },
  '高风险': { label: '高', color: 'var(--color-risk-high)' },
}

const riskInfo = computed(() => (props.riskLevel ? riskLevelMap[props.riskLevel] : undefined) || { label: '-', color: 'var(--color-text-tertiary)' })
</script>

<template>
  <CollapsibleCard title="风险指标" :default-collapsed="false" class="card-container" body-max-height="300px">
    <template #header-extra>
      <div v-if="riskLevel" class="risk-level-badge" :style="{ background: colorWithAlpha(riskInfo.color, 0.12), color: riskInfo.color }">
        {{ riskInfo.label }}
      </div>
    </template>

    <div v-if="risk.volatility_1y == null && risk.max_drawdown_1y == null" class="empty-hint">
      暂无风险指标数据
    </div>
    <div v-else class="risk-grid">
      <div v-for="item in riskItems" :key="item.label" class="risk-item">
        <div class="risk-label-row">
          <span class="risk-label">{{ item.label }}</span>
          <span class="risk-desc">{{ item.desc }}</span>
        </div>
        <div class="risk-value-row">
          <span class="risk-value" :class="{ 'text-down': item.value < 0 }">
            {{ (item.value ?? 0) > 0 && item.unit === '%' ? '+' : '' }}{{ (item.value ?? 0).toFixed(2) }}{{ item.unit }}
          </span>
        </div>
      </div>
    </div>
  </CollapsibleCard>
</template>

<style scoped>
.risk-level-badge {
  font-size: var(--fs-xs); padding: 2px var(--sp-2); border-radius: var(--radius-full);
  font-weight: var(--fw-medium); margin-right: auto;
}

.risk-grid {
  display: grid; grid-template-columns: repeat(2, 1fr); gap: var(--sp-4);
}

.risk-item {
  min-width: 0;
}

.risk-label-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--sp-1); }
.risk-label { font-size: var(--fs-sm); color: var(--color-text-primary); font-weight: var(--fw-medium); }
.risk-desc { font-size: var(--fs-xs); color: var(--color-text-tertiary); }

.risk-value-row {
  overflow: hidden;
}
.risk-value { font-size: var(--fs-xl); font-weight: var(--fw-bold); color: var(--color-text-primary); font-family: var(--font-mono); }

@media (max-width: 768px) {
  .risk-grid { grid-template-columns: 1fr; }
}
</style>
