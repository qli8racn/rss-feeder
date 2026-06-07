package domain

import "time"

type Feed struct {
	ID          int64
	FeedURL     string
	Title       string
	LastFetched time.Time
	CreatedAt   time.Time
}
