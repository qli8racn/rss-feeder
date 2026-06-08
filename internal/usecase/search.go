package usecase

import (
	"context"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type SearchUsecase struct {
	articleRepo articlerepo.Repository
}

func NewSearchUsecase(articleRepo articlerepo.Repository) *SearchUsecase {
	return &SearchUsecase{articleRepo: articleRepo}
}

func (uc *SearchUsecase) Execute(ctx context.Context, keyword string, bookmarkedOnly bool) ([]domain.Article, error) {
	return uc.articleRepo.Search(ctx, keyword, bookmarkedOnly)
}
