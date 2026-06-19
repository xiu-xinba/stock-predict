<template>
  <div :class="['rank-panel', 'card', type]">
    <div class="rank-body">
      <!-- Error -->
      <template v-if="error">
        <div class="rank-state error">{{ error }}</div>
      </template>

      <!-- Loading -->
      <template v-else-if="loading">
        <div v-for="i in 5" :key="i" class="rank-row">
          <div class="sk-rank skeleton-pulse"></div>
          <div class="rank-info">
            <div class="sk-name skeleton-pulse"></div>
            <div v-if="subField" class="sk-type skeleton-pulse"></div>
          </div>
          <div class="sk-pct skeleton-pulse"></div>
        </div>
      </template>

      <!-- Data -->
      <template v-else-if="rows.length > 0">
        <div
          v-for="(item, i) in rows"
          :key="item.code"
          class="rank-row"
          :class="type"
          role="button"
          tabindex="0"
          :style="{ '--i': i }"
          @click="goTo(item.code)"
          @keydown.enter="goTo(item.code)"
        >
          <span :class="['rank-num', { top: i < 3 }]">{{ i + 1 }}</span>
          <div class="rank-info">
            <span class="rank-name">{{ item.name }}</span>
            <span v-if="item.sub" class="rank-type">{{ item.sub }}</span>
          </div>
          <span
            class="rank-pct"
            :class="pctClass(item.change_pct)"
            :title="item.quote_date || undefined"
          >
            {{ formatPct(item.change_pct) }}
          </span>
        </div>
      </template>

      <!-- Empty -->
      <template v-else>
        <div class="rank-state">暂无排行数据</div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
/** 涨跌排行组件，通过 type 区分涨幅榜/跌幅榜 */
import { computed } from 'vue'
import { useRouter } from 'vue-router'

defineOptions({ name: 'RankingList' })

interface RankingRecord {
  rank: number
  change_pct: number
  quote_date?: string
  update_time?: string
  data_source?: string
  [key: string]: unknown
}

const props = withDefaults(
  defineProps<{
    items: unknown[]
    type: 'up' | 'down'
    routePrefix: string
    codeField: string
    nameField: string
    subField?: string
    loading?: boolean
    error?: string | null
  }>(),
  {
    subField: '',
    loading: false,
    error: null,
  },
)

const router = useRouter()

/** 根据涨跌幅实际值格式化，正数加+，负数加-，零不加符号 */
function formatPct(val: number | undefined): string {
  const v = val ?? 0
  const abs = Math.abs(v).toFixed(2)
  return (v > 0 ? '+' : v < 0 ? '-' : '') + abs + '%'
}

/** 根据涨跌幅实际值返回样式类 */
function pctClass(val: number | undefined): string {
  const v = val ?? 0
  if (v > 0) return 'up'
  if (v < 0) return 'down'
  return ''
}

function toRankingRecord(item: unknown): RankingRecord {
  return (item ?? {}) as RankingRecord
}

const rows = computed(() =>
  props.items.map((item) => {
    const record = toRankingRecord(item)
    const code = record[props.codeField]
    const name = record[props.nameField]
    const sub = props.subField ? record[props.subField] : ''
    return {
      rank: Number(record.rank ?? 0),
      code: code == null ? '' : String(code),
      name: name == null ? '' : String(name),
      sub: sub == null ? '' : String(sub),
      change_pct: Number(record.change_pct ?? 0),
      quote_date: typeof record.quote_date === 'string' ? record.quote_date : undefined,
    }
  }),
)

function goTo(code: string) {
  if (!code) return
  router.push(`${props.routePrefix}/${code}`)
}
</script>

<style scoped>
/* ── Card container ── */
.rank-panel {
  position: relative;
  overflow: hidden;
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  flex: 1;
  transition:
    border-color var(--transition-spring),
    box-shadow var(--transition-spring);
}

.rank-panel::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  pointer-events: none;
  opacity: 0;
  transition: opacity var(--transition-spring);
  background: radial-gradient(
    320px circle at var(--mouse-x, 50%) var(--mouse-y, 0%),
    color-mix(in srgb, var(--color-brand) 6%, transparent),
    transparent 60%
  );
}

.rank-panel:hover {
  border-color: var(--color-brand-muted);
  box-shadow: 0 4px 16px color-mix(in srgb, var(--color-brand) 10%, transparent);
}

.rank-panel:hover::after {
  opacity: 1;
}

/* ── Body ── */
.rank-body {
  padding: var(--sp-1) 0 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: space-evenly;
}

/* ── Empty / Error states ── */
.rank-state {
  min-height: 54px;
  display: flex;
  align-items: center;
  padding: 0 var(--sp-4);
  padding-left: 44px;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  position: relative;
}

