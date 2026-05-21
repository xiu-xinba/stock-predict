<template>
  <div v-if="pred" class="prediction-result">
    <!-- 顶部：基金信息 + 核心预测 -->
    <div class="result-top">
      <div class="fund-info">
        <div :class="['direction-badge', direction]">
          {{ direction === 'up' ? '↑' : direction === 'down' ? '↓' : '→' }}
        </div>
        <div class="fund-meta">
          <h2 class="fund-name">{{ pred.fund_name }}</h2>
          <span class="fund-code">{{ pred.fund_code }}</span>
        </div>
      </div>
      <div v-if="result" class="prediction-core">
        <div :class="['pct-value', direction]">
          {{ direction === 'up' ? '+' : '' }}{{ result.predicted_change_pct }}%
        </div>
        <div :class="['direction-text', direction]">
          {{ direction === 'up' ? '预测上涨' : direction === 'down' ? '预测下跌' : '预测平盘' }}
        </div>
        <div class="range-text">
          区间 {{ result.change_range.low }}% ~ {{ result.change_range.high }}%
        </div>
      </div>
    </div>

    <!-- 指标条 -->
    <div v-if="result" class="metrics-strip">
      <div class="metric">
        <span class="metric-label">置信度</span>
        <div class="metric-bar">
          <div :class="['metric-fill', direction]" :style="{ width: confidencePct + '%' }"></div>
        </div>
        <span :class="['metric-val', direction]">{{ confidencePct }}%</span>
      </div>
      <div class="metric-divider"></div>
      <div class="metric">
        <span class="metric-label">波动区间</span>
        <span class="metric-val brand">{{ spread }}%</span>
      </div>
      <div class="metric-divider"></div>
      <div v-if="snapshot" class="market-strip">
        <span class="metric-label">市场</span>
        <span :class="['market-tag', formatChangePct(snapshot.sh_index_change_pct).cls]">
          沪 {{ formatChangePct(snapshot.sh_index_change_pct).text }}
        </span>
        <span :class="['market-tag', formatChangePct(snapshot.sz_index_change_pct).cls]">
          深 {{ formatChangePct(snapshot.sz_index_change_pct).text }}
        </span>
        <span :class="['market-tag', formatChangePct(snapshot.cyb_index_change_pct).cls]">
          创 {{ formatChangePct(snapshot.cyb_index_change_pct).text }}
        </span>
      </div>
    </div>

    <!-- 双栏：因子分析 + 图表 -->
    <div v-if="result" class="content-grid">
      <div class="panel factors-panel">
        <div class="panel-header">
          <span class="panel-icon">🔑</span> 关键预测因子
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
          <span class="panel-icon">📊</span> 因子重要性分布
        </div>
        <div class="panel-body chart-body">
          <div ref="chartRef" role="img" aria-label="预测因子重要性柱状图"></div>
        </div>
      </div>
    </div>

    <!-- 可靠性警告 -->
    <div v-if="result?.reliability && result.reliability !== 'model'" class="reliability-warning">
      <span class="warning-icon">⚠️</span>
      <span>{{ result.reliability_note || '预测结果可靠性较低，仅供参考' }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import echarts from '@/utils/echarts'
import { useECharts } from '@/composables/useECharts'
import { usePredictionStore } from '@/stores/prediction'

const store = usePredictionStore()

const pred = computed(() => store.prediction)
const result = computed(() => pred.value?.prediction ?? null)
const snapshot = computed(() => pred.value?.market_snapshot ?? null)

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
    return {
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
      grid: { left: 90, right: 30, top: 10, bottom: 20 },
      xAxis: {
        type: 'value',
        max: factors.length > 0 ? Math.max(0.4, ...factors.map(f => f.importance)) * 1.1 : 0.4,
        axisLabel: { formatter: (v: number) => `${(v * 100).toFixed(0)}%`, fontSize: 11 },
      },
      yAxis: {
        type: 'category',
        data: factors.map(f => f.name).reverse(),
        axisLabel: { width: 70, overflow: 'truncate', fontSize: 11 },
      },
      series: [{
        type: 'bar',
        data: factors.map(f => f.importance).reverse(),
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
            { offset: 0, color: '#b3d8ff' },
            { offset: 1, color: '#3366ff' },
          ]),
          borderRadius: [0, 4, 4, 0],
        },
        barWidth: 14,
      }],
    }
  },
  () => result.value?.top_factors
)
</script>

<style scoped>
.prediction-result {
  animation: fadeInUp 0.35s ease;
}
@keyframes fadeInUp {
  from { opacity: 0; transform: translateY(12px); }
  to { opacity: 1; transform: translateY(0); }
}

