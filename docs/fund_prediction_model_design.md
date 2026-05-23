# 指数基金预测模型搭建设计报告

编写日期：2026-05-23

## 1. 本次修订结论

本报告的设计目标从“适配现有训练框架”调整为“从预测效果出发，选择多组最优候选模型并滚动验证择优”。现有工程只作为后续部署载体，不限制模型选择。

修订后的核心判断：

1. 预测对象以指数基金为主，包括场外指数基金、ETF、LOF、联接基金、港股指数基金和 QDII 指数基金。
2. 日内短周期预测和次日/本周预测必须使用不同模型族、不同特征窗口和不同验证方式。
3. 恐慌数据不是展示层附加项，而是训练特征的一部分，必须进入样本表、回测和特征重要性分析。
4. 指数基金预测不能只看基金成交量。应围绕“跟踪指数收益 + 成分股贡献 + 期货/期权定价 + 商品冲击 + 利率汇率 + 跨市场传导 + 资金流 + 舆情恐慌”建立多资产特征体系。
5. 最终模型不预设为某一个算法胜出，而采用 Champion/Challenger 机制：多模型同数据训练、滚动回测、分市场状态评估，择优或集成上线。

## 2. 目标与边界

### 2.1 预测目标

本项目建设两类模型：

1. 长周期模型
   - 预测下一个交易日涨跌幅。
   - 预测未来一周涨跌幅。
   - 输出方向：上涨、下跌、震荡。
   - 适用于指数基金、ETF、LOF、港股/QDII 指数基金。

2. 日内短周期模型
   - 在交易时段预测未来 3 分钟或 5 分钟涨跌走势。
   - 优先适用于高流动性 ETF、LOF、港股 ETF。
   - 对场外指数基金只输出“盘中估值走势”，不声称预测真实盘中净值。

### 2.2 误差目标的专业化定义

用户提出的目标是：

- 次日/本周预测与实际偏差不超过 10%。
- 3/5 分钟预测偏差不超过 5%。

从金融建模角度，这两个目标不能理解为“每一笔样本都保证误差小于阈值”。更可执行的定义如下：

| 目标 | 建议验收口径 |
|---|---|
| 长周期 10% 偏差 | 在高置信样本上，方向正确且收益率相对误差或分位区间覆盖达到目标；同时报告覆盖率 |
| 日内 5% 偏差 | 在高置信样本上，概率校准误差、方向准确率和交易成本后收益达到目标；同时报告覆盖率 |
| 非高置信样本 | 输出 `no_signal` 或 `low_confidence`，不作为可行动预测 |

这样做不是降低要求，而是避免在高噪声市场中给出不可验证的确定性承诺。金融机构内部也通常使用“准确率 + 覆盖率 + 回测收益 + 风险控制”的联合口径。

## 3. 选择的方法是什么

### 3.1 总体方法：模型锦标赛而非单模型押注

本项目采用模型锦标赛：

1. 为长周期、日内、舆情恐慌分别设立候选模型池。
2. 每个候选模型使用同一批 point-in-time 数据训练。
3. 通过滚动回测、验证集、测试集和纸面交易选择 Champion 模型。
4. 未胜出的模型作为 Challenger 定期复训，若连续跑赢 Champion 再替换。
5. 若不同模型在不同市场状态下各有优势，则采用动态集成，而不是只选一个模型。

### 3.2 长周期模型候选池

长周期模型面向次日和未来一周，重点捕捉指数、期货、商品、宏观、风格和恐慌状态的传导。

| 候选模型 | 作用 | 入选原因 |
|---|---|---|
| Temporal Fusion Transformer, TFT | 多周期收益预测和可解释注意力 | 能处理静态特征、已知未来特征、历史多变量特征，适合多资产因子输入 |
| PatchTST / iTransformer | 多变量长序列预测 | 对长窗口时间序列更强，适合指数、期货、商品、汇率、利率同步建模 |
| TimesNet / TimeMixer | 多尺度周期和趋势建模 | 适合处理日频、周频、跨资产不同周期的混合信号 |
| Temporal Graph Neural Network / Graph Attention Network | 指数与成分股、行业、期货、商品之间的传导 | 指数基金天然是网络结构，成分股权重、行业关联、期货基差都可以建图 |
| CatBoost / XGBoost / LightGBM | 强 tabular 基线和集成成员 | 对稀疏、缺失、非线性特征稳健，便于解释和做特征消融 |
| Time-series foundation model, 如 MOMENT / Lag-Llama / Chronos 类 | 实验性 challenger | 用于测试预训练时序模型能否在微调后提升泛化，不作为第一版唯一主模型 |

长周期模型的最终形态建议是“深度时序模型 + 图模型 + GBDT 校准器”的集成：

```text
指数/期货/商品/宏观序列 -> TFT/PatchTST/iTransformer
成分股/行业/期货/商品关系图 -> Temporal GNN
结构化因子、恐慌因子、估值因子 -> CatBoost/XGBoost/LightGBM
三者输出 -> stacking/calibration -> 最终收益率、方向和置信区间
```

