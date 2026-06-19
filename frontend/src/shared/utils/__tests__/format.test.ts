import { describe, it, expect } from 'vitest'
import {
  formatValue,
  formatVolume,
  formatSignedPct,
  getDirection,
  colorWithAlpha,
} from '@/shared/utils/format'

describe('formatValue', () => {
  it('returns "--" for NaN', () => {
    expect(formatValue(NaN)).toBe('--')
  })

  it('formats values below 10000 with 2 decimal places', () => {
    expect(formatValue(1234.567)).toBe('1234.57')
  })

  it('formats values >= 10000 with locale string', () => {
    expect(formatValue(12345)).toBe('12,345')
  })

  it('formats zero correctly', () => {
    expect(formatValue(0)).toBe('0.00')
  })
})

describe('formatVolume', () => {
  it('returns "--" for NaN', () => {
    expect(formatVolume(NaN)).toBe('--')
  })

  it('formats values >= 1e8 with 亿', () => {
    expect(formatVolume(2.5e8)).toBe('2.50亿')
  })

  it('formats values >= 1e4 with 万', () => {
    expect(formatVolume(50000)).toBe('5万')
  })

  it('formats small values with locale string', () => {
    expect(formatVolume(1234)).toBe('1,234')
  })
})

describe('formatSignedPct', () => {
  it('returns "--%" for null', () => {
    expect(formatSignedPct(null)).toBe('--%')
  })

  it('adds + sign for positive values', () => {
    expect(formatSignedPct(1.23)).toBe('+1.2300%')
  })

  it('keeps - sign for negative values', () => {
    expect(formatSignedPct(-0.5)).toBe('-0.5000%')
  })

  it('respects custom digits parameter', () => {
    expect(formatSignedPct(1.23, 2)).toBe('+1.23%')
  })
})

describe('getDirection', () => {
  it('returns "text-up" for positive values', () => {
    expect(getDirection(1)).toBe('text-up')
  })

  it('returns "text-down" for negative values', () => {
    expect(getDirection(-1)).toBe('text-down')
  })

  it('returns "text-flat" for zero', () => {
    expect(getDirection(0)).toBe('text-flat')
  })

  it('returns "text-flat" for null', () => {
    expect(getDirection(null)).toBe('text-flat')
  })
})

describe('colorWithAlpha', () => {
  it('converts hex color to rgba', () => {
    expect(colorWithAlpha('#ff0000', 0.5)).toBe('rgba(255, 0, 0, 0.5)')
  })

  it('converts rgb color to rgba', () => {
    expect(colorWithAlpha('rgb(255, 0, 0)', 0.5)).toBe('rgba(255, 0, 0, 0.5)')
  })

  it('returns color as-is for unrecognized format', () => {
    expect(colorWithAlpha('red', 0.5)).toBe('red')
  })
})
