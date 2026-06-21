package anthropic

import "context"

// EnrichOptions は EnrichAgent.Run のオプション。
type EnrichOptions struct {
	Limit       int
	Force       bool
	FeedURL     string // 指定時はそのフィードのみを対象にする（空文字列なら全フィード対象）
	BatchSize   int    // 1回のAPI呼び出しで処理する記事数（0以下ならデフォルト値を使う）
	Concurrency int    // バッチを並列実行する際の最大同時実行数（0以下ならデフォルト値を使う）
}

// EnrichAgent は記事に要約・カテゴリを付与して DB に保存する。
type EnrichAgent interface {
	Run(ctx context.Context, opts EnrichOptions) (int, error)
}
