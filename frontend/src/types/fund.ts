/** Fund search result item. */
export interface FundItem {
  /** Fund code, six digits. */
  fund_code: string
  /** Fund display name. */
  fund_name: string
  /** Fund category, such as mixed, index, or stock fund. */
  fund_type: string
  /** Fund company. */
  company?: string
  /** Fund manager. */
  manager?: string
  /** Latest NAV. */
  latest_nav?: number
  /** Accumulated NAV. */
  cumulative_nav?: number
  /** Estimated NAV. */
  estimated_nav?: number
  /** Daily percentage change. */
  change_pct?: number
  /** One-month return. */
  return_1m?: number
  /** Three-month return. */
  return_3m?: number
  /** Six-month return. */
  return_6m?: number
  /** One-year return. */
  return_1y?: number
  /** Three-year return. */
  return_3y?: number
  /** Risk level. */
  risk_level?: string
  /** Fund inception date. */
  inception_date?: string
  /** Quote date or estimated quote time. */
  quote_date?: string
  /** Quote data source. */
  quote_source?: string
}

export interface FundSearchData {
  items: FundItem[]
  total: number
  page: number
  size: number
}

export interface FundFilters {
  types: string[]
  companies: string[]
  risk_levels: string[]
}
