# Stock Predict - 基金预测系统

基于机器学习的基金净值预测系统，提供市场行情展示、自选基金管理和智能预测功能。

## 功能概览

- **市场行情** — A股/港股/美股主要指数实时展示，迷你走势图，基金涨跌排行
- **自选基金** — 添加/删除关注基金，实时报价刷新，排序筛选
- **智能预测** — 基于 ONNX 模型的基金净值趋势预测，多因子分析

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
| ASP.NET Core 8 | Web API |
| Entity Framework Core | ORM (SQLite) |
| ONNX Runtime | 模型推理 |
| Polly | HTTP 重试策略 |
| Swagger | API 文档 |

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
├── backend-dotnet/          # .NET 后端
│   ├── Controllers/        # API 控制器
│   ├── Services/           # 业务逻辑
│   ├── Models/             # 数据模型
│   ├── Dtos/               # 数据传输对象
│   └── Data/               # 数据库上下文
└── docs/                   # 项目文档
```

## 快速开始

### 环境要求

- Node.js >= 18
- .NET SDK 8.0
- SQLite

### 启动后端

```bash
cd backend-dotnet
dotnet restore
dotnet run
```

后端默认运行在 `http://localhost:5000`

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
| `/market/indices` | GET | 获取市场指数数据 |
| `/market/ranking/{type}` | GET | 获取基金涨跌排行 |
| `/watchlist` | GET | 获取自选列表 |
| `/watchlist` | POST | 添加自选基金 |
| `/watchlist/{code}` | DELETE | 删除自选基金 |
| `/watchlist/quotes` | POST | 批量获取基金报价 |
| `/predict/{code}` | GET | 获取基金预测数据 |
| `/funds/search` | GET | 搜索基金 |

## 设计特色

- **Inter 字体体系** — 专为数据密集型界面设计的字体层级系统
- **市场主题色** — A股红/港股橙/美股蓝，左侧色轨快速区分市场
- **深色模式** — 完整的 light/dark 主题切换
- **响应式布局** — 适配桌面、平板和手机
- **流畅动画** — 统一的缓动函数和时长规范，交错入场动画
- **等宽数字** — `font-variant-numeric: tabular-nums` 确保数值对齐

## License

MIT
