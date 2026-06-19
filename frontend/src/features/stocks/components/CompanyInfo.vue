<script setup lang="ts">
/** 公司概况组件，F10 风格展示基本信息、股本结构、上市信息。
 * 宽屏 3 列网格，窄屏 2 列，紧凑 info-row。
 * 使用 secondary tier + COMPANY PROFILE eyebrow。
 */
import { computed } from 'vue'
import { formatVolume } from '@/shared/utils/format'
import type { StockBasicInfo, StockQuote, StockFinancials } from '@/features/stocks/types'

defineOptions({ name: 'CompanyInfo' })

const props = defineProps<{
  basic: StockBasicInfo
  quote: StockQuote
  financials?: StockFinancials | null
}>()

const marketLabel = computed(() => {
  const m = props.basic.market
  if (m === 'sh') return '上海证券交易所'
  if (m === 'sz') return '深圳证券交易所'
  if (m === 'bj') return '北京证券交易所'
  return m || '--'
})

const totalMarketCap = computed(() => {
  if (!props.basic.total_shares || !props.quote.price) return '--'
  return formatVolume(props.basic.total_shares * props.quote.price)
})

const floatMarketCap = computed(() => {
  if (!props.basic.float_shares || !props.quote.price) return '--'
  return formatVolume(props.basic.float_shares * props.quote.price)
})

const floatRatio = computed(() => {
  if (!props.basic.total_shares || !props.basic.float_shares) return '--'
  return ((props.basic.float_shares / props.basic.total_shares) * 100).toFixed(2) + '%'
})

const infoRows = computed(() => {
  const rows: Array<{ label: string; value: string }> = [
    { label: '交易所', value: marketLabel.value },
    { label: '所属行业', value: props.basic.industry || '--' },
    { label: '上市日期', value: props.basic.list_date || '--' },
    {
      label: '总股本',
      value: props.basic.total_shares ? formatVolume(props.basic.total_shares) : '--',
    },
    {
      label: '流通股本',
      value: props.basic.float_shares ? formatVolume(props.basic.float_shares) : '--',
    },
    { label: '流通占比', value: floatRatio.value },
    { label: '总市值', value: totalMarketCap.value },
    { label: '流通市值', value: floatMarketCap.value },
  ]
  return rows
})

const financialRows = computed(() => {
  if (!props.financials) return []
  const f = props.financials
  const rows: Array<{ label: string; value: string }> = []
  if (f.eps != null) rows.push({ label: '每股收益', value: f.eps.toFixed(2) + ' 元' })
  if (f.roe != null) rows.push({ label: '净资产收益率', value: (f.roe * 100).toFixed(2) + '%' })
  if (f.gross_margin != null)
    rows.push({ label: '毛利率', value: (f.gross_margin * 100).toFixed(2) + '%' })
  if (f.net_margin != null)
    rows.push({ label: '净利率', value: (f.net_margin * 100).toFixed(2) + '%' })
  if (f.revenue != null) rows.push({ label: '营业收入', value: formatVolume(f.revenue) })
  if (f.net_profit != null) rows.push({ label: '净利润', value: formatVolume(f.net_profit) })
  return rows
})
</script>

<template>
  <section class="card card-tier-secondary company-info-card fade-slide-up" style="--delay: 3">
    <div class="card-header">
      <div class="card-title-wrap">
        <span class="card-eyebrow">COMPANY PROFILE</span>
        <h2 class="card-title">公司概况</h2>
      </div>
    </div>
    <div class="card-body">
      <div class="info-section">
        <div class="section-label">基本信息</div>
        <div class="info-grid">
          <div v-for="row in infoRows" :key="row.label" class="info-row">
            <span class="info-label">{{ row.label }}</span>
            <span class="info-value">{{ row.value }}</span>
          </div>
        </div>
      </div>

      <div v-if="financialRows.length" class="info-section">
        <div class="section-label">核心财务</div>
        <div class="info-grid">
          <div v-for="row in financialRows" :key="row.label" class="info-row">
            <span class="info-label">{{ row.label }}</span>
            <span class="info-value">{{ row.value }}</span>
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

.info-section + .info-section {
  margin-top: var(--sp-4);
}

.section-label {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  color: var(--color-text-tertiary);
  letter-spacing: var(--ls-wider);
  text-transform: uppercase;
  margin-bottom: var(--sp-2);
  margin-top: var(--sp-1);
  padding-left: var(--sp-2);
  border-left: 2px solid var(--color-brand);
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--sp-1);
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--sp-1) var(--sp-2);
  border-radius: var(--radius-sm);
  transition: background var(--transition-fast);
  gap: var(--sp-2);
}

.info-row:hover {
  background: var(--color-bg-hover);
}

.info-label {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
  white-space: nowrap;
  letter-spacing: var(--ls-wide);
}

.info-value {
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  color: var(--color-text-primary);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1024px) {
  .info-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .info-grid {
    grid-template-columns: 1fr;
  }
}
</style>
