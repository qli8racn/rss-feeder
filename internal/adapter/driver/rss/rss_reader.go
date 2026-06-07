package rss

import (
	"context"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

type RSSReader interface {
	Fetch(ctx context.Context, feedURL string) (feedTitle string, articles []domain.Article, err error)
}
