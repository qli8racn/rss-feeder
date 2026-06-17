import { describe, expect, it } from 'vitest'
import { buildFilterQuery, parseFilterState } from './filter'

describe('parseFilterState', () => {
  it('returns defaults when the query string is empty', () => {
    expect(parseFilterState('')).toEqual({
      keyword: '',
      category: '',
      sort: 'published_at',
      order: 'desc',
      bookmarkedOnly: false,
      page: 1,
      perPage: 25,
    })
  })

  it('reads valid values from the query string', () => {
    const state = parseFilterState(
      '?q=react&category=Tech&sort=title&order=asc&bookmarked=true&page=3&per_page=50',
    )
    expect(state).toEqual({
      keyword: 'react',
      category: 'Tech',
      sort: 'title',
      order: 'asc',
      bookmarkedOnly: true,
      page: 3,
      perPage: 50,
    })
  })

  it('falls back to defaults for invalid sort/order/page/per_page values', () => {
    const state = parseFilterState('?sort=bogus&order=bogus&page=-1&per_page=10')
    expect(state.sort).toBe('published_at')
    expect(state.order).toBe('desc')
    expect(state.page).toBe(1)
    expect(state.perPage).toBe(25)
  })

  it('treats any non-"true" bookmarked value as false', () => {
    expect(parseFilterState('?bookmarked=1').bookmarkedOnly).toBe(false)
    expect(parseFilterState('?bookmarked=false').bookmarkedOnly).toBe(false)
  })
})

describe('buildFilterQuery', () => {
  it('omits default values', () => {
    expect(
      buildFilterQuery({
        keyword: '',
        category: '',
        sort: 'published_at',
        order: 'desc',
        bookmarkedOnly: false,
        page: 1,
        perPage: 25,
      }),
    ).toBe('')
  })

  it('includes only non-default values', () => {
    const qs = buildFilterQuery({
      keyword: 'react',
      category: 'Tech',
      sort: 'title',
      order: 'asc',
      bookmarkedOnly: true,
      page: 2,
      perPage: 50,
    })
    const params = new URLSearchParams(qs)
    expect(params.get('q')).toBe('react')
    expect(params.get('category')).toBe('Tech')
    expect(params.get('sort')).toBe('title')
    expect(params.get('order')).toBe('asc')
    expect(params.get('bookmarked')).toBe('true')
    expect(params.get('page')).toBe('2')
    expect(params.get('per_page')).toBe('50')
  })

  it('round-trips through parseFilterState', () => {
    const original = {
      keyword: 'go',
      category: 'AI',
      sort: 'category' as const,
      order: 'asc' as const,
      bookmarkedOnly: true,
      page: 5,
      perPage: 100 as const,
    }
    expect(parseFilterState(`?${buildFilterQuery(original)}`)).toEqual(original)
  })
})
