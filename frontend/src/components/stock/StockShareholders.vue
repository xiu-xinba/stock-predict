<script setup lang="ts">
import { getDirection, formatSignedPct } from '@/utils/format'
import CollapsibleCard from '@/components/CollapsibleCard.vue'
import type { StockShareholders } from '@/types'

defineOptions({ name: 'StockShareholders' })

defineProps<{
  shareholders: StockShareholders
}>()
</script>

<template>
  <CollapsibleCard title="股东信息" :default-collapsed="false" body-max-height="600px">
    <div class="institution-stats">
      <div class="kv-item">
        <span class="kv-label">机构数量</span>
        <span class="kv-value">{{ shareholders.institution_count ?? '--' }}</span>
      </div>
      <div class="kv-item">
        <span class="kv-label">持仓比例</span>
        <span class="kv-value">{{ shareholders.institution_ratio != null ? (shareholders.institution_ratio * 100).toFixed(2) + '%' : '--' }}</span>
      </div>
    </div>

    <table v-if="shareholders.top10 && shareholders.top10.length" class="shareholders-table">
      <thead>
        <tr>
          <th class="col-name">股东名称</th>
          <th class="col-ratio">持股比例</th>
          <th class="col-change">增减</th>
          <th class="col-type">类型</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(holder, idx) in shareholders.top10" :key="idx">
          <td class="col-name">{{ holder.name }}</td>
          <td class="col-ratio">{{ holder.ratio != null ? (holder.ratio * 100).toFixed(2) + '%' : '--' }}</td>
          <td class="col-change" :class="getDirection(holder.change)">
            {{ formatSignedPct(holder.change, 2) }}
          </td>
          <td class="col-type">{{ holder.share_type }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty-hint">暂无股东数据</div>
  </CollapsibleCard>
</template>

<style scoped>
.institution-stats {
  display: flex;
  gap: var(--sp-6);
  margin-bottom: var(--sp-4);
}

.shareholders-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--fs-sm);
}

.shareholders-table th {
  text-align: left;
  padding: var(--sp-2) var(--sp-2);
  color: var(--color-text-tertiary);
  font-weight: var(--fw-medium);
  font-size: var(--fs-xs);
  border-bottom: 1px solid var(--color-border);
}

.shareholders-table td {
  padding: var(--sp-2) var(--sp-2);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text-primary);
}

.col-name {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.col-ratio,
.col-change {
  font-family: var(--font-mono);
  text-align: right;
}

.col-type {
  color: var(--color-text-tertiary);
}

.text-up { color: var(--color-up); }
.text-down { color: var(--color-down); }
.text-flat { color: var(--color-flat); }

@media (max-width: 768px) {
  .col-type { display: none; }
  .col-name { max-width: 120px; }
}
</style>
