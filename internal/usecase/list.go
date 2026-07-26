package usecase

import (
	"context"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

// ListFilterOptions は Web API（記事一覧）向けの絞り込み・並び替え・ページネーション条件を表す。
type ListFilterOptions struct {
	Mode     ListMode
	Category string
	Sort     string
	Order    string
	Page     int
	PerPage  int
	// SkipMarkAsRead が true の場合、取得した記事を既読にマークする副作用を行わない
	// （MCPサーバーの list ツールなど、閲覧そのものを既読化のトリガーにしたくない呼び出し元向け）。
	SkipMarkAsRead bool
}

type ListMode int

const (
	ListModeUnread ListMode = iota
	ListModeAll
	ListModeBookmarked
)

type ListUsecase struct {
	articleRepo articlerepo.Repository
	userID      int64
}

func NewListUsecase(articleRepo articlerepo.Repository, userID int64) *ListUsecase {
	return &ListUsecase{articleRepo: articleRepo, userID: userID}
}

func (uc *ListUsecase) Execute(ctx context.Context, mode ListMode) ([]domain.Article, error) {
	var (
		articles []domain.Article
		err      error
	)

	switch mode {
	case ListModeAll:
		articles, err = uc.articleRepo.FindAll(ctx, uc.userID)
	case ListModeBookmarked:
		articles, err = uc.articleRepo.FindBookmarked(ctx, uc.userID)
	default:
		articles, err = uc.articleRepo.FindUnread(ctx, uc.userID)
	}
	if err != nil {
		return nil, err
	}

	if err := uc.markUnreadAsRead(ctx, articles); err != nil {
		return nil, err
	}

	return articles, nil
}

// ExecuteFiltered は Web API 向けに、カテゴリ・並び替え・ページネーションに対応した記事一覧を取得する。
// Execute と同様、取得した記事のうち未読のものを既読としてマークする。
func (uc *ListUsecase) ExecuteFiltered(ctx context.Context, opts ListFilterOptions) ([]domain.Article, int64, error) {
	filter := articlerepo.ListFilter{
		Category: opts.Category,
		Sort:     opts.Sort,
		Order:    opts.Order,
		Page:     opts.Page,
		PerPage:  opts.PerPage,
	}
	switch opts.Mode {
	case ListModeBookmarked:
		filter.BookmarkedOnly = true
	case ListModeUnread:
		filter.Unread = true
	}

	articles, total, err := uc.articleRepo.FindFiltered(ctx, filter, uc.userID)
	if err != nil {
		return nil, 0, err
	}

	if !opts.SkipMarkAsRead {
		if err := uc.markUnreadAsRead(ctx, articles); err != nil {
			return nil, 0, err
		}
	}

	return articles, total, nil
}

func (uc *ListUsecase) markUnreadAsRead(ctx context.Context, articles []domain.Article) error {
	var unreadIDs []int64
	for _, a := range articles {
		if !a.Read {
			unreadIDs = append(unreadIDs, a.ID)
		}
	}
	if len(unreadIDs) == 0 {
		return nil
	}
	return uc.articleRepo.MarkAsRead(ctx, unreadIDs, uc.userID)
}
