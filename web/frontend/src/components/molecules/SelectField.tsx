import type { ChangeEvent, ReactNode } from 'react'
import { ChevronDownIcon } from '../atoms/icons'

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
        className="appearance-none rounded border border-slate-400/10 bg-surface-raised py-2 pl-3 pr-8 font-mono text-caption text-text-primary focus:outline-none"
      >
        {children}
      </select>
      <ChevronDownIcon className="pointer-events-none absolute right-2 top-1/2 size-3 -translate-y-1/2 text-text-secondary" />
    </div>
  )
}

export default SelectField