/* === 顶部：基金信息 + 核心预测 === */
.result-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-4);
  padding: var(--sp-4) var(--sp-5);
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  margin-bottom: var(--sp-3);
  flex-wrap: wrap;
}
.fund-info {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
}
.direction-badge {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 800;
  flex-shrink: 0;
}
.direction-badge.up { background: rgba(228, 57, 60, 0.1); color: var(--color-up); }
.direction-badge.down { background: rgba(46, 139, 87, 0.1); color: var(--color-down); }
.direction-badge.flat { background: rgba(144, 147, 153, 0.1); color: var(--color-flat); }
.fund-name { font-size: var(--fs-lg); font-weight: 700; margin: 0; }
.fund-code { font-size: var(--fs-sm); color: var(--color-text-secondary); }
.prediction-core {
  text-align: right;
}
.pct-value {
  font-size: 36px;
  font-weight: 800;
  line-height: 1.1;
  letter-spacing: -1px;
}
.pct-value.up { color: var(--color-up); }
.pct-value.down { color: var(--color-down); }
.pct-value.flat { color: var(--color-flat); }
.direction-text {
  font-size: var(--fs-sm);
  font-weight: 600;
  letter-spacing: 1px;
}
.direction-text.up { color: var(--color-up); }
.direction-text.down { color: var(--color-down); }
.direction-text.flat { color: var(--color-flat); }
.range-text {
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
  margin-top: 2px;
}

/* === 指标条 === */
.metrics-strip {
  display: flex;
  align-items: center;
  gap: var(--sp-4);
  padding: var(--sp-3) var(--sp-5);
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  margin-bottom: var(--sp-3);
  flex-wrap: wrap;
}
.metric {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}
.metric-label {
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
  white-space: nowrap;
}
.metric-bar {
  width: 80px;
  height: 6px;
  background: var(--color-bg-hover);
  border-radius: 3px;
  overflow: hidden;
}
.metric-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.6s ease;
}
.metric-fill.up { background: var(--color-up); }
.metric-fill.down { background: var(--color-down); }
.metric-fill.flat { background: var(--color-flat); }
.metric-val {
  font-size: var(--fs-sm);
  font-weight: 700;
}
.metric-val.brand { color: var(--color-brand); }
.metric-val.up { color: var(--color-up); }
.metric-val.down { color: var(--color-down); }
.metric-divider {
  width: 1px;
  height: 20px;
  background: var(--color-border);
}
.market-strip {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}
.market-tag {
  font-size: var(--fs-xs);
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
}
.market-tag.up { background: rgba(228, 57, 60, 0.08); color: var(--color-up); }
.market-tag.down { background: rgba(46, 139, 87, 0.08); color: var(--color-down); }
.market-tag.flat { background: var(--color-bg-hover); color: var(--color-text-secondary); }

/* === 双栏内容 === */
.content-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--sp-3);
  margin-bottom: var(--sp-3);
}
.panel {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  overflow: hidden;
}
.panel-header {
  padding: var(--sp-3) var(--sp-4);
  border-bottom: 1px solid var(--color-border);
  font-size: var(--fs-sm);
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}
.panel-icon { font-size: 14px; }
.panel-body {
  padding: var(--sp-3) var(--sp-4);
}
.chart-body > div {
  width: 100%;
  height: 240px;
}

/* 因子行 */
.factor-row {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-2) 0;
  border-bottom: 1px solid var(--color-border-light, #f2f3f5);
}
.factor-row:last-child { border-bottom: none; }
.factor-rank {
  width: 20px;
  height: 20px;
  border-radius: 5px;
  font-size: 10px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  flex-shrink: 0;
}
.factor-rank.top { background: rgba(51, 102, 255, 0.1); color: var(--color-brand); }
.factor-info {
  flex: 1;
  min-width: 0;
}
.factor-name { font-size: var(--fs-xs); font-weight: 600; display: block; }
.factor-desc { font-size: 10px; color: var(--color-text-secondary); display: block; margin-top: 1px; }
.factor-bar-wrap {
  width: 60px;
  height: 4px;
  background: var(--color-bg-hover);
  border-radius: 2px;
  flex-shrink: 0;
  overflow: hidden;
}
.factor-bar {
  height: 100%;
  border-radius: 2px;
  background: var(--color-brand);
  transition: width 0.5s ease;
}
.factor-pct {
  font-size: 10px;
  color: var(--color-text-secondary);
  width: 30px;
  text-align: right;
  flex-shrink: 0;
}

/* 可靠性警告 */
.reliability-warning {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-3) var(--sp-4);
  background: rgba(230, 162, 60, 0.08);
  border: 1px solid rgba(230, 162, 60, 0.2);
  border-radius: var(--radius-md);
  font-size: var(--fs-sm);
  color: var(--color-warning, #e6a23c);
}
.warning-icon { flex-shrink: 0; }

/* 响应式 */
@media (max-width: 767px) {
  .result-top {
    flex-direction: column;
    align-items: flex-start;
  }
  .prediction-core {
    text-align: left;
  }
  .pct-value {
    font-size: 28px;
  }
  .content-grid {
    grid-template-columns: 1fr;
  }
  .metrics-strip {
    flex-wrap: wrap;
  }
  .metric-divider {
    display: none;
  }
}
</style>