### 3.3 日内短周期模型候选池

日内模型面向 3/5 分钟，预测目标更接近微观结构和跨市场即时冲击，不能照搬日频模型。

| 候选模型 | 作用 | 入选原因 |
|---|---|---|
| DeepLOB | 盘口和限价订单簿特征 | 适合处理买卖盘、盘口深度、成交冲击 |
| LOBTransformer / 短序列 Transformer | 分钟级和 tick 级状态变化 | 适合捕捉短时注意力和突发冲击 |
| TCN / WaveNet 类时序卷积 | 低延迟短序列预测 | 推理速度快，适合线上 3/5 分钟模型 |
| CatBoost/XGBoost/LightGBM | 微观结构强基线 | 可快速验证盘口、基差、资金流、波动率等特征是否有效 |
| 强化学习策略模型, 如 FinRL 方向 | 交易执行与仓位决策验证 | 不作为收益预测主模型，只用于后续交易策略层评估 |

日内模型推荐架构：

```text
盘口/成交/IOPV/溢折价 -> DeepLOB/LOBTransformer
指数期货、成分股、行业 ETF 分钟序列 -> TCN/短序列 Transformer
恐慌、新闻突发、资金流、期权 IV -> GBDT/Transformer covariates
模型输出 -> 交易成本过滤 -> 3m/5m 方向、幅度、可行动标记
```

### 3.4 舆情与恐慌模型候选池

舆情模型不是单独给用户看分数，而是生成可训练特征。

| 候选模型 | 作用 | 入选原因 |
|---|---|---|
| 中文 FinBERT / 金融 BERT | 新闻、公告、研报、社媒情绪 | 中文金融文本语义更贴近市场语言 |
| FinGPT 类金融大模型 | 事件抽取和主题分类 | 适合从新闻中抽取政策、监管、业绩、风险事件 |
| 弱监督事件分类器 | 扩大训练样本 | 用关键词、市场反应和人工小样本生成初始标签 |
| 恐慌因子合成模型 | 构造 A 股/港股 panic score | 将期权、跌停、资金流、新闻负面情绪融合成训练特征 |

舆情与恐慌输出字段：

```text
sentiment_score
panic_score
policy_risk_score
liquidity_stress_score
geopolitical_risk_score
negative_news_intensity
topic_embedding
entity_embedding
source_reliability
```

这些字段按 5 分钟、30 分钟、1 日、5 日窗口聚合后进入长周期和日内模型训练。

## 4. 为什么这样选择

### 4.1 指数基金的收益来源不是基金本身，而是跟踪资产

指数基金的短中期收益主要来自：

1. 跟踪指数涨跌。
2. 成分股权重与成分股波动。
3. 指数期货对现货指数的领先或反馈。
4. ETF 溢折价、IOPV、申赎套利和流动性。
5. 市场风格、行业轮动、宏观和政策冲击。
6. 全球资产价格变化，尤其是商品、汇率、利率和海外股指。

所以模型不能以“基金历史净值 + 成交量”为核心。正确做法是先预测跟踪指数及其传导链，再映射到具体基金收益：

```text
fund_return
  = tracking_index_return * beta
  + tracking_error
  + ETF_premium_discount_change
  + flow_liquidity_effect
  - fee_drag
  + residual
```

### 4.2 长周期与日内周期的市场机制不同

长周期模型关心：

- 隔夜信息消化。
- 海外市场和商品市场冲击。
- 期货基差和期限结构。
- 政策、宏观、资金流和舆情。
- 风格轮动和行业景气。

日内模型关心：

- 盘口深度和买卖价差。
- ETF 申赎套利、IOPV 和溢折价。
- 期指分钟级基差变化。
- 成分股即时异动和权重贡献。
- 突发新闻和恐慌扩散速度。

因此两类任务应使用不同模型、不同数据窗口、不同标签和不同评价指标。

### 4.3 期货、商品、利率和汇率是指数基金预测的核心外生变量

对于指数基金，跨资产冲击常常早于基金净值反映：

- 沪深 300、中证 500、中证 1000 股指期货影响 A 股宽基指数预期。
- 恒指期货、恒生科技期货影响港股指数基金。
- 布伦特原油、WTI、国内原油期货影响能源、化工、航空、通胀预期和风险偏好。
- 铜、铁矿石、螺纹钢、焦煤焦炭影响周期股、地产链和工业需求预期。
- 黄金影响避险状态。
- 美债收益率、美元指数、USD/CNH、USD/CNY 影响港股、成长股、外资流向和估值折现率。
- 中国利率、DR007、SHIBOR、国债收益率和央行公开市场操作影响流动性。

这些变量必须进入训练数据，且要按照真实发布时间和交易时区对齐，不能使用预测时点之后才可见的数据。

### 4.4 恐慌因子要进入模型，而不是事后解释

