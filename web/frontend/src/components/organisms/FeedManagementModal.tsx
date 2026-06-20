import { useCallback, useEffect, useState } from 'react'
import { addFeed, fetchFeeds, removeFeed } from '../../api'
import type { Feed } from '../../types'
import { PlusIcon } from '../atoms/icons'
import IconButton from '../molecules/IconButton'
import TextField from '../molecules/TextField'
import Button from '../atoms/Button'
import Table, { TABLE_HEADER_ROW_CLASS, TABLE_ROW_CLASS } from '../molecules/Table'

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
const ACTION_COL_CLASS = 'w-8'

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
      <div className="max-h-[80vh] w-full max-w-lg overflow-y-auto rounded border border-slate-400/10 bg-surface-base text-text-primary">
        <div className="flex items-center justify-between border-b border-slate-400/10 px-[18px] py-3.5">
          <h2 className="font-mono text-body font-bold text-text-primary/90">フィード管理</h2>
          <IconButton icon="close" onClick={onClose} ariaLabel="閉じる" />
        </div>

        <form
          onSubmit={handleAdd}
          className="flex items-center gap-2 border-b border-slate-400/10 px-[18px] py-3.5"
        >
          <TextField
            className="flex-1"
            value={newUrl}
            onChange={(e) => setNewUrl(e.target.value)}
            placeholder="https://example.com/feed.xml"
            disabled={submitting}
          />
          <Button type="submit" disabled={submitting || !newUrl.trim()}>
            <PlusIcon className="size-3" />
            <span>{submitting ? '追加中...' : '追加'}</span>
          </Button>
        </form>

        {error && <p className="px-[18px] py-2 text-small text-rose-400">{error}</p>}

        {loading ? (
          <p className="px-[18px] py-8 text-center text-small text-text-secondary">読み込み中...</p>
        ) : feeds.length === 0 ? (
          <p className="px-[18px] py-8 text-center text-small text-text-secondary">
            登録済みのフィードがありません。上の入力欄から追加してください。
          </p>
        ) : (
          <Table>
            <thead>
              <tr className={TABLE_HEADER_ROW_CLASS}>
                <th className={`${URL_COL_CLASS} py-2.5 pl-[18px] pr-3 text-left font-mono text-micro text-text-secondary`}>
                  URL
                </th>
                <th className="py-2.5 pr-3 text-left font-mono text-micro text-text-secondary">タイトル</th>
                <th className={`${LAST_FETCHED_COL_CLASS} py-2.5 pr-3 text-left font-mono text-micro text-text-secondary`}>
                  最終取得
                </th>
                <th className={`${ACTION_COL_CLASS} py-2.5 pr-[18px]`} />
              </tr>
            </thead>
            <tbody>
              {feeds.map((feed) => (
                <tr key={feed.id} className={TABLE_ROW_CLASS}>
                  <td className="break-all py-3 pl-[18px] pr-3 align-top font-mono text-micro text-text-secondary">
                    {feed.feed_url}
                  </td>
                  <td className="py-3 pr-3 align-top text-caption text-text-primary/90">
                    {feed.title || '(未取得)'}
                  </td>
                  <td className="py-3 pr-3 align-top font-mono text-micro text-text-secondary">
                    {formatLastFetched(feed.last_fetched)}
                  </td>
                  <td className="py-3 pr-[18px] align-top">
                    <IconButton icon="trash" onClick={() => handleRemove(feed)} ariaLabel={`${feed.feed_url} を削除`} />
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
        )}
      </div>
    </div>
  )
}

export default FeedManagementModal
