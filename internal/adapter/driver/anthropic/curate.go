package anthropic

import "context"

// CurateAgent は直近記事とブックマーク趣向から「今日読む価値がある記事」を選定・提示する。
type CurateAgent interface {
	Run(ctx context.Context, opts CurateOptions) (string, error)
}

// CurateOptions は CurateAgent.Run の実行オプション。
type CurateOptions struct {
	// Limit は考慮する直近記事数の上限。0 の場合はエージェント側のデフォルト値を使う。
	Limit int
}
