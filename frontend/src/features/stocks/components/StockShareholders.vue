<script setup lang="ts">
/** 股东信息组件，展示机构持仓统计和前十大股东列表。
 * 紧凑化表格行高，机构统计改为 inline chip 样式节省垂直空间。
 * 使用 auxiliary tier + SHAREHOLDERS eyebrow。
 */
import { getDirection, formatSignedPct } from '@/shared/utils/format'
import type { StockShareholders } from '@/features/stocks/types'

defineOptions({ name: 'StockShareholders' })

defineProps<{
  shareholders: StockShareholders
}>()
</script>

<template>
  <section class="card card-tier-auxiliary fade-slide-up" style="--delay: 3">
    <div class="card-header">
      <div class="card-title-wrap">
        <span class="card-eyebrow">SHAREHOLDERS</span>
        <h2 class="card-title">股东信息</h2>
      </div>
    </div>
    <div class="card-body">
      <div
        v-if="shareholders.institution_count != null || shareholders.institution_ratio != null"
        class="institution-chips"
      >
        <div v-if="shareholders.institution_count != null" class="inst-chip">
          <span class="inst-label">机构数量</span>
          <span class="inst-value">{{ shareholders.institution_count }}</span>
        </div>
        <div v-if="shareholders.institution_ratio != null" class="inst-chip">
          <span class="inst-label">持仓比例</span>
          <span class="inst-value">{{
            (shareholders.institution_ratio * 100).toFixed(2) + '%'
          }}</span>
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
          <tr
            v-for="(holder, idx) in shareholders.top10"
            :key="idx"
            :class="{ 'row-alt': idx % 2 === 1 }"
          >
            <td class="col-name">{{ holder.name }}</td>
            <td class="col-ratio">
              {{ holder.ratio != null ? (holder.ratio * 100).toFixed(2) + '%' : '--' }}
            </td>
            <td class="col-change" :class="getDirection(holder.change)">
              {{ formatSignedPct(holder.change, 2) }}
            </td>
            <td class="col-type">{{ holder.share_type }}</td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty-hint">暂无股东数据</div>
    </div>
  </section>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.institution-chips {
  display: flex;
  gap: var(--sp-2);
  margin-bottom: var(--sp-3);
  flex-wrap: wrap;
}

.inst-chip {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-1_5);
  padding: var(--sp-1_5) var(--sp-3);
  border-radius: var(--radius-full);
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-light);
}

.inst-label {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
  letter-spacing: var(--ls-wide);
}

.inst-value {
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  color: var(--color-text-primary);
}

.shareholders-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--fs-sm);
}

.shareholders-table th {
  text-align: left;
  padding: var(--sp-1_5) var(--sp-2);
  color: var(--color-text-tertiary);
  font-weight: var(--fw-medium);
  font-size: var(--fs-xs);
  letter-spacing: var(--ls-wide);
  border-bottom: 1px solid var(--color-border);
}

.shareholders-table td {
  padding: var(--sp-1_5) var(--sp-2);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text-primary);
  transition: background var(--transition-fast);
}

.shareholders-table tbody tr {
  transition: background var(--transition-fast);
}

.shareholders-table tbody tr:hover {
  background: var(--color-bg-elevated);
}

.shareholders-table tbody tr:active {
  background: color-mix(in srgb, var(--color-brand) 8%, var(--color-bg-elevated));
}

.row-alt {
  background: var(--color-bg-elevated);
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
  font-variant-numeric: tabular-nums;
  text-align: right;
}

.col-type {
  color: var(--color-text-tertiary);
}

.text-up {
  color: var(--color-up);
}
.text-down {
  color: var(--color-down);
}
.text-flat {
  color: var(--color-flat);
}

@media (max-width: 768px) {
  .col-type {
    display: none;
  }
  .col-name {
    max-width: 120px;
  }
}
</style>
