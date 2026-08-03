package usecase

import (
	"context"

	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type ListFeedsUsecase struct {
	feedRepo feedrepo.Repository
	userID   int64
}

func NewListFeedsUsecase(feedRepo feedrepo.Repository, userID int64) *ListFeedsUsecase {
	return &ListFeedsUsecase{feedRepo: feedRepo, userID: userID}
}

func (uc *ListFeedsUsecase) Execute(ctx context.Context) ([]domain.Feed, error) {
	return uc.feedRepo.ListAll(ctx, uc.userID)
}