恐慌数据应作为训练样本的一部分：

- A 股恐慌：期权隐含波动率、波动率偏斜、put/call 成交量比、跌停家数、破位指数数量、融资余额变化、北向资金急撤、负面新闻强度。
- 港股恐慌：VHSI、恒指/恒科期货基差、港股卖空比例、南向资金、美元/港元流动性、负面新闻强度。
- 商品恐慌：原油和工业金属的大幅波动、黄金避险上涨、美元指数走强。

训练时必须做消融实验：

```text
全特征模型
去掉恐慌因子模型
去掉期货/商品模型
去掉舆情模型
去掉跨市场模型
```

只有确认某类数据在验证集和测试集稳定增益，才保留到线上模型。

## 5. 数据设计

### 5.1 训练数据范围

以指数基金为主，建议第一批覆盖：

| 市场 | 基金类型 | 示例方向 |
|---|---|---|
| A 股 | 沪深 300、中证 500、中证 1000、上证 50、创业板、科创、红利、消费、医药、新能源、半导体 ETF/指数基金 | 宽基 + 重点行业 |
| 港股 | 恒生指数、恒生科技、港股通、国企指数 ETF/指数基金 | 港股宽基和科技 |
| QDII | 纳指、标普、全球科技、原油、黄金相关指数基金 | 跨市场和商品冲击 |

### 5.2 数据类别

| 类别 | 必要字段 | 作用 |
|---|---|---|
| 基金数据 | 净值、估算净值、ETF 成交价、IOPV、溢折价、规模、份额、申赎、费率、跟踪指数 | 标签、基金收益映射、跟踪误差 |
| 跟踪指数 | OHLCV、涨跌幅、成交额、换手、估值、行业权重、成分股权重 | 指数基金核心收益来源 |
| 成分股 | 加权收益、加权成交额、涨跌停、北向持仓、融资融券、公告事件 | 指数涨跌贡献分解 |
| 行业/风格 | 行业指数、主题指数、大小盘、成长/价值、红利/质量/低波 | 风格轮动 |
| 股指期货 | IF、IH、IC、IM 价格、基差、期限结构、持仓量、成交量 | 领先指数预期 |
| 期权/波动率 | ETF/指数期权 IV、skew、put/call、隐含波动率期限结构 | 恐慌和尾部风险 |
| 商品期货 | Brent、WTI、国内原油、铜、铝、铁矿、螺纹、焦煤焦炭、黄金、农产品 | 通胀、周期、风险偏好 |
| 利率/信用 | DR007、SHIBOR、国债收益率、信用利差、央行公开市场操作 | 流动性和估值折现 |
| 汇率 | USD/CNY、USD/CNH、DXY、HKD 利率和汇率 | 外资流向、港股估值、QDII |
| 海外市场 | 标普、纳指、道指、VIX、日经、韩国、台湾、欧洲主要指数 | 隔夜风险和区域传导 |
| 资金流 | 北向、南向、ETF 申赎、主力资金、融资余额变化 | 供需和风险偏好 |
| 新闻公告 | 政策、监管、行业、公司、宏观、地缘风险 | 事件冲击 |
| 社媒舆情 | 热度、负面比例、恐慌扩散、关键词主题 | 短期情绪 |
| 交易日历 | A/H/美/商品交易日、节假日、半日市、夜盘 | 时区与可见性对齐 |

### 5.3 数据源候选

开发期可用开源或低门槛数据，生产期建议接入授权数据。

| 数据 | 开发期来源 | 生产期建议 |
|---|---|---|
| 基金、A 股、ETF、部分港股 | AkShare、Tushare Pro、BaoStock、东方财富公开接口 | Wind、同花顺 iFinD、聚宽、米筐、券商行情 |
| 交易所行情 | 上交所、深交所、中金所、上期所、大商所、郑商所、港交所公开数据 | 交易所授权行情、券商 Level-1/Level-2 |
| 商品与海外 | 交易所公开数据、Yahoo/Stooq 等开发期源 | Bloomberg、Refinitiv、Wind、ICE/CME/LME 授权源 |
| 宏观利率汇率 | 央行、外汇交易中心、国家统计局、FRED、HKMA | Wind、Bloomberg、Refinitiv |
| 新闻舆情 | GDELT、交易所公告、财经网站公开源 | 财联社、Wind 新闻、同花顺、专业舆情数据商 |
| 恐慌指数 | 港股 VHSI、A 股自建合成恐慌因子 | 恒生指数公司、期权链授权数据、专业波动率数据 |

### 5.4 训练数据库表

