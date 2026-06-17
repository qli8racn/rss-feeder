import { memo } from 'react'
import { BookmarkIcon, RssIcon } from './icons'

interface HeaderProps {
  bookmarkedCount: number
  bookmarkedOnly: boolean
  onToggleBookmarkedOnly: () => void
}

function Header({ bookmarkedCount, bookmarkedOnly, onToggleBookmarkedOnly }: HeaderProps) {
  return (
    <header className="flex items-center justify-between border-b border-slate-400/10 bg-[#0d1117]/95 px-6 py-4">
      <div className="flex items-center gap-3">
        <RssIcon className="size-[18px] text-amber-500" />
        <h1 className="font-mono text-[13px] font-bold uppercase tracking-[2px] text-slate-200/90">
          RSS Feed Viewer
        </h1>
      </div>
      <button
        type="button"
        onClick={onToggleBookmarkedOnly}
        aria-pressed={bookmarkedOnly}
        className={`flex items-center gap-1.5 rounded border px-3 py-1.5 font-mono text-[11px] transition-colors ${
          bookmarkedOnly ? 'border-amber-500/60 text-amber-400' : 'border-slate-400/10 text-slate-500'
        }`}
      >
        <BookmarkIcon filled={bookmarkedOnly} className="size-3" />
        <span>ブックマーク</span>
        <span className="rounded bg-[#1e2733] px-1.5 py-0.5 text-[10px] font-bold text-slate-500">
          {bookmarkedCount}
        </span>
      </button>
    </header>
  )
}

export default memo(Header)
