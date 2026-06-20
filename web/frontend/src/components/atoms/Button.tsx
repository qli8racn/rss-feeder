import type { ReactNode } from 'react'

// Atom: 枠線・角丸・背景色の見た目だけを持つ最小単位。アイコン・ラベルは呼び出し側がchildrenで合成する
interface ButtonProps {
  type?: 'button' | 'submit'
  onClick?: () => void
  children: ReactNode
  disabled?: boolean
  active?: boolean
  ariaLabel?: string
}

function Button({ type = 'button', onClick, children, disabled, active, ariaLabel }: ButtonProps) {
  const borderColor = active ? 'border-amber-500/60 text-accent-default' : 'border-slate-400/10 text-text-secondary'

  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      aria-label={ariaLabel}
      aria-pressed={active}
      className={`flex items-center gap-1.5 whitespace-nowrap rounded border bg-surface-raised px-3 py-2 font-mono text-caption transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${borderColor}`}
    >
      {children}
    </button>
  )
}

export default Button
