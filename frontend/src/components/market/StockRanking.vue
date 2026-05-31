<template>
  <RankingList
    :title="title"
    :items="items"
    :type="type"
    route-prefix="/stock"
    code-field="stock_code"
    name-field="stock_name"
    sub-field="stock_code"
  />
</template>

<script setup lang="ts">
import RankingList from './RankingList.vue'
import type { StockRankingItem } from '@/types/stock'

defineOptions({ name: 'StockRanking' })

defineProps<{
  title: string
  items: StockRankingItem[]
  type: 'gainers' | 'losers'
}>()
</script>
