<script setup lang="ts">
import { ref } from 'vue'
import { useECharts } from '@/composables/useECharts'
import { useTheme } from '@/composables/useTheme'
import { cssVar } from '@/utils/format'
import CollapsibleCard from '@/components/CollapsibleCard.vue'
import type { FundPortfolioData } from '@/types'

defineOptions({ name: 'FundPortfolio' })

const props = defineProps<{ portfolio: FundPortfolioData }>()
const { isDark } = useTheme()
const pieRef = ref<HTMLElement>()

function getPalette() {
  return Array.from({ length: 10 }, (_, i) => cssVar(`--color-chart-p${i + 1}`))
}

function getPieOption() {
  const sectors = props.portfolio.sector_allocation
  if (!sectors.length) return {}

  return {
    tooltip: {
      trigger: 'item',
      backgroundColor: cssVar('--color-bg-card'),
      borderColor: cssVar('--color-border'),
      textStyle: { color: cssVar('--color-text-primary'), fontSize: Number(cssVar('--fs-sm').replace('px','')) },
      formatter: '{b}: {d}%',
    },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      center: ['50%', '50%'],
      avoidLabelOverlap: true,
      itemStyle: { borderRadius: Number(cssVar('--radius-sm').replace('px','')), borderColor: cssVar('--color-bg-card'), borderWidth: 2 },
      label: { show: true, fontSize: Number(cssVar('--fs-xs').replace('px','')), color: cssVar('--color-text-secondary') },
      data: sectors.map((s, i) => ({ value: s.ratio, name: s.name, itemStyle: { color: getPalette()[i % 10] } })),
    }],
    animation: true,
    animationDuration: 600,
  }
}

useECharts(pieRef, getPieOption, () => [props.portfolio.sector_allocation, isDark.value])
</script>

<template>
  <CollapsibleCard title="投资组合" :default-collapsed="false" class="card-container" body-max-height="500px">
    <div class="portfolio-grid">
      <div class="holdings-section">
        <h3 class="sub-title">前十大持仓</h3>
        <div class="holdings-list">
          <div class="holding-row" v-for="(item, i) in portfolio.top_holdings" :key="item.code">
            <span class="holding-rank">{{ i + 1 }}</span>
            <span class="holding-name">{{ item.name }}</span>
            <span class="holding-ratio">{{ item.ratio.toFixed(1) }}%</span>
            <div class="holding-bar-wrap">
              <div class="holding-bar" :style="{ width: (item.ratio / 10 * 100) + '%' }" />
            </div>
          </div>
        </div>
      </div>

      <div class="sector-section">
        <h3 class="sub-title">行业分布</h3>
        <div class="pie-wrap" ref="pieRef" />
      </div>
    </div>
  </CollapsibleCard>
</template>

<style scoped>
.portfolio-grid {
  display: grid; grid-template-columns: 1fr 1fr; gap: var(--sp-5);
}

.sub-title {
  font-size: var(--fs-sm); font-weight: var(--fw-medium); color: var(--color-text-secondary);
  margin: 0 0 var(--sp-2) 0;
}

.holding-row {
  display: grid; grid-template-columns: 20px 1fr 48px 60px; align-items: center; gap: var(--sp-2);
  padding: var(--sp-1) 0; font-size: var(--fs-xs);
}

.holding-rank { color: var(--color-text-disabled); text-align: center; }
.holding-name { color: var(--color-text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.holding-ratio { color: var(--color-text-secondary); text-align: right; font-family: var(--font-mono, monospace); }

.holding-bar-wrap {
  height: 4px; background: var(--color-bg-elevated); border-radius: var(--radius-sm); overflow: hidden;
}

.holding-bar {
  height: 100%; background: var(--color-brand); border-radius: var(--radius-sm);
  transition: width var(--transition-normal);
}

.pie-wrap { width: 100%; height: 200px; }

@media (max-width: 768px) {
  .portfolio-grid { grid-template-columns: 1fr; }
  .pie-wrap { height: 180px; }
}
</style>
