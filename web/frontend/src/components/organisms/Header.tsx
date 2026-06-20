import { memo } from 'react'
import { BookmarkIcon, ListIcon, RefreshIcon, RssIcon } from '../atoms/icons'
import Button from '../atoms/Button'

interface HeaderProps {
  bookmarkedCount: number
  bookmarkedOnly: boolean
  onToggleBookmarkedOnly: () => void
  fetching: boolean
  onFetchLatest: () => void
  onOpenFeedManagement: () => void
}

function Header({
  bookmarkedCount,
  bookmarkedOnly,
  onToggleBookmarkedOnly,
  fetching,
  onFetchLatest,
  onOpenFeedManagement,
}: HeaderProps) {
  return (
    <header className="flex items-center justify-between border-b border-slate-400/10 bg-surface-base/95 px-6 py-4">
      <div className="flex items-center gap-3">
        <RssIcon className="size-[18px] text-amber-500" />
        <h1 className="font-mono text-body font-bold uppercase tracking-[2px] text-text-primary/90">
          RSS Feed Viewer
        </h1>
      </div>
      <div className="flex items-center gap-2">
        <Button onClick={onFetchLatest} disabled={fetching}>
          <RefreshIcon className={`size-3 ${fetching ? 'animate-spin' : ''}`} />
          <span>{fetching ? '取得中...' : '最新フィードを取得'}</span>
        </Button>
        <Button onClick={onToggleBookmarkedOnly} active={bookmarkedOnly}>
          <BookmarkIcon filled={bookmarkedOnly} className="size-3" />
          <span>ブックマーク</span>
          <span className="rounded bg-surface-raised px-1.5 py-0.5 text-micro font-bold text-text-secondary">
            {bookmarkedCount}
          </span>
        </Button>
        <Button onClick={onOpenFeedManagement}>
          <ListIcon className="size-3" />
          <span>フィード管理</span>
        </Button>
      </div>
    </header>
  )
}

export default memo(Header)
