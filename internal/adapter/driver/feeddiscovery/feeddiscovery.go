package feeddiscovery

import (
	"context"
	"errors"
)

// ErrAgentUnavailable は rss-agent バイナリが存在しない・実行できない場合に返される。
var ErrAgentUnavailable = errors.New("rss-agent is not available")

// Agent はURLからRSS/AtomフィードのURLを推測する。
// 実装は cmd/web（サブプロセス経由）と cmd/rss-feeder（インプロセス）で異なる。
type Agent interface {
	Discover(ctx context.Context, url string) (feedURL string, err error)
}
