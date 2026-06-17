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
}

const DEFAULT_STYLE: CategoryStyle = { text: '#94a3b8', bg: 'rgba(148,163,184,0.1)' }

export function categoryStyle(category: string): CategoryStyle {
  return CATEGORY_STYLES[category] ?? DEFAULT_STYLE
}
