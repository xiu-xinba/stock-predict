/** @module market — 行情模块公共入口，导出组件、store 和类型 */
/** 导出市场健康度组件 */
export { default as HealthWidget } from './components/HealthWidget.vue'
/** 导出沪深港通资金流向图表组件 */
export { default as HSGTFlowChart } from './components/HSGTFlowChart.vue'
/** 导出市场行情 store */
export { useMarketStore } from './store/market'
export type * from './types'
