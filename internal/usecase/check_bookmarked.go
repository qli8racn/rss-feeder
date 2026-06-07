package usecase

import (
	"context"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
)

type CheckBookmarkedUsecase struct {
	articleRepo articlerepo.Repository
}

func NewCheckBookmarkedUsecase(repo articlerepo.Repository) *CheckBookmarkedUsecase {
	return &CheckBookmarkedUsecase{articleRepo: repo}
}

func (uc *CheckBookmarkedUsecase) Execute(ctx context.Context) (int64, error) {
	return uc.articleRepo.CountBookmarked(ctx)
}
