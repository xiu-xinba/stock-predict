export interface WatchlistItem {
  fund_code: string
  fund_name: string
  fund_type: string
  estimated_nav: number
  change_pct: number
  direction: 'up' | 'down' | 'flat'
  added_at: number
  quote_date?: string
  quote_source?: string
}
