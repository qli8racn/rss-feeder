package anthropic

import "context"

// FeedDiscoveryAgent は HTML からRSS/AtomフィードへのリンクURLを推測する。
type FeedDiscoveryAgent interface {
	Discover(ctx context.Context, url string, html string) (feedURL string, err error)
}
