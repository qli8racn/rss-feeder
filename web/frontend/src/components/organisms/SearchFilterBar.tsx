import { memo, useEffect, useState } from 'react'
import { SearchIcon } from '../atoms/icons'
import SelectField from '../molecules/SelectField'
import TextField from '../molecules/TextField'
import { PER_PAGE_OPTIONS } from '../../types'
import type { PerPage } from '../../types'

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
      <TextField
        className="flex-1"
        icon={<SearchIcon className="size-3.5 text-text-secondary" />}
        value={input}
        onChange={(e) => setInput(e.target.value)}
        placeholder="タイトル・メディア・サマリーで検索..."
      />
      <div className="flex gap-2">
        <SelectField value={category} onChange={(e) => onCategoryChange(e.target.value)}>
          <option value="">すべてのカテゴリ</option>
          {categories.map((c) => (
            <option key={c} value={c}>
              {c}
            </option>
          ))}
        </SelectField>
        <SelectField value={perPage} onChange={(e) => onPerPageChange(Number(e.target.value) as PerPage)}>
          {PER_PAGE_OPTIONS.map((n) => (
            <option key={n} value={n}>
              {n}件
            </option>
          ))}
        </SelectField>
      </div>
    </form>
  )
}

export default memo(SearchFilterBar)
