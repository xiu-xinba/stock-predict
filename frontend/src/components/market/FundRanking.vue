<template>
  <RankingList
    :title="title"
    :items="items"
    :type="type"
    route-prefix="/fund"
    code-field="fund_code"
    name-field="fund_name"
    sub-field="fund_type"
  />
</template>

<script setup lang="ts">
import RankingList from './RankingList.vue'
import type { FundRankingItem } from '@/types'

defineOptions({ name: 'FundRanking' })

defineProps<{
  title: string
  items: FundRankingItem[]
  type: 'gainers' | 'losers'
}>()
</script>
