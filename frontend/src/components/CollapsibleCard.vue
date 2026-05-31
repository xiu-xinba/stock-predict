<script setup lang="ts">
import { ref, watch } from 'vue'

defineOptions({ name: 'CollapsibleCard' })

const props = withDefaults(defineProps<{
  title: string
  defaultCollapsed?: boolean
  bodyMaxHeight?: string
}>(), {
  defaultCollapsed: false,
  bodyMaxHeight: '500px',
})

const collapsed = ref(props.defaultCollapsed)

watch(() => props.defaultCollapsed, (v) => {
  collapsed.value = v
})

function toggle() {
  collapsed.value = !collapsed.value
}
</script>

<template>
  <section class="collapsible-card card">
    <div class="card-header" @click="toggle">
      <h2 class="card-title">{{ title }}</h2>
      <slot name="header-extra" />
      <span class="collapse-icon" :class="{ rotated: collapsed }">▾</span>
    </div>

    <div class="card-body" :class="{ collapsed }" :style="{ '--body-max-height': bodyMaxHeight }">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.collapsible-card {
  padding: var(--sp-4);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
  user-select: none;
  margin-bottom: var(--sp-3);
  gap: var(--sp-2);
}

.card-title {
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  color: var(--color-text-primary);
  margin: 0;
}

.collapse-icon {
  font-size: var(--fs-sm);
  color: var(--color-text-tertiary);
  transition: transform var(--transition-fast);
}

.collapse-icon.rotated {
  transform: rotate(-90deg);
}

.card-body {
  max-height: var(--body-max-height);
  opacity: 1;
  overflow: hidden;
  transition: max-height var(--transition-spring), opacity var(--transition-normal);
}

.card-body.collapsed {
  max-height: 0;
  opacity: 0;
  padding-top: 0;
  padding-bottom: 0;
  pointer-events: none;
}
</style>
