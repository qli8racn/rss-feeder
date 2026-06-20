import type { ChangeEvent, ReactNode } from 'react'
import { ChevronDownIcon } from '../icons'

interface SelectFieldProps {
  value: string | number
  onChange: (e: ChangeEvent<HTMLSelectElement>) => void
  children: ReactNode
}

function SelectField({ value, onChange, children }: SelectFieldProps) {
  return (
    <div className="relative">
      <select
        value={value}
        onChange={onChange}
        className="appearance-none rounded border border-slate-400/10 bg-surface-raised py-2 pl-3 pr-8 font-mono text-caption text-slate-200 focus:outline-none"
      >
        {children}
      </select>
      <ChevronDownIcon className="pointer-events-none absolute right-2 top-1/2 size-3 -translate-y-1/2 text-slate-500" />
    </div>
  )
}

export default SelectField
