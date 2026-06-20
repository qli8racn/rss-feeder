import type { ReactNode } from 'react'
import { BookmarkIcon, ListIcon, PlusIcon, RefreshIcon } from '../icons'

// Figma の Button コンポーネントが持つ `Icon` / `Variant` バリアントプロパティに合わせる
type ButtonIcon = 'refresh' | 'list' | 'plus' | 'bookmark'
type ButtonVariant = 'outline' | 'filled'

interface ButtonProps {
  icon: ButtonIcon
  variant?: ButtonVariant
  type?: 'button' | 'submit'
  onClick?: () => void
  children: ReactNode
  disabled?: boolean
  active?: boolean
  spinning?: boolean
  badge?: number
}

function Button({
  icon,
  variant = 'outline',
  type = 'button',
  onClick,
  children,
  disabled,
  active,
  spinning,
  badge,
}: ButtonProps) {
  const borderColor = active ? 'border-amber-500/60 text-amber-400' : 'border-slate-400/10 text-slate-500'
  const sizing = variant === 'filled' ? 'whitespace-nowrap px-3 py-2' : 'px-3 py-1.5'
  const background = variant === 'filled' ? 'bg-surface-raised' : ''

  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      aria-pressed={icon === 'bookmark' ? active : undefined}
      className={`flex items-center gap-1.5 rounded border font-mono text-caption transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${borderColor} ${sizing} ${background}`}
    >
      {icon === 'bookmark' ? (
        <BookmarkIcon filled={active} className="size-3" />
      ) : icon === 'list' ? (
        <ListIcon className="size-3" />
      ) : icon === 'plus' ? (
        <PlusIcon className="size-3" />
      ) : (
        <RefreshIcon className={`size-3 ${spinning ? 'animate-spin' : ''}`} />
      )}
      <span>{children}</span>
      {badge !== undefined && (
        <span className="rounded bg-surface-raised px-1.5 py-0.5 text-micro font-bold text-slate-500">{badge}</span>
      )}
    </button>
  )
}

export default Button