| 表名 | 粒度 | 关键字段 |
|---|---|---|
| `dim_fund` | 基金 | `fund_code`, `fund_type`, `tracking_index`, `market`, `is_etf`, `is_lof`, `fee_rate` |
| `fund_daily` | 日 | `fund_code`, `trade_date`, `nav`, `adjusted_nav`, `estimated_nav`, `share`, `aum`, `flow` |
| `fund_intraday` | 1 分钟/tick | `fund_code`, `timestamp`, `price`, `iopv`, `premium_pct`, `volume`, `amount`, `bid_ask_spread` |
| `index_daily` | 日 | `index_code`, `trade_date`, `open`, `high`, `low`, `close`, `volume`, `amount`, `valuation` |
| `index_intraday` | 1 分钟 | `index_code`, `timestamp`, `price`, `return`, `volume`, `amount` |
| `index_constituent` | 调仓日 | `index_code`, `stock_code`, `effective_date`, `weight`, `industry`, `free_float_mktcap` |
| `stock_daily_intraday` | 日/分钟 | `stock_code`, `timestamp`, `return`, `volume`, `turnover`, `limit_status`, `northbound_holding` |
| `futures_bar` | 日/分钟 | `contract`, `underlying`, `timestamp`, `price`, `basis`, `open_interest`, `term_structure` |
| `commodity_bar` | 日/分钟 | `symbol`, `asset_class`, `timestamp`, `price`, `return`, `volatility` |
| `option_volatility` | 日/分钟 | `underlying`, `timestamp`, `iv`, `skew`, `put_call_ratio`, `volume`, `open_interest` |
| `macro_rate_fx` | 日/分钟 | `symbol`, `timestamp`, `value`, `change`, `release_time` |
| `cross_market` | 日/分钟 | `market`, `timestamp`, `index_return`, `vix`, `risk_on_off` |
| `capital_flow` | 日/分钟 | `market`, `timestamp`, `northbound`, `southbound`, `etf_flow`, `margin_balance` |
| `sentiment_event` | 事件 | `event_time`, `release_time`, `entity`, `topic`, `sentiment_score`, `panic_score`, `source` |
| `panic_factor` | 日/分钟 | `market`, `timestamp`, `fear_score`, `iv_component`, `flow_component`, `news_component`, `limit_component` |
| `label_daily_weekly` | 日 | `fund_code`, `trade_date`, `return_1d`, `return_1w`, `direction_1d`, `direction_1w` |
| `label_intraday` | 分钟 | `fund_code`, `timestamp`, `return_3m`, `return_5m`, `direction_3m`, `direction_5m` |

所有表必须保留 `release_time` 或 `available_time`。模型只能使用预测时点已经可见的数据。

## 6. 特征工程

### 6.1 指数基金核心特征

1. 跟踪指数特征
   - 指数 1/3/5/10/20/60 日收益率。
   - 指数波动率、最大回撤、成交额、换手率、估值分位。
   - 指数成分股涨跌贡献、权重集中度、行业权重变化。

2. 基金映射特征
   - 跟踪误差、历史 beta、alpha 残差。
   - ETF 溢折价、IOPV 偏离、申赎份额、流动性。
   - 同跟踪指数不同基金之间的价差和资金流分化。

3. 成分股和行业传导
   - 前 10/20/50 大权重股加权收益和加权成交冲击。
   - 涨停/跌停权重占比。
   - 行业指数相对强弱。
   - 北向持仓变化和成交净额。

### 6.2 期货、期权与商品特征

1. 股指期货
   - 主力合约收益、基差、年化基差、期限结构。
   - 成交量、持仓量、移仓换月状态。
   - IF/IH/IC/IM 相对强弱，映射大盘/小盘/成长风格。

2. 期权和波动率
   - 隐含波动率水平、波动率偏斜、期限结构。
   - put/call 成交量和持仓量。
   - 波动率分位数和跳升幅度。

3. 商品冲击
   - Brent、WTI、国内原油收益和波动。
   - 铜、铁矿、螺纹、煤炭、有色金属。
   - 黄金和美元指数的避险组合信号。
   - 商品对行业指数的映射：油价对能源/化工/航空，铜和黑色对周期/地产链。

### 6.3 宏观、汇率和跨市场特征

1. 利率流动性
   - DR007、SHIBOR、国债收益率曲线。
   - 信用利差、央行公开市场操作、逆回购到期。

2. 汇率
   - USD/CNY、USD/CNH、DXY。
   - CNH-CNY 价差，港元流动性指标。

3. 跨市场
   - 美股三大指数、VIX、纳指期货。
   - 日经、韩国、台湾、欧洲主要指数。
   - 恒生指数、恒生科技、A/H 溢价。
   - 美股隔夜收益对 A 股开盘、港股科技和 QDII 的影响。

### 6.4 舆情与恐慌特征

1. 新闻公告
   - 政策、监管、行业、公司、宏观、地缘风险主题。
   - 正负面情绪分数、情绪变化率、热度变化率。

2. 社媒恐慌
   - 负面评论占比、恐慌关键词密度、热帖扩散速度。
   - 与指数、行业、重仓股实体对齐。

