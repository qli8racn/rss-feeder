import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import ArticleListTemplate from '../templates/ArticleListTemplate'
import { fetchArticles, fetchCategories, fetchLatestArticles, searchArticles, toggleBookmark } from '../../api'
import { parseFilterState } from '../../domain/filter'
import { syncFilterStateToURL } from '../../usecase/syncFilterStateToURL'
import type { Article, PerPage, SortField, SortOrder } from '../../domain/article'

function ArticleListPage() {
  const [initialFilters] = useState(() => parseFilterState(window.location.search))
  const [articles, setArticles] = useState<Article[]>([])
  const [total, setTotal] = useState(0)
  const [perPage, setPerPage] = useState<PerPage>(initialFilters.perPage)
  const [page, setPage] = useState(initialFilters.page)
  const [keyword, setKeyword] = useState(initialFilters.keyword)
  const [category, setCategory] = useState(initialFilters.category)
  const [sort, setSort] = useState<SortField>(initialFilters.sort)
  const [order, setOrder] = useState<SortOrder>(initialFilters.order)
  const [bookmarkedOnly, setBookmarkedOnly] = useState(initialFilters.bookmarkedOnly)
  const [bookmarkedTotal, setBookmarkedTotal] = useState(0)
  const [categories, setCategories] = useState<string[]>([])
  const [error, setError] = useState<string | null>(null)
  const [reloadToken, setReloadToken] = useState(0)
  const [fetching, setFetching] = useState(false)
  const [feedModalOpen, setFeedModalOpen] = useState(false)

  const refreshBookmarkedTotal = useCallback(() => {
    // バッジの件数だけが必要なので、記事本文を含むフルデータは取得しない
    fetchArticles({ mode: 'bookmarked', page: 1, perPage: 1 })
      .then((res) => setBookmarkedTotal(res.total))
      .catch(() => {
        // ヘッダーのバッジ表示のみに使うため、失敗してもエラー表示はしない
      })
  }, [])

  useEffect(() => {
    fetchCategories().then(setCategories).catch(() => setCategories([]))
    refreshBookmarkedTotal()
  }, [refreshBookmarkedTotal])

  // フィルター（キーワード・カテゴリ・ブックマーク絞り込み・件数）が変わった場合のみページを1に戻す。
  // ページリセットと記事取得を同一effect内で行うことで、古いページ番号での無駄なフェッチを避ける。
  const filtersRef = useRef({ keyword, category, bookmarkedOnly, perPage })

  useEffect(() => {
    const filtersChanged =
      filtersRef.current.keyword !== keyword ||
      filtersRef.current.category !== category ||
      filtersRef.current.bookmarkedOnly !== bookmarkedOnly ||
      filtersRef.current.perPage !== perPage
    filtersRef.current = { keyword, category, bookmarkedOnly, perPage }

    if (filtersChanged && page !== 1) {
      setPage(1)
      return
    }

    let cancelled = false
    setError(null)

    const load = keyword
      ? searchArticles({ q: keyword, bookmarked: bookmarkedOnly, category, sort, order, page, perPage })
      : fetchArticles({ mode: bookmarkedOnly ? 'bookmarked' : 'all', category, sort, order, page, perPage })

    load
      .then((res) => {
        if (cancelled) return
        setArticles(res.articles)
        setTotal(res.total)
      })
      .catch((err: Error) => {
        if (!cancelled) setError(err.message)
      })

    return () => {
      cancelled = true
    }
  }, [keyword, category, sort, order, page, bookmarkedOnly, perPage, reloadToken])

  useEffect(() => {
    syncFilterStateToURL({ keyword, category, sort, order, bookmarkedOnly, page, perPage })
  }, [keyword, category, sort, order, bookmarkedOnly, page, perPage])

  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / perPage)), [total, perPage])

  const handleSortChange = useCallback((s: SortField, o: SortOrder) => {
    setSort(s)
    setOrder(o)
  }, [])

  const handleToggleBookmarkedOnly = useCallback(() => setBookmarkedOnly((v) => !v), [])

  const handlePerPageChange = useCallback((value: PerPage) => setPerPage(value), [])

  const handleFetchLatest = useCallback(async () => {
    setFetching(true)
    setError(null)
    try {
      await fetchLatestArticles()
      setReloadToken((v) => v + 1)
      fetchCategories().then(setCategories).catch(() => {})
      refreshBookmarkedTotal()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setFetching(false)
    }
  }, [refreshBookmarkedTotal])

  const handleToggleBookmark = useCallback(
    async (id: number) => {
      try {
        const updated = await toggleBookmark(id)
        setArticles((prev) => prev.map((a) => (a.id === id ? updated : a)))
        refreshBookmarkedTotal()
      } catch (err) {
        setError((err as Error).message)
      }
    },
    [refreshBookmarkedTotal],
  )

  return (
    <ArticleListTemplate
      bookmarkedCount={bookmarkedTotal}
      bookmarkedOnly={bookmarkedOnly}
      onToggleBookmarkedOnly={handleToggleBookmarkedOnly}
      fetching={fetching}
      onFetchLatest={handleFetchLatest}
      onOpenFeedManagement={() => setFeedModalOpen(true)}
      feedModalOpen={feedModalOpen}
      onCloseFeedManagement={() => setFeedModalOpen(false)}
      initialKeyword={initialFilters.keyword}
      onKeywordCommit={setKeyword}
      category={category}
      onCategoryChange={setCategory}
      categories={categories}
      perPage={perPage}
      onPerPageChange={handlePerPageChange}
      total={total}
      page={page}
      totalPages={totalPages}
      onPageChange={setPage}
      error={error}
      articles={articles}
      sort={sort}
      order={order}
      onSortChange={handleSortChange}
      onToggleBookmark={handleToggleBookmark}
    />
  )
}

export default ArticleListPage
