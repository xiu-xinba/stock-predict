<template>
  <RankingList
    :items="items"
    :type="type"
    route-prefix="/stock"
    code-field="stock_code"
    name-field="stock_name"
    :loading="loading"
    :error="error"
  />
</template>

<script setup lang="ts">
/** 股票排行卡片 */
import RankingList from './RankingList.vue'
import type { StockRankingItem } from '../types'

defineOptions({ name: 'StockRanking' })

defineProps<{
  items: StockRankingItem[]
  type: 'up' | 'down'
  loading?: boolean
  error?: string | null
}>()
</script>
