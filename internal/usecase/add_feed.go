package usecase

import (
	"context"
	"errors"
	"fmt"

	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type AddFeedUsecase struct {
	feedRepo feedrepo.Repository
	userID   int64
}

func NewAddFeedUsecase(feedRepo feedrepo.Repository, userID int64) *AddFeedUsecase {
	return &AddFeedUsecase{feedRepo: feedRepo, userID: userID}
}

func (uc *AddFeedUsecase) Execute(ctx context.Context, url string) (*domain.Feed, error) {
	if err := uc.feedRepo.Register(ctx, url, uc.userID); err != nil {
		if errors.Is(err, feedrepo.ErrAlreadyExists) {
			return nil, fmt.Errorf("すでに登録済みです %s: %w", url, err)
		}
		return nil, err
	}
	feed, err := uc.feedRepo.FindByURL(ctx, url, uc.userID)
	if err != nil {
		return nil, err
	}
	if feed == nil {
		// Register 成功直後なので通常到達しないが、FindByURL は未存在時に (nil, nil) を返す実装
		// (internal/driver/readerdb/feed/feed.go) のため、nil ポインタを呼び出し元に渡さないよう防御する。
		return nil, fmt.Errorf("フィード登録直後の取得に失敗しました %s", url)
	}
	return feed, nil
}
