package feed

import (
	"context"
	"database/sql"
	"time"

	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/samber/do/v2"
)

type repository struct {
	db *sql.DB
}

func NewRepository(i do.Injector) (feedrepo.Repository, error) {
	return &repository{db: do.MustInvoke[*sql.DB](i)}, nil
}

func (r *repository) Save(ctx context.Context, feed domain.Feed) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO feeds (feed_url, title, last_fetched)
		VALUES (?, ?, ?)
		ON CONFLICT(feed_url) DO UPDATE SET
			title        = excluded.title,
			last_fetched = excluded.last_fetched
	`, feed.FeedURL, feed.Title, time.Now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *repository) FindByURL(ctx context.Context, url string) (*domain.Feed, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, feed_url, title, last_fetched, created_at
		FROM feeds WHERE feed_url = ?
	`, url)

	var f domain.Feed
	var lastFetched, createdAt sql.NullTime
	if err := row.Scan(&f.ID, &f.FeedURL, &f.Title, &lastFetched, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if lastFetched.Valid {
		f.LastFetched = lastFetched.Time
	}
	if createdAt.Valid {
		f.CreatedAt = createdAt.Time
	}
	return &f, nil
}
