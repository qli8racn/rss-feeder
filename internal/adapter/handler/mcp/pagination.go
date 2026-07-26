package mcp

// list・search ツールは CLI と異なり、返した記事がそのまま LLM のコンテキストに入るため、
// CLI 同様の「実用上全件」な件数（旧: 10000件）をデフォルトにすると summary 付き記事数百〜数千件で
// コンテキストを圧迫する。そのため limit・page を LLM 側から指定可能にし、デフォルト・上限を
// 小さめに設定する。
const (
	defaultPerPage = 50
	maxPerPage     = 200
)

// resolvePerPage は limit の未指定（0以下）をデフォルト値に、上限超過をクランプする。
func resolvePerPage(limit int) int {
	switch {
	case limit <= 0:
		return defaultPerPage
	case limit > maxPerPage:
		return maxPerPage
	default:
		return limit
	}
}

// resolvePage はページ番号の未指定・不正値（1未満）を1ページ目に補正する。
func resolvePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}
