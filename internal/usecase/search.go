package usecase

import (
	"context"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type SearchUsecase struct {
	articleRepo articlerepo.Repository
	userID      int64
}

func NewSearchUsecase(articleRepo articlerepo.Repository, userID int64) *SearchUsecase {
	return &SearchUsecase{articleRepo: articleRepo, userID: userID}
}

func (uc *SearchUsecase) Execute(ctx context.Context, keyword string, bookmarkedOnly bool) ([]domain.Article, error) {
	return uc.articleRepo.Search(ctx, keyword, bookmarkedOnly, uc.userID)
}

// SearchFilterOptions は Web API（検索）向けのカテゴリ・並び替え・ページネーション条件を表す。
type SearchFilterOptions struct {
	Keyword        string
	BookmarkedOnly bool
	Category       string
	Sort           string
	Order          string
	Page           int
	PerPage        int
}

func (uc *SearchUsecase) ExecuteFiltered(ctx context.Context, opts SearchFilterOptions) ([]domain.Article, int64, error) {
	return uc.articleRepo.FindFiltered(ctx, articlerepo.ListFilter{
		Keyword:        opts.Keyword,
		BookmarkedOnly: opts.BookmarkedOnly,
		Category:       opts.Category,
		Sort:           opts.Sort,
		Order:          opts.Order,
		Page:           opts.Page,
		PerPage:        opts.PerPage,
	}, uc.userID)
}
