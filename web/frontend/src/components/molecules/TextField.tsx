import type { ChangeEvent, ReactNode } from 'react'

interface TextFieldProps {
  value: string
  onChange: (e: ChangeEvent<HTMLInputElement>) => void
  placeholder?: string
  disabled?: boolean
  icon?: ReactNode
  className?: string
}

function TextField({ value, onChange, placeholder, disabled, icon, className = '' }: TextFieldProps) {
  return (
    <div className={`relative ${className}`}>
      {icon && (
        <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2">{icon}</span>
      )}
      <input
        type="text"
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        disabled={disabled}
        className={`w-full rounded border border-slate-400/10 bg-surface-raised text-text-primary focus:outline-none disabled:opacity-50 ${
          icon
            ? 'py-2 pl-9 pr-4 text-body placeholder:text-text-secondary'
            : 'px-3 py-2 text-caption placeholder:text-text-secondary/60'
        }`}
      />
    </div>
  )
}

export default TextField
