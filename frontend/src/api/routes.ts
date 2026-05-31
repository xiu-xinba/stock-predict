export const API_ROUTES = {
  search: '/search',
  funds: {
    search: '/funds/search',
    filters: '/funds/filters',
  },
  market: {
    indices: '/market/indices',
    ranking: (type: 'gainers' | 'losers') => `/market/ranking/${type}`,
  },
  prediction: {
    fund: (fundCode: string) => `/predict/${fundCode}`,
  },
  fund: {
    detail: (fundCode: string) => `/fund/${fundCode}/detail`,
  },
  watchlist: {
    quotes: '/watchlist/quotes',
  },
  stock: {
    search: '/stocks/search',
    filters: '/stocks/filters',
    detail: (code: string) => `/stock/${code}/detail`,
    predict: (code: string) => `/stock/${code}/predict`,
    quotes: '/stocks/quotes',
    ranking: (type: string) => `/market/stock-ranking/${type}`,
    sync: '/stocks/sync',
  },
} as const
