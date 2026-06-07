package rss

import (
	"context"
	"time"

	"github.com/mmcdole/gofeed"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/samber/do/v2"
)

type reader struct {
	parser *gofeed.Parser
}

func NewReader(_ do.Injector) (adapterrss.RSSReader, error) {
	return &reader{parser: gofeed.NewParser()}, nil
}

func (r *reader) Fetch(ctx context.Context, feedURL string) (string, []domain.Article, error) {
	feed, err := r.parser.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return "", nil, err
	}

	articles := make([]domain.Article, 0, len(feed.Items))
	for _, item := range feed.Items {
		var publishedAt time.Time
		if item.PublishedParsed != nil {
			publishedAt = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			publishedAt = *item.UpdatedParsed
		}

		content := item.Description
		if item.Content != "" {
			content = item.Content
		}

		articles = append(articles, domain.Article{
			URL:         item.Link,
			Title:       item.Title,
			Content:     content,
			PublishedAt: publishedAt,
		})
	}

	return feed.Title, articles, nil
}
