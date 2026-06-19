import { useCallback, useEffect, useState } from 'react'
import { addFeed, fetchFeeds, removeFeed } from '../api'
import type { Feed } from '../types'
import { CloseIcon, PlusIcon, TrashIcon } from './icons'

interface FeedManagementModalProps {
  open: boolean
  onClose: () => void
}

function formatLastFetched(value: string | null): string {
  if (!value) return 'なし'
  return new Date(value).toLocaleString('ja-JP')
}

const URL_COL_CLASS = 'w-[170px]'
const LAST_FETCHED_COL_CLASS = 'w-[100px]'
const ACTION_COL_CLASS = 'w-6'

function FeedManagementModal({ open, onClose }: FeedManagementModalProps) {
  const [feeds, setFeeds] = useState<Feed[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [newUrl, setNewUrl] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const loadFeeds = useCallback(() => {
    setLoading(true)
    setError(null)
    fetchFeeds()
      .then(setFeeds)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (open) loadFeeds()
  }, [open, loadFeeds])

  if (!open) return null

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    const url = newUrl.trim()
    if (!url) return
    setSubmitting(true)
    setError(null)
    try {
      const feed = await addFeed(url)
      setFeeds((prev) => [...prev, feed])
      setNewUrl('')
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setSubmitting(false)
    }
  }

  const handleRemove = async (feed: Feed) => {
    if (!window.confirm(`「${feed.title || feed.feed_url}」を削除しますか?`)) return
    setError(null)
    try {
      await removeFeed(feed.id)
      setFeeds((prev) => prev.filter((f) => f.id !== feed.id))
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4">
      <div className="max-h-[80vh] w-full max-w-lg overflow-y-auto rounded border border-slate-400/10 bg-[#0d1117] text-slate-200">
        <div className="flex items-center justify-between border-b border-slate-400/10 px-[18px] py-3.5">
          <h2 className="font-mono text-[13px] font-bold text-slate-200/90">フィード管理</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="閉じる"
            className="rounded border border-slate-400/10 p-1.5"
          >
            <CloseIcon className="size-3.5 text-slate-500" />
          </button>
        </div>

        <form
          onSubmit={handleAdd}
          className="flex items-center gap-2 border-b border-slate-400/10 px-[18px] py-3.5"
        >
          <input
            type="text"
            value={newUrl}
            onChange={(e) => setNewUrl(e.target.value)}
            placeholder="https://example.com/feed.xml"
            disabled={submitting}
            className="flex-1 rounded border border-slate-400/10 bg-[#1e2733] px-3 py-2 text-[11px] text-slate-200 placeholder:text-slate-500/60 focus:outline-none disabled:opacity-50"
          />
          <button
            type="submit"
            disabled={submitting || !newUrl.trim()}
            className="flex items-center gap-1.5 whitespace-nowrap rounded border border-slate-400/10 bg-[#1e2733] px-3 py-2 font-mono text-[11px] text-slate-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <PlusIcon className="size-3" />
            <span>{submitting ? '追加中...' : '追加'}</span>
          </button>
        </form>

        {error && <p className="px-[18px] py-2 text-[12px] text-rose-400">{error}</p>}

        {loading ? (
          <p className="px-[18px] py-8 text-center text-[12px] text-slate-500">読み込み中...</p>
        ) : feeds.length === 0 ? (
          <p className="px-[18px] py-8 text-center text-[12px] text-slate-500">
            登録済みのフィードがありません。上の入力欄から追加してください。
          </p>
        ) : (
          <div>
            <div className="flex items-center gap-3 bg-[#1e2733]/60 px-[18px] py-2.5 font-mono text-[10.5px] text-slate-500">
              <span className={URL_COL_CLASS}>URL</span>
              <span className="flex-1">タイトル</span>
              <span className={LAST_FETCHED_COL_CLASS}>最終取得</span>
              <span className={ACTION_COL_CLASS} />
            </div>
            {feeds.map((feed) => (
              <div
                key={feed.id}
                className="flex items-center gap-3 border-b border-slate-400/10 px-[18px] py-3"
              >
                <span className={`${URL_COL_CLASS} break-all font-mono text-[10.5px] text-slate-500`}>
                  {feed.feed_url}
                </span>
                <span className="flex-1 text-[11.25px] text-slate-200/90">
                  {feed.title || '(未取得)'}
                </span>
                <span className={`${LAST_FETCHED_COL_CLASS} font-mono text-[10.5px] text-slate-500`}>
                  {formatLastFetched(feed.last_fetched)}
                </span>
                <button
                  type="button"
                  onClick={() => handleRemove(feed)}
                  aria-label={`${feed.feed_url} を削除`}
                  className={`${ACTION_COL_CLASS} rounded border border-slate-400/10 p-1.5`}
                >
                  <TrashIcon className="size-3.5 text-slate-500" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export default FeedManagementModal
