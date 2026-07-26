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

func (r *repository) Save(ctx context.Context, feed domain.Feed, userID int64) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO feeds (user_id, feed_url, title, last_fetched)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, feed_url) DO UPDATE SET
			title        = excluded.title,
			last_fetched = excluded.last_fetched
	`, userID, feed.FeedURL, feed.Title, time.Now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *repository) FindByURL(ctx context.Context, url string, userID int64) (*domain.Feed, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, feed_url, COALESCE(title, ''), last_fetched, created_at
		FROM feeds WHERE feed_url = ? AND user_id = ?
	`, url, userID)

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

func (r *repository) Register(ctx context.Context, url string, userID int64) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO feeds (user_id, feed_url) VALUES (?, ?) ON CONFLICT(user_id, feed_url) DO NOTHING`,
		userID, url)
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

func (r *repository) ListAll(ctx context.Context, userID int64) ([]domain.Feed, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, feed_url, COALESCE(title, ''), last_fetched, created_at
		FROM feeds WHERE user_id = ? ORDER BY created_at
	`, userID)
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

// Remove はフィードと、それに紐づく記事を削除する。SQLiteのforeign_keys制約は
// 有効化していない（audit_log.article_idなど他のFKにも影響するため）ため、
// articlesの削除はON DELETE CASCADEに頼らずトランザクション内で明示的に行う。
// id が userID の所有するフィードでない場合（他ユーザーのフィードID、または存在しないID）は
// ErrNotFound を返す（記事IDと同様、フィードIDの推測による他ユーザーのフィード操作を防ぐ）。
func (r *repository) Remove(ctx context.Context, id int64, userID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var exists int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM feeds WHERE id = ? AND user_id = ?`, id, userID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return feedrepo.ErrNotFound
		}
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM articles WHERE feed_id = ?`, id); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM feeds WHERE id = ? AND user_id = ?`, id, userID); err != nil {
		return err
	}
	return tx.Commit()
}
