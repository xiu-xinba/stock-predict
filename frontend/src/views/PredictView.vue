<template>
  <div class="predict-view">
    <FundSearch />

    <transition name="fade" mode="out-in">
      <div v-if="store.loading" key="loading" class="state-loading">
        <div class="skeleton-hero"></div>
        <div class="skeleton-row">
          <div class="skeleton-block"></div>
          <div class="skeleton-block"></div>
        </div>
      </div>

      <div v-else-if="store.prediction" key="result">
        <PredictionCard />
      </div>

      <div v-else-if="store.error" key="error" class="state-error">
        <div class="error-visual">⚠️</div>
        <div class="error-title">预测请求失败</div>
        <div class="error-desc">{{ store.error }}</div>
      </div>

      <div v-else key="empty" class="state-empty">
        <div class="empty-visual">
          <svg viewBox="0 0 120 120" fill="none" xmlns="http://www.w3.org/2000/svg">
            <circle cx="60" cy="60" r="56" stroke="var(--color-brand)" stroke-width="2" stroke-dasharray="8 4" opacity="0.3"/>
            <circle cx="60" cy="60" r="40" fill="var(--color-brand)" opacity="0.06"/>
            <path d="M50 52L60 42L70 52" stroke="var(--color-brand)" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M60 42V78" stroke="var(--color-brand)" stroke-width="3" stroke-linecap="round"/>
          </svg>
        </div>
        <div class="empty-title">搜索基金开始预测</div>
        <div class="empty-desc">输入基金代码或名称，获取 AI 驱动的涨跌预测</div>
        <div class="feature-grid">
          <div class="feature-item">
            <div class="feature-icon-wrap up"><span>↑</span></div>
            <div class="feature-text">
              <div class="feature-name">方向预测</div>
              <div class="feature-desc">多因子模型涨跌方向判断</div>
            </div>
          </div>
          <div class="feature-item">
            <div class="feature-icon-wrap brand"><span>◆</span></div>
            <div class="feature-text">
              <div class="feature-name">因子分析</div>
              <div class="feature-desc">关键预测因子贡献度</div>
            </div>
          </div>
          <div class="feature-item">
            <div class="feature-icon-wrap down"><span>↓</span></div>
            <div class="feature-text">
              <div class="feature-name">市场快照</div>
              <div class="feature-desc">实时指数辅助判断</div>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { usePredictionStore } from '@/stores/prediction'
import FundSearch from '@/components/FundSearch.vue'
import PredictionCard from '@/components/PredictionCard.vue'
const store = usePredictionStore()
const route = useRoute()

function loadPrediction() {
  const fundCode = route.params.fundCode as string
  if (fundCode && /^\d{6}$/.test(fundCode)) {
    store.predict(fundCode)
  }
}

onMounted(loadPrediction)

watch(() => route.params.fundCode, loadPrediction)
</script>

<style scoped>
.predict-view {
  padding: var(--sp-6) 0;
}

.state-loading {
  padding: 40px 0;
}
.skeleton-hero {
  height: 80px;
  border-radius: var(--radius-lg);
  margin-bottom: var(--sp-6);
  background: linear-gradient(90deg, var(--color-bg-hover) 25%, var(--color-bg-card) 50%, var(--color-bg-hover) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}
.skeleton-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--sp-4);
}
.skeleton-block {
  height: 180px;
  border-radius: var(--radius-md);
  background: linear-gradient(90deg, var(--color-bg-hover) 25%, var(--color-bg-card) 50%, var(--color-bg-hover) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}
@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.state-empty {
  text-align: center;
  padding: 48px 0;
}
.empty-visual {
  width: 100px;
  height: 100px;
  margin: 0 auto var(--sp-5);
}
.empty-visual svg {
  width: 100%;
  height: 100%;
}
.empty-title {
  font-size: var(--fs-lg);
  font-weight: 600;
  margin-bottom: var(--sp-2);
}
.empty-desc {
  font-size: var(--fs-base);
  color: var(--color-text-secondary);
}
.feature-grid {
  display: flex;
  justify-content: center;
  gap: var(--sp-3);
  margin-top: var(--sp-6);
}
.feature-item {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  background: var(--color-bg-card);
  border-radius: var(--radius-md);
  padding: var(--sp-3) var(--sp-4);
  border: 1px solid var(--color-border);
  transition: transform 0.2s, box-shadow 0.2s;
}
.feature-item:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}
.feature-icon-wrap {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 700;
  flex-shrink: 0;
}
.feature-icon-wrap.up { background: rgba(228, 57, 60, 0.1); color: var(--color-up); }
.feature-icon-wrap.down { background: rgba(46, 139, 87, 0.1); color: var(--color-down); }
.feature-icon-wrap.brand { background: rgba(51, 102, 255, 0.1); color: var(--color-brand); }
.feature-text { text-align: left; }
.feature-name {
  font-size: var(--fs-sm);
  font-weight: 600;
}
.feature-desc {
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
}

.state-error {
  text-align: center;
  padding: 60px 0;
}
.error-visual {
  width: 120px;
  height: 120px;
  margin: 0 auto var(--sp-6);
  background: linear-gradient(135deg, #fef0f0, #fff5f5);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 48px;
}
html.dark .error-visual {
  background: linear-gradient(135deg, #2a1a1a, #2d1f1f);
}
.error-title {
  font-size: var(--fs-lg);
  font-weight: 600;
  margin-bottom: var(--sp-2);
  color: var(--color-up);
}
.error-desc {
  font-size: var(--fs-base);
  color: var(--color-text-secondary);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (max-width: 767px) {
  .feature-grid {
    grid-template-columns: 1fr;
  }
  .skeleton-row {
    grid-template-columns: 1fr;
  }
}
</style>
