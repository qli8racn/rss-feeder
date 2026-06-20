package feedenrich

import (
	"context"
	"errors"
)

// ErrAgentUnavailable は rss-agent バイナリが存在しない・実行できない場合に返される。
var ErrAgentUnavailable = errors.New("rss-agent is not available")

// Agent は指定フィードの最新記事を対象に要約・カテゴライズをトリガーする。
type Agent interface {
	Enrich(ctx context.Context, feedURL string, limit int) error
}
