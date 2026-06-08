package usecase

import (
	"context"
	"errors"
	"fmt"

	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
)

type RemoveFeedUsecase struct {
	feedRepo feedrepo.Repository
}

func NewRemoveFeedUsecase(feedRepo feedrepo.Repository) *RemoveFeedUsecase {
	return &RemoveFeedUsecase{feedRepo: feedRepo}
}

func (uc *RemoveFeedUsecase) Execute(ctx context.Context, id int64) error {
	if err := uc.feedRepo.Remove(ctx, id); err != nil {
		if errors.Is(err, feedrepo.ErrNotFound) {
			return fmt.Errorf("フィードが見つかりません id=%d: %w", id, err)
		}
		return err
	}
	return nil
}
