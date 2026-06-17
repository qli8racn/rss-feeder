import { memo, useEffect, useState } from 'react'
import { ChevronDownIcon, SearchIcon } from './icons'
import type { SortField, SortOrder } from '../types'

const SEARCH_DEBOUNCE_MS = 300

const SORT_OPTIONS: { label: string; sort: SortField; order: SortOrder }[] = [
  { label: '新しい順', sort: 'published_at', order: 'desc' },
  { label: '古い順', sort: 'published_at', order: 'asc' },
  { label: 'タイトル (A→Z)', sort: 'title', order: 'asc' },
  { label: 'タイトル (Z→A)', sort: 'title', order: 'desc' },
]

interface SearchFilterBarProps {
  initialKeyword: string
  onKeywordCommit: (value: string) => void
  category: string
  onCategoryChange: (value: string) => void
  categories: string[]
  sort: SortField
  order: SortOrder
  onSortOrderChange: (sort: SortField, order: SortOrder) => void
}

function SearchFilterBar({
  initialKeyword,
  onKeywordCommit,
  category,
  onCategoryChange,
  categories,
  sort,
  order,
  onSortOrderChange,
}: SearchFilterBarProps) {
  const [input, setInput] = useState(initialKeyword)

  // 入力中の毎キー入力でAPIを呼ばないよう、確定値は親へデバウンスして伝播する
  useEffect(() => {
    const handle = setTimeout(() => onKeywordCommit(input.trim()), SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(handle)
  }, [input, onKeywordCommit])

  const currentSortIndex = Math.max(
    0,
    SORT_OPTIONS.findIndex((o) => o.sort === sort && o.order === order),
  )

  return (
    <form
      role="search"
      onSubmit={(e) => {
        e.preventDefault()
        onKeywordCommit(input.trim())
      }}
      className="flex flex-col gap-3 sm:flex-row sm:items-start"
    >
      <div className="relative flex-1">
        <SearchIcon className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-slate-500" />
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="タイトル・メディア・サマリーで検索..."
          className="w-full rounded border border-slate-400/10 bg-[#1e2733] py-2 pl-9 pr-4 text-[13px] text-slate-200 placeholder:text-slate-500 focus:outline-none"
        />
      </div>
      <div className="flex gap-2">
        <div className="relative">
          <select
            value={category}
            onChange={(e) => onCategoryChange(e.target.value)}
            className="appearance-none rounded border border-slate-400/10 bg-[#1e2733] py-2 pl-3 pr-8 font-mono text-[11px] text-slate-200 focus:outline-none"
          >
            <option value="">すべてのカテゴリ</option>
            {categories.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
          <ChevronDownIcon className="pointer-events-none absolute right-2 top-1/2 size-3 -translate-y-1/2 text-slate-500" />
        </div>
        <div className="relative">
          <select
            value={currentSortIndex}
            onChange={(e) => {
              const opt = SORT_OPTIONS[Number(e.target.value)]
              onSortOrderChange(opt.sort, opt.order)
            }}
            className="appearance-none rounded border border-slate-400/10 bg-[#1e2733] py-2 pl-3 pr-8 font-mono text-[11px] text-slate-200 focus:outline-none"
          >
            {SORT_OPTIONS.map((opt, i) => (
              <option key={opt.label} value={i}>
                {opt.label}
              </option>
            ))}
          </select>
          <ChevronDownIcon className="pointer-events-none absolute right-2 top-1/2 size-3 -translate-y-1/2 text-slate-500" />
        </div>
      </div>
    </form>
  )
}

export default memo(SearchFilterBar)