.rank-state::before {
  position: absolute;
  left: var(--sp-4);
  top: 50%;
  transform: translateY(-50%);
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--color-bg-elevated);
  content: '';
  border: 2px solid var(--color-border-light);
}

.rank-state.error {
  color: var(--color-warning);
}

.rank-state.error::before {
  border-color: var(--color-warning);
  background: color-mix(in srgb, var(--color-warning) 10%, transparent);
}

/* ── Row ── */
.rank-row {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr) 96px;
  align-items: center;
  gap: var(--sp-2);
  min-height: 48px;
  padding: 0 var(--sp-4);
  border-bottom: 1px solid var(--color-border-light);
  border-left: 3px solid transparent;
  cursor: pointer;
  transition:
    background-color var(--transition-fast),
    transform var(--transition-fast),
    border-left-color var(--transition-fast);
}

.rank-row.up:hover {
  border-left-color: var(--color-up);
  background: linear-gradient(90deg, var(--color-up-bg), transparent);
  transform: translateX(3px);
}

.rank-row.down:hover {
  border-left-color: var(--color-down);
  background: linear-gradient(90deg, var(--color-down-bg), transparent);
  transform: translateX(3px);
}

.rank-row:focus-visible {
  outline: 2px solid var(--color-brand);
  outline-offset: -2px;
}

.rank-body .rank-row:last-child {
  border-bottom: none;
}

/* ── Rank badge ── */
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
  color: var(--color-text-on-accent);
  font-weight: var(--fw-bold);
  font-size: var(--fs-sm);
  width: 30px;
  height: 30px;
  border-radius: var(--radius-md);
}

/* Up top badges */
.rank-row.up:nth-child(1) .rank-num.top {
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--color-warning) 80%, var(--color-accent)),
    color-mix(in srgb, var(--color-warning) 60%, var(--color-accent))
  );
  box-shadow: 0 2px 8px color-mix(in srgb, var(--color-warning) 35%, transparent);
}

.rank-row.up:nth-child(2) .rank-num.top {
  background: linear-gradient(135deg, var(--color-text-secondary), var(--color-text-disabled));
  box-shadow: 0 2px 6px color-mix(in srgb, var(--color-text-secondary) 30%, transparent);
}

.rank-row.up:nth-child(3) .rank-num.top {
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--color-warning) 50%, var(--color-text-secondary)),
    color-mix(in srgb, var(--color-warning) 35%, var(--color-text-disabled))
  );
  box-shadow: 0 2px 6px color-mix(in srgb, var(--color-warning) 25%, transparent);
}

/* Down top badges */
.rank-row.down:nth-child(1) .rank-num.top {
  background: linear-gradient(
    135deg,
    var(--color-down),
    color-mix(in srgb, var(--color-down) 70%, var(--color-accent))
  );
  box-shadow: 0 2px 8px color-mix(in srgb, var(--color-down) 35%, transparent);
}

.rank-row.down:nth-child(2) .rank-num.top {
  background: linear-gradient(135deg, var(--color-text-secondary), var(--color-text-disabled));
  box-shadow: 0 2px 6px color-mix(in srgb, var(--color-text-secondary) 30%, transparent);
}

.rank-row.down:nth-child(3) .rank-num.top {
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--color-down) 40%, var(--color-text-secondary)),
    color-mix(in srgb, var(--color-down) 25%, var(--color-text-disabled))
  );
  box-shadow: 0 2px 6px color-mix(in srgb, var(--color-down) 20%, transparent);
}

/* ── Info ── */
.rank-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow: hidden;
}

.rank-name {
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.3;
}

.rank-type {
  font-size: 10px;
  color: var(--color-text-disabled);
  line-height: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ── Change % badge ── */
.rank-pct {
  justify-self: end;
  font-size: var(--fs-base);
  font-weight: var(--fw-bold);
  font-family: var(--font-mono);
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  text-align: right;
}

.rank-pct.up {
  color: var(--color-up);
  background: var(--color-up-bg);
}

.rank-pct.down {
  color: var(--color-down);
  background: var(--color-down-bg);
}

/* ── Skeleton ── */
.sk-rank {
  width: 30px;
  height: 30px;
  border-radius: var(--radius-md);
}

.sk-name {
  width: 60%;
  height: 14px;
  border-radius: var(--radius-md);
}

.sk-type {
  width: 40%;
  height: 10px;
  margin-top: 2px;
  border-radius: var(--radius-md);
}

.sk-pct {
  width: 64px;
  height: 20px;
  border-radius: var(--radius-md);
  justify-self: end;
}

/* ── Reduced motion ── */
@media (prefers-reduced-motion: reduce) {
  .rank-panel,
  .rank-panel:hover,
  .rank-panel::after,
  .rank-panel:hover::after,
  .rank-row,
  .rank-row:hover {
    transition-duration: 0.01ms !important;
  }
}
</style>
