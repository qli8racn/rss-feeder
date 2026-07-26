package article

import (
	"context"
	"errors"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

var ErrDuplicate = errors.New("duplicate article")

// DefaultPerPage は ListFilter.PerPage 未指定時のデフォルト値（ハンドラ層・ドライバ層で共有する）。
const DefaultPerPage = 25

// ValidSortFields は ListFilter.Sort に指定可能な値（ハンドラ層の検証とドライバ層のSQLカラム解決で共有する正とする定義）。
var ValidSortFields = map[string]bool{
	"title":        true,
	"publisher":    true,
	"category":     true,
	"published_at": true,
}

// ListFilter は Web API（記事一覧・検索）向けの絞り込み・並び替え・ページネーション条件を表す。
type ListFilter struct {
	Unread         bool
	BookmarkedOnly bool
	Keyword        string
	Category       string
	Sort           string // "title" | "publisher" | "category" | "published_at"
	Order          string // "asc" | "desc"
	Page           int
	PerPage        int
}

// EnrichmentUpdate は UpdateEnrichmentBatch に渡す1件分の更新内容。
type EnrichmentUpdate struct {
	ID       int64
	Summary  string
	Category string
}

// MetadataUpdate は UpdateMetadataBatch に渡す1件分の更新内容。
// URL で記事を特定する（バックフィル時はフィードから再取得した記事に DB 上の ID が無いため）。
type MetadataUpdate struct {
	URL          string
	Publisher    string
	ThumbnailURL string
}

// Repository は記事を操作する。一覧・検索・更新系のメソッドは userID でスコープし、
// JOIN feeds ON articles.feed_id = feeds.id AND feeds.user_id = ? を条件に加えることで
// 他ユーザーの記事IDを推測して操作されることを防ぐ（記事IDはユーザー横断の連番のため）。
//
// FetchLatest・UpdateEnrichmentBatch・FindWithoutSummary・UpdateMetadataBatch は
// enrich/summarize/backfill_metadataの対象記事選定・メタデータ補完に使われ、記事内容自体は
// 所有ユーザーによらず共通に扱って問題ないため、userIDでのスコープは行わない
// （docs/steering/20260726_mcp_user_management/design.md 参照）。
type Repository interface {
	Save(ctx context.Context, article domain.Article) error
	FindAll(ctx context.Context, userID int64) ([]domain.Article, error)
	FindUnread(ctx context.Context, userID int64) ([]domain.Article, error)
	FindBookmarked(ctx context.Context, userID int64) ([]domain.Article, error)
	FindByID(ctx context.Context, id int64, userID int64) (*domain.Article, error)
	Update(ctx context.Context, article domain.Article, userID int64) error
	MarkAsRead(ctx context.Context, ids []int64, userID int64) error
	DeleteNonBookmarked(ctx context.Context, userID int64) (int64, error)
	CountNonBookmarked(ctx context.Context, userID int64) (int64, error)
	CountBookmarked(ctx context.Context, userID int64) (int64, error)
	FetchLatest(ctx context.Context, limit int, feedURL string) ([]domain.Article, error)
	Search(ctx context.Context, keyword string, bookmarkedOnly bool, userID int64) ([]domain.Article, error)
	// UpdateEnrichmentBatch は複数件の要約・カテゴリ更新を1トランザクションでまとめて行う。
	UpdateEnrichmentBatch(ctx context.Context, updates []EnrichmentUpdate) error
	FindWithoutSummary(ctx context.Context, limit int) ([]domain.Article, error)
	// UpdateMetadataBatch は既存記事への出版元・サムネイルのバックフィル用。
	// 既に値が設定されている列は上書きせず、空文字の列のみ埋める。更新件数を返す。
	UpdateMetadataBatch(ctx context.Context, updates []MetadataUpdate) (int64, error)
	FindFiltered(ctx context.Context, filter ListFilter, userID int64) ([]domain.Article, int64, error)
	DistinctCategories(ctx context.Context, userID int64) ([]string, error)
}
