<script setup lang="ts">
/** 股东人数变化组件，展示最新股东人数及股东人数/户均持股趋势图。
 * 人数减少（筹码集中）视为利好，人数增加（筹码分散）视为利空。
 * 使用 auxiliary tier + SHAREHOLDER TREND eyebrow。
 */
import { ref, computed } from 'vue'
import { useECharts } from '@/shared/charts/useECharts'
import { useTheme } from '@/shared/composables/useTheme'
import { cssVar, formatSignedPct } from '@/shared/utils/format'
import { getBaseChartOption } from '@/shared/charts/echarts'
import type { StockShareholderTrend } from '@/features/stocks/types'

defineOptions({ name: 'StockShareholderTrend' })

const props = defineProps<{
  shareholderTrend: StockShareholderTrend
}>()

const { isDark } = useTheme()
const chartRef = ref<HTMLElement>()

/** 最新一期变化方向（负值=人数减少=利好） */
const latestChange = computed(() => {
  const trend = props.shareholderTrend.trend || []
  if (!trend.length) return null
  return trend[trend.length - 1].change
})

const changeDir = computed(() => {
  const v = latestChange.value
  if (v == null) return 'text-flat'
  // 人数减少为利好（up色），人数增加为利空（down色）
  if (v < 0) return 'text-up'
  if (v > 0) return 'text-down'
  return 'text-flat'
})

function getChartOption() {
  const base = getBaseChartOption()
  const trend = props.shareholderTrend.trend || []
  if (!trend.length) return {}

  const dates = trend.map((t) => t.date)
  const counts = trend.map((t) => t.count)
  const avgHoldings = trend.map((t) => t.avg_holding)

  return {
    ...base,
    legend: {
      data: ['股东人数', '户均持股'],
      bottom: 0,
      textStyle: {
        color: cssVar('--color-chart-axis'),
        fontSize: Number(cssVar('--fs-xs').replace('px', '')),
      },
      itemWidth: 12,
      itemHeight: 8,
    },
    xAxis: {
      ...base.xAxis,
      type: 'category' as const,
      data: dates,
      axisLabel: {
        ...base.xAxis.axisLabel,
        formatter: (val: string) => val.slice(2, 7),
      },
    },
    yAxis: [
      {
        ...base.yAxis,
        type: 'value' as const,
        name: '人数',
        nameTextStyle: {
          color: cssVar('--color-chart-axis'),
          fontSize: Number(cssVar('--fs-xs').replace('px', '')),
        },
      },
      {
        ...base.yAxis,
        type: 'value' as const,
        name: '股',
        splitLine: { show: false },
        nameTextStyle: {
          color: cssVar('--color-chart-axis'),
          fontSize: Number(cssVar('--fs-xs').replace('px', '')),
        },
      },
    ],
    series: [
      {
        name: '股东人数',
        type: 'bar',
        data: counts,
        itemStyle: {
          color: cssVar('--color-brand'),
          opacity: 0.7,
        },
        barMaxWidth: 16,
      },
      {
        name: '户均持股',
        type: 'line',
        yAxisIndex: 1,
        data: avgHoldings,
        smooth: 0.3,
        showSymbol: false,
        lineStyle: { width: 2, color: cssVar('--color-up') },
      },
    ],
    animation: true,
    animationDuration: 600,
  }
}

useECharts(chartRef, getChartOption, () => [props.shareholderTrend.trend, isDark.value])
</script>

<template>
  <section class="card card-tier-auxiliary fade-slide-up" style="--delay: 5">
    <div class="card-header">
      <div class="card-title-wrap">
        <span class="card-eyebrow">SHAREHOLDER TREND</span>
        <h2 class="card-title">股东人数变化</h2>
      </div>
    </div>
    <div class="card-body">
      <div class="trend-overview">
        <div class="overview-item">
          <span class="overview-label">最新股东人数</span>
          <span class="overview-value numeric">{{
            shareholderTrend.latest_count != null
              ? shareholderTrend.latest_count.toLocaleString('zh-CN')
              : '--'
          }}</span>
        </div>
        <div v-if="latestChange != null" class="overview-item">
          <span class="overview-label">环比变化</span>
          <span class="overview-value numeric" :class="changeDir">
            {{ formatSignedPct(latestChange, 2) }}
          </span>
        </div>
      </div>

      <div
        v-if="shareholderTrend.trend && shareholderTrend.trend.length"
        ref="chartRef"
        class="chart-wrap"
      />
      <div v-else class="empty-hint">暂无股东人数数据</div>
    </div>
  </section>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.trend-overview {
  display: flex;
  gap: var(--sp-6);
  margin-bottom: var(--sp-3);
  padding: var(--sp-3);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
}

.overview-item {
  display: flex;
  flex-direction: column;
  gap: var(--sp-0_5);
}

.overview-label {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
  letter-spacing: var(--ls-wide);
}

.overview-value {
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--color-text-primary);
}

.overview-value.numeric {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.overview-value.text-up {
  color: var(--color-up);
}
.overview-value.text-down {
  color: var(--color-down);
}
.overview-value.text-flat {
  color: var(--color-text-secondary);
}

.chart-wrap {
  width: 100%;
  height: 240px;
}

@media (max-width: 768px) {
  .chart-wrap {
    height: 180px;
  }
}
</style>
