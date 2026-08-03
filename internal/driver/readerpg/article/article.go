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

var _ articlerepo.Repository = (*repository)(nil)

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

// ownedByUserSubquery は articles.feed_id が userID の所有するフィードを指しているかを
// 判定するサブクエリを返す（SQLite版のownedByUserSubqueryのPostgres版）。Postgresは
// 位置引数（$1, $2, ...）のため、SQLite版のような固定文字列定数にはできず、userIDが
// 引数リストの何番目（argIndex）に来るかを呼び出し側から渡す必要がある。
// alias には、articles に "a" エイリアスが付くクエリ（feedsとのJOINを伴うクエリ）では "a."
// を、それ以外では "" を渡す。
func ownedByUserSubquery(argIndex int, alias string) string {
	return fmt.Sprintf("%sfeed_id IN (SELECT id FROM feeds WHERE user_id = $%d)", alias, argIndex)
}

// Save は article.FeedID の所有権チェックを行わない。呼び出し元が事前に
// userIDスコープの feedRepo（例: feedRepo.Save の戻り値）から取得した FeedID を
// 渡すことが前提（articlerepo.Repository のインターフェースコメント参照）。
func (r *repository) Save(ctx context.Context, a domain.Article) error {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO articles (feed_id, url, title, content, published_at, fetched_at, publisher, thumbnail_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (feed_id, url) DO NOTHING
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
	q := fmt.Sprintf("SELECT %s FROM articles WHERE %s ORDER BY published_at DESC NULLS LAST, id DESC", articleColumns, ownedByUserSubquery(1, ""))
	return r.queryArgs(ctx, q, userID)
}

func (r *repository) FindUnread(ctx context.Context, userID int64) ([]domain.Article, error) {
	q := fmt.Sprintf("SELECT %s FROM articles WHERE read = FALSE AND %s ORDER BY published_at DESC NULLS LAST, id DESC", articleColumns, ownedByUserSubquery(1, ""))
	return r.queryArgs(ctx, q, userID)
}

func (r *repository) FindBookmarked(ctx context.Context, userID int64) ([]domain.Article, error) {
	q := fmt.Sprintf(
		"SELECT %s, COALESCE(f.feed_url, '') FROM articles a LEFT JOIN feeds f ON a.feed_id = f.id WHERE a.bookmarked = TRUE AND %s ORDER BY a.published_at DESC NULLS LAST, a.id DESC",
		aliasedArticleColumns, ownedByUserSubquery(1, "a."))
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanArticlesWithFeed(rows)
}

func (r *repository) FetchLatest(ctx context.Context, limit int, feedURL string, userID int64) ([]domain.Article, error) {
	args := []any{userID}
	q := fmt.Sprintf(
		"SELECT %s, COALESCE(f.feed_url, '') FROM articles a LEFT JOIN feeds f ON a.feed_id = f.id WHERE %s",
		aliasedArticleColumns, ownedByUserSubquery(1, "a."))
	if feedURL != "" {
		args = append(args, feedURL)
		q += fmt.Sprintf(" AND f.feed_url = $%d", len(args))
	}
	args = append(args, limit)
	q += fmt.Sprintf(" ORDER BY a.published_at DESC NULLS LAST, a.id DESC LIMIT $%d", len(args))

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanArticlesWithFeed(rows)
}

func (r *repository) FindByID(ctx context.Context, id int64, userID int64) (*domain.Article, error) {
	q := fmt.Sprintf("SELECT %s FROM articles WHERE id = $1 AND %s", articleColumns, ownedByUserSubquery(2, ""))
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
	q := fmt.Sprintf(`UPDATE articles SET read = $1, bookmarked = $2 WHERE id = $3 AND %s`, ownedByUserSubquery(4, ""))
	_, err := r.db.ExecContext(ctx, q, a.Read, a.Bookmarked, a.ID, userID)
	return err
}

func (r *repository) MarkAsRead(ctx context.Context, ids []int64, userID int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids)+1)
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	args[len(ids)] = userID
	q := fmt.Sprintf(`UPDATE articles SET read = TRUE WHERE id IN (%s) AND %s`,
		strings.Join(placeholders, ","), ownedByUserSubquery(len(ids)+1, ""))
	_, err := r.db.ExecContext(ctx, q, args...)
	return err
}

