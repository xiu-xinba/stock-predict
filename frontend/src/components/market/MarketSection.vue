<template>
  <section class="mkt-section">
    <div :class="['sec-grid', market === 'cn' ? 'grid-3' : 'grid-auto']">
      <IndexCard v-for="idx in indices" :key="idx.code" :index="idx" />
    </div>
  </section>
</template>

<script setup lang="ts">
import IndexCard from './IndexCard.vue'
import type { MarketIndex } from '@/types'

const props = defineProps<{
  title: string
  market: string
  indices: MarketIndex[]
}>()
</script>

<style scoped>
.mkt-section { margin-bottom: var(--sp-4); }

.grid-3 {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--sp-3);
}
.grid-auto {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: var(--sp-3);
}

@media (max-width: 768px) {
  .grid-3 { grid-template-columns: 1fr 1fr; }
  .grid-auto { grid-template-columns: 1fr 1fr; }
}
@media (max-width: 480px) {
  .grid-3, .grid-auto { grid-template-columns: 1fr; }
}
</style>
