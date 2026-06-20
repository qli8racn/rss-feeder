import { memo } from 'react'
import { ChevronLeftIcon, ChevronRightIcon } from '../atoms/icons'
import PageButton from '../molecules/PageButton'

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
      <PageButton variant="nav" disabled={page <= 1} onClick={() => onPageChange(page - 1)} ariaLabel="前のページ">
        <ChevronLeftIcon className="size-3.5 text-slate-400" />
      </PageButton>
      {getPageNumbers(page, totalPages).map((p) => (
        <PageButton key={p} variant="number" active={p === page} onClick={() => onPageChange(p)}>
          {p}
        </PageButton>
      ))}
      <PageButton
        variant="nav"
        disabled={page >= totalPages}
        onClick={() => onPageChange(page + 1)}
        ariaLabel="次のページ"
      >
        <ChevronRightIcon className="size-3.5 text-slate-400" />
      </PageButton>
    </div>
  )
}

export default memo(Pagination)
