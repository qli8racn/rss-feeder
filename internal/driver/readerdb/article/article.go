package article

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/samber/do/v2"
)

type repository struct {
	db *sql.DB
}

const (
	articleColumns        = "id, feed_id, url, title, content, published_at, read, bookmarked, fetched_at, publisher, thumbnail_url, summary, category"
	aliasedArticleColumns = "a.id, a.feed_id, a.url, a.title, a.content, a.published_at, a.read, a.bookmarked, a.fetched_at, a.publisher, a.thumbnail_url, a.summary, a.category"
)

func NewRepository(i do.Injector) (articlerepo.Repository, error) {
	return &repository{db: do.MustInvoke[*sql.DB](i)}, nil
}

func (r *repository) Save(ctx context.Context, a domain.Article) error {
	res, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO articles (feed_id, url, title, content, published_at, fetched_at, publisher, thumbnail_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, a.FeedID, a.URL, a.Title, a.Content, a.PublishedAt, time.Now(), a.Publisher, a.ThumbnailURL)
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
	q := fmt.Sprintf("SELECT %s FROM articles ORDER BY published_at DESC", articleColumns)
	return r.query(ctx, q)
}

func (r *repository) FindUnread(ctx context.Context) ([]domain.Article, error) {
	q := fmt.Sprintf("SELECT %s FROM articles WHERE read = 0 ORDER BY published_at DESC", articleColumns)
	return r.query(ctx, q)
}

func (r *repository) FindBookmarked(ctx context.Context) ([]domain.Article, error) {
	q := fmt.Sprintf("SELECT %s, COALESCE(f.feed_url, '') FROM articles a LEFT JOIN feeds f ON a.feed_id = f.id WHERE a.bookmarked = 1 ORDER BY a.published_at DESC", aliasedArticleColumns)
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanArticlesWithFeed(rows)
}

func (r *repository) FetchLatest(ctx context.Context, limit int, feedURL string) ([]domain.Article, error) {
	q := fmt.Sprintf("SELECT %s, COALESCE(f.feed_url, '') FROM articles a LEFT JOIN feeds f ON a.feed_id = f.id", aliasedArticleColumns)
	args := []any{}
	if feedURL != "" {
		q += " WHERE f.feed_url = ?"
		args = append(args, feedURL)
	}
	q += " ORDER BY a.published_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanArticlesWithFeed(rows)
}

func (r *repository) FindByID(ctx context.Context, id int64) (*domain.Article, error) {
	q := fmt.Sprintf("SELECT %s FROM articles WHERE id = ?", articleColumns)
	row := r.db.QueryRowContext(ctx, q, id)
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

func (r *repository) MarkAsRead(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE articles SET read = 1 WHERE id IN (`+placeholders+`)`,
		args...)
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

func (r *repository) Search(ctx context.Context, keyword string, bookmarkedOnly bool) ([]domain.Article, error) {
	q := fmt.Sprintf("SELECT %s FROM articles WHERE (title LIKE ? OR content LIKE ?)", articleColumns)
	like := "%" + keyword + "%"
	args := []any{like, like}
	if bookmarkedOnly {
		q += " AND bookmarked = 1"
	}
	q += " ORDER BY published_at DESC"
	return r.queryArgs(ctx, q, args...)
}

func (r *repository) UpdateEnrichment(ctx context.Context, id int64, summary, category string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE articles SET summary = ?, category = ? WHERE id = ?
	`, summary, category, id)
	return err
}

func (r *repository) FindWithoutSummary(ctx context.Context, limit int) ([]domain.Article, error) {
	q := fmt.Sprintf("SELECT %s FROM articles WHERE summary IS NULL OR summary = '' ORDER BY published_at DESC LIMIT ?", articleColumns)
	return r.queryArgs(ctx, q, limit)
}

func (r *repository) CountBookmarked(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM articles WHERE bookmarked = 1`).Scan(&count)
	return count, err
}

func (r *repository) FindFiltered(ctx context.Context, filter articlerepo.ListFilter) ([]domain.Article, int64, error) {
	var conditions []string
	var args []any

	if filter.Unread {
		conditions = append(conditions, "read = 0")
	}
	if filter.BookmarkedOnly {
		conditions = append(conditions, "bookmarked = 1")
	}
	if filter.Keyword != "" {
		conditions = append(conditions, "(title LIKE ? OR content LIKE ?)")
		like := "%" + filter.Keyword + "%"
		args = append(args, like, like)
	}
	if filter.Category != "" {
		conditions = append(conditions, "category = ?")
		args = append(args, filter.Category)
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	countQ := "SELECT COUNT(*) FROM articles" + where
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	sortColumn := filter.Sort
	if !articlerepo.ValidSortFields[sortColumn] {
		sortColumn = "published_at"
	}
	orderDir := "DESC"
	if filter.Order == "asc" {
		orderDir = "ASC"
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 {
		perPage = articlerepo.DefaultPerPage
	}

	q := fmt.Sprintf("SELECT %s FROM articles%s ORDER BY %s %s LIMIT ? OFFSET ?", articleColumns, where, sortColumn, orderDir)
	queryArgs := append(append([]any{}, args...), perPage, (page-1)*perPage)

	articles, err := r.queryArgs(ctx, q, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	return articles, total, nil
}

func (r *repository) DistinctCategories(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT category FROM articles WHERE category != '' ORDER BY category ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var categories []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *repository) query(ctx context.Context, q string) ([]domain.Article, error) {
	return r.queryArgs(ctx, q)
}

func (r *repository) queryArgs(ctx context.Context, q string, args ...any) ([]domain.Article, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

func scanArticlesWithFeed(rows *sql.Rows) ([]domain.Article, error) {
	var articles []domain.Article
	for rows.Next() {
		a, err := scanArticleWithFeed(rows.Scan)
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
	var publisher, thumbnailURL, summary, category sql.NullString
	err := scan(&a.ID, &a.FeedID, &a.URL, &a.Title, &a.Content,
		&publishedAt, &a.Read, &a.Bookmarked, &fetchedAt,
		&publisher, &thumbnailURL, &summary, &category)
	if err != nil {
		return nil, err
	}
	if publishedAt.Valid {
		a.PublishedAt = publishedAt.Time
	}
	if fetchedAt.Valid {
		a.FetchedAt = fetchedAt.Time
	}
	a.Publisher = publisher.String
	a.ThumbnailURL = thumbnailURL.String
	a.Summary = summary.String
	a.Category = category.String
	return &a, nil
}

func scanArticleWithFeed(scan func(dest ...any) error) (*domain.Article, error) {
	var a domain.Article
	var publishedAt, fetchedAt sql.NullTime
	var publisher, thumbnailURL, summary, category sql.NullString
	err := scan(&a.ID, &a.FeedID, &a.URL, &a.Title, &a.Content,
		&publishedAt, &a.Read, &a.Bookmarked, &fetchedAt,
		&publisher, &thumbnailURL, &summary, &category, &a.FeedURL)
	if err != nil {
		return nil, err
	}
	if publishedAt.Valid {
		a.PublishedAt = publishedAt.Time
	}
	if fetchedAt.Valid {
		a.FetchedAt = fetchedAt.Time
	}
	a.Publisher = publisher.String
	a.ThumbnailURL = thumbnailURL.String
	a.Summary = summary.String
	a.Category = category.String
	return &a, nil
}
