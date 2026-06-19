/** @module market/utils/marketTime — 市场交易时间判断工具 */
/** 交易市场类型 */
export type ExchangeMarket = 'cn' | 'hk' | 'us'

interface Session {
  start: number
  end: number
}

const MARKET_TIME_ZONES: Record<ExchangeMarket, string> = {
  cn: 'Asia/Shanghai',
  hk: 'Asia/Hong_Kong',
  us: 'America/New_York',
}

const MARKET_SESSIONS: Record<ExchangeMarket, Session[]> = {
  cn: [
    { start: 9 * 60 + 30, end: 11 * 60 + 30 },
    { start: 13 * 60, end: 15 * 60 },
  ],
  hk: [
    { start: 9 * 60 + 30, end: 12 * 60 },
    { start: 13 * 60, end: 16 * 60 },
  ],
  us: [{ start: 9 * 60 + 30, end: 16 * 60 }],
}

/**
 * 根据指数代码判断所属市场
 * @param code - 指数代码
 * @returns 市场类型
 */
export function detectIndexMarket(code: string): ExchangeMarket {
  if (['hsi', 'hstech'].includes(code)) return 'hk'
  if (['dji', 'ixic', 'spx'].includes(code)) return 'us'
  return 'cn'
}

/**
 * 判断指定市场当前是否处于交易时段
 * @param market - 市场类型
 * @param now - 当前时间，默认为 new Date()
 * @returns 是否为交易时段
 */
export function isMarketInSession(market: ExchangeMarket, now: Date = new Date()): boolean {
  const formatter = new Intl.DateTimeFormat('en-US', {
    timeZone: MARKET_TIME_ZONES[market],
    weekday: 'short',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  })
  const parts = Object.fromEntries(
    formatter.formatToParts(now).map((part) => [part.type, part.value]),
  )
  if (parts.weekday === 'Sat' || parts.weekday === 'Sun') return false

  const minutes = Number(parts.hour) * 60 + Number(parts.minute)
  return MARKET_SESSIONS[market].some(
    (session) => minutes >= session.start && minutes <= session.end,
  )
}
