<script setup lang="ts">
/** 详情页布局组件，统一处理加载骨架屏、错误状态与内容展示的切换逻辑 */
import ErrorState from '@/shared/components/ErrorState.vue'
import type { AppError } from '@/shared/types/errors'

defineOptions({ name: 'DetailPageLayout' })

withDefaults(
  defineProps<{
    loading: boolean
    error: AppError | null
    code: string
    hasContent?: boolean
    skeletonCount?: number
    wide?: boolean
  }>(),
  {
    hasContent: false,
    skeletonCount: 5,
    wide: false,
  },
)

const emit = defineEmits<{
  retry: []
}>()
</script>

<template>
  <div class="detail-page-layout" :class="{ wide }">
    <div v-if="loading && !hasContent" class="skeleton-wrap">
      <div
        v-for="i in skeletonCount"
        :key="i"
        class="skeleton-card skeleton-pulse"
        :style="{ '--skeleton-delay': i }"
      />
    </div>

    <ErrorState v-else-if="error" :message="error.message" @retry="emit('retry')" />

    <div v-else-if="hasContent" class="detail-content fade-slide-up" style="--delay: 0">
      <slot name="header" />
      <slot />
      <slot name="footer" />
    </div>
  </div>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.detail-content {
  animation: content-enter 0.4s var(--ease-out-expo) both;
}

.detail-page-layout {
  padding: var(--sp-4) var(--sp-4) calc(var(--dock-height, 80px) + var(--sp-8));
  max-width: 800px;
  margin: 0 auto;
}

.detail-page-layout.wide {
  max-width: 1280px;
  padding-left: var(--sp-5);
  padding-right: var(--sp-5);
}

.skeleton-wrap {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}

.skeleton-card {
  height: 180px;
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
  position: relative;
  overflow: hidden;
  animation: skeleton-enter 0.4s var(--ease-out-expo) both;
  animation-delay: calc(var(--skeleton-delay, 0) * 80ms);
}

.skeleton-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, var(--color-brand), transparent);
  opacity: 0.5;
}

@keyframes skeleton-enter {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes content-enter {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: var(--sp-5);
}

.detail-page-layout.wide .detail-content {
  gap: var(--sp-5);
}

.detail-page-layout.wide .detail-content > :first-child {
  margin-bottom: var(--sp-1);
}
</style>
