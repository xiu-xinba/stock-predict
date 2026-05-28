/**
 * TypeScript 类型定义 — 前后端数据契约
 *
 * 本文件定义了与后端 API 响应结构一一对应的 TypeScript 接口，
 * 确保前端对 API 返回数据的访问具有类型安全和自动补全。
 *
 * 命名约定：字段名使用 snake_case 与后端 JSON 保持一致，
 * 避免额外的 camelCase 转换层，降低维护成本。
 */

/** 预测因子项 — 模型中贡献度最高的特征 */
export interface FactorItem {
  /** 因子英文标识，如 momentum_5d、return_10d */
  name: string
  /** 因子重要性归一化值 0-1，所有因子之和为 1 */
  importance: number
  /** 因子中文描述，面向用户展示 */
  description: string
}

/** 涨跌幅预测区间 */
export interface ChangeRange {
  /** 区间下限（%） */
  low: number
  /** 区间上限（%） */
  high: number
}

/** 经验残差预测区间元数据 */
export interface PredictionInterval {
  low: number
  high: number
  method?: string
  level?: number | null
  empirical_coverage?: number | null
}

export interface ReturnDecomposition {
  enabled: boolean
  method: string
  formula: string
  index_return_pct: number | null
  tracking_error_pct: number | null
  direct_fund_return_pct: number | null
  index_return_target?: string
  tracking_error_target?: string
}

export interface ActionabilityGate {
  actionable: boolean
  reason?: string
  min_high_confidence_accuracy?: number | null
  min_high_confidence_coverage?: number | null
  high_confidence_accuracy?: number | null
  high_confidence_coverage?: number | null
  max_calibration_ece?: number | null
  calibration_ece?: number | null
}

/** 预测可靠性级别 */
export type PredictionReliability =
  | 'model'
  | 'model_service'
  | 'model_mvp'
  | 'weekly_model_mvp'
  | 'intraday_model_mvp'
  | 'model_no_features'
  | 'baseline'
  | 'baseline_no_realtime'
  | 'mock'

export type PredictionSignalStatus = 'actionable' | 'low_confidence' | 'no_signal'

export type PredictionModelSource = 'python_model_service' | 'go_baseline'

export type PredictionModelCoverageStatus =
  | 'model_supported'
  | 'baseline_only'
  | 'unsupported_fund'
  | 'model_unavailable'

/** 预测结果 — 核心业务数据 */
export interface PredictionResult {
  /** 预测周期，如 next_day / intraday_5m */
  horizon: string
  /** 预测窗口文案 */
  target_window: string
  /** 预测来源：Python 模型服务或 Go 基线 */
  model_source: PredictionModelSource
  /** 模型候选器名称，如 extra_trees */
  model_candidate?: string
  /** 特征集标识 */
  feature_set?: string
  /** 模型输入样本时间 */
  model_asof_time?: string
  /** 模型覆盖状态 */
  model_coverage_status: PredictionModelCoverageStatus
  /** 模型覆盖状态说明 */
  model_coverage_note?: string
  /** 预测方向：'up' 上涨 / 'down' 下跌 / 'flat' 平盘 */
  direction: 'up' | 'down' | 'flat'
  /** 方向置信度 0-1，越接近 1 表示模型越确信 */
  direction_confidence: number
  /** 预测涨跌幅（%），正数上涨负数下跌 */
  predicted_change_pct: number
  /** 涨跌幅预测区间 */
  change_range: ChangeRange
  /** 预测区间校准信息 */
  prediction_interval?: PredictionInterval | null
  /** 指数基金收益拆解 */
  return_decomposition?: ReturnDecomposition | null
  /** 可行动信号质量闸门 */
  actionability_gate?: ActionabilityGate | null
  /** 关键预测因子列表（按重要性降序，最多 5 个） */
  top_factors: FactorItem[]
  /** 预测可靠性级别 */
  reliability: PredictionReliability
  /** 可靠性说明 */
  reliability_note: string
  /** 目标准确率阈值 */
  accuracy_target: number
  /** 是否已通过目标准确率验证 */
  meets_accuracy_target: boolean
  /** 设计信号状态：可行动、低置信或无信号 */
  signal_status: PredictionSignalStatus
  /** 是否可作为高置信动作信号 */
  is_actionable: boolean
  /** 校准与回测说明 */
  calibration_note: string
}

