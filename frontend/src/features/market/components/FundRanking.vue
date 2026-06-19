<template>
  <RankingList
    :items="items"
    :type="type"
    route-prefix="/fund"
    code-field="fund_code"
    name-field="fund_name"
    sub-field="fund_type"
    :loading="loading"
    :error="error"
  />
</template>

<script setup lang="ts">
/** 基金排行卡片 */
import RankingList from './RankingList.vue'
import type { FundRankingItem } from '@/features/market/types'

defineOptions({ name: 'FundRanking' })

defineProps<{
  items: FundRankingItem[]
  type: 'up' | 'down'
  loading?: boolean
  error?: string | null
}>()
</script>
