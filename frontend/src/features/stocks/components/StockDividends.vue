<script setup lang="ts">
/** 分红送配组件，展示累计分红及历史分红送配记录表。
 * 使用 auxiliary tier + DIVIDENDS eyebrow。
 */
import { formatVolume } from '@/shared/utils/format'
import type { StockDividends } from '@/features/stocks/types'

defineOptions({ name: 'StockDividends' })

defineProps<{
  dividends: StockDividends
}>()
</script>

<template>
  <section class="card card-tier-auxiliary fade-slide-up" style="--delay: 4">
    <div class="card-header">
      <div class="card-title-wrap">
        <span class="card-eyebrow">DIVIDENDS</span>
        <h2 class="card-title">分红送配</h2>
      </div>
    </div>
    <div class="card-body">
      <div class="dividend-overview">
        <div class="overview-item">
          <span class="overview-label">累计分红</span>
          <span class="overview-value numeric">{{
            dividends.total_dividend != null ? formatVolume(dividends.total_dividend) + '元' : '--'
          }}</span>
        </div>
        <div class="overview-item">
          <span class="overview-label">分红次数</span>
          <span class="overview-value numeric">{{ dividends.records?.length || 0 }}</span>
        </div>
      </div>

      <table v-if="dividends.records && dividends.records.length" class="dividend-table">
        <thead>
          <tr>
            <th class="col-date">除权除息日</th>
            <th class="col-bonus">送股(每股)</th>
            <th class="col-transfer">转增(每股)</th>
            <th class="col-dividend">分红(每股)</th>
            <th class="col-progress">进度</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(record, idx) in dividends.records"
            :key="idx"
            :class="{ 'row-alt': idx % 2 === 1 }"
          >
            <td class="col-date">{{ record.date }}</td>
            <td class="col-bonus">{{ record.bonus != null ? record.bonus.toFixed(2) : '--' }}</td>
            <td class="col-transfer">
              {{ record.transfer != null ? record.transfer.toFixed(2) : '--' }}
            </td>
            <td class="col-dividend">
              {{ record.dividend != null ? record.dividend.toFixed(4) : '--' }}
            </td>
            <td class="col-progress">
              <span class="progress-tag" :class="{ 'progress-done': record.progress === '实施' }">{{
                record.progress || '--'
              }}</span>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty-hint">暂无分红数据</div>
    </div>
  </section>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.dividend-overview {
  display: flex;
  gap: var(--sp-6);
  margin-bottom: var(--sp-3);
  padding: var(--sp-3);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
}

.overview-item {
  display: flex;
  flex-direction: column;
  gap: var(--sp-0_5);
}

.overview-label {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
  letter-spacing: var(--ls-wide);
}

.overview-value {
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--color-text-primary);
}

.overview-value.numeric {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.dividend-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--fs-sm);
}

.dividend-table th {
  text-align: left;
  padding: var(--sp-1_5) var(--sp-2);
  color: var(--color-text-tertiary);
  font-weight: var(--fw-medium);
  font-size: var(--fs-xs);
  letter-spacing: var(--ls-wide);
  border-bottom: 1px solid var(--color-border);
}

.dividend-table td {
  padding: var(--sp-1_5) var(--sp-2);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text-primary);
  transition: background var(--transition-fast);
}

.dividend-table tbody tr {
  transition: background var(--transition-fast);
}

.dividend-table tbody tr:hover {
  background: var(--color-bg-elevated);
}

.row-alt {
  background: var(--color-bg-elevated);
}

.col-date,
.col-bonus,
.col-transfer,
.col-dividend {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.col-bonus,
.col-transfer,
.col-dividend {
  text-align: right;
}

.progress-tag {
  display: inline-block;
  padding: 1px var(--sp-1_5);
  border-radius: var(--radius-sm);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  color: var(--color-text-secondary);
  background: var(--color-border-light);
}

.progress-tag.progress-done {
  color: var(--color-up);
  background: var(--color-up-bg);
}

@media (max-width: 768px) {
  .col-transfer {
    display: none;
  }
}
</style>
