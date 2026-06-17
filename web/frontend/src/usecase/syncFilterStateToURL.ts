import { buildFilterQuery } from '../domain/filter'
import type { FilterState } from '../domain/filter'

// ブラウザ履歴は積まず置き換えるのみ
// （フィルター操作1回ごとに戻るボタンの履歴が増えるのを避けるため replaceState を使う）。
export function syncFilterStateToURL(state: FilterState): void {
  const qs = buildFilterQuery(state)
  const url = qs ? `${window.location.pathname}?${qs}` : window.location.pathname
  window.history.replaceState(null, '', url)
}
