import type { components } from '../api/schema.gen'

export type Article = components['schemas']['Article']
export type ArticlesResponse = components['schemas']['PagedArticles']
export type FetchLatestResult = components['schemas']['FetchResult']

export type Mode = components['schemas']['Mode']
export type SortField = components['schemas']['Sort']
export type SortOrder = components['schemas']['Order']

export const SORT_FIELDS: readonly SortField[] = ['title', 'publisher', 'category', 'published_at']

export const PER_PAGE_OPTIONS = [25, 50, 100] as const
export type PerPage = (typeof PER_PAGE_OPTIONS)[number]
export const DEFAULT_PER_PAGE: PerPage = 25
