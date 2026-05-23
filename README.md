# Stock Predict - 基金预测系统

基于机器学习的基金净值预测系统，提供市场行情展示、自选基金管理和智能预测功能。

## 功能概览

- **市场行情** — A股/港股/美股主要指数实时展示，迷你走势图，基金涨跌排行
- **自选基金** — 添加/删除关注基金，实时报价刷新，排序筛选
- **智能预测** — 提供隔日和盘中 5 分钟预测接口，当前为 Go 基线逻辑，预留新模型接入

## 技术栈

### 前端
| 技术 | 用途 |
|------|------|
| Vue 3 + TypeScript | 核心框架 |
| Pinia | 状态管理 |
| ECharts | 数据可视化 |
| Element Plus | UI 组件库 |
| Vite | 构建工具 |
| Axios | HTTP 客户端 |

### 后端
| 技术 | 用途 |
|------|------|
| Go 1.22+ | Web API |
| Go 标准库 net/http | 路由、中间件、HTTP 服务 |
| 内存种子数据 | 当前开发期数据仓库 |
| Python / ONNX 预留 | 后续训练模型服务或运行时模型接入 |

## 项目结构

```
stock-predict/
├── frontend/                # Vue 3 前端
│   ├── src/
│   │   ├── api/            # API 请求层
│   │   ├── components/     # 组件
│   │   │   ├── market/     # 行情组件
│   │   │   └── watchlist/  # 自选组件
│   │   ├── composables/    # 组合式函数
│   │   ├── router/         # 路由配置
│   │   ├── stores/         # Pinia Store
│   │   ├── types/          # TypeScript 类型
│   │   ├── utils/          # 工具函数
│   │   └── views/          # 页面视图
│   └── vite.config.ts
├── backend-go/              # Go 后端
├── model-training/          # 模型训练项目
└── docs/                   # 项目文档
```

## 快速开始

### 环境要求

- Node.js >= 18
- Go >= 1.22
- Python 3.11/3.12（训练模型时需要）

### 启动后端

```bash
cd backend-go
go run ./cmd/api
```

后端默认运行在 `http://localhost:5070`

### 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端默认运行在 `http://localhost:5173`

### 构建生产版本

```bash
cd frontend
npm run build
```

## API 接口

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/health` | GET | 后端健康检查 |
| `/api/v1/market/indices` | GET | 获取市场指数数据，包含 A 股、港股、美股与标普 500 |
| `/api/v1/market/ranking/{gainers\|losers}` | GET | 获取基金涨跌排行 |
| `/api/v1/watchlist/quotes` | POST | 批量获取自选基金报价 |
| `/api/v1/predict/{code}` | GET | 获取基金预测数据 |
| `/api/v1/funds/search` | GET | 搜索基金 |
| `/api/v1/funds/filters` | GET | 获取基金筛选项 |
| `/api/v1/funds/sync` | POST | 开发期触发基金同步占位接口 |

## 设计特色

- **Inter 字体体系** — 专为数据密集型界面设计的字体层级系统
- **市场主题色** — A股红/港股橙/美股蓝，左侧色轨快速区分市场
- **深色模式** — 完整的 light/dark 主题切换
- **响应式布局** — 适配桌面、平板和手机
- **流畅动画** — 统一的缓动函数和时长规范，交错入场动画
- **等宽数字** — `font-variant-numeric: tabular-nums` 确保数值对齐

## License

MIT
