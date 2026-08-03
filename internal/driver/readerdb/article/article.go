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

	// ownedByUserSubquery は articles.feed_id が userID の所有するフィードを指しているかを
	// 判定するサブクエリ。記事IDはユーザー横断の連番のため、これを条件に加えないと、
	// 他ユーザーの記事IDを推測して操作できてしまう。
	ownedByUserSubquery = "feed_id IN (SELECT id FROM feeds WHERE user_id = ?)"
)

func NewRepository(i do.Injector) (articlerepo.Repository, error) {
	return &repository{db: do.MustInvoke[*sql.DB](i)}, nil
}

// Save は a.FeedID が呼び出し元によって既にuserIDスコープで検証済みであることを前提とし、
// 所有権の再チェックは行わない（articlerepo.Repository のインターフェースコメント参照）。
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

func (r *repository) FindAll(ctx context.Context, userID int64) ([]domain.Article, error) {
	q := fmt.Sprintf("SELECT %s FROM articles WHERE %s ORDER BY published_at DESC", articleColumns, ownedByUserSubquery)
	return r.queryArgs(ctx, q, userID)
}

func (r *repository) FindUnread(ctx context.Context, userID int64) ([]domain.Article, error) {
	q := fmt.Sprintf("SELECT %s FROM articles WHERE read = 0 AND %s ORDER BY published_at DESC", articleColumns, ownedByUserSubquery)
	return r.queryArgs(ctx, q, userID)
}

func (r *repository) FindBookmarked(ctx context.Context, userID int64) ([]domain.Article, error) {
	q := fmt.Sprintf(
		"SELECT %s, COALESCE(f.feed_url, '') FROM articles a LEFT JOIN feeds f ON a.feed_id = f.id WHERE a.bookmarked = 1 AND a.%s ORDER BY a.published_at DESC",
		aliasedArticleColumns, ownedByUserSubquery)
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanArticlesWithFeed(rows)
}

