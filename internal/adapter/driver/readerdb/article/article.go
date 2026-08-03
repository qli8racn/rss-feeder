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

// Repository は記事を操作する。ほぼ全メソッドを userID でスコープし、
// feed_id IN (SELECT id FROM feeds WHERE user_id = ?) を条件に加えることで
// 他ユーザーの記事IDを推測して操作されることを防ぐ（記事IDはユーザー横断の連番のため）。
// FetchLatest・FindWithoutSummary・UpdateEnrichmentBatch は enrich/summarize/curate の
// 対象記事選定・要約結果の保存に使われ、他ユーザーの記事が課金・処理対象に混入することを
// 防ぐためスコープが必須（過去にenrich.goのみ対応漏れがあった不具合を踏まえ全Agentで統一する）。
//
// UpdateMetadataBatch のみ URL 一致で全ユーザー横断に補完するため userID を取らない
// （メタデータ補完はどのユーザーの記事に適用しても実害がなく、むしろ複数ユーザーが同じ
// 外部記事URLを保存している場合に全員分を一度に補完できる方が望ましいため。
// docs/steering/20260726_mcp_user_management/design.md 参照）。
type Repository interface {
	// Save は article.FeedID の所有権チェックを行わない。呼び出し元が事前に
	// userIDスコープの feedRepo（例: feedRepo.Save の戻り値）から取得した FeedID を
	// 渡すことが前提（design.md参照）。
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
	FetchLatest(ctx context.Context, limit int, feedURL string, userID int64) ([]domain.Article, error)
	Search(ctx context.Context, keyword string, bookmarkedOnly bool, userID int64) ([]domain.Article, error)
	// UpdateEnrichmentBatch は複数件の要約・カテゴリ更新を1トランザクションでまとめて行う。
	UpdateEnrichmentBatch(ctx context.Context, updates []EnrichmentUpdate, userID int64) error
	FindWithoutSummary(ctx context.Context, limit int, userID int64) ([]domain.Article, error)
	// UpdateMetadataBatch は既存記事への出版元・サムネイルのバックフィル用。
	// 既に値が設定されている列は上書きせず、空文字の列のみ埋める。更新件数を返す。
	UpdateMetadataBatch(ctx context.Context, updates []MetadataUpdate) (int64, error)
	FindFiltered(ctx context.Context, filter ListFilter, userID int64) ([]domain.Article, int64, error)
	DistinctCategories(ctx context.Context, userID int64) ([]string, error)
}
