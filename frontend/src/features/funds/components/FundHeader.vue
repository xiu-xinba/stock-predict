<script setup lang="ts">
/** 基金头部信息卡片，展示基金名称、净值、涨跌幅、自选操作等 */
import { computed } from 'vue'
import { ElMessage } from 'element-plus'
import { useWatchlistStore } from '@/features/watchlist'
import { formatSignedPct } from '@/shared/utils/format'
import AssetHeader from '@/shared/components/AssetHeader.vue'
import type { FundItem } from '@/features/funds/types'

defineOptions({ name: 'FundHeader' })

const props = defineProps<{
  basic: FundItem
  quote: FundItem
}>()

const watchlistStore = useWatchlistStore()

const navPriceLabel = computed(() => {
  const nav = (props.quote.estimated_nav || props.quote.latest_nav || 0).toFixed(4)
  return props.quote.quote_source === 'eastmoney_fundgz' ? `估值 ${nav}` : nav
})

const isInWatchlist = computed(() => watchlistStore.isInWatchlist(props.basic.fund_code))

const infoItems = computed(() => {
  const items: Array<{ label: string; value: string }> = []
  if (props.basic.company) items.push({ label: '基金公司', value: props.basic.company })
  if (props.basic.manager) items.push({ label: '基金经理', value: props.basic.manager })
  if (props.basic.latest_nav)
    items.push({ label: '最新净值', value: props.basic.latest_nav.toFixed(4) })
  if (props.basic.cumulative_nav)
    items.push({ label: '累计净值', value: props.basic.cumulative_nav.toFixed(4) })
  if (props.basic.inception_date)
    items.push({ label: '成立日期', value: props.basic.inception_date })
  if (props.quote.quote_date) items.push({ label: '数据时间', value: props.quote.quote_date })
  if (props.quote.quote_source)
    items.push({ label: '数据来源', value: quoteSourceLabel(props.quote.quote_source) })
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
      ElMessage.warning('该基金已在自选中')
    } else if (result === 'limit') {
      ElMessage.warning('自选列表已满（最多50只）')
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
  />
</template>
