export const API_ROUTES = {
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
  watchlist: {
    quotes: '/watchlist/quotes',
  },
} as const
