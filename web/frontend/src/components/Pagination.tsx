import { memo } from 'react'
import { ChevronLeftIcon, ChevronRightIcon } from './icons'

interface PaginationProps {
  page: number
  totalPages: number
  onPageChange: (page: number) => void
}

function getPageNumbers(current: number, total: number, maxVisible = 7): number[] {
  if (total <= maxVisible) return Array.from({ length: total }, (_, i) => i + 1)
  let start = Math.max(1, current - Math.floor(maxVisible / 2))
  let end = start + maxVisible - 1
  if (end > total) {
    end = total
    start = end - maxVisible + 1
  }
  return Array.from({ length: end - start + 1 }, (_, i) => start + i)
}

function Pagination({ page, totalPages, onPageChange }: PaginationProps) {
  if (totalPages <= 1) return null

  return (
    <div className="flex items-center justify-center gap-2 pt-6">
      <button
        type="button"
        disabled={page <= 1}
        onClick={() => onPageChange(page - 1)}
        aria-label="前のページ"
        className="rounded border border-slate-400/10 p-2 disabled:opacity-30"
      >
        <ChevronLeftIcon className="size-3.5 text-slate-400" />
      </button>
      {getPageNumbers(page, totalPages).map((p) => (
        <button
          key={p}
          type="button"
          onClick={() => onPageChange(p)}
          className={`size-8 rounded border font-mono text-caption ${
            p === page
              ? 'border-amber-500/60 bg-amber-500/10 text-amber-400'
              : 'border-slate-400/10 text-slate-500'
          }`}
        >
          {p}
        </button>
      ))}
      <button
        type="button"
        disabled={page >= totalPages}
        onClick={() => onPageChange(page + 1)}
        aria-label="次のページ"
        className="rounded border border-slate-400/10 p-2 disabled:opacity-30"
      >
        <ChevronRightIcon className="size-3.5 text-slate-400" />
      </button>
    </div>
  )
}

export default memo(Pagination)
