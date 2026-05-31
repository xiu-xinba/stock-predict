<template>
  <div v-if="pred" class="prediction-result">
    <section class="result-top card card-accent-top">
      <div class="fund-info">
        <div :class="['direction-badge', direction]">
          {{ direction === 'up' ? '↑' : direction === 'down' ? '↓' : '→' }}
        </div>
        <div class="fund-meta">
          <span class="fund-code">{{ pred.fund_code }}</span>
          <h2 class="fund-name">{{ pred.fund_name }}</h2>
        </div>
      </div>
      <div v-if="result" class="prediction-core">
        <div :class="['pct-value', direction]">
          {{ formatSignedPct(result.predicted_change_pct) }}
        </div>
        <div :class="['direction-text', direction]">
          {{ direction === 'up' ? '预测上涨' : direction === 'down' ? '预测下跌' : '预测平盘' }}
        </div>
        <div :class="['signal-status', signalStatus]">
          {{ signalStatusLabel(signalStatus) }}
        </div>
      </div>
      <div v-if="isMoneyFund(pred?.fund_type)" class="money-fund-notice">
        <span class="notice-icon">ℹ</span>
        <span>货币基金展示万份收益与七日年化口径，不适用涨跌方向预测</span>
      </div>
    </section>

    <HorizonGrid :items="horizonResults" />

    <section v-if="result" class="metrics-grid card card-accent-top">
      <div class="metric">
        <span class="metric-label">置信度</span>
        <strong :class="direction">{{ confidencePct }}%</strong>
        <div class="metric-bar">
          <div :class="['metric-fill', direction]" :style="{ width: confidencePct + '%' }"></div>
        </div>
      </div>
      <div class="metric">
        <span class="metric-label">预测区间</span>
        <strong>{{ result.change_range.low }}% ~ {{ result.change_range.high }}%</strong>
        <span class="metric-sub">{{ intervalMeta }}</span>
      </div>
      <div v-if="snapshot" class="metric compact">
        <span class="metric-label">上证</span>
        <strong :class="formatChangePct(snapshot.sh_index_change_pct).cls">{{ formatChangePct(snapshot.sh_index_change_pct).text }}</strong>
      </div>
      <div v-if="snapshot" class="metric compact">
        <span class="metric-label">深证 / 创业</span>
        <strong>
          <span :class="formatChangePct(snapshot.sz_index_change_pct).cls">{{ formatChangePct(snapshot.sz_index_change_pct).text }}</span>
          <span class="slash">/</span>
          <span :class="formatChangePct(snapshot.cyb_index_change_pct).cls">{{ formatChangePct(snapshot.cyb_index_change_pct).text }}</span>
        </strong>
      </div>
    </section>

    <section v-if="result" class="content-grid">
      <FactorPanel :factors="result.top_factors" />
      <ReturnDecomposition :decomposition="returnDecomposition" />
    </section>

    <ReliabilityWarning
      :result="result"
      :quality="quality"
      :fund-type="pred.fund_type"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatSignedPct } from '@/utils/format'
import { usePredictionStore } from '@/stores/prediction'
import type { PredictionSignalStatus } from '@/types/predict'
import HorizonGrid from './HorizonGrid.vue'
import type { HorizonItem } from './HorizonGrid.vue'
import FactorPanel from './FactorPanel.vue'
import ReturnDecomposition from './ReturnDecomposition.vue'
import ReliabilityWarning from './ReliabilityWarning.vue'

defineOptions({ name: 'PredictionCard' })

const store = usePredictionStore()

const pred = computed(() => store.prediction)
const result = computed(() => pred.value?.next_day_prediction ?? pred.value?.prediction ?? null)
const weeklyResult = computed(() => pred.value?.weekly_prediction ?? null)
const intradayResult = computed(() => pred.value?.intraday_prediction ?? null)
const snapshot = computed(() => pred.value?.market_snapshot ?? null)
const quality = computed(() => pred.value?.data_quality ?? null)
const returnDecomposition = computed(() => result.value?.return_decomposition ?? null)

const horizonResults = computed<HorizonItem[]>(() => {
  const items: HorizonItem[] = []
  if (result.value) items.push({ key: 'next-day', label: '隔日', result: result.value })
  if (weeklyResult.value) items.push({ key: 'weekly', label: '未来一周', result: weeklyResult.value })
  if (intradayResult.value) items.push({ key: 'intraday', label: '盘中5分钟', result: intradayResult.value })
  return items
})

const direction = computed(() => result.value?.direction ?? 'flat')
const signalStatus = computed(() => result.value?.signal_status ?? 'low_confidence')

const confidencePct = computed(() =>
  Math.round((result.value?.direction_confidence ?? 0) * 100)
)

const spread = computed(() => {
  const range = result.value?.change_range
  if (!range) return 0
  return Math.round(Math.abs(range.high - range.low) * 100) / 100
})

const intervalMeta = computed(() => {
  const interval = result.value?.prediction_interval
  if (!interval) return `Spread ${spread.value}%`
  const parts = []
  if (interval.level != null) parts.push(`${Math.round(interval.level * 100)}%经验区间`)
  if (interval.empirical_coverage != null) parts.push(`覆盖${Math.round(interval.empirical_coverage * 100)}%`)
  return parts.length ? parts.join(' / ') : `Spread ${spread.value}%`
})

