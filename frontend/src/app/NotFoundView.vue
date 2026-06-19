<template>
  <div class="not-found-page">
    <div class="gradient-orb" aria-hidden="true" />
    <div class="not-found-number" aria-hidden="true">404</div>
    <h1 class="not-found-title">页面未找到</h1>
    <p class="not-found-desc">
      您访问的页面不存在或已被移除，请检查地址是否正确，或返回首页继续浏览。
    </p>
    <router-link to="/watchlist" class="btn-primary not-found-btn">返回首页</router-link>
  </div>
</template>

<script setup lang="ts">
/**
 * 404 页面未找到视图
 *
 * 当用户访问不存在的路由时展示，提供返回首页的引导入口。
 */
defineOptions({ name: 'NotFoundView' })
</script>

<style scoped>
.not-found-page {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  padding: var(--sp-20) var(--sp-4);
  text-align: center;
  overflow: hidden;
  animation: fade-scale-in 0.5s var(--ease-out-expo) both;
}

.not-found-page::before,
.not-found-page::after {
  content: '';
  position: absolute;
  border-radius: 50%;
  background: var(--color-brand);
  opacity: 0.15;
  pointer-events: none;
  animation: float 8s ease-in-out infinite;
}

.not-found-page::before {
  width: 8px;
  height: 8px;
  top: 25%;
  left: 20%;
  animation-delay: 0s;
}

.not-found-page::after {
  width: 6px;
  height: 6px;
  bottom: 30%;
  right: 18%;
  animation-delay: 2s;
}

.gradient-orb {
  position: absolute;
  width: 400px;
  height: 400px;
  border-radius: 50%;
  background: radial-gradient(circle, var(--color-brand-light) 0%, transparent 70%);
  filter: blur(60px);
  opacity: 0.5;
  top: 10%;
  right: -10%;
  pointer-events: none;
}

.gradient-orb::after {
  content: '';
  position: absolute;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--color-accent);
  opacity: 0.2;
  top: 30%;
  left: 40%;
  animation: float 7s ease-in-out infinite;
  animation-delay: 1s;
}

.not-found-number {
  font-size: clamp(120px, 20vw, 200px);
  font-weight: var(--fw-black);
  background: linear-gradient(135deg, var(--color-brand), var(--color-accent));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  opacity: 1;
  line-height: 1;
  letter-spacing: var(--ls-tighter);
  margin-bottom: var(--sp-4);
  animation: float 6s ease-in-out infinite;
  user-select: none;
}

@keyframes float {
  0%,
  100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-10px);
  }
}

.not-found-title {
  margin: 0 0 var(--sp-2);
  color: var(--color-text-primary);
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
}

.not-found-desc {
  margin: 0 0 var(--sp-8);
  color: var(--color-text-secondary);
  font-size: var(--fs-base);
  line-height: var(--lh-relaxed);
  max-width: 360px;
  text-wrap: balance;
}

.not-found-btn {
  font-size: var(--fs-base);
  padding: var(--sp-3) var(--sp-8);
  transition:
    background var(--transition-fast),
    transform 0.1s ease,
    box-shadow var(--transition-fast);
  box-shadow: 0 1px 3px color-mix(in srgb, var(--color-brand) 20%, transparent);
}

.not-found-btn:hover {
  background: var(--color-brand-hover);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px color-mix(in srgb, var(--color-brand) 30%, transparent);
}

.not-found-btn:active {
  transform: translateY(0) scale(0.97);
  box-shadow: 0 1px 3px color-mix(in srgb, var(--color-brand) 20%, transparent);
}

@media (prefers-reduced-motion: reduce) {
  .not-found-page,
  .not-found-page::before,
  .not-found-page::after,
  .gradient-orb::after,
  .not-found-number {
    animation-duration: 0.01ms !important;
  }
}
</style>
