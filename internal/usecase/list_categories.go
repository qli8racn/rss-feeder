package usecase

import (
	"context"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
)

type ListCategoriesUsecase struct {
	articleRepo articlerepo.Repository
}

func NewListCategoriesUsecase(articleRepo articlerepo.Repository) *ListCategoriesUsecase {
	return &ListCategoriesUsecase{articleRepo: articleRepo}
}

func (uc *ListCategoriesUsecase) Execute(ctx context.Context) ([]string, error) {
	return uc.articleRepo.DistinctCategories(ctx)
}
