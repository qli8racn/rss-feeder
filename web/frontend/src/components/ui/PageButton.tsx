import type { ReactNode } from 'react'

// Figma の PageButton コンポーネントが持つ `Variant` バリアントプロパティに合わせる
type PageButtonVariant = 'nav' | 'number'

interface PageButtonProps {
  variant: PageButtonVariant
  onClick: () => void
  children: ReactNode
  disabled?: boolean
  active?: boolean
  ariaLabel?: string
}

function PageButton({ variant, onClick, children, disabled, active, ariaLabel }: PageButtonProps) {
  if (variant === 'nav') {
    return (
      <button
        type="button"
        onClick={onClick}
        disabled={disabled}
        aria-label={ariaLabel}
        className="rounded border border-slate-400/10 p-2 disabled:opacity-30"
      >
        {children}
      </button>
    )
  }

  return (
    <button
      type="button"
      onClick={onClick}
      className={`size-8 rounded border font-mono text-caption ${
        active ? 'border-amber-500/60 bg-amber-500/10 text-amber-400' : 'border-slate-400/10 text-slate-500'
      }`}
    >
      {children}
    </button>
  )
}

export default PageButton
