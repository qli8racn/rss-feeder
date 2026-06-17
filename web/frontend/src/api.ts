import type { Article, ArticlesResponse } from './types'

const PER_PAGE = 25

function buildQuery(params: Record<string, string | number | boolean | undefined>): string {
  const usp = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === '') continue
    usp.set(key, String(value))
  }
  const qs = usp.toString()
  return qs ? `?${qs}` : ''
}

export interface ListParams {
  mode?: 'all' | 'unread' | 'bookmarked'
  category?: string
  sort?: string
  order?: string
  page?: number
  perPage?: number
}

export async function fetchArticles(params: ListParams): Promise<ArticlesResponse> {
  const { perPage, ...rest } = params
  const qs = buildQuery({ ...rest, per_page: perPage ?? PER_PAGE })
  const res = await fetch(`/api/articles${qs}`)
  if (!res.ok) throw new Error(`記事の取得に失敗しました (${res.status})`)
  return res.json()
}

export interface SearchParams {
  q: string
  bookmarked?: boolean
  category?: string
  sort?: string
  order?: string
  page?: number
  perPage?: number
}

export async function searchArticles(params: SearchParams): Promise<ArticlesResponse> {
  const { perPage, ...rest } = params
  const qs = buildQuery({ ...rest, per_page: perPage ?? PER_PAGE })
  const res = await fetch(`/api/articles/search${qs}`)
  if (!res.ok) throw new Error(`検索に失敗しました (${res.status})`)
  return res.json()
}

export async function fetchCategories(): Promise<string[]> {
  const res = await fetch('/api/categories')
  if (!res.ok) throw new Error(`カテゴリの取得に失敗しました (${res.status})`)
  return res.json()
}

export async function toggleBookmark(id: number): Promise<Article> {
  const res = await fetch(`/api/articles/${id}/bookmark`, { method: 'POST' })
  if (!res.ok) throw new Error(`ブックマーク更新に失敗しました (${res.status})`)
  return res.json()
}

export interface FetchLatestResult {
  saved: number
  skipped: number
  errors: number
}

export async function fetchLatestArticles(): Promise<FetchLatestResult> {
  const res = await fetch('/api/articles/fetch', { method: 'POST' })
  if (!res.ok) throw new Error(`最新フィードの取得に失敗しました (${res.status})`)
  return res.json()
}
