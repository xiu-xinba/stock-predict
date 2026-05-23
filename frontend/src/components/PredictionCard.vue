<template>
  <div v-if="pred" class="prediction-result">
    <section class="result-top">
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
          {{ direction === 'up' ? '+' : '' }}{{ result.predicted_change_pct }}%
        </div>
        <div :class="['direction-text', direction]">
          {{ direction === 'up' ? '预测上涨' : direction === 'down' ? '预测下跌' : '预测平盘' }}
        </div>
      </div>
    </section>

    <section v-if="horizonResults.length" class="horizon-grid">
      <div v-for="item in horizonResults" :key="item.key" class="horizon-card">
        <div class="horizon-head">
          <span class="horizon-label">{{ item.label }}</span>
          <span :class="['target-pill', item.result.meets_accuracy_target ? 'ok' : 'warn']">
            {{ item.result.meets_accuracy_target ? '已达98%' : '未达98%' }}
          </span>
        </div>
        <div class="horizon-main">
          <strong :class="item.result.direction">
            {{ item.result.direction === 'up' ? '+' : '' }}{{ item.result.predicted_change_pct }}%
          </strong>
          <span :class="['horizon-dir', item.result.direction]">
            {{ item.result.direction === 'up' ? '上涨' : item.result.direction === 'down' ? '下跌' : '平盘' }}
          </span>
        </div>
        <div class="horizon-meta">
          <span>置信 {{ Math.round(item.result.direction_confidence * 100) }}%</span>
          <span>{{ item.result.change_range.low }}% ~ {{ item.result.change_range.high }}%</span>
        </div>
      </div>
    </section>

    <section v-if="result" class="metrics-grid">
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
        <span class="metric-sub">Spread {{ spread }}%</span>
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
      <div class="panel factors-panel">
        <div class="panel-header">
          <span class="panel-mark"></span>
          <span>关键预测因子</span>
        </div>
        <div class="panel-body">
          <div v-for="(f, i) in result.top_factors" :key="f.name" class="factor-row">
            <span :class="['factor-rank', { top: i === 0 }]">{{ i + 1 }}</span>
            <div class="factor-info">
              <span class="factor-name">{{ f.name }}</span>
              <span class="factor-desc">{{ f.description }}</span>
            </div>
            <div class="factor-bar-wrap">
              <div class="factor-bar" :style="{ width: (f.importance * 100) + '%' }"></div>
            </div>
            <span class="factor-pct">{{ (f.importance * 100).toFixed(0) }}%</span>
          </div>
        </div>
      </div>

      <div class="panel chart-panel">
        <div class="panel-header">
          <span class="panel-mark"></span>
          <span>因子重要性分布</span>
        </div>
        <div class="panel-body chart-body">
          <div ref="chartRef" role="img" aria-label="预测因子重要性柱状图"></div>
        </div>
      </div>
    </section>

    <div v-if="result && result.reliability !== 'model'" class="reliability-warning">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2 1 21h22L12 2zm1 15h-2v-2h2v2zm0-4h-2V8h2v5z"/></svg>
      <span>{{ result.reliability_note || '预测结果可靠性较低，仅供参考' }}</span>
    </div>

    <div v-if="quality" class="reliability-warning quality">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 4h18v2H3V4zm0 7h18v2H3v-2zm0 7h18v2H3v-2z"/></svg>
      <span>{{ quality.note }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useECharts } from '@/composables/useECharts'
import { usePredictionStore } from '@/stores/prediction'
import { useTheme } from '@/composables/useTheme'
import { cssVar } from '@/utils/format'
import type { PredictionResult } from '@/types'

const store = usePredictionStore()
const { isDark } = useTheme()

const pred = computed(() => store.prediction)
const result = computed(() => pred.value?.next_day_prediction ?? pred.value?.prediction ?? null)
const intradayResult = computed(() => pred.value?.intraday_prediction ?? null)
const snapshot = computed(() => pred.value?.market_snapshot ?? null)
const quality = computed(() => pred.value?.data_quality ?? null)

const horizonResults = computed(() => {
  const items: Array<{ key: string; label: string; result: PredictionResult }> = []
  if (result.value) items.push({ key: 'next-day', label: '隔日', result: result.value })
  if (intradayResult.value) items.push({ key: 'intraday', label: '盘中5分钟', result: intradayResult.value })
  return items
})

const direction = computed(() => result.value?.direction ?? 'flat')

const confidencePct = computed(() =>
  Math.round((result.value?.direction_confidence ?? 0) * 100)
)

const spread = computed(() => {
  const range = result.value?.change_range
  if (!range) return 0
  return Math.round(Math.abs(range.high - range.low) * 100) / 100
})

function formatChangePct(val: number | undefined): { text: string; cls: string } {
  if (val == null) return { text: '--%', cls: 'flat' }
  const sign = val >= 0 ? '+' : ''
  return { text: `${sign}${val.toFixed(2)}%`, cls: val >= 0 ? 'up' : 'down' }
}

