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

/** 预测结果 — 核心业务数据 */
export interface PredictionResult {
  /** 预测方向：'up' 上涨 / 'down' 下跌 / 'flat' 平盘 */
  direction: 'up' | 'down' | 'flat'
  /** 方向置信度 0-1，越接近 1 表示模型越确信 */
  direction_confidence: number
  /** 预测涨跌幅（%），正数上涨负数下跌 */
  predicted_change_pct: number
  /** 涨跌幅预测区间 */
  change_range: ChangeRange
  /** 关键预测因子列表（按重要性降序，最多 5 个） */
  top_factors: FactorItem[]
  /** 预测可靠性级别: 'model' | 'model_no_features' | 'mock' */
  reliability?: string
  /** 可靠性说明 */
  reliability_note?: string
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
