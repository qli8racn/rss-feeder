interface CategoryStyle {
  text: string
  bg: string
}

// Figma デザインで確認できた配色（Tech は実測値、他は同系統の配色パターンを踏襲した参考値）。
// 固定マスタではないため、未知のカテゴリは DEFAULT_STYLE にフォールバックする。
const CATEGORY_STYLES: Record<string, CategoryStyle> = {
  Tech: { text: '#00bcff', bg: 'rgba(0,188,255,0.1)' },
  AI: { text: '#a78bfa', bg: 'rgba(167,139,250,0.1)' },
  Work: { text: '#fb7185', bg: 'rgba(251,113,133,0.1)' },
  Science: { text: '#34d399', bg: 'rgba(52,211,153,0.1)' },
  Finance: { text: '#fbbf24', bg: 'rgba(251,191,36,0.1)' },
  Design: { text: '#e879f9', bg: 'rgba(232,121,249,0.1)' },
  Business: { text: '#60a5fa', bg: 'rgba(96,165,250,0.1)' },
  Career: { text: '#fb923c', bg: 'rgba(251,146,60,0.1)' },
  Security: { text: '#f87171', bg: 'rgba(248,113,113,0.1)' },
  QA: { text: '#2dd4bf', bg: 'rgba(45,212,191,0.1)' },
  Entertainment: { text: '#f472b6', bg: 'rgba(244,114,182,0.1)' },
  Education: { text: '#818cf8', bg: 'rgba(129,140,248,0.1)' },
  Sports: { text: '#a3e635', bg: 'rgba(163,230,53,0.1)' },
  Project: { text: '#22d3ee', bg: 'rgba(34,211,238,0.1)' },
  Personal: { text: '#4ade80', bg: 'rgba(74,222,128,0.1)' },
}

const DEFAULT_STYLE: CategoryStyle = { text: '#94a3b8', bg: 'rgba(148,163,184,0.1)' }

export function categoryStyle(category: string): CategoryStyle {
  return CATEGORY_STYLES[category] ?? DEFAULT_STYLE
}
