import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import SkeletonTable from '@/shared/components/SkeletonTable.vue'

describe('SkeletonTable', () => {
  it('renders with default 5 rows', () => {
    const wrapper = mount(SkeletonTable)
    const rows = wrapper.findAll('.skeleton-row')
    expect(rows.length).toBe(5)
  })

  it('renders correct number of rows when rowCount is specified', () => {
    const wrapper = mount(SkeletonTable, {
      props: { rowCount: 3 },
    })
    const rows = wrapper.findAll('.skeleton-row')
    expect(rows.length).toBe(3)
  })

  it('renders single row when rowCount is 1', () => {
    const wrapper = mount(SkeletonTable, {
      props: { rowCount: 1 },
    })
    const rows = wrapper.findAll('.skeleton-row')
    expect(rows.length).toBe(1)
  })

  it('renders 10 rows when rowCount is 10', () => {
    const wrapper = mount(SkeletonTable, {
      props: { rowCount: 10 },
    })
    const rows = wrapper.findAll('.skeleton-row')
    expect(rows.length).toBe(10)
  })

  it('each row contains exactly 4 skeleton cells', () => {
    const wrapper = mount(SkeletonTable, { props: { rowCount: 5 } })
    const rows = wrapper.findAll('.skeleton-row')

    rows.forEach((row) => {
      const cells = row.findAll('.sk-cell')
      expect(cells.length).toBe(4)
    })
  })

  it('each row has the correct cell types in order', () => {
    const wrapper = mount(SkeletonTable, { props: { rowCount: 2 } })
    const firstRow = wrapper.find('.skeleton-row')

    expect(firstRow.find('.sk-code').exists()).toBe(true)
    expect(firstRow.find('.sk-name').exists()).toBe(true)
    expect(firstRow.find('.sk-nav').exists()).toBe(true)
    expect(firstRow.find('.sk-pct').exists()).toBe(true)
  })

  it('all cells have skeleton-pulse class for animation', () => {
    const wrapper = mount(SkeletonTable, { props: { rowCount: 1 } })
    const pulseCells = wrapper.findAll('.skeleton-pulse')

    expect(pulseCells.length).toBe(4)
  })

  it('has root container with skeleton-strip class', () => {
    const wrapper = mount(SkeletonTable)
    expect(wrapper.find('.skeleton-strip').exists()).toBe(true)
  })

  it('last row has no bottom border (border-bottom removed)', () => {
    const wrapper = mount(SkeletonTable, { props: { rowCount: 5 } })
    const rows = wrapper.findAll('.skeleton-row')
    const lastRow = rows[rows.length - 1]

    expect(lastRow.classes()).not.toContain('with-border')
  })

  it('handles zero rowCount gracefully by rendering no rows', () => {
    const wrapper = mount(SkeletonTable, {
      props: { rowCount: 0 },
    })
    const rows = wrapper.findAll('.skeleton-row')
    expect(rows.length).toBe(0)
  })
})
