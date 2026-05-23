<template>
  <div class="empty-state">
    <div class="empty-plate">
      <svg viewBox="0 0 1024 1024" class="empty-svg" aria-hidden="true">
        <path fill="currentColor" d="M128 192a64 64 0 0 1 64-64h640a64 64 0 0 1 64 64v640a64 64 0 0 1-64 64H192a64 64 0 0 1-64-64V192zm96 64v512h576V256H224zm96 96h384v64H320v-64zm0 128h256v64H320v-64zm0 128h320v64H320v-64z"/>
      </svg>
    </div>
    <h3 class="empty-title">暂无自选基金</h3>
    <p class="empty-desc">上方搜索添加后，行情会出现在这里。</p>
  </div>
</template>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 280px;
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
  text-align: center;
}

.empty-plate {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  margin-bottom: var(--sp-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-hover);
}

.empty-svg {
  width: 28px;
  height: 28px;
  color: var(--color-text-secondary);
}

.empty-title {
  margin: 0 0 var(--sp-1);
  color: var(--color-text-primary);
  font-size: var(--fs-base);
  font-weight: var(--fw-bold);
}

.empty-desc {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
}
</style>
