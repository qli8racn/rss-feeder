package usecase

import (
	"context"
	"errors"
	"fmt"

	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
)

type AddFeedUsecase struct {
	feedRepo feedrepo.Repository
}

func NewAddFeedUsecase(feedRepo feedrepo.Repository) *AddFeedUsecase {
	return &AddFeedUsecase{feedRepo: feedRepo}
}

func (uc *AddFeedUsecase) Execute(ctx context.Context, url string) error {
	if err := uc.feedRepo.Register(ctx, url); err != nil {
		if errors.Is(err, feedrepo.ErrAlreadyExists) {
			return fmt.Errorf("すでに登録済みです %s: %w", url, err)
		}
		return err
	}
	return nil
}
