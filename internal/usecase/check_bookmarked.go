package usecase

import (
	"context"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
)

type CheckBookmarkedUsecase struct {
	articleRepo articlerepo.Repository
	userID      int64
}

func NewCheckBookmarkedUsecase(repo articlerepo.Repository, userID int64) *CheckBookmarkedUsecase {
	return &CheckBookmarkedUsecase{articleRepo: repo, userID: userID}
}

func (uc *CheckBookmarkedUsecase) Execute(ctx context.Context) (int64, error) {
	return uc.articleRepo.CountBookmarked(ctx, uc.userID)
}
