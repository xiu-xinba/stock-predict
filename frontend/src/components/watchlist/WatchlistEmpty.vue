<template>
  <div class="empty-state">
    <div class="empty-visual">
      <svg viewBox="0 0 48 48" class="empty-svg" aria-hidden="true">
        <rect x="6" y="8" width="36" height="32" rx="4" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.3"/>
        <line x1="14" y1="18" x2="34" y2="18" stroke="currentColor" stroke-width="1.5" opacity="0.2" stroke-linecap="round"/>
        <line x1="14" y1="24" x2="28" y2="24" stroke="currentColor" stroke-width="1.5" opacity="0.2" stroke-linecap="round"/>
        <line x1="14" y1="30" x2="22" y2="30" stroke="currentColor" stroke-width="1.5" opacity="0.2" stroke-linecap="round"/>
        <circle cx="36" cy="14" r="6" fill="currentColor" opacity="0.08"/>
        <path d="M34 14l2 2 4-4" stroke="currentColor" stroke-width="1.5" fill="none" stroke-linecap="round" stroke-linejoin="round" opacity="0.4"/>
      </svg>
    </div>
    <h3 class="empty-title">暂无自选基金</h3>
    <p class="empty-desc">点击右上角搜索图标，搜索基金后点击星标即可添加到自选。</p>
  </div>
</template>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 320px;
  padding: var(--sp-8) var(--sp-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
  text-align: center;
}

.empty-visual {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  margin-bottom: var(--sp-5);
  border-radius: var(--radius-lg);
  background: var(--color-bg-elevated);
}

.empty-svg {
  width: 36px;
  height: 36px;
  color: var(--color-text-secondary);
}

.empty-title {
  margin: 0 0 var(--sp-2);
  color: var(--color-text-primary);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  letter-spacing: var(--ls-tight);
}

.empty-desc {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  line-height: var(--lh-relaxed);
  max-width: 260px;
}
</style>
