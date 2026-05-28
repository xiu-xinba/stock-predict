# Stock Predict - 基金预测系统

基于机器学习的基金净值预测系统，提供市场行情展示、自选基金管理和智能预测功能。

## 功能概览

- **市场行情** — A股/港股/美股主要指数实时展示，迷你走势图，基金涨跌排行
- **自选基金** — 添加/删除关注基金，实时报价刷新，排序筛选
- **智能预测** — 提供隔日、未来一周和盘中 3/5 分钟预测接口；可分别接入 Python 冠军模型服务，输出 `signal_status`、经验预测区间和收益拆解，异常时回退 Go 基线逻辑

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
| Gin + net/http | 路由、中间件、HTTP 服务 |
| 内存种子数据 | 当前开发期数据仓库 |
| Python 模型服务 / ONNX 预留 | 训练模型在线推理，后续可扩展运行时模型接入 |

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
- Conda + Python 3.11-3.13（训练模型时需要，依赖通过 conda 环境内的 pip 安装）

### 启动后端

```bash
cd backend-go
go run ./cmd/api
```

后端默认运行在 `http://localhost:5070`

如需启用当前训练出的指数基金冠军模型，先启动模型服务，再启动后端：

```powershell
cd model-training
$env:PYTHONPATH="src"
python -m fund_model_training.serve_model --model artifacts/public_mvp_index_fund_tournament_champion.joblib --samples data/processed/public_mvp_daily_weekly_index_fund_samples.csv --port 8090

cd ../backend-go
$env:MODEL_SERVICE_URL="http://127.0.0.1:8090"
go run ./cmd/api
```

未来一周模型和短周期模型可用独立模型服务接入；其中周频模型需要先通过
`retraining_cycle_weekly` 的高置信准确率门禁并生成
`model_registry/weekly_index_fund/current.json`：
`WEEKLY_MODEL_SERVICE_URL=http://127.0.0.1:8092`，
`INTRADAY_MODEL_SERVICE_URL=http://127.0.0.1:8091`。

交付前可运行统一检查脚本：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\prediction-model-delivery-check.ps1
```

需要重跑训练和端到端 smoke 时增加参数：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\prediction-model-delivery-check.ps1 -RunTraining -RunSmoke
```

多基金 API 验收可运行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\start-acceptance-services.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\prediction-api-acceptance.ps1
```

本地验收服务启动后可用下面命令停止：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\stop-acceptance-services.ps1
```

基金搜索底库默认会从东财全量基金代码列表补全，并合并东财净值排行里的
最新净值、日增长率和近 1 月/1 年收益；如果本地
`backend-go/data/funds.json` 少于 1000 条，或旧数据没有可信
`quote_source`，开发启动时会自动同步。自选刷新会优先用腾讯场内基金行情
和东财基金估值覆盖本地净值排行数据。
也可以手动触发：

```powershell
cd backend-go
$env:FUND_UNIVERSE_URL="https://fund.eastmoney.com/js/fundcode_search.js"
$env:FUND_REALTIME_QUOTES_ENABLED="true"
Invoke-RestMethod -Method Post -Uri "http://localhost:5070/api/v1/funds/sync"
```

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
