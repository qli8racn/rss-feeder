import type { ReactNode } from 'react'

interface IconButtonProps {
  onClick: () => void
  ariaLabel: string
  className?: string
  children: ReactNode
}

function IconButton({ onClick, ariaLabel, className = '', children }: IconButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={ariaLabel}
      className={`rounded border border-slate-400/10 p-1.5 ${className}`}
    >
      {children}
    </button>
  )
}

export default IconButton