3. 合成恐慌指数
   - A 股 `CN_Fear_Score`：期权 IV、put/call、跌停权重、北向撤出、融资余额下降、新闻负面强度。
   - 港股 `HK_Fear_Score`：VHSI、卖空比例、恒指期货基差、南向资金、美元/港元流动性、新闻负面强度。
   - 全球 `Global_Risk_Off`：VIX、美元指数、美债收益率、黄金、原油波动。

## 7. 标签与预测对象

### 7.1 长周期标签

```text
fund_return_1d = adjusted_nav[t+1] / adjusted_nav[t] - 1
fund_return_1w = adjusted_nav[t+5] / adjusted_nav[t] - 1
index_return_1d = tracking_index_close[t+1] / tracking_index_close[t] - 1
tracking_error_1d = fund_return_1d - index_return_1d * beta
```

模型可以同时预测：

- 基金收益。
- 跟踪指数收益。
- 跟踪误差。

对指数基金，建议把“预测指数收益 + 预测跟踪误差”作为主路径，比直接预测基金净值更稳。

### 7.2 日内标签

ETF/LOF：

```text
return_3m = trade_price[t+3m] / trade_price[t] - 1
return_5m = trade_price[t+5m] / trade_price[t] - 1
```

场外指数基金：

```text
proxy_return_3m = estimated_nav_or_index_proxy[t+3m] / estimated_nav_or_index_proxy[t] - 1
proxy_return_5m = estimated_nav_or_index_proxy[t+5m] / estimated_nav_or_index_proxy[t] - 1
```

场外基金日内预测必须标注 `proxy_intraday=true`。

## 8. 训练、验证与择优

### 8.1 数据切分

采用滚动 walk-forward，不使用普通随机切分。

| 数据段 | 作用 |
|---|---|
| 训练集 | 拟合模型参数 |
| 验证集 | 选择特征、调参、设置信号阈值 |
| 测试集 | 最终离线评估，只用一次 |
| 纸面交易集 | 上线前实时验证 |

分钟级样本必须使用 purged split 和 embargo，避免 3/5 分钟重叠标签导致分数虚高。

### 8.2 择优指标

长周期模型：

- MAE、RMSE、SMAPE。
- 方向准确率、Balanced Accuracy。
- IC、RankIC、分组收益。
- 高置信覆盖率和高置信准确率。
- 置信区间覆盖率。
- 分市场状态表现：牛市、熊市、震荡、高恐慌、低恐慌、高波动、低波动。

日内模型：

- 3/5 分钟方向准确率。
- 交易成本后收益。
- 换手率、最大回撤、盈亏比。
- 开盘、午盘、尾盘分时段效果。
- 高置信覆盖率。
- 推理延迟和数据延迟。

### 8.3 模型选择规则

1. 任何模型必须跑赢 naive baseline：
   - 长周期 baseline：跟踪指数动量/均值回归、昨日收益为预测。
   - 日内 baseline：短时动量、IOPV 偏离回归、期指基差。
2. 候选模型必须在验证集和测试集都稳定，不接受只在单一年份有效。
3. 模型胜出不只看准确率，还看校准、覆盖率、回撤和极端行情表现。
4. 若深度模型效果不稳定，允许 GBDT 或集成模型成为 Champion。
5. 每次加入新数据类别必须做消融实验，证明不是噪声。

## 9. 持续学习与自动微调

### 9.1 核心原则

模型不能只训练一次。指数基金所处的交易环境会持续变化，包括市场制度、交易拥挤度、政策方向、资金风格、期货基差、商品冲击、外资行为和舆情扩散方式。因此系统必须具备持续学习能力。

但持续学习不等于让线上生产模型在每次预测后自动改参数。金融场景中，未经验证的实时自学习容易把异常行情、脏数据、操纵性噪声或短期恐慌错误固化进模型。正确做法是：

```text
线上模型稳定服务
  -> 实时记录预测、特征、市场状态和实际结果
  -> 漂移检测和表现监控
  -> 触发滚动重训或增量微调
  -> 新模型进入 shadow/challenger
  -> 通过回测和纸面交易门槛
  -> 自动或人工审批晋升为 champion
```

### 9.2 持续学习分层方案

| 层级 | 更新方式 | 适用对象 | 风险控制 |
|---|---|---|---|
| 数据层 | 每分钟/每日持续入库 | 行情、期货、商品、资金流、恐慌、舆情、预测结果 | 校验缺失、异常值、延迟和时间戳 |
| 特征层 | 每分钟/每日刷新 | 日内特征、日频特征、恐慌因子、跨市场因子 | 保留 `available_time`，禁止未来函数 |
| 监控层 | 实时/每日监控漂移 | 特征分布、预测分布、误差、置信度、覆盖率 | 触发告警，不直接改线上模型 |
| 轻量更新层 | 每日/每周增量微调 | GBDT、线性校准器、概率校准器、部分深度模型 adapter | 只在验证通过后替换 |
| 完整重训层 | 每周/每月/事件触发 | TFT、PatchTST、iTransformer、GNN、DeepLOB、LOBTransformer | walk-forward 回测、shadow 验证 |
| 部署层 | champion/challenger 切换 | 长周期模型、日内模型、舆情模型 | 可回滚、可灰度、可审计 |

