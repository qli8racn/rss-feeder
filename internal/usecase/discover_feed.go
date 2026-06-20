package usecase

import (
	"context"
	"fmt"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	"github.com/qli8racn/rss-feeder/internal/adapter/driver/htmlfetch"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
)

// DiscoverFeedUsecase は通常のサイトURLからRSS/AtomフィードのURLを推測する。
type DiscoverFeedUsecase struct {
	htmlFetcher htmlfetch.Fetcher
	agent       adapteranthropic.FeedDiscoveryAgent
	rssReader   adapterrss.RSSReader
}

func NewDiscoverFeedUsecase(
	htmlFetcher htmlfetch.Fetcher,
	agent adapteranthropic.FeedDiscoveryAgent,
	rssReader adapterrss.RSSReader,
) *DiscoverFeedUsecase {
	return &DiscoverFeedUsecase{htmlFetcher: htmlFetcher, agent: agent, rssReader: rssReader}
}

// Execute は url のHTMLを取得し、Claudeにフィード URL を推測させた上で、
// 候補URLが実際にRSS/Atomとしてパース可能かを再検証する。
func (uc *DiscoverFeedUsecase) Execute(ctx context.Context, url string) (string, error) {
	html, err := uc.htmlFetcher.Fetch(ctx, url)
	if err != nil {
		return "", fmt.Errorf("HTMLの取得に失敗しました %s: %w", url, err)
	}

	candidate, err := uc.agent.Discover(ctx, url, html)
	if err != nil {
		return "", fmt.Errorf("フィードURLの推測に失敗しました %s: %w", url, err)
	}

	if _, _, err := uc.rssReader.Fetch(ctx, candidate); err != nil {
		return "", fmt.Errorf("推測されたURLはRSS/Atomとして解釈できませんでした %s: %w", candidate, err)
	}

	return candidate, nil
}
