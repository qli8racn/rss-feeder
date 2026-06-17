import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import Header from './components/Header'
import SearchFilterBar from './components/SearchFilterBar'
import ArticleTable from './components/ArticleTable'
import Pagination from './components/Pagination'
import Footer from './components/Footer'
import { fetchArticles, fetchCategories, searchArticles, toggleBookmark } from './api'
import type { Article, SortField, SortOrder } from './types'

function App() {
  const [articles, setArticles] = useState<Article[]>([])
  const [total, setTotal] = useState(0)
  const [perPage, setPerPage] = useState(25)
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [category, setCategory] = useState('')
  const [sort, setSort] = useState<SortField>('published_at')
  const [order, setOrder] = useState<SortOrder>('desc')
  const [bookmarkedOnly, setBookmarkedOnly] = useState(false)
  const [bookmarkedTotal, setBookmarkedTotal] = useState(0)
  const [categories, setCategories] = useState<string[]>([])
  const [error, setError] = useState<string | null>(null)

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

  // フィルター（キーワード・カテゴリ・ブックマーク絞り込み）が変わった場合のみページを1に戻す。
  // ページリセットと記事取得を同一effect内で行うことで、古いページ番号での無駄なフェッチを避ける。
  const filtersRef = useRef({ keyword, category, bookmarkedOnly })

  useEffect(() => {
    const filtersChanged =
      filtersRef.current.keyword !== keyword ||
      filtersRef.current.category !== category ||
      filtersRef.current.bookmarkedOnly !== bookmarkedOnly
    filtersRef.current = { keyword, category, bookmarkedOnly }

    if (filtersChanged && page !== 1) {
      setPage(1)
      return
    }

    let cancelled = false
    setError(null)

    const load = keyword
      ? searchArticles({ q: keyword, bookmarked: bookmarkedOnly, category, sort, order, page })
      : fetchArticles({ mode: bookmarkedOnly ? 'bookmarked' : 'all', category, sort, order, page })

    load
      .then((res) => {
        if (cancelled) return
        setArticles(res.articles)
        setTotal(res.total)
        setPerPage(res.per_page)
      })
      .catch((err: Error) => {
        if (!cancelled) setError(err.message)
      })

    return () => {
      cancelled = true
    }
  }, [keyword, category, sort, order, page, bookmarkedOnly])

  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / perPage)), [total, perPage])

  const handleToggleBookmark = async (id: number) => {
    try {
      const updated = await toggleBookmark(id)
      setArticles((prev) => prev.map((a) => (a.id === id ? updated : a)))
      refreshBookmarkedTotal()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <div className="min-h-screen bg-[#0d1117] text-slate-200">
      <Header
        bookmarkedCount={bookmarkedTotal}
        bookmarkedOnly={bookmarkedOnly}
        onToggleBookmarkedOnly={() => setBookmarkedOnly((v) => !v)}
      />
      <main className="mx-auto max-w-6xl px-4 py-6 sm:px-6">
        <SearchFilterBar
          onKeywordCommit={setKeyword}
          category={category}
          onCategoryChange={setCategory}
          categories={categories}
          sort={sort}
          order={order}
          onSortOrderChange={(s, o) => {
            setSort(s)
            setOrder(o)
          }}
        />
        <div className="mt-4 flex items-center justify-between text-[12px] text-slate-500">
          <span>{total} 件</span>
          <span>
            {page} / {totalPages} ページ
          </span>
        </div>
        <div className="mt-3">
          {error ? (
            <p className="py-12 text-center text-sm text-rose-400">{error}</p>
          ) : (
            <ArticleTable
              articles={articles}
              page={page}
              perPage={perPage}
              sort={sort}
              order={order}
              onSortChange={(s, o) => {
                setSort(s)
                setOrder(o)
              }}
              onToggleBookmark={handleToggleBookmark}
            />
          )}
        </div>
        <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
        <Footer totalArticles={total} />
      </main>
    </div>
  )
}

export default App