/** 预测数据覆盖情况 */
export interface PredictionDataQuality {
  has_realtime_quote: boolean
  has_market_indices: boolean
  has_holdings_data: boolean
  has_intraday_constituent_data: boolean
  has_etf_flow_data: boolean
  coverage_score: number
  missing_sources: string[]
  note: string
}

/** 市场快照 — 三大 A 股指数实时数据 */
export interface MarketSnapshot {
  /** 上证综合指数点位 */
  sh_index: number
  /** 上证指数涨跌幅（%） */
  sh_index_change_pct: number
  /** 深证成份指数点位 */
  sz_index: number
  /** 深证指数涨跌幅（%） */
  sz_index_change_pct: number
  /** 创业板指数点位 */
  cyb_index: number
  /** 创业板指数涨跌幅（%） */
  cyb_index_change_pct: number
  /** 数据更新时间（ISO 格式字符串） */
  update_time: string
}

/** 预测接口响应的 data 字段 */
export interface PredictionData {
  /** 基金代码（6 位数字） */
  fund_code: string
  /** 基金名称 */
  fund_name: string
  /** 预测结果 */
  prediction: PredictionResult
  /** 隔日预测 */
  next_day_prediction: PredictionResult
  /** 未来一周预测 */
  weekly_prediction: PredictionResult
  /** 盘中未来五分钟预测 */
  intraday_prediction: PredictionResult
  /** 数据覆盖情况 */
  data_quality: PredictionDataQuality
  /** 市场快照 */
  market_snapshot: MarketSnapshot
}

/** 后端统一响应格式 — 所有 API 均使用此包装 */
export interface ApiResponse<T> {
  /** 业务状态码，0 表示成功 */
  code: number
  /** 提示信息，失败时包含错误描述 */
  message: string
  /** 响应数据，失败时为 null */
  data: T | null
}

/** 基金搜索结果项（扩展版） */
export interface FundItem {
  /** 基金代码（6 位数字） */
  fund_code: string
  /** 基金名称 */
  fund_name: string
  /** 基金类型，如"混合型"、"指数型"、"股票型" */
  fund_type: string
  /** 基金公司 */
  company?: string
  /** 基金经理 */
  manager?: string
  /** 最新净值 */
  latest_nav?: number
  /** 累计净值 */
  cumulative_nav?: number
  /** 近1月收益率(%) */
  return_1m?: number
  /** 近3月收益率(%) */
  return_3m?: number
  /** 近6月收益率(%) */
  return_6m?: number
  /** 近1年收益率(%) */
  return_1y?: number
  /** 近3年收益率(%) */
  return_3y?: number
  /** 风险等级 */
  risk_level?: string
  /** 成立日期 */
  inception_date?: string
  /** 行情日期或估值时间 */
  quote_date?: string
  /** 行情来源 */
  quote_source?: string
}

/** 基金搜索接口响应的 data 字段 */
export interface FundSearchData {
  /** 搜索结果列表 */
  items: FundItem[]
  /** 总匹配数 */
  total: number
  /** 当前页码 */
  page: number
  /** 每页条数 */
  size: number
}

/** 基金筛选选项 */
export interface FundFilters {
  /** 基金类型列表 */
  types: string[]
  /** 基金公司列表 */
  companies: string[]
  /** 风险等级列表 */
  risk_levels: string[]
}

/** 基金搜索完整响应类型 */
export type FundSearchResponse = ApiResponse<FundSearchData>

/** 基金筛选选项完整响应类型 */
export type FundFiltersResponse = ApiResponse<FundFilters>

/** 预测完整响应类型 */
export type PredictResponse = ApiResponse<PredictionData>
