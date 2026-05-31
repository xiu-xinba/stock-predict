<template>
  <div v-if="loading" class="empty-hint">加载预测中...</div>
  <div v-else-if="error" class="empty-hint">{{ error }}<br><span class="hint-reason">请稍后重试或检查基金代码</span></div>
  <div v-else-if="prediction" class="prediction-summary">
    <div class="pred-item" v-if="prediction.next_day_prediction">
      <span class="pred-label">隔日预测</span>
      <span class="pred-direction" :class="dirClass(prediction.next_day_prediction.direction)">
        {{ dirLabel(prediction.next_day_prediction.direction) }}
      </span>
      <span class="pred-change" :class="dirClass(prediction.next_day_prediction.direction)">
        {{ formatSignedPct(prediction.next_day_prediction.predicted_change_pct, 2) }}
      </span>
      <span class="pred-confidence">
        置信度 {{ (prediction.next_day_prediction.direction_confidence * 100).toFixed(0) }}%
      </span>
    </div>

    <div class="pred-item" v-if="prediction.weekly_prediction">
      <span class="pred-label">周频预测</span>
      <span class="pred-direction" :class="dirClass(prediction.weekly_prediction.direction)">
        {{ dirLabel(prediction.weekly_prediction.direction) }}
      </span>
      <span class="pred-change" :class="dirClass(prediction.weekly_prediction.direction)">
        {{ formatSignedPct(prediction.weekly_prediction.predicted_change_pct, 2) }}
      </span>
      <span class="pred-confidence">
        置信度 {{ (prediction.weekly_prediction.direction_confidence * 100).toFixed(0) }}%
      </span>
    </div>

    <div class="pred-item" v-if="prediction.intraday_prediction">
      <span class="pred-label">盘中5分钟</span>
      <span class="pred-direction" :class="dirClass(prediction.intraday_prediction.direction)">
        {{ dirLabel(prediction.intraday_prediction.direction) }}
      </span>
      <span class="pred-change" :class="dirClass(prediction.intraday_prediction.direction)">
        {{ formatSignedPct(prediction.intraday_prediction.predicted_change_pct, 2) }}
      </span>
      <span class="pred-confidence">
        置信度 {{ (prediction.intraday_prediction.direction_confidence * 100).toFixed(0) }}%
      </span>
    </div>

    <button class="btn-primary" @click="$emit('view-full')">
      查看完整预测分析 →
    </button>
  </div>
  <div v-else class="empty-hint">暂无预测数据<span class="hint-reason">模型可能暂未覆盖该基金</span></div>
</template>

<script setup lang="ts">
import { formatSignedPct, dirClass } from '@/utils/format'
import type { PredictionResult } from '@/types'

interface PredictionDisplayData {
  next_day_prediction?: PredictionResult
  weekly_prediction?: PredictionResult
  intraday_prediction?: PredictionResult
}

defineOptions({ name: 'PredictionDisplay' })

defineProps<{
  prediction: PredictionDisplayData | null
  loading: boolean
  error?: string
}>()

defineEmits<{
  'view-full': []
}>()

function dirLabel(dir: string) {
  if (dir === 'up') return '↑ 看涨'
  if (dir === 'down') return '↓ 看跌'
  return '→ 震荡'
}
</script>

<style scoped>
.prediction-summary {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
}

.pred-item {
  display: grid;
  grid-template-columns: 80px 1fr auto auto;
  align-items: center;
  gap: var(--sp-3);
  padding: var(--sp-2) 0;
  border-bottom: 1px solid var(--color-border);
}

.pred-item:last-of-type {
  border-bottom: none;
}

.pred-label {
  font-size: var(--fs-sm);
  color: var(--color-text-secondary);
}

.pred-direction {
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

.pred-change {
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  font-family: var(--font-mono);
}

.pred-confidence {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
}

.btn-primary {
  margin-top: var(--sp-2);
  padding: var(--sp-2) var(--sp-4);
  border-radius: var(--radius-md);
  background: transparent;
  border: 1px solid var(--color-brand);
  color: var(--color-brand);
  font-size: var(--fs-sm);
  cursor: pointer;
  transition: all var(--transition-fast);
  align-self: center;
}

.btn-primary:hover {
  background: var(--color-brand);
  color: var(--color-brand-contrast);
}

.empty-hint {
  font-size: var(--fs-sm);
  color: var(--color-text-tertiary);
  text-align: center;
  padding: var(--sp-4) 0;
}

.hint-reason {
  display: block;
  margin-top: var(--sp-1);
  color: var(--color-text-disabled);
  font-size: var(--fs-xs);
}

@media (max-width: 768px) {
  .pred-item {
    grid-template-columns: 70px 1fr auto;
  }
  .pred-confidence {
    display: none;
  }
}
</style>
