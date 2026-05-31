<script setup lang="ts">
import ErrorState from '@/components/ErrorState.vue'
import type { AppError } from '@/types'

defineOptions({ name: 'DetailPageLayout' })

withDefaults(defineProps<{
  loading: boolean
  error: AppError | null
  code: string
  hasContent?: boolean
  skeletonCount?: number
}>(), {
  hasContent: false,
  skeletonCount: 5,
})

const emit = defineEmits<{
  retry: []
}>()
</script>

<template>
  <div class="detail-page-layout">
    <div v-if="loading && !hasContent" class="skeleton-wrap">
      <div class="skeleton-card skeleton-pulse" v-for="i in skeletonCount" :key="i" />
    </div>

    <ErrorState v-else-if="error" :message="error.message" @retry="emit('retry')" />

    <div v-else-if="hasContent" class="detail-content">
      <slot name="header" />
      <slot />
      <slot name="footer" />
    </div>
  </div>
</template>

<style scoped>
.detail-page-layout {
  padding: var(--sp-4) var(--sp-4) calc(var(--dock-height, 80px) + var(--sp-8));
  max-width: 800px;
  margin: 0 auto;
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
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}
</style>