### 9.3 更新频率建议

| 模型/组件 | 常规更新 | 事件触发更新 |
|---|---|---|
| 日内 3/5 分钟模型 | 每周重训，日内只刷新特征和校准阈值 | 盘口结构突变、交易制度变化、期指基差关系失效 |
| 次日/本周模型 | 每周或每月滚动重训 | 政策 regime 切换、商品/汇率大幅波动、恐慌因子跳升 |
| 恐慌因子模型 | 每日更新统计参数，每周重估权重 | 突发政策、地缘风险、市场连续暴跌 |
| 舆情模型 | 每月微调，事件词典每日更新 | 新政策词、新行业热点、新平台文本风格 |
| 概率校准器 | 每日或每周更新 | 置信度与真实胜率明显脱钩 |

日内模型标签在 3/5 分钟后即可形成，但不要立即更新生产模型。建议盘后统一生成当日训练切片，完成质量检查后进入 challenger。长周期模型的标签有 T+1 或 T+5 延迟，应等待标签完整后再训练。

### 9.4 漂移检测与触发条件

需要同时监控数据漂移、概念漂移和表现漂移：

| 漂移类型 | 监控对象 | 触发示例 |
|---|---|---|
| 数据漂移 | 特征分布、缺失率、异常值、预测分布 | 期指基差分布偏离过去 60 日，恐慌因子进入极端分位 |
| 概念漂移 | 特征与标签关系 | 油价上涨过去利空某类指数，现在因政策或供需变化转为利好 |
| 表现漂移 | 预测误差、方向准确率、校准误差、覆盖率 | 高置信样本连续 5 个交易日低于门槛 |
| 市场状态漂移 | 牛熊、震荡、高波动、风险偏好 | VIX/VHSI/自建恐慌因子进入高波动 regime |

技术上可使用：

- 统计漂移：KS 检验、PSI、Jensen-Shannon、Wasserstein distance。
- 流式漂移：ADWIN、Page-Hinkley、KSWIN。
- 模型不确定性漂移：预测熵、分位区间宽度、集成模型分歧度。
- 金融专用漂移：IC 衰减、RankIC 衰减、收益分组单调性破坏、成本后收益转负。

触发后不直接替换模型，而是启动：

```text
漂移告警 -> 数据质量检查 -> 训练窗口重选 -> challenger 重训 -> shadow 运行 -> 晋升或回滚
```

### 9.5 增量微调策略

不同模型的微调方式不同：

1. GBDT/CatBoost/XGBoost/LightGBM
   - 适合每日或每周做增量训练、追加树或重新训练最近窗口。
   - 更推荐“滑动窗口全量重训 + 保留旧模型对比”，因为金融数据漂移明显，盲目追加树可能固化旧 regime。

2. TFT、PatchTST、iTransformer、TimesNet、TimeMixer
   - 使用最近 1-3 年数据滚动微调。
   - 大模型参数低学习率更新，避免过拟合最近几天噪声。
   - 可冻结底层表示层，只微调输出头或 adapter。
   - 保留 replay buffer，把历史极端行情样本混入训练，防止灾难性遗忘。

3. Temporal GNN
   - 图结构按指数成分股、行业、期货和商品关系定期更新。
   - 指数调仓、权重变化、行业关系变化后重新训练或更新图边权。

4. DeepLOB / LOBTransformer / TCN
   - 盘后用当日新增分钟样本训练 challenger。
   - 高频模型对微观结构变化敏感，必须监控买卖价差、成交深度和滑点。

5. 舆情/恐慌模型
   - 舆情模型按月或重大事件后微调。
   - 恐慌因子权重可每日重估，但必须设置平滑，避免单日噪声过度影响。

### 9.6 训练窗口管理

建议使用多窗口并行，而不是只用一个固定历史窗口：

| 窗口 | 用途 |
|---|---|
| 短窗口：最近 20-60 个交易日 | 捕捉最新资金风格、恐慌扩散、期货基差变化 |
| 中窗口：最近 1-2 年 | 训练当前市场主结构 |
| 长窗口：最近 5-8 年 | 保留牛熊、政策、商品周期和极端行情样本 |
| 事件窗口 | 专门保存暴跌、熔断、政策突变、商品暴涨暴跌、汇率急变样本 |

最终模型可以使用多窗口集成：

```text
short_window_model
medium_window_model
long_window_model
event_memory_model
regime_gate
  -> 根据当前市场状态动态加权
```

### 9.7 自动晋升与回滚

每次重训产生的新模型默认是 challenger，不直接成为 champion。晋升条件建议：

1. 在最近滚动测试集上跑赢当前 champion。
2. 在高恐慌、低恐慌、高波动、低波动、牛市、熊市、震荡等分组中没有明显短板。
3. 高置信覆盖率不能靠大幅减少信号数量来虚高准确率。
4. 交易成本后收益不能低于 champion。
5. 置信度校准不能恶化。
6. 连续 shadow 运行 5-20 个交易日达标。

