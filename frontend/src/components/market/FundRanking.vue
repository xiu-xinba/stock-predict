<template>
  <div :class="['rank-panel', type]">
    <div class="rank-head">
      <div class="rank-head-left">
        <span :class="['rank-indicator', type]"></span>
        <span :class="['rank-title', type]">{{ title }}</span>
      </div>
      <span class="rank-sub">{{ type === 'gainers' ? '今日领涨' : '今日领跌' }}</span>
    </div>
    <div class="rank-body">
      <div
        v-for="item in items"
        :key="item.fund_code"
        class="rank-row"
        role="button"
        tabindex="0"
        @click="goToPredict(item.fund_code)"
        @keydown.enter="goToPredict(item.fund_code)"
      >
        <span :class="['rank-num', { top: item.rank <= 3 }]">{{ item.rank }}</span>
        <div class="rank-info">
          <span class="rank-name">{{ item.fund_name }}</span>
          <span class="rank-type">{{ item.fund_type }}</span>
        </div>
        <span :class="['rank-pct', item.change_pct > 0 ? 'up' : item.change_pct < 0 ? 'down' : 'flat']">
          {{ item.change_pct > 0 ? '+' : '' }}{{ item.change_pct.toFixed(2) }}%
        </span>
      </div>
      <div v-if="items.length === 0" class="rank-empty">
        <svg viewBox="0 0 24 24" width="24" height="24" class="empty-icon"><path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z" opacity="0.3"/></svg>
        <span>暂无数据</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import type { FundRankingItem } from '@/types'

defineProps<{
  title: string
  items: FundRankingItem[]
  type: 'gainers' | 'losers'
}>()

const router = useRouter()
function goToPredict(fundCode: string) {
  router.push(`/predict/${fundCode}`)
}
</script>

<style scoped>
.rank-panel {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  overflow: hidden;
  border: 1px solid var(--color-border-light);
  box-shadow: var(--shadow-sm);
  transition: box-shadow 300ms cubic-bezier(0.4, 0, 0.2, 1),
              transform 300ms cubic-bezier(0.4, 0, 0.2, 1);
}
.rank-panel:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-1px);
}

.rank-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sp-3) var(--sp-4);
  border-bottom: 1px solid var(--color-border-light);
}
.rank-head-left {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}

/* 标题前的小圆点指示器 */
.rank-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.rank-indicator.gainers { background: var(--color-up); box-shadow: 0 0 6px rgba(207, 46, 46, 0.3); }
.rank-indicator.losers { background: var(--color-down); box-shadow: 0 0 6px rgba(26, 153, 86, 0.3); }

.rank-title {
  font-size: var(--fs-md);
  font-weight: var(--fw-bold);
  letter-spacing: var(--ls-wide);
  line-height: var(--lh-snug);
}
.rank-title.gainers { color: var(--color-up); }
.rank-title.losers { color: var(--color-down); }
.rank-sub {
  font-size: var(--fs-xs);
  color: var(--color-text-disabled);
  letter-spacing: var(--ls-wide);
  line-height: var(--lh-normal);
}

.rank-body { padding: 0 var(--sp-3); }
.rank-row {
  display: flex;
  align-items: center;
  padding: var(--sp-3) var(--sp-1);
  border-bottom: 1px solid var(--color-border-light);
  cursor: pointer;
  transition: all var(--transition-fast);
  border-radius: var(--radius-sm);
}
.rank-row:hover { background: var(--color-bg-hover); transform: translateX(2px); }
.rank-row:focus-visible { outline: 2px solid var(--color-brand); outline-offset: -2px; }
.rank-row:last-child { border-bottom: none; }

.rank-num {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  flex-shrink: 0;
  margin-right: var(--sp-2);
  transition: all var(--transition-fast);
  font-variant-numeric: tabular-nums;
}
.rank-num.top {
  background: var(--color-brand-light);
  color: var(--color-brand);
}
.rank-row:hover .rank-num.top {
  transform: scale(1.1);
}

.rank-info { flex: 1; min-width: 0; }
.rank-name {
  font-size: var(--fs-base);
  font-weight: var(--fw-medium);
  color: var(--color-text-primary);
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: var(--lh-snug);
}
.rank-type { font-size: var(--fs-xs); color: var(--color-text-secondary); letter-spacing: var(--ls-wide); line-height: var(--lh-normal); }

.rank-pct {
  font-size: var(--fs-base);
  font-weight: var(--fw-bold);
  flex-shrink: 0;
  margin-left: var(--sp-2);
  font-variant-numeric: tabular-nums;
  letter-spacing: var(--ls-wide);
}
.rank-pct.up { color: var(--color-up); }
.rank-pct.down { color: var(--color-down); }
.rank-pct.flat { color: var(--color-text-regular); }

.rank-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--color-text-secondary);
  padding: var(--sp-6) 0;
  font-size: var(--fs-sm);
  line-height: var(--lh-normal);
  gap: var(--sp-2);
}
.empty-icon {
  opacity: 0.3;
}
</style>
