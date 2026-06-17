package anthropic

import "context"

// PreferenceAgent はブックマーク済み記事から趣向を分析する。
type PreferenceAgent interface {
	Run(ctx context.Context) (string, error)
}
