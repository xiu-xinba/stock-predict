# Design

## Visual Theme

暗色优先的金融工具界面。默认暗色主题，支持亮色切换。信息密度适中，每个屏幕聚焦一个核心任务。视觉层次通过背景色层级（page → card → elevated → hover）而非边框或阴影来建立。克制用色，避免 AI slop 模式（无紫色、无发光阴影、无过度渐变）。Sonoma 暗色 + Ventura 亮色双主题，中性色偏暖灰色调，Sonoma 风格暖灰背景搭配 Apple 风格品牌色。

背景层级：
- 暗色：page(#1a1a1f) → card(#242429) → elevated(#2e2e35) → hover(#36363e)
- 亮色：page(#f5f5f7) → card(#ffffff) → elevated(#eeeef0) → hover(#e8e8eb)

## Design Dials (taste-skill v2)

- **DESIGN_VARIANCE: 6** — 结构化但不死板
- **MOTION_INTENSITY: 5** — 有动效但不花哨
- **VISUAL_DENSITY: 7** — 金融数据密集型，但留呼吸空间

## Color Strategy

Restrained。品牌蓝色占界面面积不超过 10%，仅用于交互元素（链接、按钮、选中态）和关键语义。中性色偏暖灰色调，涨跌色使用中国 A 股惯例（红涨绿跌）。强调色使用深琥珀色（非紫色），仅用于 kicker 标签等极少量场景。

## Color Palette

### Brand

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `--color-brand` | `#007AFF` | `#5e9eff` | 交互元素、链接、选中态 |
| `--color-brand-hover` | `#0062d4` | `#7ab5ff` | 品牌色 hover 态 |
| `--color-brand-contrast` | `#ffffff` | `#0a0e1a` | 品牌色上的文字色 |
| `--color-brand-muted` | `#4d9aff` | `#4a7cc7` | 品牌色弱化态 |
| `--color-brand-light` | `rgba(0,122,255,0.14)` | `rgba(94,158,255,0.18)` | 品牌色浅背景（focus ring、selection） |
| `--color-brand-soft` | `rgba(0,122,255,0.08)` | `rgba(94,158,255,0.10)` | 品牌色低饱和背景 |

### Accent

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `--color-accent` | `#c2410c` | `#f0a050` | kicker 标签（极少量使用） |
| `--color-accent-hover` | `#9a3412` | `#f5b870` | accent hover 态 |
| `--color-accent-soft` | `rgba(194,65,12,0.08)` | `rgba(240,160,80,0.10)` | accent 低饱和背景 |
| `--color-accent-muted` | `#fdba74` | `rgba(240,160,80,0.30)` | accent 弱化态 |

### Semantic

| Token | Light | Dark | Meaning |
|-------|-------|------|---------|
| `--color-up` | `#d92b2b` | `#e8605a` | 涨（A股红） |
| `--color-down` | `#1a9a52` | `#3dbe7a` | 跌（A股绿） |
| `--color-flat` | `#86868b` | `#8e8ea0` | 持平 |
| `--color-warning` | `#c27803` | `#f0a050` | 警告 |
| `--color-danger` | `#d92b2b` | `#e8605a` | 危险 |
| `--color-info` | `#86868b` | `#8e8ea0` | 信息 |

### Direction backgrounds

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `--color-up-bg` | `rgba(217,43,43,0.08)` | `rgba(232,96,90,0.12)` | 上涨背景 |
| `--color-up-border` | `rgba(217,43,43,0.2)` | `rgba(232,96,90,0.25)` | 上涨边框 |
| `--color-down-bg` | `rgba(26,154,82,0.08)` | `rgba(61,190,122,0.12)` | 下跌背景 |
| `--color-down-border` | `rgba(26,154,82,0.2)` | `rgba(61,190,122,0.25)` | 下跌边框 |

### Neutral

| Token | Light | Dark |
|-------|-------|------|
| `--color-bg-page` | `#f5f5f7` | `#1a1a1f` |
| `--color-bg-card` | `#ffffff` | `#242429` |
| `--color-bg-elevated` | `#eeeef0` | `#2e2e35` |
| `--color-bg-hover` | `#e8e8eb` | `#36363e` |
| `--color-bg-overlay` | `rgba(0,0,0,0.5)` | `rgba(0,0,0,0.6)` |
| `--color-bg-nav` | `#ffffff` | `#242429` |
| `--color-bg-topbar` | `rgba(245,245,247,0.72)` | `rgba(26,26,31,0.78)` |
| `--color-border` | `rgba(0,0,0,0.08)` | `rgba(255,255,255,0.08)` |
| `--color-border-light` | `rgba(0,0,0,0.04)` | `rgba(255,255,255,0.04)` |
| `--color-text-primary` | `#1d1d1f` | `#f0f0f5` |
| `--color-text-regular` | `#424245` | `#c0c0cc` |
| `--color-text-secondary` | `#86868b` | `#8e8ea0` |
| `--color-text-tertiary` | `#aeaeb2` | `#5a5a6e` |
| `--color-text-disabled` | `#aeaeb2` | `#5a5a6e` |

### Market region accents

| Token | Light | Dark | Region |
|-------|-------|------|--------|
| `--color-hk` | `#b54708` | `#f0a050` | 港股 |
| `--color-hk-bg` | `rgba(181,71,8,0.08)` | `rgba(240,160,80,0.10)` | 港股背景 |
| `--color-us` | `#3538cd` | `#8e9eff` | 美股 |
| `--color-us-bg` | `rgba(53,56,205,0.08)` | `rgba(142,158,255,0.10)` | 美股背景 |

### Chart

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `--color-chart-axis` | `#86868b` | `#6a6a7e` | 坐标轴文字 |
| `--color-chart-grid` | `#e8e8eb` | `rgba(255,255,255,0.04)` | 网格线 |
| `--color-chart-ma5` | `#d97706` | `#fbbf24` | MA5 均线 |
| `--color-chart-ma10` | `#2563eb` | `#60a5fa` | MA10 均线 |
| `--color-chart-ma20` | `#0891b2` | `#22d3ee` | MA20 均线 |

### Chart palette

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `--color-chart-p1` | `#007AFF` | `#5e9eff` | 主色 |
| `--color-chart-p2` | `#1a9a52` | `#3dbe7a` | 辅色 |
| `--color-chart-p3` | `#f0a050` | `#f0a050` | 第三色 |
| `--color-chart-p4` | `#d92b2b` | `#e8605a` | 第四色 |
| `--color-chart-p5` | `#0891b2` | `#22d3ee` | 第五色 |
| `--color-chart-p6` | `#db2777` | `#f472b6` | 第六色 |
| `--color-chart-p7` | `#65a30d` | `#a3e635` | 第七色 |
| `--color-chart-p8` | `#ea580c` | `#fb923c` | 第八色 |
| `--color-chart-p9` | `#475569` | `#94a3b8` | 第九色 |
| `--color-chart-p10` | `#0d9488` | `#2dd4bf` | 第十色 |

### Risk

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `--color-risk-low` | `#12b76a` | `#3dbe7a` | 低风险 |
| `--color-risk-medium-low` | `#17b26a` | `#5cd89a` | 中低风险 |
| `--color-risk-medium` | `#f79009` | `#f0a050` | 中风险 |
| `--color-risk-medium-high` | `#f04438` | `#e87060` | 中高风险 |
| `--color-risk-high` | `#d92d20` | `#e8605a` | 高风险 |

## Typography

Font stack: 'Inter', -apple-system, BlinkMacSystemFont, 'PingFang SC', 'Segoe UI', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif.

Mono stack: 'JetBrains Mono', 'SF Mono', 'Menlo', 'Cascadia Code', ui-monospace, monospace. 用于数值展示（价格、涨跌幅、净值）。

### Scale

| Token | Size | Usage |
|-------|------|-------|
| `--fs-xs` | 12px | 辅助标注 |
| `--fs-sm` | 13px | 次要文字 |
| `--fs-base` | 15px | 正文 |
| `--fs-md` | 17px | 小标题 |
| `--fs-lg` | 20px | 区块标题 |
| `--fs-xl` | 24px | 页面标题 |
| `--fs-2xl` | 32px | 大数字 |
| `--fs-3xl` | 40px | 关键指标 |
| `--fs-4xl` | 48px | 英雄数字 |
| `--fs-5xl` | 48px | 预测百分比主展示 |

### Line height

| Token | Value | Usage |
|-------|-------|-------|
| `--lh-tight` | 1.2 | 大数字 |
| `--lh-snug` | 1.35 | 标题 |
| `--lh-normal` | 1.5 | 正文 |
| `--lh-relaxed` | 1.65 | 长文本 |

### Letter spacing

| Token | Value | Usage |
|-------|-------|-------|
| `--ls-tighter` | -0.03em | 大标题 |
| `--ls-tight` | -0.01em | 中标题 |
| `--ls-normal` | 0 | 正文 |
| `--ls-wide` | 0.02em | 小标签 |
| `--ls-wider` | 0.04em | 全大写标签 |
| `--ls-widest` | 0.08em | kicker |

### Weight

| Token | Value | Usage |
|-------|-------|-------|
| `--fw-regular` | 400 | 正文 |
| `--fw-medium` | 500 | 强调 |
| `--fw-semibold` | 600 | 标题 |
| `--fw-bold` | 700 | 关键数字 |
| `--fw-extrabold` | 800 | 英雄数字 |
| `--fw-black` | 900 | 最大数字 |

## Spacing

Base unit: 4px.

| Token | Value | Usage |
|-------|-------|-------|
| `--sp-0_5` | 2px | 微间距（badge 内间距、紧凑 gap） |
| `--sp-1` | 4px | 紧凑间距 |
| `--sp-2` | 8px | 元素内间距 |
| `--sp-3` | 12px | 小区块间距 |
| `--sp-4` | 16px | 标准间距 |
| `--sp-5` | 20px | 区块间距 |
| `--sp-6` | 24px | 大区块间距 |
| `--sp-8` | 32px | 区段间距 |
| `--sp-10` | 40px | 大区段间距 |
| `--sp-12` | 48px | 页面区段间距 |
| `--sp-16` | 64px | 最大区段间距 |

## Border Radius

| Token | Value | Usage |
|-------|-------|-------|
| `--radius-sm` | 6px | 小元素（标签、badge） |
| `--radius-md` | 10px | 按钮、输入框 |
| `--radius-lg` | 14px | 卡片、面板 |
| `--radius-xl` | 18px | dock、弹出层 |
| `--radius-full` | 9999px | 圆形头像、pill |

## Elevation

通过背景色层级而非阴影建立深度。阴影仅用于弹出层和模态框。禁止发光阴影（glow）。三层阴影系统搭配 inset 高光：

### Light

| Token | Value | Usage |
|-------|-------|-------|
| `--shadow-sm` | `inset 0 0.5px 0 rgba(255,255,255,0.8), 0 1px 2px rgba(0,0,0,0.04)` | 卡片默认 |
| `--shadow-md` | `inset 0 0.5px 0 rgba(255,255,255,0.8), 0 1px 2px rgba(0,0,0,0.04), 0 4px 12px rgba(0,0,0,0.06)` | 悬浮卡片 |
| `--shadow-lg` | `0 2px 4px rgba(0,0,0,0.06), 0 8px 24px rgba(0,0,0,0.1), 0 16px 48px rgba(0,0,0,0.08)` | 弹出层、模态框 |

### Dark

| Token | Value | Usage |
|-------|-------|-------|
| `--shadow-sm` | `inset 0 0.5px 0 rgba(255,255,255,0.06), 0 1px 2px rgba(0,0,0,0.2)` | 卡片默认 |
| `--shadow-md` | `inset 0 0.5px 0 rgba(255,255,255,0.06), 0 1px 2px rgba(0,0,0,0.2), 0 4px 12px rgba(0,0,0,0.15)` | 悬浮卡片 |
| `--shadow-lg` | `inset 0 0.5px 0 rgba(255,255,255,0.08), 0 2px 4px rgba(0,0,0,0.25), 0 12px 32px rgba(0,0,0,0.3)` | 弹出层、模态框 |

背景层级（暗色主题）：page(#1a1a1f) → card(#242429) → elevated(#2e2e35) → hover(#36363e)

## Motion

| Token | Value | Usage |
|-------|-------|-------|
| `--transition-fast` | 0.15s ease | 颜色变化、hover |
| `--transition-normal` | 0.25s ease | 展开/收起 |
| `--transition-spring` | 0.4s cubic-bezier(0.16,1,0.3,1) | 弹性过渡 |
| `--ease-out-quart` | cubic-bezier(0.25,1,0.5,1) | 进度条、滑入 |
| `--ease-out-expo` | cubic-bezier(0.16,1,0.3,1) | 弹出层 |

禁止：bounce、elastic、layout 属性动画、发光阴影、过度渐变。尊重 prefers-reduced-motion。

### Transition catalog

| Name | Trigger | Curve |
|------|---------|-------|
| `fade` | 页面内容切换 | `opacity 0.12s ease` |
| `dock` | 底部导航展开/收起 | enter: `0.3s cubic-bezier(0.2,0.8,0.2,1)` / leave: `0.18s ease` |
| `pill` | 导航药丸出现/消失 | `0.18s ease` |
| `row` | 自选列表项增删 | `opacity + transform, var(--transition-normal)` |
| `fab-tooltip` | 刷新按钮提示 | enter: `0.2s cubic-bezier(0.16,1,0.3,1)` / leave: `0.15s ease` |
| `filter-panel` | 搜索筛选面板 | `var(--transition-fast)` |

### Animation catalog

| Name | Usage | Parameters |
|------|-------|-----------|
| `spin` | 加载旋转 | `0.8s linear infinite` |
| `shimmer` | 骨架屏闪烁 | `1.5s infinite` |
| `pulse-dot` | 实时指示点 | `2s ease-in-out infinite` |
| `fab-spin` | 刷新按钮旋转 | `0.8s cubic-bezier(0.4,0,0.2,1) infinite` |
| `vt-fade-out` | View Transition 退出 | `0.2s ease-out` + scale(0.98) |
| `vt-fade-in` | View Transition 进入 | `0.25s ease-out` + scale(1.01) |

### Stagger entry

`useStaggerEntry(selector, options)` 使用 IntersectionObserver 实现列表项交错入场。初始状态 `opacity:0 translateY(12px) scale(0.98)`，进入视口后 `opacity:1 translateY(0) scale(1)`，过渡 `0.5s ease-out-expo`，每项延迟 `staggerMs`（默认 60ms）。

### Accessibility

全局 `prefers-reduced-motion: reduce` 将所有动画和过渡时长压缩到 0.01ms。组件级覆盖确保 dock、排行、搜索等过渡在 reduced-motion 下也正确降级。`prefers-reduced-transparency: reduce` 移除 topbar/dock/fab 的 backdrop-filter，改用纯色背景。

## Vibrancy

Topbar 和 dock 使用 `backdrop-filter: blur(20px) saturate(180%)` 搭配半透明 rgba 背景实现毛玻璃效果：

- 亮色：`--color-bg-topbar: rgba(245,245,247,0.72)`
- 暗色：`--color-bg-topbar: rgba(26,26,31,0.78)`

`prefers-reduced-transparency: reduce` 时移除 backdrop-filter，改用纯色背景。

## Anti-Patterns (taste-skill v2 LILA Compliance)

- 禁止紫色（#8b5cf6 等）作为主色或强调色
- 禁止发光阴影（shadow-glow-*）
- 禁止 brand+accent 双色渐变装饰线
- 禁止 body::after 彩色光晕
- 禁止 fractalNoise 纹理
- 禁止 hover 时 translateY(-8px) 以上位移
- 禁止 gradient text
- 禁止 em-dash（—）

## Components

### Card

圆角 `--radius-lg`，背景 `--color-bg-card`，边框 `1px solid --color-border`。无阴影默认。内容区间距 `--sp-4`。

### CollapsibleCard

可折叠卡片。Props: `title`, `defaultCollapsed`, `bodyMaxHeight`。展开/收起使用 `--transition-spring`。Slots: `default`, `header-extra`。

### Button

品牌色按钮：背景 `--color-brand`，文字 `--color-brand-contrast`，圆角 `--radius-md`，内间距 `--sp-2 --sp-4`。

### Badge/Tag

圆角 `--radius-sm`，字号 `--fs-xs`，内间距 `2px --sp-2`。语义色使用对应的 soft 背景 + 主色文字。涨跌 badge 使用 `--color-up-bg`/`--color-down-bg` + `--color-up`/`--color-down`。

### Input

圆角 `--radius-md`，边框 `1px solid --color-border`，focus 态边框 `--color-brand`。

### Chart

ECharts 图表使用 CSS 变量读取主题色。`getBaseChartOption()` 提供统一的 tooltip、grid、axis 基础配置。网格线 `--color-chart-grid`，轴线 `--color-chart-axis`。涨跌色使用 `--color-up` / `--color-down`。图表调色板使用 `--color-chart-p1` 至 `--color-chart-p10`。`cssVar()` 工具函数从 CSS 变量读取值，带主题感知缓存（暗色切换时自动失效）。

### AssetHeader

通用资产头部组件。Props: `name`, `code`, `price`, `change`, `changePercent`, `isUp`, `infoItems`, `isInWatchlist`, `watchlistLoading`, `gridColumns`(默认3), `badges`, `liveDotTitle`。Emits: `toggleWatchlist`。Slots: `#actions`。基金和股票头部均为其薄包装层。

### DetailPageLayout

通用详情页布局。Props: `loading`, `error`(AppError), `code`, `hasContent`, `skeletonCount`(默认5)。三态切换：骨架屏 / 错误状态 / 内容。Slots: `#header`, `#default`, `#footer`。最大宽度 800px 居中。

### RankingList

通用排行列表。Props: `title`, `items`, `type`('gainers'|'losers'), `routePrefix`, `codeField`, `nameField`, `subField`。基金和股票排行均为其封装。

### PredictionDisplay

通用预测摘要展示。Props: `prediction`(PredictionDisplayData), `loading`, `error`。Emits: `view-full`。

### ErrorState

错误状态展示。Props: `message`(默认'加载失败'), `retryLabel`(默认'重试'), `compact`。Emits: `retry`。

### SearchOverlay

全屏搜索浮层。支持基金搜索（300ms 防抖）和股票搜索（300ms 防抖）。键盘导航：ArrowUp/Down 循环高亮，Enter 选择，Escape 关闭。搜索历史持久化到 localStorage。

## Icons

使用内联 SVG，尺寸 16px/20px/24px。颜色继承父元素 `currentColor`。空状态使用 `--color-icon-empty`。

## Layout Architecture

### Global layout (App.vue)

```
┌─────────────────────────────────────┐
│  topbar (sticky, 毛玻璃 backdrop)    │  品牌标识 + 搜索 + 主题切换
├─────────────────────────────────────┤
│  main-content                       │  width: min(100%, 1180px), 居中
│  (router-view)                      │  padding-bottom: 116px (为 dock 留空)
├─────────────────────────────────────┤
│  dock-hotspot (28px)                │  底部触发热区
│  dock-pill / dock (浮动导航)         │  3 tab: 自选/行情/预测
└─────────────────────────────────────┘
│  MarketDock (浮动)                  │  A股/港股/美股指数面板
│  RefreshFab (浮动)                  │  右下角刷新按钮
│  SearchOverlay (全屏浮层)           │  搜索面板
```

### Dock interaction

macOS 风格浮动导航栏。默认显示 dock-pill（小药丸），鼠标/触摸底部热区后展开完整 dock。900ms 无交互后自动收起。页面加载 1800ms 后自动收起。支持鼠标悬停保持、触摸保持。收起/展开有 `dock-enter/leave` 过渡动画。

### Route structure

| Path | Component | Meta |
|------|-----------|------|
| `/watchlist` | WatchlistView | 自选 |
| `/market` | MarketView | 行情 |
| `/predict` | PredictView | 预测 |
| `/predict/:fundCode` | PredictView | 预测详情 |
| `/fund/:fundCode` | FundDetailView | 基金详情 |
| `/stock/:stockCode` | StockDetailView | 股票详情 |
| `/:pathMatch(.*)*` | NotFoundView | 404 |

所有路由组件懒加载。`beforeEach` 设置 `document.title`。

### Component tree

```
App.vue
├── topbar (header)
├── <router-view>
├── dock-pill / dock (nav)
├── SearchOverlay
└── RefreshFab

MarketView → FundRanking/StockRanking → RankingList + MarketDock
PredictView → PredictionCard → useECharts
WatchlistView → ErrorState + WatchlistEmpty
FundDetailView → DetailPageLayout → AssetHeader(FundHeader) + Fund*组件
StockDetailView → DetailPageLayout → AssetHeader(StockHeader) + Stock*组件
```

## Responsive

### Breakpoints

| Breakpoint | Target | Key changes |
|-----------|--------|-------------|
| `>1024px` | 桌面 | dock 居中浮动，3 等宽 item(58px)，多列布局 |
| `768-1024px` | 平板 | skeleton 单列，metrics 2 列 |
| `<768px` | 手机 | dock 全宽展开，item 31%，单列布局，topbar 缩小 |

### Container Queries

`.card-container` 定义 `container-type: inline-size`，`@container card (max-width: 480px)` 时 info-grid 2 列、return-grid 3 列、risk/portfolio 单列、manager-stats 2 列。与视口断点解耦，用于卡片内部布局自适应。

## Dark Mode

默认暗色。通过 `html.dark` 类切换。所有颜色通过 CSS 变量定义，切换时仅修改变量值。`useTheme` composable 管理状态，localStorage 持久化（key: `theme-preference`），优先级：localStorage > `prefers-color-scheme` > 默认 light。

主题传播路径：`toggleTheme()` → `mode` ref 变化 → `watch` 触发 → `localStorage.setItem` + `applyTheme()` → `html.dark` class 切换 → CSS 变量自动切换 → Element Plus 暗色覆盖生效 → ECharts `cssVar` 缓存失效 → 图表下次渲染自动更新。

## State Architecture

### Stores (Pinia)

| Store | Purpose | Persistence |
|-------|---------|-------------|
| `useWatchlistStore` | 自选列表门面（聚合基金+股票） | - |
| `useFundWatchlistStore` | 基金自选 | `fund-watchlist` |
| `useStockWatchlistStore` | 股票自选 | `stock-watchlist` |
| `useMarketStore` | 行情数据（指数+排行，30s 自动刷新） | - |
| `usePredictionStore` | 预测数据+基金搜索 | - |
| `useFundDetailStore` | 基金详情 | - |
| `useStockDetailStore` | 股票详情 | - |

### Data flow

```
API Layer (axios) → Store (Pinia) → Component
     ↑                    ↑
  api/index.ts      stores/*.ts
  (请求去重/重试/    (状态管理/序列号
   取消/CSRF/错误    防竞态/持久化/
   分类AppError)     缓存节流)
```

关键模式：
- 请求去重：相同请求 abort 前一个
- 竞态防护：递增序列号防止过期响应覆盖新数据
- 自动刷新：MarketStore 引用计数管理 30s 定时刷新，配合 `visibilitychange` 暂停
- 缓存节流：MarketStore 30s 客户端缓存，WatchlistStore 5s 刷新节流
- 错误处理：API 层统一转换为 `AppError`（code/message/retryable/type），Store 层区分 `CancelError`

### Composables

| Composable | Purpose |
|-----------|---------|
| `useTheme` | 主题切换（亮/暗），localStorage 持久化 |
| `useECharts` | ECharts 生命周期（init/resize/dispose），ResizeObserver + RAF 节流 |
| `useStaggerEntry` | 列表项交错入场动画，IntersectionObserver |
| `useFundCodeRoute` | 从路由参数提取基金代码 |
| `useFundSearch` | 基金搜索（防抖、下拉建议、历史记录） |
| `useSparkline` | 迷你折线图（市场指数），共享 ResizeObserver |

## Type System

类型文件按领域拆分：`types/index.ts`（统一导出 + AppError）、`predict.ts`、`watchlist.ts`、`market.ts`、`stock.ts`、`fundDetail.ts`。

关键设计：
- snake_case 字段名与后端 JSON 对齐，无 camelCase 转换层
- `ApiResponse<T>` 统一响应包装（code/message/data）
- 联合类型枚举（`'up'|'down'|'flat'`）而非 TypeScript enum
- 聚合类型（`StockDetailData`、`FundDetailData`）组合多个子类型
- `AppError` 结构化错误（code/message/retryable/type）
