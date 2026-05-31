<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useWatchlistStore } from '@/stores/watchlist'
import { formatSignedPct } from '@/utils/format'
import AssetHeader from '@/components/common/AssetHeader.vue'
import type { FundItem } from '@/types'

defineOptions({ name: 'FundHeader' })

const props = defineProps<{
  basic: FundItem
  quote: FundItem
}>()

const router = useRouter()
const watchlistStore = useWatchlistStore()

const navPriceLabel = computed(() => {
  const nav = (props.quote.estimated_nav || props.quote.latest_nav || 0).toFixed(4)
  return props.quote.quote_source === 'eastmoney_fundgz' ? `估值 ${nav}` : nav
})

const isInWatchlist = computed(() =>
  watchlistStore.isInWatchlist(props.basic.fund_code)
)

const infoItems = computed(() => {
  const items: Array<{ label: string; value: string }> = []
  if (props.basic.company) items.push({ label: '基金公司', value: props.basic.company })
  if (props.basic.manager) items.push({ label: '基金经理', value: props.basic.manager })
  if (props.basic.latest_nav) items.push({ label: '最新净值', value: props.basic.latest_nav.toFixed(4) })
  if (props.basic.cumulative_nav) items.push({ label: '累计净值', value: props.basic.cumulative_nav.toFixed(4) })
  if (props.basic.inception_date) items.push({ label: '成立日期', value: props.basic.inception_date })
  if (props.quote.quote_date) items.push({ label: '数据时间', value: props.quote.quote_date })
  if (props.quote.quote_source) items.push({ label: '数据来源', value: quoteSourceLabel(props.quote.quote_source) })
  return items
})

const badges = computed(() => {
  const result: Array<{ text: string; type: 'primary' | 'secondary' }> = []
  result.push({ text: props.basic.fund_type, type: 'primary' })
  if (props.basic.risk_level) result.push({ text: props.basic.risk_level, type: 'secondary' })
  return result
})

function toggleWatchlist() {
  if (isInWatchlist.value) {
    watchlistStore.removeItem(props.basic.fund_code)
  } else {
    const result = watchlistStore.addItem({
      fund_code: props.basic.fund_code,
      fund_name: props.basic.fund_name,
      fund_type: props.basic.fund_type,
    })
    if (result === 'duplicate') {
      alert('该基金已在自选中')
    } else if (result === 'limit') {
      alert('自选列表已满（最多50只）')
    }
  }
}

function quoteSourceLabel(source: string): string {
  const labels: Record<string, string> = {
    tencent_quote: '实时',
    eastmoney_fundgz: '估值',
    money_fund_yield: '收益',
  }
  return labels[source] || source
}

function goToPredict() {
  router.push(`/predict/${props.basic.fund_code}`)
}
</script>

<template>
  <AssetHeader
    :name="basic.fund_name"
    :code="basic.fund_code"
    :price="navPriceLabel"
    :change="formatSignedPct(quote.change_pct, 2)"
    :change-percent="quote.change_pct ?? 0"
    :is-up="(quote.change_pct ?? 0) > 0"
    :info-items="infoItems"
    :is-in-watchlist="isInWatchlist"
    :watchlist-loading="false"
    :grid-columns="3"
    :badges="badges"
    live-dot-title="实时估值"
    @toggle-watchlist="toggleWatchlist"
  >
    <template #actions>
      <button class="predict-btn" @click="goToPredict">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
        </svg>
        查看预测
      </button>
    </template>
  </AssetHeader>
</template>

<style scoped>
.predict-btn {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-1);
  padding: var(--sp-2) var(--sp-4);
  border-radius: var(--radius-md);
  background: var(--color-brand);
  color: var(--color-brand-contrast);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  cursor: pointer;
  border: none;
  transition: opacity var(--transition-fast);
}

.predict-btn:hover {
  background: var(--color-brand-hover);
}
</style>
