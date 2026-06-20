import Header from '../organisms/Header'
import SearchFilterBar from '../organisms/SearchFilterBar'
import ArticleTable from '../organisms/ArticleTable'
import Pagination from '../organisms/Pagination'
import Footer from '../organisms/Footer'
import FeedManagementModal from '../organisms/FeedManagementModal'
import type { Article, PerPage, SortField, SortOrder } from '../../domain/article'

interface ArticleListTemplateProps {
  bookmarkedCount: number
  bookmarkedOnly: boolean
  onToggleBookmarkedOnly: () => void
  fetching: boolean
  onFetchLatest: () => void
  onOpenFeedManagement: () => void
  feedModalOpen: boolean
  onCloseFeedManagement: () => void
  initialKeyword: string
  onKeywordCommit: (value: string) => void
  category: string
  onCategoryChange: (value: string) => void
  categories: string[]
  perPage: PerPage
  onPerPageChange: (perPage: PerPage) => void
  total: number
  page: number
  totalPages: number
  onPageChange: (page: number) => void
  error: string | null
  articles: Article[]
  sort: SortField
  order: SortOrder
  onSortChange: (sort: SortField, order: SortOrder) => void
  onToggleBookmark: (id: number) => void
}

function ArticleListTemplate({
  bookmarkedCount,
  bookmarkedOnly,
  onToggleBookmarkedOnly,
  fetching,
  onFetchLatest,
  onOpenFeedManagement,
  feedModalOpen,
  onCloseFeedManagement,
  initialKeyword,
  onKeywordCommit,
  category,
  onCategoryChange,
  categories,
  perPage,
  onPerPageChange,
  total,
  page,
  totalPages,
  onPageChange,
  error,
  articles,
  sort,
  order,
  onSortChange,
  onToggleBookmark,
}: ArticleListTemplateProps) {
  return (
    <div className="min-h-screen bg-surface-base text-text-primary">
      <Header
        bookmarkedCount={bookmarkedCount}
        bookmarkedOnly={bookmarkedOnly}
        onToggleBookmarkedOnly={onToggleBookmarkedOnly}
        fetching={fetching}
        onFetchLatest={onFetchLatest}
        onOpenFeedManagement={onOpenFeedManagement}
      />
      <FeedManagementModal open={feedModalOpen} onClose={onCloseFeedManagement} />
      <main className="mx-auto max-w-6xl px-4 py-6 sm:px-6">
        <SearchFilterBar
          initialKeyword={initialKeyword}
          onKeywordCommit={onKeywordCommit}
          category={category}
          onCategoryChange={onCategoryChange}
          categories={categories}
          perPage={perPage}
          onPerPageChange={onPerPageChange}
        />
        <div className="mt-4 flex items-center justify-between text-small text-text-secondary">
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
              onSortChange={onSortChange}
              onToggleBookmark={onToggleBookmark}
            />
          )}
        </div>
        <Pagination page={page} totalPages={totalPages} onPageChange={onPageChange} />
        <Footer totalArticles={total} />
      </main>
    </div>
  )
}

export default ArticleListTemplate
