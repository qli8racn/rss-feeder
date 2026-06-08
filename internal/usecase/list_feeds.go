package usecase

import (
	"context"

	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type ListFeedsUsecase struct {
	feedRepo feedrepo.Repository
}

func NewListFeedsUsecase(feedRepo feedrepo.Repository) *ListFeedsUsecase {
	return &ListFeedsUsecase{feedRepo: feedRepo}
}

func (uc *ListFeedsUsecase) Execute(ctx context.Context) ([]domain.Feed, error) {
	return uc.feedRepo.ListAll(ctx)
}
