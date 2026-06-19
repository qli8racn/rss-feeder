export interface Article {
  id: number
  feed_id: number
  feed_url: string
  url: string
  title: string
  content: string
  published_at: string
  read: boolean
  bookmarked: boolean
  fetched_at: string
  publisher: string
  thumbnail_url: string
  summary: string
  category: string
}

export interface ArticlesResponse {
  articles: Article[]
  total: number
  page: number
  per_page: number
}

export type SortField = 'title' | 'publisher' | 'category' | 'published_at'
export type SortOrder = 'asc' | 'desc'

export const SORT_FIELDS: readonly SortField[] = ['title', 'publisher', 'category', 'published_at']

export const PER_PAGE_OPTIONS = [25, 50, 100] as const
export type PerPage = (typeof PER_PAGE_OPTIONS)[number]
export const DEFAULT_PER_PAGE: PerPage = 25

export interface Feed {
  id: number
  feed_url: string
  title: string
  last_fetched: string | null
  created_at: string
}
