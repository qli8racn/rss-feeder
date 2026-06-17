import { memo } from 'react'
import type { Article, SortField, SortOrder } from '../types'
import { BookmarkIcon } from './icons'
import { formatDate } from '../domain/date'
import { categoryStyle } from '../domain/category'

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
      <div className="rounded border border-slate-400/10 bg-[rgba(30,39,51,0.3)] py-24 text-center">
        <p className="font-mono text-[11px] uppercase tracking-widest text-slate-500">
          該当する記事がありません
        </p>
      </div>
    )
  }

  return (
    <div className="overflow-hidden rounded border border-slate-400/10">
      <table className="w-full table-fixed text-left">
        <thead>
          <tr className="border-b border-slate-400/10 bg-[rgba(30,39,51,0.6)]">
            <th className="w-10 px-3 py-3 font-mono text-[11px] uppercase tracking-wider text-slate-500">
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
                  className="flex items-center gap-1 font-mono text-[11px] uppercase tracking-wider text-slate-500 hover:text-slate-300"
                >
                  {col.label}
                  {sort === col.field && <span className="text-[10px]">{order === 'asc' ? '▲' : '▼'}</span>}
                </button>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {articles.map((article, i) => {
            const style = categoryStyle(article.category)
            return (
              <tr key={article.id} className="border-b border-slate-400/10 last:border-b-0">
                <td className="px-3 py-4 align-top font-mono text-[11px] text-slate-500/50">
                  {String((page - 1) * perPage + i + 1).padStart(2, '0')}
                </td>
                <td className="py-4 align-top">
                  <button type="button" onClick={() => onToggleBookmark(article.id)} aria-label="ブックマーク切り替え">
                    <BookmarkIcon
                      filled={article.bookmarked}
                      className={`size-4 ${article.bookmarked ? 'text-amber-400' : 'text-slate-500'}`}
                    />
                  </button>
                </td>
                <td className="py-4 align-top">
                  <a
                    href={article.url}
                    target="_blank"
                    rel="noreferrer"
                    className="text-[13px] font-medium text-slate-200 hover:underline"
                  >
                    {article.title}
                  </a>
                  {article.summary && (
                    <p className="mt-1 line-clamp-1 text-[11px] text-slate-500">{article.summary}</p>
                  )}
                  <div className="mt-1 flex flex-wrap gap-3 text-[11px] text-slate-500 md:hidden">
                    {article.publisher && <span>{article.publisher}</span>}
                    {article.category && (
                      <span className="rounded px-1.5" style={{ color: style.text, backgroundColor: style.bg }}>
                        {article.category}
                      </span>
                    )}
                    <span>{formatDate(article.published_at)}</span>
                  </div>
                </td>
                <td className="hidden py-4 align-top text-[11px] text-slate-500 md:table-cell">
                  {article.publisher}
                </td>
                <td className="hidden py-4 align-top md:table-cell">
                  {article.category && (
                    <span
                      className="rounded px-2 py-1 text-[11px]"
                      style={{ color: style.text, backgroundColor: style.bg }}
                    >
                      {article.category}
                    </span>
                  )}
                </td>
                <td className="hidden py-4 align-top font-mono text-[11px] text-slate-500 md:table-cell">
                  {formatDate(article.published_at)}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

export default memo(ArticleTable)