const chartRef = ref<HTMLElement>()
useECharts(
  chartRef,
  () => {
    const factors = result.value?.top_factors ?? []
    const brand = cssVar('--color-brand', '#175cd3')
    const axis = cssVar('--color-chart-axis', '#98a2b3')
    const gridLine = cssVar('--color-chart-grid', '#f2f4f7')
    return {
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
      grid: { left: 90, right: 24, top: 10, bottom: 20 },
      xAxis: {
        type: 'value',
        max: factors.length > 0 ? Math.max(0.4, ...factors.map(f => f.importance)) * 1.1 : 0.4,
        axisLabel: { color: axis, formatter: (v: number) => `${(v * 100).toFixed(0)}%`, fontSize: 11 },
        splitLine: { lineStyle: { color: gridLine } },
      },
      yAxis: {
        type: 'category',
        data: factors.map(f => f.name).reverse(),
        axisLabel: { color: axis, width: 72, overflow: 'truncate', fontSize: 11 },
        axisLine: { lineStyle: { color: gridLine } },
        axisTick: { lineStyle: { color: gridLine } },
      },
      series: [{
        type: 'bar',
        data: factors.map(f => f.importance).reverse(),
        itemStyle: {
          color: brand,
          borderRadius: [0, 4, 4, 0],
        },
        barWidth: 14,
      }],
    }
  },
  () => [result.value?.top_factors, isDark.value]
)
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
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
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
  width: 44px;
  height: 44px;
  flex-shrink: 0;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: 22px;
  font-weight: var(--fw-extrabold);
}

.direction-badge.up { color: var(--color-up); background: var(--color-up-bg); border-color: var(--color-up-border); }
.direction-badge.down { color: var(--color-down); background: var(--color-down-bg); border-color: var(--color-down-border); }
.direction-badge.flat { color: var(--color-flat); background: var(--color-bg-hover); }

.fund-meta {
  min-width: 0;
}

.fund-code {
  display: block;
  margin-bottom: 2px;
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
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

.horizon-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--sp-4);
}

.horizon-card {
  min-width: 0;
  padding: var(--sp-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
}

.horizon-head,
.horizon-main,
.horizon-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-3);
}

.horizon-label {
  color: var(--color-text-primary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}

.target-pill {
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
}

.target-pill.ok {
  color: var(--color-success);
  background: var(--color-down-bg);
}

.target-pill.warn {
  color: var(--color-warning);
  background: var(--color-warning-bg);
}

.horizon-main {
  margin-top: var(--sp-3);
}

.horizon-main strong {
  font-size: var(--fs-2xl);
  line-height: var(--lh-tight);
}

.horizon-dir {
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}

.horizon-meta {
  margin-top: var(--sp-2);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
}

.pct-value {
  font-size: 34px;
  font-weight: var(--fw-extrabold);
  line-height: var(--lh-tight);
}

.pct-value.up,
.direction-text.up,
.metric strong.up,
.up {
  color: var(--color-up);
}

.pct-value.down,
.direction-text.down,
.metric strong.down,
.down {
  color: var(--color-down);
}

.pct-value.flat,
.direction-text.flat,
.metric strong.flat,
.flat {
  color: var(--color-flat);
}

.direction-text {
  margin-top: var(--sp-1);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}

.metrics-grid {
  display: grid;
  grid-template-columns: 1.2fr 1.2fr 0.8fr 1fr;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--color-bg-card);
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
  transition: width 0.45s ease;
}

.metric-fill.up { background: var(--color-up); }
.metric-fill.down { background: var(--color-down); }
.metric-fill.flat { background: var(--color-flat); }

.content-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: var(--sp-4);
}

.panel {
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
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

.chart-body > div {
  width: 100%;
  height: 260px;
}

.factor-row {
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr) 72px 36px;
  align-items: center;
  gap: var(--sp-2);
  min-height: 48px;
  border-bottom: 1px solid var(--color-border-light);
}

.factor-row:last-child {
  border-bottom: 0;
}

.factor-rank {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: var(--radius-sm);
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
}

.factor-rank.top {
  color: var(--color-brand);
  background: var(--color-brand-soft);
}

.factor-info {
  min-width: 0;
}

.factor-name {
  display: block;
  overflow: hidden;
  color: var(--color-text-primary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.factor-desc {
  display: block;
  overflow: hidden;
  margin-top: 1px;
  color: var(--color-text-secondary);
  font-size: var(--fs-2xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.factor-bar-wrap {
  width: 72px;
  height: 5px;
  overflow: hidden;
  border-radius: var(--radius-sm);
  background: var(--color-bg-hover);
}

.factor-bar {
  height: 100%;
  border-radius: var(--radius-sm);
  background: var(--color-brand);
  transition: width 0.45s ease;
}

.factor-pct {
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  text-align: right;
}

.reliability-warning {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-3) var(--sp-4);
  border: 1px solid var(--color-warning-border);
  border-radius: var(--radius-md);
  background: var(--color-warning-bg);
  color: var(--color-warning);
  font-size: var(--fs-sm);
}

.reliability-warning svg {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.reliability-warning.quality {
  border-color: var(--color-border);
  background: var(--color-bg-card);
  color: var(--color-text-secondary);
}

@media (max-width: 1040px) {
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

  .horizon-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .result-top {
    align-items: flex-start;
    flex-direction: column;
  }

  .prediction-core {
    text-align: left;
  }

  .pct-value {
    font-size: 30px;
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

  .factor-row {
    grid-template-columns: 24px minmax(0, 1fr) 36px;
  }

  .factor-bar-wrap {
    display: none;
  }
}
</style>
