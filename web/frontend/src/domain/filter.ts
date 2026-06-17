import { DEFAULT_PER_PAGE, PER_PAGE_OPTIONS, SORT_FIELDS } from '../types'
import type { PerPage, SortField, SortOrder } from '../types'

export interface FilterState {
  keyword: string
  category: string
  sort: SortField
  order: SortOrder
  bookmarkedOnly: boolean
  page: number
  perPage: PerPage
}

const DEFAULT_SORT: SortField = 'published_at'
const DEFAULT_ORDER: SortOrder = 'desc'

function isSortField(value: string | null): value is SortField {
  return value !== null && (SORT_FIELDS as readonly string[]).includes(value)
}

function isSortOrder(value: string | null): value is SortOrder {
  return value === 'asc' || value === 'desc'
}

function isPerPage(value: number): value is PerPage {
  return (PER_PAGE_OPTIONS as readonly number[]).includes(value)
}

export function parseFilterState(search: string): FilterState {
  const params = new URLSearchParams(search)
  const page = Number(params.get('page'))
  const sort = params.get('sort')
  const order = params.get('order')
  const perPage = Number(params.get('per_page'))

  return {
    keyword: params.get('q') ?? '',
    category: params.get('category') ?? '',
    sort: isSortField(sort) ? sort : DEFAULT_SORT,
    order: isSortOrder(order) ? order : DEFAULT_ORDER,
    bookmarkedOnly: params.get('bookmarked') === 'true',
    page: Number.isInteger(page) && page > 0 ? page : 1,
    perPage: isPerPage(perPage) ? perPage : DEFAULT_PER_PAGE,
  }
}

// デフォルト値はクエリに残さず短く保つ
export function buildFilterQuery(state: FilterState): string {
  const params = new URLSearchParams()
  if (state.keyword) params.set('q', state.keyword)
  if (state.category) params.set('category', state.category)
  if (state.sort !== DEFAULT_SORT) params.set('sort', state.sort)
  if (state.order !== DEFAULT_ORDER) params.set('order', state.order)
  if (state.bookmarkedOnly) params.set('bookmarked', 'true')
  if (state.page !== 1) params.set('page', String(state.page))
  if (state.perPage !== DEFAULT_PER_PAGE) params.set('per_page', String(state.perPage))
  return params.toString()
}
