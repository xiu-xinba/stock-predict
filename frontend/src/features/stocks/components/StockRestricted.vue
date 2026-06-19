<script setup lang="ts">
/** 限售解禁组件，展示下次解禁信息及历史解禁记录表。
 * 使用 auxiliary tier + RESTRICTED eyebrow。
 */
import { formatVolume } from '@/shared/utils/format'
import type { StockRestricted } from '@/features/stocks/types'

defineOptions({ name: 'StockRestricted' })

defineProps<{
  restricted: StockRestricted
}>()
</script>

<template>
  <section class="card card-tier-auxiliary fade-slide-up" style="--delay: 5">
    <div class="card-header">
      <div class="card-title-wrap">
        <span class="card-eyebrow">RESTRICTED</span>
        <h2 class="card-title">限售解禁</h2>
      </div>
    </div>
    <div class="card-body">
      <div v-if="restricted.next_release" class="next-release">
        <div class="next-release-header">
          <span class="overview-label">下次解禁</span>
          <span class="next-release-date">{{ restricted.next_release.date }}</span>
        </div>
        <div class="next-release-body">
          <div class="overview-item">
            <span class="overview-label">解禁数量</span>
            <span class="overview-value numeric">{{
              formatVolume(restricted.next_release.volume) + '股'
            }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">占总股本</span>
            <span class="overview-value numeric">{{
              restricted.next_release.ratio != null
                ? (restricted.next_release.ratio * 100).toFixed(2) + '%'
                : '--'
            }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">解禁类型</span>
            <span class="overview-value">{{ restricted.next_release.type || '--' }}</span>
          </div>
        </div>
      </div>

      <table v-if="restricted.history && restricted.history.length" class="restricted-table">
        <thead>
          <tr>
            <th class="col-date">解禁日期</th>
            <th class="col-volume">解禁数量</th>
            <th class="col-ratio">占总股本</th>
            <th class="col-type">类型</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(record, idx) in restricted.history"
            :key="idx"
            :class="{ 'row-alt': idx % 2 === 1 }"
          >
            <td class="col-date">{{ record.date }}</td>
            <td class="col-volume">{{ formatVolume(record.volume) }}</td>
            <td class="col-ratio">
              {{ record.ratio != null ? (record.ratio * 100).toFixed(2) + '%' : '--' }}
            </td>
            <td class="col-type">{{ record.type || '--' }}</td>
          </tr>
        </tbody>
      </table>
      <div v-else-if="!restricted.next_release" class="empty-hint">暂无解禁数据</div>
    </div>
  </section>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.next-release {
  margin-bottom: var(--sp-3);
  padding: var(--sp-3);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
  border-left: 3px solid var(--color-brand);
}

.next-release-header {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  margin-bottom: var(--sp-2);
}

.next-release-date {
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  color: var(--color-brand);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.next-release-body {
  display: flex;
  gap: var(--sp-6);
  flex-wrap: wrap;
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
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--color-text-primary);
}

.overview-value.numeric {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.restricted-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--fs-sm);
}

.restricted-table th {
  text-align: left;
  padding: var(--sp-1_5) var(--sp-2);
  color: var(--color-text-tertiary);
  font-weight: var(--fw-medium);
  font-size: var(--fs-xs);
  letter-spacing: var(--ls-wide);
  border-bottom: 1px solid var(--color-border);
}

.restricted-table td {
  padding: var(--sp-1_5) var(--sp-2);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text-primary);
  transition: background var(--transition-fast);
}

.restricted-table tbody tr {
  transition: background var(--transition-fast);
}

.restricted-table tbody tr:hover {
  background: var(--color-bg-elevated);
}

.row-alt {
  background: var(--color-bg-elevated);
}

.col-date,
.col-volume,
.col-ratio {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.col-volume,
.col-ratio {
  text-align: right;
}

@media (max-width: 768px) {
  .col-type {
    display: none;
  }
}
</style>
