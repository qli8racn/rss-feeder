package anthropic

import "context"

// EnrichOptions は EnrichAgent.Run のオプション。
type EnrichOptions struct {
	Limit int
	Force bool
}

// EnrichAgent は記事に要約・カテゴリを付与して DB に保存する。
type EnrichAgent interface {
	Run(ctx context.Context, opts EnrichOptions) (int, error)
}
