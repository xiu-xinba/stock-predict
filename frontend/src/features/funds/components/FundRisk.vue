<script setup lang="ts">
/** 基金风险指标卡片，展示波动率、最大回撤、夏普比率、Beta 系数等 */
import { computed } from 'vue'
import { colorWithAlpha } from '@/shared/utils/format'
import type { FundRiskMetrics } from '@/features/funds/types'

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
  低风险: { label: '低', color: 'var(--color-risk-low)' },
  中低风险: { label: '中低', color: 'var(--color-risk-medium-low)' },
  中风险: { label: '中', color: 'var(--color-risk-medium)' },
  中高风险: { label: '中高', color: 'var(--color-risk-medium-high)' },
  高风险: { label: '高', color: 'var(--color-risk-high)' },
}

const riskInfo = computed(
  () =>
    (props.riskLevel ? riskLevelMap[props.riskLevel] : undefined) || {
      label: '-',
      color: 'var(--color-text-tertiary)',
    },
)
</script>

<template>
  <section class="card card-tier-auxiliary card-container fade-slide-up" style="--delay: 2">
    <div class="card-header">
      <div class="card-title-wrap">
        <h2 class="card-title">风险指标</h2>
      </div>
      <div
        v-if="riskLevel"
        class="risk-level-badge"
        :style="{ background: colorWithAlpha(riskInfo.color, 0.12), color: riskInfo.color }"
      >
        {{ riskInfo.label }}
      </div>
    </div>
    <div class="card-body">
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
              {{ (item.value ?? 0) > 0 && item.unit === '%' ? '+' : ''
              }}{{ (item.value ?? 0).toFixed(2) }}{{ item.unit }}
            </span>
          </div>
          <div class="risk-bar-wrap">
            <div
              class="risk-bar"
              :class="{ 'text-down': item.value < 0 }"
              :style="{
                width:
                  Math.min((Math.abs(item.value ?? 0) / (item.unit === '%' ? 50 : 3)) * 100, 100) +
                  '%',
              }"
            />
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.risk-level-badge {
  font-size: var(--fs-xs);
  padding: 2px var(--sp-2);
  border-radius: var(--radius-full);
  font-weight: var(--fw-medium);
  margin-right: auto;
  box-shadow: 0 0 8px currentColor;
}

.risk-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--sp-4);
}

.risk-item {
  min-width: 0;
}

.risk-label-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--sp-1);
}
.risk-label {
  font-size: var(--fs-sm);
  color: var(--color-text-primary);
  font-weight: var(--fw-medium);
}
.risk-desc {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
}

.risk-value-row {
  overflow: hidden;
}
.risk-value {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  color: var(--color-text-primary);
  font-family: var(--font-mono);
}

.risk-bar-wrap {
  height: 3px;
  background: var(--color-bg-elevated);
  border-radius: var(--radius-sm);
  margin-top: var(--sp-1);
  overflow: hidden;
}

.risk-bar {
  height: 100%;
  background: var(--color-brand);
  border-radius: var(--radius-sm);
  transition: width var(--transition-normal);
}

.risk-bar.text-down {
  background: var(--color-down);
}

@media (max-width: 768px) {
  .risk-grid {
    grid-template-columns: 1fr;
  }
}
</style>
