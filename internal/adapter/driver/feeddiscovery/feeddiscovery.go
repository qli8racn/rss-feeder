package feeddiscovery

import (
	"context"
	"errors"
)

// ErrAgentUnavailable は rss-agent バイナリが存在しない・実行できない場合に返される。
var ErrAgentUnavailable = errors.New("rss-agent is not available")

// Agent はURLからRSS/AtomフィードのURLを推測する。
// cmd/web・cmd/rss-feeder とも同じサブプロセス実装（bin/rss-agent discover-feed）を使う。
type Agent interface {
	Discover(ctx context.Context, url string) (feedURL string, err error)
}
