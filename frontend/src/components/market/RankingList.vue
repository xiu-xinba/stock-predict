<template>
  <div :class="['rank-panel', 'card', type]">
    <div class="rank-head">
      <div class="rank-head-left">
        <span :class="['rank-indicator', type]"></span>
        <span :class="['rank-title', type]">{{ title }}</span>
      </div>
      <span class="rank-sub">{{ type === 'gainers' ? '最新涨幅' : '最新跌幅' }}</span>
    </div>
    <div class="rank-body">
      <template v-if="items.length > 0">
        <div
          v-for="item in items"
          :key="item[codeField]"
          class="rank-row"
          role="button"
          tabindex="0"
          @click="goTo(item[codeField])"
          @keydown.enter="goTo(item[codeField])"
        >
          <span :class="['rank-num', { top: item.rank <= 3 }]">{{ item.rank }}</span>
          <div class="rank-info">
            <span class="rank-name">{{ item[nameField] }}</span>
            <span class="rank-type">{{ item[subField] }}</span>
          </div>
          <span
            :class="['rank-pct', item.change_pct > 0 ? 'up' : item.change_pct < 0 ? 'down' : 'flat']"
            :title="item.quote_date || undefined"
          >
            {{ item.change_pct > 0 ? '+' : '' }}{{ (item.change_pct ?? 0).toFixed(2) }}%
          </span>
        </div>
      </template>
      <template v-else>
        <div v-for="i in 5" :key="'sk-' + i" class="rank-row">
          <div class="sk-rank skeleton-pulse"></div>
          <div class="rank-info">
            <div class="sk-name skeleton-pulse"></div>
            <div class="sk-type skeleton-pulse"></div>
          </div>
          <div class="sk-pct skeleton-pulse"></div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

defineOptions({ name: 'RankingList' })

const props = withDefaults(defineProps<{
  title: string
  items: any[]
  type: 'gainers' | 'losers'
  routePrefix: string
  codeField: string
  nameField: string
  subField?: string
}>(), {
  subField: '',
})

const router = useRouter()
function goTo(code: string) {
  router.push(`${props.routePrefix}/${code}`)
}
</script>

<style scoped>
.rank-panel {
  position: relative;
  overflow: hidden;
  transition:
    border-color var(--transition-spring),
    box-shadow var(--transition-spring);
}

.rank-panel:hover {
  border-color: var(--color-brand-muted);
}

.rank-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sp-3) var(--sp-4);
  border-bottom: 1px solid var(--color-border-light);
  position: relative;
}

.rank-head::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--color-brand);
}
.rank-head-left {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}

.rank-indicator {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}
.rank-indicator.gainers { background: var(--color-up); }
.rank-indicator.losers { background: var(--color-down); }

.rank-title {
  font-size: var(--fs-base);
  font-weight: var(--fw-bold);
  line-height: var(--lh-snug);
}
.rank-title.gainers { color: var(--color-up); }
.rank-title.losers { color: var(--color-down); }
.rank-sub {
  font-size: var(--fs-xs);
  color: var(--color-text-disabled);
  line-height: var(--lh-normal);
}

.rank-body { padding: 0; }
.rank-row {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr) 84px;
  align-items: center;
  gap: var(--sp-2);
  min-height: 54px;
  padding: 0 var(--sp-4);
  border-bottom: 1px solid var(--color-border-light);
  cursor: pointer;
  transition: background-color var(--transition-fast), transform var(--transition-fast);
}
.rank-row:hover { background: var(--color-bg-hover); transform: translateX(2px); }
.rank-row:focus-visible { outline: 2px solid var(--color-brand); outline-offset: -2px; }
.rank-row:last-child { border-bottom: none; }

.rank-num {
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
  flex-shrink: 0;
}

.rank-num.top {
  background: var(--color-brand-soft);
  color: var(--color-brand);
}

.rank-info { flex: 1; min-width: 0; }
.rank-name {
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  color: var(--color-text-primary);
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: var(--lh-snug);
}
.rank-type { font-size: var(--fs-xs); color: var(--color-text-secondary); line-height: var(--lh-normal); }

.rank-pct {
  justify-self: end;
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
  font-family: var(--font-mono);
  flex-shrink: 0;
}

.rank-pct.up { color: var(--color-up); }
.rank-pct.down { color: var(--color-down); }
.rank-pct.flat { color: var(--color-text-regular); }

.sk-rank {
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
}

.sk-name {
  width: 60%;
  height: 14px;
  border-radius: var(--radius-sm);
}

.sk-type {
  width: 40%;
  height: 10px;
  margin-top: var(--sp-1);
  border-radius: var(--radius-sm);
}

.sk-pct {
  width: 52px;
  height: 14px;
  border-radius: var(--radius-sm);
  justify-self: end;
}

@media (prefers-reduced-motion: reduce) {
  .rank-panel,
  .rank-panel:hover,
  .rank-row,
  .rank-row:hover {
    transition-duration: 0.01ms !important;
  }
}
</style>
