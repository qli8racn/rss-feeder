package mcp

import (
	"time"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

// ArticleDTO は MCP ツールの出力用に domain.Article を変換したもの。
// MCPクライアント（LLM）が解釈しやすいよう、日時は RFC3339 文字列で表現する。
type ArticleDTO struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	PublishedAt  string `json:"published_at,omitempty"`
	Bookmarked   bool   `json:"bookmarked"`
	Read         bool   `json:"read"`
	Category     string `json:"category,omitempty"`
	Summary      string `json:"summary,omitempty"`
	Publisher    string `json:"publisher,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	FeedURL      string `json:"feed_url,omitempty"`
}

func toArticleDTO(a domain.Article) ArticleDTO {
	dto := ArticleDTO{
		ID:           a.ID,
		Title:        a.Title,
		URL:          a.URL,
		Bookmarked:   a.Bookmarked,
		Read:         a.Read,
		Category:     a.Category,
		Summary:      a.Summary,
		Publisher:    a.Publisher,
		ThumbnailURL: a.ThumbnailURL,
		FeedURL:      a.FeedURL,
	}
	if !a.PublishedAt.IsZero() {
		dto.PublishedAt = a.PublishedAt.Format(time.RFC3339)
	}
	return dto
}

func toArticleDTOs(articles []domain.Article) []ArticleDTO {
	dtos := make([]ArticleDTO, len(articles))
	for i, a := range articles {
		dtos[i] = toArticleDTO(a)
	}
	return dtos
}

// FeedDTO は MCP ツールの出力用に domain.Feed を変換したもの。
type FeedDTO struct {
	ID          int64  `json:"id"`
	FeedURL     string `json:"feed_url"`
	Title       string `json:"title,omitempty"`
	LastFetched string `json:"last_fetched,omitempty"`
}

func toFeedDTO(f domain.Feed) FeedDTO {
	dto := FeedDTO{ID: f.ID, FeedURL: f.FeedURL, Title: f.Title}
	if !f.LastFetched.IsZero() {
		dto.LastFetched = f.LastFetched.Format(time.RFC3339)
	}
	return dto
}

func toFeedDTOs(feeds []domain.Feed) []FeedDTO {
	dtos := make([]FeedDTO, len(feeds))
	for i, f := range feeds {
		dtos[i] = toFeedDTO(f)
	}
	return dtos
}
