import { CloseIcon, TrashIcon } from '../icons'

// Figma の IconButton コンポーネントセットが持つ `Icon` バリアントプロパティ（Close / Trash）に合わせる
type IconButtonIcon = 'close' | 'trash'

interface IconButtonProps {
  icon: IconButtonIcon
  onClick: () => void
  ariaLabel: string
  className?: string
}

const ICONS: Record<IconButtonIcon, typeof CloseIcon> = {
  close: CloseIcon,
  trash: TrashIcon,
}

function IconButton({ icon, onClick, ariaLabel, className = '' }: IconButtonProps) {
  const Icon = ICONS[icon]
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={ariaLabel}
      className={`rounded border border-slate-400/10 p-1.5 ${className}`}
    >
      <Icon className="size-3.5 text-slate-500" />
    </button>
  )
}

export default IconButton