func (r *repository) DeleteNonBookmarked(ctx context.Context, userID int64) (int64, error) {
	q := fmt.Sprintf(`DELETE FROM articles WHERE bookmarked = FALSE AND %s`, ownedByUserSubquery(1, ""))
	res, err := r.db.ExecContext(ctx, q, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *repository) CountNonBookmarked(ctx context.Context, userID int64) (int64, error) {
	var count int64
	q := fmt.Sprintf(`SELECT COUNT(*) FROM articles WHERE bookmarked = FALSE AND %s`, ownedByUserSubquery(1, ""))
	err := r.db.QueryRowContext(ctx, q, userID).Scan(&count)
	return count, err
}

// Search は ILIKE を使う。PostgresのLIKEはSQLiteと異なりASCII範囲でも大文字小文字を
// 区別するため、SQLite実装（LIKE）と挙動を揃えるためILIKEを使う。
func (r *repository) Search(ctx context.Context, keyword string, bookmarkedOnly bool, userID int64) ([]domain.Article, error) {
	q := fmt.Sprintf("SELECT %s FROM articles WHERE (title ILIKE $1 OR content ILIKE $2)", articleColumns)
	like := "%" + keyword + "%"
	args := []any{like, like}
	if bookmarkedOnly {
		q += " AND bookmarked = TRUE"
	}
	args = append(args, userID)
	q += fmt.Sprintf(" AND %s", ownedByUserSubquery(len(args), ""))
	q += " ORDER BY published_at DESC NULLS LAST, id DESC"
	return r.queryArgs(ctx, q, args...)
}

// UpdateEnrichmentBatch は複数件の要約・カテゴリ更新を1トランザクションでまとめて行う。
// SQLite実装と同様、1件でも失敗すればトランザクション全体をロールバックする。
func (r *repository) UpdateEnrichmentBatch(ctx context.Context, updates []articlerepo.EnrichmentUpdate, userID int64) error {
	if len(updates) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf(`UPDATE articles SET summary = $1, category = $2 WHERE id = $3 AND %s`, ownedByUserSubquery(4, "")))
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
	q := fmt.Sprintf("SELECT %s FROM articles WHERE (summary IS NULL OR summary = '') AND %s ORDER BY published_at DESC NULLS LAST, id DESC LIMIT $2", articleColumns, ownedByUserSubquery(1, ""))
	return r.queryArgs(ctx, q, userID, limit)
}

// UpdateMetadataBatch は既存記事への出版元・サムネイルのバックフィル用。
// SQLite実装（sql.Named）と異なり、Postgres実装では名前付きパラメータの互換性が
// 不確実なため位置引数（$1,$2,$3）で書く（design.md参照）。UpdateMetadataBatchのみ
// URL一致で全ユーザー横断に補完するためuserIDを取らない（理由はarticlerepo.Repository
// のインターフェースコメント参照）。
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
			publisher = CASE WHEN COALESCE(publisher, '') = '' AND $1 != '' THEN $1 ELSE publisher END,
			thumbnail_url = CASE WHEN COALESCE(thumbnail_url, '') = '' AND $2 != '' THEN $2 ELSE thumbnail_url END
		WHERE url = $3
			AND (
				(COALESCE(publisher, '') = '' AND $1 != '')
				OR (COALESCE(thumbnail_url, '') = '' AND $2 != '')
			)
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var total int64
	for _, u := range updates {
		res, err := stmt.ExecContext(ctx, u.Publisher, u.ThumbnailURL, u.URL)
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
	q := fmt.Sprintf(`SELECT COUNT(*) FROM articles WHERE bookmarked = TRUE AND %s`, ownedByUserSubquery(1, ""))
	err := r.db.QueryRowContext(ctx, q, userID).Scan(&count)
	return count, err
}

func (r *repository) FindFiltered(ctx context.Context, filter articlerepo.ListFilter, userID int64) ([]domain.Article, int64, error) {
	conditions := []string{ownedByUserSubquery(1, "")}
	args := []any{userID}

	if filter.Unread {
		conditions = append(conditions, "read = FALSE")
	}
	if filter.BookmarkedOnly {
		conditions = append(conditions, "bookmarked = TRUE")
	}
	if filter.Keyword != "" {
		// ILIKE: PostgresのLIKEはSQLiteと異なりASCII範囲でも大文字小文字を区別するため、
		// SQLite実装（LIKE）と挙動を揃える。
		like := "%" + filter.Keyword + "%"
		args = append(args, like, like)
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR content ILIKE $%d)", len(args)-1, len(args)))
	}
	if filter.Category != "" {
		args = append(args, filter.Category)
		conditions = append(conditions, fmt.Sprintf("category = $%d", len(args)))
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
	// NULLS FIRST/LAST: SQLiteはASCでNULLSFIRST・DESCでNULLSLASTがデフォルトだが、
	// Postgresはその逆（ASCでNULLSLAST・DESCでNULLSFIRST）がデフォルトのため、
	// sortColumn（category・publisher等はNULL/空になりうる）でSQLite側の挙動と揃えるために明示する。
	orderDir := "DESC"
	nullsOrder := "NULLS LAST"
	if filter.Order == "asc" {
		orderDir = "ASC"
		nullsOrder = "NULLS FIRST"
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 {
		perPage = articlerepo.DefaultPerPage
	}

	limitArgs := append(append([]any{}, args...), perPage, (page-1)*perPage)
	q := fmt.Sprintf("SELECT %s FROM articles%s ORDER BY %s %s %s, id %s LIMIT $%d OFFSET $%d",
		articleColumns, where, sortColumn, orderDir, nullsOrder, orderDir, len(limitArgs)-1, len(limitArgs))

	articles, err := r.queryArgs(ctx, q, limitArgs...)
	if err != nil {
		return nil, 0, err
	}
	return articles, total, nil
}

func (r *repository) DistinctCategories(ctx context.Context, userID int64) ([]string, error) {
	q := fmt.Sprintf(`SELECT DISTINCT category FROM articles WHERE category != '' AND %s ORDER BY category ASC`, ownedByUserSubquery(1, ""))
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