func (r *repository) FetchLatest(ctx context.Context, limit int, feedURL string, userID int64) ([]domain.Article, error) {
	q := fmt.Sprintf(
		"SELECT %s, COALESCE(f.feed_url, '') FROM articles a LEFT JOIN feeds f ON a.feed_id = f.id WHERE a.%s",
		aliasedArticleColumns, ownedByUserSubquery)
	args := []any{userID}
	if feedURL != "" {
		q += " AND f.feed_url = ?"
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

func (r *repository) FindByID(ctx context.Context, id int64, userID int64) (*domain.Article, error) {
	q := fmt.Sprintf("SELECT %s FROM articles WHERE id = ? AND %s", articleColumns, ownedByUserSubquery)
	row := r.db.QueryRowContext(ctx, q, id, userID)
	a, err := scanArticle(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

func (r *repository) Update(ctx context.Context, a domain.Article, userID int64) error {
	q := fmt.Sprintf(`UPDATE articles SET read = ?, bookmarked = ? WHERE id = ? AND %s`, ownedByUserSubquery)
	_, err := r.db.ExecContext(ctx, q, a.Read, a.Bookmarked, a.ID, userID)
	return err
}

func (r *repository) MarkAsRead(ctx context.Context, ids []int64, userID int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, 0, len(ids)+1)
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, userID)
	q := fmt.Sprintf(`UPDATE articles SET read = 1 WHERE id IN (%s) AND %s`, placeholders, ownedByUserSubquery)
	_, err := r.db.ExecContext(ctx, q, args...)
	return err
}

func (r *repository) DeleteNonBookmarked(ctx context.Context, userID int64) (int64, error) {
	q := fmt.Sprintf(`DELETE FROM articles WHERE bookmarked = 0 AND %s`, ownedByUserSubquery)
	res, err := r.db.ExecContext(ctx, q, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *repository) CountNonBookmarked(ctx context.Context, userID int64) (int64, error) {
	var count int64
	q := fmt.Sprintf(`SELECT COUNT(*) FROM articles WHERE bookmarked = 0 AND %s`, ownedByUserSubquery)
	err := r.db.QueryRowContext(ctx, q, userID).Scan(&count)
	return count, err
}

func (r *repository) Search(ctx context.Context, keyword string, bookmarkedOnly bool, userID int64) ([]domain.Article, error) {
	q := fmt.Sprintf("SELECT %s FROM articles WHERE (title LIKE ? OR content LIKE ?)", articleColumns)
	like := "%" + keyword + "%"
	args := []any{like, like}
	if bookmarkedOnly {
		q += " AND bookmarked = 1"
	}
	q += " AND " + ownedByUserSubquery
	args = append(args, userID)
	q += " ORDER BY published_at DESC"
	return r.queryArgs(ctx, q, args...)
}

// UpdateEnrichmentBatch は複数件の要約・カテゴリ更新を1トランザクションでまとめて行う。
// 1件ずつ ExecContext するより、ジャーナルへのfsyncをまとめられる分高速。
// 1件でも失敗すればトランザクション全体をロールバックする（呼び出し元はこの単位を
// 「DBへの保存単位」として扱うため、部分成功させたい場合は呼び出し元で複数回に分けて呼ぶ）。
func (r *repository) UpdateEnrichmentBatch(ctx context.Context, updates []articlerepo.EnrichmentUpdate, userID int64) error {
	if len(updates) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf(`UPDATE articles SET summary = ?, category = ? WHERE id = ? AND %s`, ownedByUserSubquery))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, u := range updates {
		if _, err := stmt.ExecContext(ctx, u.Summary, u.Category, u.ID, userID); err != nil {
			return fmt.Errorf("記事 %d の更新に失敗しました: %w", u.ID, err)
		}
	}
	return tx.Commit()
}

func (r *repository) FindWithoutSummary(ctx context.Context, limit int, userID int64) ([]domain.Article, error) {
	q := fmt.Sprintf("SELECT %s FROM articles WHERE (summary IS NULL OR summary = '') AND %s ORDER BY published_at DESC LIMIT ?", articleColumns, ownedByUserSubquery)
	return r.queryArgs(ctx, q, userID, limit)
}

// UpdateMetadataBatch は既存記事への出版元・サムネイルのバックフィル用。
// publisher・thumbnail_url は列ごとに「現在空（NULL または空文字）、かつ新しい値が空でない場合のみ」更新し、
// 既に値がある列は上書きしない（何度実行しても安全、かつ手動で編集された値を壊さない）。
// `ALTER TABLE ... ADD COLUMN`（migration.go）で追加した既存行は NULL になるため、
// 空文字判定だけでは検出できず COALESCE で NULL も空として扱う。
// 新しい値も空（フィードがそもそもサムネイルを提供しない等）の場合は対象外とする
// （WHERE句に含めないと、永遠に「補完」件数として数えられてしまう）。
func (r *repository) UpdateMetadataBatch(ctx context.Context, updates []articlerepo.MetadataUpdate) (int64, error) {
	if len(updates) == 0 {
		return 0, nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		UPDATE articles SET
			publisher = CASE WHEN COALESCE(publisher, '') = '' AND :publisher != '' THEN :publisher ELSE publisher END,
			thumbnail_url = CASE WHEN COALESCE(thumbnail_url, '') = '' AND :thumbnail_url != '' THEN :thumbnail_url ELSE thumbnail_url END
		WHERE url = :url
			AND (
				(COALESCE(publisher, '') = '' AND :publisher != '')
				OR (COALESCE(thumbnail_url, '') = '' AND :thumbnail_url != '')
			)
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var total int64
	for _, u := range updates {
		res, err := stmt.ExecContext(ctx,
			sql.Named("publisher", u.Publisher),
			sql.Named("thumbnail_url", u.ThumbnailURL),
			sql.Named("url", u.URL),
		)
		if err != nil {
			return 0, fmt.Errorf("記事 %s の更新に失敗しました: %w", u.URL, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, tx.Commit()
}

func (r *repository) CountBookmarked(ctx context.Context, userID int64) (int64, error) {
	var count int64
	q := fmt.Sprintf(`SELECT COUNT(*) FROM articles WHERE bookmarked = 1 AND %s`, ownedByUserSubquery)
	err := r.db.QueryRowContext(ctx, q, userID).Scan(&count)
	return count, err
}

func (r *repository) FindFiltered(ctx context.Context, filter articlerepo.ListFilter, userID int64) ([]domain.Article, int64, error) {
	conditions := []string{ownedByUserSubquery}
	args := []any{userID}

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

	where := " WHERE " + strings.Join(conditions, " AND ")

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

func (r *repository) DistinctCategories(ctx context.Context, userID int64) ([]string, error) {
	q := fmt.Sprintf(`SELECT DISTINCT category FROM articles WHERE category != '' AND %s ORDER BY category ASC`, ownedByUserSubquery)
	rows, err := r.db.QueryContext(ctx, q, userID)
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
