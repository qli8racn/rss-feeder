import { memo, useEffect, useState } from 'react'
import { ChevronDownIcon, SearchIcon } from './icons'
import { PER_PAGE_OPTIONS } from '../types'
import type { PerPage } from '../types'

const SEARCH_DEBOUNCE_MS = 300

interface SearchFilterBarProps {
  initialKeyword: string
  onKeywordCommit: (value: string) => void
  category: string
  onCategoryChange: (value: string) => void
  categories: string[]
  perPage: PerPage
  onPerPageChange: (perPage: PerPage) => void
}

function SearchFilterBar({
  initialKeyword,
  onKeywordCommit,
  category,
  onCategoryChange,
  categories,
  perPage,
  onPerPageChange,
}: SearchFilterBarProps) {
  const [input, setInput] = useState(initialKeyword)

  // 入力中の毎キー入力でAPIを呼ばないよう、確定値は親へデバウンスして伝播する
  useEffect(() => {
    const handle = setTimeout(() => onKeywordCommit(input.trim()), SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(handle)
  }, [input, onKeywordCommit])

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
            value={perPage}
            onChange={(e) => onPerPageChange(Number(e.target.value) as PerPage)}
            className="appearance-none rounded border border-slate-400/10 bg-[#1e2733] py-2 pl-3 pr-8 font-mono text-[11px] text-slate-200 focus:outline-none"
          >
            {PER_PAGE_OPTIONS.map((n) => (
              <option key={n} value={n}>
                {n}件
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
