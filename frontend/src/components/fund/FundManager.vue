<script setup lang="ts">
import type { FundManagerInfo } from '@/types'
import CollapsibleCard from '@/components/CollapsibleCard.vue'

defineOptions({ name: 'FundManager' })

defineProps<{ manager: FundManagerInfo }>()
</script>

<template>
  <CollapsibleCard title="基金经理" :default-collapsed="false" body-max-height="300px">
    <div v-if="!manager.name" class="empty-hint">暂无经理信息</div>
    <template v-else>
      <div class="manager-name">{{ manager.name }}</div>
      <div class="manager-stats">
        <div v-if="manager.tenure_days > 0" class="kv-item">
          <span class="kv-label">任职天数</span>
          <span class="kv-value">{{ manager.tenure_days }} 天</span>
        </div>
        <div v-if="manager.managed_size" class="kv-item">
          <span class="kv-label">管理规模</span>
          <span class="kv-value">{{ manager.managed_size }}</span>
        </div>
        <div v-if="manager.fund_count > 0" class="kv-item">
          <span class="kv-label">在管基金</span>
          <span class="kv-value">{{ manager.fund_count }} 只</span>
        </div>
      </div>
      <p v-if="manager.bio" class="manager-bio">{{ manager.bio }}</p>
    </template>
  </CollapsibleCard>
</template>

<style scoped>
.manager-name {
  font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--color-text-primary);
  margin-bottom: var(--sp-3);
}

.manager-stats {
  display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--sp-3);
  margin-bottom: var(--sp-3);
}

.kv-item { display: flex; flex-direction: column; gap: var(--sp-0_5); }
.kv-label { font-size: var(--fs-xs); color: var(--color-text-tertiary); }
.kv-value { font-size: var(--fs-sm); color: var(--color-text-primary); font-weight: var(--fw-medium); }

.manager-bio {
  font-size: var(--fs-sm); color: var(--color-text-secondary); line-height: var(--lh-relaxed);
  margin: 0;
}

@media (max-width: 768px) {
  .manager-stats { grid-template-columns: repeat(2, 1fr); }
}
</style>