function formatChangePct(val: number | undefined): { text: string; cls: string } {
  if (val == null) return { text: '--%', cls: 'flat' }
  const sign = val >= 0 ? '+' : ''
  return { text: `${sign}${val.toFixed(2)}%`, cls: val >= 0 ? 'up' : 'down' }
}

function signalStatusLabel(status: PredictionSignalStatus): string {
  if (status === 'actionable') return '可行动'
  if (status === 'no_signal') return '无信号'
  return '低置信'
}

function isMoneyFund(fundType: string | undefined): boolean {
  if (!fundType) return false
  const t = fundType.toLowerCase()
  return t.includes('货币') || t.includes('money')
}
</script>

<style scoped>
.prediction-result {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}

.result-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-4);
  padding: var(--sp-5);
  position: relative;
  overflow: hidden;
  transition: border-color var(--transition-spring), box-shadow var(--transition-spring);
}

.result-top:hover {
  border-color: var(--color-brand-muted);
  box-shadow: var(--shadow-md);
}

.fund-info {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  min-width: 0;
}

.direction-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  font-size: var(--fs-2xl);
  font-weight: var(--fw-black);
  transition: transform var(--transition-spring), box-shadow var(--transition-spring);
}

.direction-badge:hover {
  transform: scale(1.04);
}

.direction-badge.up { color: var(--color-up); background: var(--color-up-bg); border-color: var(--color-up-border); }
.direction-badge.down { color: var(--color-down); background: var(--color-down-bg); border-color: var(--color-down-border); }
.direction-badge.flat { color: var(--color-flat); background: var(--color-bg-hover); }

.fund-meta {
  min-width: 0;
}

.fund-code {
  display: block;
  margin-bottom: var(--sp-0_5);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  letter-spacing: var(--ls-wide);
}

.fund-name {
  margin: 0;
  overflow: hidden;
  color: var(--color-text-primary);
  font-size: var(--fs-xl);
  font-weight: var(--fw-bold);
  line-height: var(--lh-snug);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.prediction-core {
  text-align: right;
}

.pct-value {
  font-size: var(--fs-5xl);
  font-weight: var(--fw-black);
  line-height: var(--lh-tight);
  letter-spacing: var(--ls-tighter);
}

.pct-value.up,
.direction-text.up,
.metric strong.up {
  color: var(--color-up);
}

.pct-value.down,
.direction-text.down,
.metric strong.down {
  color: var(--color-down);
}

.pct-value.flat,
.direction-text.flat,
.metric strong.flat {
  color: var(--color-flat);
}

.direction-text {
  margin-top: var(--sp-1);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}

.signal-status {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  line-height: var(--lh-tight);
  white-space: nowrap;
}

.signal-status.actionable {
  color: var(--color-success);
  background: var(--color-down-bg);
  border-color: var(--color-up-border);
}

.signal-status.low_confidence {
  color: var(--color-warning);
  background: var(--color-warning-bg);
  border-color: var(--color-warning-border);
}

.signal-status.no_signal {
  color: var(--color-flat);
  background: var(--color-bg-hover);
}

.metrics-grid {
  display: grid;
  grid-template-columns: 1.2fr 1.2fr 0.8fr 1fr;
  overflow: hidden;
  position: relative;
}

.metrics-grid::before {
  display: none;
}

.metric {
  display: flex;
  flex-direction: column;
  gap: var(--sp-2);
  min-height: 92px;
  padding: var(--sp-4);
  border-right: 1px solid var(--color-border-light);
}

.metric:last-child {
  border-right: 0;
}

.metric.compact {
  justify-content: center;
}

.metric-label {
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.metric strong {
  color: var(--color-text-primary);
  font-size: var(--fs-lg);
  line-height: var(--lh-tight);
}

.metric-sub,
.slash {
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
}

.metric-bar {
  width: 100%;
  height: 6px;
  overflow: hidden;
  border-radius: var(--radius-sm);
  background: var(--color-bg-hover);
}

.metric-fill {
  height: 100%;
  border-radius: var(--radius-sm);
  transition: width 0.5s var(--ease-out-quart);
}

.metric-fill.up { background: var(--color-up); }
.metric-fill.down { background: var(--color-down); }
.metric-fill.flat { background: var(--color-flat); }

.content-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: var(--sp-4);
}

.money-fund-notice {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  border-radius: var(--radius-md);
  background: var(--color-brand-soft);
  border: 1px solid var(--color-brand-muted);
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  margin-bottom: var(--sp-3);
}

.notice-icon {
  font-size: var(--fs-base);
}

@media (max-width: 1024px) {
  .metrics-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .metric:nth-child(2) {
    border-right: 0;
  }

  .metric:nth-child(-n + 2) {
    border-bottom: 1px solid var(--color-border-light);
  }

  .content-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .result-top {
    align-items: flex-start;
    flex-direction: column;
  }

  .prediction-core {
    text-align: left;
  }

  .pct-value {
    font-size: var(--fs-3xl);
  }

  .metrics-grid {
    grid-template-columns: 1fr;
  }

  .metric {
    border-right: 0;
    border-bottom: 1px solid var(--color-border-light);
  }

  .metric:last-child {
    border-bottom: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .result-top,
  .result-top:hover,
  .direction-badge,
  .direction-badge:hover,
  .metric-fill {
    transition-duration: 0.01ms !important;
  }
}
</style>
