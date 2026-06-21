import { memo } from 'react'
import type { Article, SortField, SortOrder } from '../../domain/article'
import { BookmarkIcon } from '../atoms/icons'
import { formatDate } from '../../domain/date'
import Table, { TABLE_HEADER_ROW_CLASS, TABLE_ROW_CLASS } from '../molecules/Table'
import CategoryBadge from '../molecules/CategoryBadge'

interface ArticleTableProps {
  articles: Article[]
  page: number
  perPage: number
  sort: SortField
  order: SortOrder
  onSortChange: (sort: SortField, order: SortOrder) => void
  onToggleBookmark: (id: number) => void
}

const COLUMNS: { field: SortField; label: string }[] = [
  { field: 'title', label: 'タイトル' },
  { field: 'publisher', label: 'メディア' },
  { field: 'category', label: 'カテゴリ' },
  { field: 'published_at', label: '日時' },
]

function ArticleTable({
  articles,
  page,
  perPage,
  sort,
  order,
  onSortChange,
  onToggleBookmark,
}: ArticleTableProps) {
  const handleHeaderClick = (field: SortField) => {
    if (field === sort) {
      onSortChange(field, order === 'asc' ? 'desc' : 'asc')
    } else {
      onSortChange(field, field === 'published_at' ? 'desc' : 'asc')
    }
  }

  if (articles.length === 0) {
    // 空状態: テーブルの形を模したスケルトン風プレースホルダー（docs/web-ui-spec.md 参照）
    return (
      <div className="rounded border border-slate-400/10 bg-surface-raised/30 py-24 text-center">
        <p className="font-mono text-caption uppercase tracking-widest text-text-secondary">
          該当する記事がありません
        </p>
      </div>
    )
  }

  return (
    <Table>
      <thead>
        <tr className={TABLE_HEADER_ROW_CLASS}>
          <th className="w-10 px-3 py-3 font-mono text-caption uppercase tracking-wider text-text-secondary">
            #
          </th>
          <th className="w-8 py-3" />
          {COLUMNS.map((col) => (
            <th
              key={col.field}
              className={col.field === 'title' ? 'py-3 text-left' : 'hidden w-32 py-3 text-left md:table-cell'}
            >
              <button
                type="button"
                onClick={() => handleHeaderClick(col.field)}
                className="flex items-center gap-1 font-mono text-caption uppercase tracking-wider text-text-secondary hover:text-slate-300"
              >
                {col.label}
                {sort === col.field && <span className="text-micro">{order === 'asc' ? '▲' : '▼'}</span>}
              </button>
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {articles.map((article, i) => {
          return (
            <tr key={article.id} className={TABLE_ROW_CLASS}>
              <td className="px-3 py-4 align-top font-mono text-caption text-text-secondary/50">
                {String((page - 1) * perPage + i + 1).padStart(2, '0')}
              </td>
              <td className="py-4 align-top">
                <button type="button" onClick={() => onToggleBookmark(article.id)} aria-label="ブックマーク切り替え">
                  <BookmarkIcon
                    filled={article.bookmarked}
                    className={`size-4 ${article.bookmarked ? 'text-accent-default' : 'text-text-secondary'}`}
                  />
                </button>
              </td>
              <td className="py-4 align-top">
                <a
                  href={article.url}
                  target="_blank"
                  rel="noreferrer"
                  className="text-body font-medium text-text-primary hover:underline"
                >
                  {article.title}
                </a>
                {article.summary && (
                  <p className="mt-1 line-clamp-1 text-caption text-text-secondary">{article.summary}</p>
                )}
                <div className="mt-1 flex flex-wrap gap-3 text-caption text-text-secondary md:hidden">
                  {article.publisher && <span>{article.publisher}</span>}
                  {article.category && <CategoryBadge category={article.category} className="px-1.5" />}
                  <span>{formatDate(article.published_at)}</span>
                </div>
              </td>
              <td className="hidden py-4 align-top text-caption text-text-secondary md:table-cell">
                {article.publisher}
              </td>
              <td className="hidden py-4 align-top md:table-cell">
                {article.category && <CategoryBadge category={article.category} className="px-2 py-1" />}
              </td>
              <td className="hidden py-4 align-top font-mono text-caption text-text-secondary md:table-cell">
                {formatDate(article.published_at)}
              </td>
            </tr>
          )
        })}
      </tbody>
    </Table>
  )
}

export default memo(ArticleTable)
