package anthropic

import "context"

// DiscoverAgent はブックマーク趣向と登録済みフィードから、未購読の RSS フィードを推薦する。
type DiscoverAgent interface {
	Run(ctx context.Context) (string, error)
}
