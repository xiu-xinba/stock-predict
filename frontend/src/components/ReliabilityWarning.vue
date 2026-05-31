<template>
  <div v-if="showReliability" class="reliability-warning">
    <svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2 1 21h22L12 2zm1 15h-2v-2h2v2zm0-4h-2V8h2v5z"/></svg>
    <span>{{ reliabilityText }}</span>
  </div>

  <div v-if="quality" class="reliability-warning quality">
    <svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 4h18v2H3V4zm0 7h18v2H3v-2zm0 7h18v2H3v-2z"/></svg>
    <span>{{ quality.note }}</span>
    <span v-if="dataQualityDescription" class="quality-desc">{{ dataQualityDescription }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { PredictionResult, PredictionDataQuality, PredictionModelCoverageStatus } from '@/types/predict'

defineOptions({ name: 'ReliabilityWarning' })

const props = defineProps<{
  result: PredictionResult | null
  quality: PredictionDataQuality | null
  fundType: string | undefined
}>()

const showReliability = computed(() => props.result && props.result.reliability !== 'model')

const reliabilityText = computed(() => {
  if (!props.result) return ''
  const base = props.result.reliability_note || '预测结果可靠性较低，仅供参考'
  const gateText = actionabilityGateLabel(props.result)
  return gateText ? `${base} ${gateText}` : base
})

const dataQualityDescription = computed(() =>
  buildDataQualityDescription(props.result?.model_coverage_status, props.fundType)
)

function actionabilityGateLabel(prediction: PredictionResult): string {
  const gate = prediction.actionability_gate
  if (!gate || gate.actionable) return ''
  if (gate.reason === 'high_confidence_coverage_below_threshold') return '高置信覆盖率不足，暂不作为行动信号。'
  if (gate.reason === 'high_confidence_accuracy_below_threshold') return '高置信准确率不足，暂不作为行动信号。'
  if (gate.reason === 'calibration_ece_above_threshold') return '概率校准误差偏高，暂不作为行动信号。'
  return '模型质量闸门未通过，暂不作为行动信号。'
}

function isMoneyFund(fundType: string | undefined): boolean {
  if (!fundType) return false
  const t = fundType.toLowerCase()
  return t.includes('货币') || t.includes('money')
}

function buildDataQualityDescription(
  status: PredictionModelCoverageStatus | undefined,
  fundType: string | undefined
): string {
  if (isMoneyFund(fundType)) {
    return '货币基金使用收益口径，不适用涨跌方向模型'
  }
  switch (status) {
    case 'model_supported': return '训练模型预测，数据质量较高'
    case 'baseline_only': return '基线估计，非训练模型预测，仅供参考'
    case 'unsupported_fund': return '该基金暂不适合现有模型预测'
    case 'model_unavailable': return '模型服务暂时不可用'
    default: return ''
  }
}
</script>

<style scoped>
.reliability-warning {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-3) var(--sp-4);
  border: 1px solid var(--color-warning-border);
  border-radius: var(--radius-md);
  background: var(--color-warning-bg);
  color: var(--color-warning);
  font-size: var(--fs-sm);
}

.reliability-warning svg {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.reliability-warning.quality {
  border-color: var(--color-border);
  background: var(--color-bg-card);
  color: var(--color-text-secondary);
}

.quality-desc {
  margin-left: var(--sp-2);
  padding-left: var(--sp-2);
  border-left: 1px solid var(--color-border);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
}
</style>
