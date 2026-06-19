<script setup lang="ts">
/** 研报评级组件，展示最新评级、评级数量及研报列表。
 * 评级按买入/增持/中性/减持/卖出进行颜色编码。
 * 使用 auxiliary tier + RESEARCH eyebrow。
 */
import type { StockResearch } from '@/features/stocks/types'

defineOptions({ name: 'StockResearch' })

defineProps<{
  research: StockResearch
}>()

/** 评级 → CSS 类名映射 */
function ratingClass(rating: string): string {
  if (!rating) return 'rating-flat'
  const r = rating.toLowerCase()
  if (r.includes('买入') || r.includes('buy') || r.includes('strong')) return 'rating-up'
  if (
    r.includes('增持') ||
    r.includes('overweight') ||
    r.includes('add') ||
    r.includes('outperform')
  )
    return 'rating-mild-up'
  if (r.includes('减持') || r.includes('reduce') || r.includes('underperform'))
    return 'rating-mild-down'
  if (r.includes('卖出') || r.includes('sell')) return 'rating-down'
  return 'rating-flat'
}
</script>

<template>
  <section class="card card-tier-auxiliary stock-research-card fade-slide-up" style="--delay: 4">
    <div class="card-header">
      <div class="card-title-wrap">
        <h2 class="card-title">研报评级</h2>
      </div>
    </div>
    <div class="card-body">
      <div class="research-overview">
        <div class="overview-item">
          <span class="overview-label">最新评级</span>
          <span class="overview-value" :class="ratingClass(research.latest_rating || '')">{{
            research.latest_rating || '--'
          }}</span>
        </div>
        <div class="overview-item">
          <span class="overview-label">评级数量</span>
          <span class="overview-value numeric">{{ research.rating_count || 0 }}</span>
        </div>
      </div>

      <table v-if="research.reports && research.reports.length" class="research-table">
        <thead>
          <tr>
            <th class="col-date">日期</th>
            <th class="col-org">机构</th>
            <th class="col-rating">评级</th>
            <th class="col-target">目标价</th>
            <th class="col-researcher">研究员</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(report, idx) in research.reports"
            :key="idx"
            :class="{ 'row-alt': idx % 2 === 1 }"
          >
            <td class="col-date">{{ report.date }}</td>
            <td class="col-org" :title="report.org_name">{{ report.org_name }}</td>
            <td class="col-rating">
              <span class="rating-tag" :class="ratingClass(report.rating || '')">{{
                report.rating || '--'
              }}</span>
            </td>
            <td class="col-target">
              {{
                report.target_price != null && report.target_price > 0
                  ? report.target_price.toFixed(2)
                  : '--'
              }}
            </td>
            <td class="col-researcher" :title="report.researcher">
              {{ report.researcher || '--' }}
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty-hint">暂无研报数据</div>
    </div>
  </section>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.research-overview {
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

.overview-value.rating-up {
  color: var(--color-up);
}
.overview-value.rating-mild-up {
  color: var(--color-up);
  opacity: 0.85;
}
.overview-value.rating-mild-down {
  color: var(--color-down);
  opacity: 0.85;
}
.overview-value.rating-down {
  color: var(--color-down);
}
.overview-value.rating-flat {
  color: var(--color-text-secondary);
}

.research-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--fs-sm);
}

.research-table th {
  text-align: left;
  padding: var(--sp-1_5) var(--sp-2);
  color: var(--color-text-tertiary);
  font-weight: var(--fw-medium);
  font-size: var(--fs-xs);
  letter-spacing: var(--ls-wide);
  border-bottom: 1px solid var(--color-border);
}

.research-table td {
  padding: var(--sp-1_5) var(--sp-2);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text-primary);
  transition: background var(--transition-fast);
}

.research-table tbody tr {
  transition: background var(--transition-fast);
}

.research-table tbody tr:hover {
  background: var(--color-bg-elevated);
}

.row-alt {
  background: var(--color-bg-elevated);
}

.col-date,
.col-target {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.col-target {
  text-align: right;
}

.col-org,
.col-researcher {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rating-tag {
  display: inline-block;
  padding: 1px var(--sp-1_5);
  border-radius: var(--radius-sm);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  white-space: nowrap;
}

.rating-tag.rating-up {
  color: var(--color-up);
  background: var(--color-up-bg);
}
.rating-tag.rating-mild-up {
  color: var(--color-up);
  background: color-mix(in srgb, var(--color-up) 12%, transparent);
}
.rating-tag.rating-mild-down {
  color: var(--color-down);
  background: color-mix(in srgb, var(--color-down) 12%, transparent);
}
.rating-tag.rating-down {
  color: var(--color-down);
  background: var(--color-down-bg);
}
.rating-tag.rating-flat {
  color: var(--color-text-secondary);
  background: var(--color-border-light);
}

@media (max-width: 768px) {
  .col-researcher {
    display: none;
  }
  .col-org {
    max-width: 90px;
  }
}
</style>