如果新模型上线后出现以下情况，应自动回滚：

- 数据质量异常。
- 预测分布异常集中或异常发散。
- 高置信样本连续低于风控门槛。
- 推理延迟超过日内模型可接受范围。
- 模型输出与基准模型严重背离且无法解释。

### 9.8 系统闭环

持续学习闭环建议如下：

```text
1. 交易中：
   采集实时行情、期货、商品、资金流、舆情、恐慌因子。
   记录每次预测输入、输出、模型版本和置信度。

2. 标签形成：
   日内标签在 3/5 分钟后形成。
   次日标签在下一个交易日净值或指数收盘后形成。
   本周标签在 5 个交易日后形成。

3. 盘后作业：
   校验数据质量。
   计算真实误差、方向准确率、覆盖率、成本后收益。
   更新漂移报告。

4. 重训触发：
   固定周期重训。
   漂移触发重训。
   人工事件触发重训。

5. 模型评审：
   challenger 回测。
   shadow 运行。
   自动或人工审批。

6. 上线：
   更新 champion alias。
   保留旧模型 rollback alias。
   持续监控。
```

### 9.9 工具建议

| 能力 | 工具候选 | 说明 |
|---|---|---|
| 流式学习与漂移检测 | River | 支持在线学习和 ADWIN 等漂移检测，适合先做监控和轻量模型 |
| 数据/模型漂移报告 | Evidently | 适合生成特征漂移、预测漂移和数据质量报告 |
| 模型注册与版本 | MLflow Model Registry | 使用 `champion`、`challenger`、`shadow`、`rollback` alias 管理版本 |
| 任务编排 | Airflow、Prefect、Dagster | 盘后训练、漂移检测、报告生成、模型晋升 |
| 特征仓库 | Feast 或自建 Feature Store | 统一离线/在线特征，减少训练服务偏差 |
| 实验追踪 | MLflow、Weights & Biases | 记录模型、数据版本、指标、参数和特征列表 |

第一阶段可以先不搭完整 MLOps 平台，但必须实现三个最小能力：

1. 每次预测都记录特征、输出、模型版本和真实结果。
2. 每天生成漂移和表现报告。
3. 新模型必须先作为 challenger/shadow，通过门槛后再替换 champion。

## 10. 实施方案

### 阶段 1：指数基金样本和多资产数据底座

预计 2-3 周。

- 建立指数基金清单和跟踪指数映射。
- 采集基金净值、ETF 行情、IOPV、溢折价、申赎和规模。
- 采集跟踪指数、成分股、行业、期货、期权、商品、利率、汇率、跨市场指数。
- 构造 A 股、港股、全球风险三个恐慌因子。
- 生成 point-in-time 训练样本。

### 阶段 2：长周期模型锦标赛

预计 2-4 周。

- 训练 TFT、PatchTST/iTransformer、TimesNet/TimeMixer、Temporal GNN、CatBoost/XGBoost/LightGBM。
- 训练收益率回归、方向分类、置信区间三个任务。
- 做滚动回测和特征消融。
- 选择 Champion 或集成方案。

### 阶段 3：日内模型锦标赛

预计 3-5 周。

- 建立 ETF/LOF 分钟级样本。
- 训练 DeepLOB、LOBTransformer、TCN、短序列 Transformer、GBDT 基线。
- 加入期指基差、IOPV、溢折价、成分股权重冲击、恐慌突变。
- 做交易成本、滑点、延迟评估。

### 阶段 4：舆情与恐慌模型

预计 2-4 周，可与阶段 2/3 并行。

- 微调中文金融情绪模型。
- 抽取政策、监管、行业、地缘、流动性风险事件。
- 生成分钟级和日级情绪/恐慌特征。
- 通过消融验证舆情恐慌对长周期和日内模型的增益。

### 阶段 5：部署与纸面交易

预计至少 4 周。

- 模型服务与现有 Go API 对接。
- 每天生成预测、实际偏差、覆盖率、成本后收益报告。
- 纸面交易通过后再允许 `is_actionable=true`。
- 保留模型漂移监控和 Challenger 自动评估。

## 11. 主要风险与应对

| 风险 | 影响 | 应对 |
|---|---|---|
| 数据时间戳不准确 | 产生未来函数 | 所有字段保留 `available_time`，严格按预测时点截断 |
| 开源数据质量不稳定 | 训练噪声和线上断档 | 开发期可用开源源，生产期切授权行情和新闻源 |
| 跨市场时区错配 | 错用未来数据 | 建立交易日历和时区对齐规则 |
| 深度模型过拟合 | 回测好、实盘差 | walk-forward、purged split、正则化、Challenger 机制 |
| 舆情噪声大 | 伪相关 | 消融实验、源可靠性权重、人工抽检 |
| 场外基金无真实日内净值 | 标签不真实 | 只输出估值代理，ETF/LOF 才做真实日内价格预测 |
| 极端行情模型失效 | 大幅偏离 | 恐慌状态分层评估、风控阈值、no_signal 机制 |
| 自动微调污染线上模型 | 异常行情或脏数据被模型学习 | 新模型必须先进入 challenger/shadow，不允许无验证直接替换 champion |
| 灾难性遗忘 | 只适应近期行情，忘记极端行情 | replay buffer、事件窗口、多窗口集成 |

