package domain

import "time"

type Article struct {
	ID           int64
	FeedID       int64
	URL          string
	Title        string
	Content      string
	PublishedAt  time.Time
	Read         bool
	Bookmarked   bool
	FetchedAt    time.Time
	FeedURL      string
	Publisher    string
	ThumbnailURL string
	Summary      string
	Category     string
}

func (a *Article) ToggleBookmark() {
	a.Bookmarked = !a.Bookmarked
}

func (a *Article) MarkAsRead() {
	a.Read = true
}
