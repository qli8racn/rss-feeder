import type { ReactNode } from 'react'

export const TABLE_HEADER_ROW_CLASS = 'border-b border-slate-400/10 bg-surface-raised/60'
export const TABLE_ROW_CLASS = 'border-b border-slate-400/10 last:border-b-0'

interface TableProps {
  children: ReactNode
}

function Table({ children }: TableProps) {
  return (
    <div className="overflow-hidden rounded border border-slate-400/10">
      <table className="w-full table-fixed text-left">{children}</table>
    </div>
  )
}

export default Table
