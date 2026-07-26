package usecase

import (
	"context"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
)

type ListCategoriesUsecase struct {
	articleRepo articlerepo.Repository
	userID      int64
}

func NewListCategoriesUsecase(articleRepo articlerepo.Repository, userID int64) *ListCategoriesUsecase {
	return &ListCategoriesUsecase{articleRepo: articleRepo, userID: userID}
}

func (uc *ListCategoriesUsecase) Execute(ctx context.Context) ([]string, error) {
	return uc.articleRepo.DistinctCategories(ctx, uc.userID)
}
