import type { ChangeEvent } from 'react'
import { SearchIcon } from '../icons'

interface TextFieldProps {
  value: string
  onChange: (e: ChangeEvent<HTMLInputElement>) => void
  placeholder?: string
  disabled?: boolean
  hasIcon?: boolean
  className?: string
}

function TextField({ value, onChange, placeholder, disabled, hasIcon, className = '' }: TextFieldProps) {
  return (
    <div className={`relative ${className}`}>
      {hasIcon && (
        <SearchIcon className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-slate-500" />
      )}
      <input
        type="text"
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        disabled={disabled}
        className={`w-full rounded border border-slate-400/10 bg-surface-raised text-slate-200 focus:outline-none disabled:opacity-50 ${
          hasIcon
            ? 'py-2 pl-9 pr-4 text-body placeholder:text-slate-500'
            : 'px-3 py-2 text-caption placeholder:text-slate-500/60'
        }`}
      />
    </div>
  )
}

export default TextField
