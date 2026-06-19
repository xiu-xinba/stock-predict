<script setup lang="ts">
/** 错误状态组件，展示错误信息与重试按钮，支持紧凑模式 */
defineOptions({ name: 'ErrorState' })

withDefaults(
  defineProps<{
    message?: string
    retryLabel?: string
    compact?: boolean
  }>(),
  {
    message: '加载失败',
    retryLabel: '重试',
    compact: false,
  },
)

const emit = defineEmits<{
  retry: []
}>()
</script>

<template>
  <div :class="['error-state', { compact }]">
    <svg class="error-icon" viewBox="0 0 24 24" aria-hidden="true">
      <path
        fill="currentColor"
        d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
      />
    </svg>
    <p class="error-text">{{ message }}</p>
    <button class="btn-primary" type="button" @click="emit('retry')">{{ retryLabel }}</button>
  </div>
</template>

<style scoped>
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: var(--sp-16) var(--sp-4);
  animation: fade-slide-up 0.4s var(--ease-out-expo) both;
}

.error-state.compact {
  flex-direction: row;
  gap: var(--sp-3);
  padding: var(--sp-3) var(--sp-4);
  border: 1px solid var(--color-warning-border);
  border-radius: var(--radius-md);
  background: var(--color-warning-bg);
  justify-content: flex-start;
}

.error-icon {
  width: 40px;
  height: 40px;
  color: var(--color-warning);
  flex-shrink: 0;
  animation: pulse-icon 2.5s ease-in-out infinite;
}

@keyframes pulse-icon {
  0%,
  100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.7;
    transform: scale(1.05);
  }
}

.compact .error-icon {
  width: 18px;
  height: 18px;
  animation: none;
}

.error-text {
  margin: var(--sp-4) 0;
  color: var(--color-text-secondary);
  font-size: var(--fs-base);
}

.compact .error-text {
  margin: 0;
  font-size: var(--fs-sm);
  color: var(--color-warning);
  line-height: var(--lh-normal);
}

.btn-primary {
  padding: var(--sp-2) var(--sp-4);
  border-radius: var(--radius-md);
  background: var(--color-brand);
  color: var(--color-brand-contrast);
  font-size: var(--fs-sm);
  cursor: pointer;
  border: none;
  transition: background var(--transition-fast);
}

.btn-primary:hover {
  background: var(--color-brand-hover);
}

.compact .btn-primary {
  margin-left: auto;
  padding: var(--sp-1) var(--sp-3);
  min-height: 28px;
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}
</style>