## 12. 第一阶段交付物

1. 指数基金清单与跟踪指数映射表。
2. 多资产数据字典和 `available_time` 规范。
3. A 股、港股、全球风险恐慌因子构造脚本。
4. 长周期训练样本：
   - `daily_weekly_index_fund_samples`
5. 日内训练样本：
   - `intraday_etf_3m_5m_samples`
6. 模型锦标赛配置：
   - `tft_daily_weekly`
   - `patchtst_daily_weekly`
   - `itransformer_daily_weekly`
   - `temporal_gnn_daily_weekly`
   - `deeplob_intraday`
   - `lobtransformer_intraday`
   - `tcn_intraday`
   - `gbdt_baselines`
7. 滚动回测报告和消融报告。
8. Champion/Challenger 选择报告。
9. 持续学习闭环：
   - 预测日志表
   - 标签回填任务
   - 每日漂移报告
   - challenger/shadow 评估任务
   - champion/rollback 版本切换记录

## 13. 参考模型、持续学习与数据来源

模型与框架：

- Microsoft Qlib：https://github.com/microsoft/qlib
- Temporal Fusion Transformer：https://arxiv.org/abs/1912.09363
- PyTorch Forecasting TFT 文档：https://pytorch-forecasting.readthedocs.io/en/latest/api/pytorch_forecasting.models.temporal_fusion_transformer.html
- PatchTST：https://github.com/yuqinie98/PatchTST
- iTransformer：https://github.com/thuml/iTransformer
- TimesNet / Time-Series-Library：https://github.com/thuml/Time-Series-Library
- TimeMixer：https://github.com/kwuking/TimeMixer
- DeepLOB：https://arxiv.org/abs/1808.03668
- LOB Transformer 示例：https://github.com/jwallbridge/translob
- FinRL：https://github.com/AI4Finance-Foundation/FinRL
- FinGPT：https://github.com/AI4Finance-Foundation/FinGPT
- 中文 FinBERT：https://github.com/valuesimplex/FinBERT
- MOMENT：https://github.com/moment-timeseries-foundation-model/moment
- Lag-Llama：https://github.com/time-series-foundation-models/lag-llama
- Chronos：https://github.com/amazon-science/chronos-forecasting

持续学习、漂移监控与模型注册：

- River：https://riverml.xyz/
- River ADWIN：https://riverml.xyz/0.21.0/api/drift/ADWIN/
- River 论文：https://arxiv.org/abs/2012.04740
- Evidently Data Drift：https://docs.evidentlyai.com/metrics/explainer_drift
- MLflow Model Registry：https://www.mlflow.org/docs/3.0.1/model-registry
- MLflow Model Registry Workflow：https://mlflow.org/docs/3.3.2/ml/model-registry/workflow/

数据：

- AkShare：https://github.com/akfamily/akshare
- Tushare Pro：https://tushare.pro/document/2
- BaoStock：http://baostock.com/baostock/index.php/Python_API%E6%96%87%E6%A1%A3
- 上交所：https://www.sse.com.cn/
- 深交所：https://www.szse.cn/
- 中金所：https://www.cffex.com.cn/
- 上期所：https://www.shfe.com.cn/
- 大商所：http://www.dce.com.cn/
- 郑商所：http://www.czce.com.cn/
- 港交所：https://www.hkex.com.hk/
- 恒生波幅指数 VHSI：https://www.hsi.com.hk/eng/indexes/all-indexes/volatilityindex
- GDELT：https://www.gdeltproject.org/
- FRED：https://fred.stlouisfed.org/

## 14. 评审结论

本项目应以指数基金为核心预测对象，采用“多资产数据 + 长短周期分离 + 恐慌因子入模 + 模型锦标赛择优”的方案。

推荐路线：

1. 长周期模型优先比较 TFT、PatchTST/iTransformer、TimesNet/TimeMixer、Temporal GNN 和 GBDT 集成。
2. 日内模型优先比较 DeepLOB、LOBTransformer、TCN、短序列 Transformer 和 GBDT 基线。
3. 恐慌因子必须作为训练特征进入两类模型，并通过消融实验验证贡献。
4. 指数基金预测主路径应先预测跟踪指数与跟踪误差，再映射到具体基金。
5. 模型必须具备持续学习闭环：固定周期重训、漂移触发重训、challenger/shadow 验证、champion/rollback 切换。
6. 现有工程只负责部署和接口，不再作为模型选择的约束。
