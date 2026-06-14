package rss

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/samber/do/v2"
)

const fetchTimeout = 30 * time.Second

type reader struct {
	parser *gofeed.Parser
}

func NewReader(_ do.Injector) (adapterrss.RSSReader, error) {
	p := gofeed.NewParser()
	p.Client = &http.Client{Timeout: fetchTimeout}
	return &reader{parser: p}, nil
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
			URL:          item.Link,
			Title:        item.Title,
			Content:      content,
			PublishedAt:  publishedAt,
			Publisher:    feed.Title,
			ThumbnailURL: extractThumbnail(item),
		})
	}

	return feed.Title, articles, nil
}

// extractThumbnail は記事のサムネイル画像URLを優先順位（media:thumbnail → enclosure(image/*) → itunes:image）で探索する。
// いずれも見つからない場合は空文字を返す。
func extractThumbnail(item *gofeed.Item) string {
	if media, ok := item.Extensions["media"]; ok {
		if thumbnails, ok := media["thumbnail"]; ok && len(thumbnails) > 0 {
			if url, ok := thumbnails[0].Attrs["url"]; ok && url != "" {
				return url
			}
		}
	}

	for _, enclosure := range item.Enclosures {
		if strings.HasPrefix(enclosure.Type, "image/") {
			return enclosure.URL
		}
	}

	if item.Image != nil {
		return item.Image.URL
	}

	return ""
}
