import { memo } from 'react'
import { RssIcon } from './icons'
import Button from './ui/Button'

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
        <h1 className="font-mono text-body font-bold uppercase tracking-[2px] text-slate-200/90">
          RSS Feed Viewer
        </h1>
      </div>
      <div className="flex items-center gap-2">
        <Button icon="refresh" onClick={onFetchLatest} disabled={fetching} spinning={fetching}>
          {fetching ? '取得中...' : '最新フィードを取得'}
        </Button>
        <Button icon="bookmark" onClick={onToggleBookmarkedOnly} active={bookmarkedOnly} badge={bookmarkedCount}>
          ブックマーク
        </Button>
        <Button icon="list" onClick={onOpenFeedManagement}>
          フィード管理
        </Button>
      </div>
    </header>
  )
}

export default memo(Header)
