package article

import (
	"context"
	"database/sql"
	"time"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/samber/do/v2"
)

type repository struct {
	db *sql.DB
}

func NewRepository(i do.Injector) (articlerepo.Repository, error) {
	return &repository{db: do.MustInvoke[*sql.DB](i)}, nil
}

func (r *repository) Save(ctx context.Context, a domain.Article) error {
	res, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO articles (feed_id, url, title, content, published_at, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, a.FeedID, a.URL, a.Title, a.Content, a.PublishedAt, time.Now())
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return articlerepo.ErrDuplicate
	}
	return nil
}

func (r *repository) FindAll(ctx context.Context) ([]domain.Article, error) {
	return r.query(ctx, `SELECT id, feed_id, url, title, content, published_at, read, bookmarked, fetched_at
		FROM articles ORDER BY published_at DESC`)
}

func (r *repository) FindUnread(ctx context.Context) ([]domain.Article, error) {
	return r.query(ctx, `SELECT id, feed_id, url, title, content, published_at, read, bookmarked, fetched_at
		FROM articles WHERE read = 0 ORDER BY published_at DESC`)
}

func (r *repository) FindBookmarked(ctx context.Context) ([]domain.Article, error) {
	return r.query(ctx, `SELECT id, feed_id, url, title, content, published_at, read, bookmarked, fetched_at
		FROM articles WHERE bookmarked = 1 ORDER BY published_at DESC`)
}

func (r *repository) FindByID(ctx context.Context, id int64) (*domain.Article, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, feed_id, url, title, content, published_at, read, bookmarked, fetched_at
		FROM articles WHERE id = ?
	`, id)
	a, err := scanArticle(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

func (r *repository) Update(ctx context.Context, a domain.Article) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE articles SET read = ?, bookmarked = ? WHERE id = ?
	`, a.Read, a.Bookmarked, a.ID)
	return err
}

func (r *repository) DeleteNonBookmarked(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM articles WHERE bookmarked = 0`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *repository) CountNonBookmarked(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM articles WHERE bookmarked = 0`).Scan(&count)
	return count, err
}

func (r *repository) query(ctx context.Context, q string) ([]domain.Article, error) {
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []domain.Article
	for rows.Next() {
		a, err := scanArticle(rows.Scan)
		if err != nil {
			return nil, err
		}
		articles = append(articles, *a)
	}
	return articles, rows.Err()
}

func scanArticle(scan func(dest ...any) error) (*domain.Article, error) {
	var a domain.Article
	var publishedAt, fetchedAt sql.NullTime
	err := scan(&a.ID, &a.FeedID, &a.URL, &a.Title, &a.Content,
		&publishedAt, &a.Read, &a.Bookmarked, &fetchedAt)
	if err != nil {
		return nil, err
	}
	if publishedAt.Valid {
		a.PublishedAt = publishedAt.Time
	}
	if fetchedAt.Valid {
		a.FetchedAt = fetchedAt.Time
	}
	return &a, nil
}
