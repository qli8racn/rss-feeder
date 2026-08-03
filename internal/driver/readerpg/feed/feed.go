package feed

import (
	"context"
	"database/sql"
	"time"

	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/samber/do/v2"
)

var _ feedrepo.Repository = (*repository)(nil)

type repository struct {
	db *sql.DB
}

func NewRepository(i do.Injector) (feedrepo.Repository, error) {
	return &repository{db: do.MustInvoke[*sql.DB](i)}, nil
}

func (r *repository) Save(ctx context.Context, feed domain.Feed) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO feeds (feed_url, title, last_fetched)
		VALUES ($1, $2, $3)
		ON CONFLICT(feed_url) DO UPDATE SET
			title        = excluded.title,
			last_fetched = excluded.last_fetched
		RETURNING id
	`, feed.FeedURL, feed.Title, time.Now()).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *repository) FindByURL(ctx context.Context, url string) (*domain.Feed, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, feed_url, COALESCE(title, ''), last_fetched, created_at
		FROM feeds WHERE feed_url = $1
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

func (r *repository) Register(ctx context.Context, url string) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO feeds (feed_url) VALUES ($1) ON CONFLICT(feed_url) DO NOTHING`, url)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return feedrepo.ErrAlreadyExists
	}
	return nil
}

func (r *repository) ListAll(ctx context.Context) ([]domain.Feed, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, feed_url, COALESCE(title, ''), last_fetched, created_at
		FROM feeds ORDER BY created_at, id
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var feeds []domain.Feed
	for rows.Next() {
		var f domain.Feed
		var lastFetched, createdAt sql.NullTime
		if err := rows.Scan(&f.ID, &f.FeedURL, &f.Title, &lastFetched, &createdAt); err != nil {
			return nil, err
		}
		if lastFetched.Valid {
			f.LastFetched = lastFetched.Time
		}
		if createdAt.Valid {
			f.CreatedAt = createdAt.Time
		}
		feeds = append(feeds, f)
	}
	return feeds, rows.Err()
}

// Remove はフィードと、それに紐づく記事を削除する。Postgres側もSQLite実装と同様に
// ON DELETE CASCADEに頼らずトランザクション内で明示的に削除する（実装方針を揃える）。
func (r *repository) Remove(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM articles WHERE feed_id = $1`, id); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM feeds WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return feedrepo.ErrNotFound
	}
	return tx.Commit()
}
