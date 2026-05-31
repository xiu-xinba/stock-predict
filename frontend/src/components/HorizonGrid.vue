<template>
  <section v-if="items.length" class="horizon-grid">
    <div v-for="item in items" :key="item.key" class="horizon-card">
      <div class="horizon-head">
        <span class="horizon-label">{{ item.label }}</span>
        <span :class="['source-pill', item.result.model_source]">
          {{ modelSourceLabel(item.result.model_source) }}
        </span>
        <span :class="['target-pill', item.result.meets_accuracy_target ? 'ok' : 'warn']">
          {{ item.result.meets_accuracy_target ? '已达98%' : '未达98%' }}
        </span>
        <span :class="['signal-pill', item.result.signal_status]">
          {{ signalStatusLabel(item.result.signal_status) }}
        </span>
      </div>
      <div class="horizon-main">
        <strong :class="item.result.direction">
          {{ formatSignedPct(item.result.predicted_change_pct) }}
        </strong>
        <span :class="['horizon-dir', item.result.direction]">
          {{ item.result.direction === 'up' ? '上涨' : item.result.direction === 'down' ? '下跌' : '平盘' }}
        </span>
      </div>
      <div class="horizon-meta">
        <span>置信 {{ Math.round(item.result.direction_confidence * 100) }}%</span>
        <span>{{ item.result.change_range.low }}% ~ {{ item.result.change_range.high }}%</span>
      </div>
      <div v-if="modelMetaText(item.result)" class="horizon-model">
        {{ modelMetaText(item.result) }}
      </div>
      <div v-if="item.result.model_asof_time" class="horizon-model muted">
        样本 {{ formatModelTime(item.result.model_asof_time) }}
      </div>
      <div
        v-if="item.result.model_coverage_note"
        :class="['horizon-coverage', item.result.model_coverage_status]"
      >
        {{ item.result.model_coverage_note }}
      </div>
      <div v-if="horizonGateLabel(item.result)" class="horizon-gate">
        {{ horizonGateLabel(item.result) }}
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { formatSignedPct } from '@/utils/format'
import type { PredictionResult, PredictionModelSource, PredictionSignalStatus } from '@/types/predict'

export interface HorizonItem {
  key: string
  label: string
  result: PredictionResult
}

defineOptions({ name: 'HorizonGrid' })

defineProps<{
  items: HorizonItem[]
}>()

function signalStatusLabel(status: PredictionSignalStatus): string {
  if (status === 'actionable') return '可行动'
  if (status === 'no_signal') return '无信号'
  return '低置信'
}

function modelSourceLabel(source: PredictionModelSource): string {
  if (source === 'python_model_service') return 'Python模型'
  return 'Go基线'
}

function modelMetaText(prediction: PredictionResult): string {
  const parts = [prediction.model_candidate, prediction.feature_set].filter(Boolean)
  return parts.join(' · ')
}

function formatModelTime(value: string): string {
  const normalized = value.replace(' ', 'T')
  const parsed = new Date(normalized)
  if (Number.isNaN(parsed.getTime())) return value
  return parsed.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

function horizonGateLabel(prediction: PredictionResult): string {
  const gate = prediction.actionability_gate
  if (!gate || gate.actionable) return ''
  if (gate.reason === 'high_confidence_coverage_below_threshold') return '高置信覆盖不足'
  if (gate.reason === 'high_confidence_accuracy_below_threshold') return '高置信准确率不足'
  if (gate.reason === 'calibration_ece_above_threshold') return '校准误差偏高'
  return '质量闸门未通过'
}
</script>

<style scoped>
.horizon-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--sp-4);
}

.horizon-card {
  min-width: 0;
  padding: var(--sp-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
}

.horizon-head,
.horizon-main,
.horizon-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-3);
}

.horizon-head {
  flex-wrap: wrap;
}

.horizon-label {
  color: var(--color-text-primary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}

.target-pill {
  flex-shrink: 0;
  padding: 2px var(--sp-2);
  border-radius: var(--radius-sm);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
}

.target-pill.ok {
  color: var(--color-success);
  background: var(--color-down-bg);
}

.target-pill.warn {
  color: var(--color-warning);
  background: var(--color-warning-bg);
}

.source-pill {
  flex-shrink: 0;
  padding: var(--sp-1) var(--sp-2);
  margin-top: var(--sp-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  line-height: var(--lh-tight);
  white-space: nowrap;
}

.source-pill.python_model_service {
  color: var(--color-brand);
  background: var(--color-brand-soft);
  border-color: var(--color-brand-soft);
}

.source-pill.go_baseline {
  color: var(--color-text-secondary);
  background: var(--color-bg-hover);
}

.signal-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  padding: 2px var(--sp-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  line-height: var(--lh-tight);
  white-space: nowrap;
}

.signal-pill.actionable {
  color: var(--color-success);
  background: var(--color-down-bg);
  border-color: var(--color-up-border);
}

.signal-pill.low_confidence {
  color: var(--color-warning);
  background: var(--color-warning-bg);
  border-color: var(--color-warning-border);
}

.signal-pill.no_signal {
  color: var(--color-flat);
  background: var(--color-bg-hover);
}

.horizon-main {
  margin-top: var(--sp-3);
}

.horizon-main strong {
  font-size: var(--fs-2xl);
  line-height: var(--lh-tight);
}

.horizon-main strong.up { color: var(--color-up); }
.horizon-main strong.down { color: var(--color-down); }
.horizon-main strong.flat { color: var(--color-flat); }

.horizon-dir {
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}

.horizon-dir.up { color: var(--color-up); }
.horizon-dir.down { color: var(--color-down); }
.horizon-dir.flat { color: var(--color-flat); }

.horizon-meta {
  margin-top: var(--sp-2);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
}

.horizon-gate {
  margin-top: var(--sp-2);
  color: var(--color-warning);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.horizon-model {
  overflow: hidden;
  margin-top: var(--sp-2);
  color: var(--color-text-primary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.horizon-model.muted {
  color: var(--color-text-secondary);
  font-weight: var(--fw-medium);
}

.horizon-coverage {
  margin-top: var(--sp-2);
  padding: var(--sp-1) var(--sp-2);
  border-radius: var(--radius-sm);
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  line-height: var(--lh-snug);
}

.horizon-coverage.model_supported {
  background: var(--color-brand-soft);
  color: var(--color-brand);
}

.horizon-coverage.unsupported_fund,
.horizon-coverage.model_unavailable {
  background: var(--color-warning-bg);
  color: var(--color-warning);
}

@media (max-width: 1024px) {
  .horizon-grid {
    grid-template-columns: 1fr;
  }
}
</style>
