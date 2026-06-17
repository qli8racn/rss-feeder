package anthropic

import "context"

// SummarizeOptions は SummarizeAgent.Run のオプション。
type SummarizeOptions struct {
	FeedURL string
	Limit   int
}

// SummarizeAgent は記事を AI で要約する。
type SummarizeAgent interface {
	Run(ctx context.Context, opts SummarizeOptions) (string, error)
}
