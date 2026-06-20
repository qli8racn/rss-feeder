package anthropic

import "context"

// EnrichOptions は EnrichAgent.Run のオプション。
type EnrichOptions struct {
	Limit   int
	Force   bool
	FeedURL string // 指定時はそのフィードのみを対象にする（空文字列なら全フィード対象）
}

// EnrichAgent は記事に要約・カテゴリを付与して DB に保存する。
type EnrichAgent interface {
	Run(ctx context.Context, opts EnrichOptions) (int, error)
}
