// 7日以内は相対表示（2h前 / 1d前 等）、7日超は絶対日付（YYYY/MM/DD）に切り替える
export function formatDate(iso: string): string {
  const date = new Date(iso)
  const diffMs = Date.now() - date.getTime()
  const diffHours = diffMs / (1000 * 60 * 60)
  const diffDays = diffHours / 24

  if (diffDays > 7) {
    const y = date.getFullYear()
    const m = String(date.getMonth() + 1).padStart(2, '0')
    const d = String(date.getDate()).padStart(2, '0')
    return `${y}/${m}/${d}`
  }
  if (diffDays >= 1) return `${Math.floor(diffDays)}d前`
  if (diffHours >= 1) return `${Math.floor(diffHours)}h前`
  const diffMinutes = Math.max(Math.floor(diffMs / (1000 * 60)), 0)
  return `${diffMinutes}m前`
}
