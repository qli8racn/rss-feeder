import { describe, expect, it } from 'vitest'
import { parseFilterState } from '../domain/filter'
import { syncFilterStateToURL } from './syncFilterStateToURL'

describe('syncFilterStateToURL', () => {
  it('clears the query string for default values', () => {
    syncFilterStateToURL({
      keyword: '',
      category: '',
      sort: 'published_at',
      order: 'desc',
      bookmarkedOnly: false,
      page: 1,
    })
    expect(window.location.search).toBe('')
  })

  it('writes non-default values to the URL', () => {
    syncFilterStateToURL({
      keyword: 'react',
      category: 'Tech',
      sort: 'title',
      order: 'asc',
      bookmarkedOnly: true,
      page: 2,
    })
    const params = new URLSearchParams(window.location.search)
    expect(params.get('q')).toBe('react')
    expect(params.get('page')).toBe('2')
  })

  it('round-trips through parseFilterState', () => {
    const original = {
      keyword: 'go',
      category: 'AI',
      sort: 'category' as const,
      order: 'asc' as const,
      bookmarkedOnly: true,
      page: 5,
    }
    syncFilterStateToURL(original)
    expect(parseFilterState(window.location.search)).toEqual(original)
  })
})
