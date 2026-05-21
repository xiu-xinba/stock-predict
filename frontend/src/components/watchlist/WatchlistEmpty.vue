<template>
  <div class="empty-state">
    <div class="empty-circle">
      <svg viewBox="0 0 1024 1024" class="empty-svg"><path fill="currentColor" d="M512 64a448 448 0 1 1 0 896 448 448 0 0 1 0-896zm0 192a58.432 58.432 0 0 0-58.24 63.744l23.36 256.384a35.072 35.072 0 0 0 69.76 0l23.296-256.384A58.432 58.432 0 0 0 512 256zm0 512a51.2 51.2 0 1 0 0-102.4 51.2 51.2 0 0 0 0 102.4z"/></svg>
    </div>
    <h3 class="empty-title">暂无自选基金</h3>
    <p class="empty-desc">在上方搜索框输入基金代码或名称，即可添加到自选列表</p>
    <div class="empty-tips">
      <div class="tip-item">
        <span class="tip-num">1</span>
        <span class="tip-text">输入基金代码搜索</span>
      </div>
      <div class="tip-item">
        <span class="tip-num">2</span>
        <span class="tip-text">从下拉列表选择基金</span>
      </div>
      <div class="tip-item">
        <span class="tip-num">3</span>
        <span class="tip-text">点击即可添加到自选</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.empty-state {
  text-align: center;
  padding: 60px 0 40px;
}
.empty-circle {
  width: 88px;
  height: 88px;
  margin: 0 auto var(--sp-5);
  border-radius: 50%;
  background: linear-gradient(135deg, var(--color-brand-light), #f0f5ff);
  display: flex;
  align-items: center;
  justify-content: center;
}
html.dark .empty-circle {
  background: linear-gradient(135deg, #1a2744, #0f1a2e);
}
.empty-svg {
  width: 40px;
  height: 40px;
  color: var(--color-brand);
}
.empty-title {
  font-size: var(--fs-lg);
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 var(--sp-2);
}
.empty-desc {
  font-size: var(--fs-base);
  color: var(--color-text-secondary);
  margin: 0 0 var(--sp-8);
  line-height: 1.6;
}
.empty-tips {
  display: flex;
  justify-content: center;
  gap: var(--sp-6);
}
.tip-item {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}
.tip-num {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--color-brand);
  color: #fff;
  font-size: var(--fs-xs);
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.tip-text {
  font-size: var(--fs-sm);
  color: var(--color-text-regular);
}
@media (max-width: 767px) {
  .empty-tips {
    flex-direction: column;
    align-items: center;
    gap: var(--sp-3);
  }
}
</style>
