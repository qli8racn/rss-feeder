package usecase

import (
	"context"
	"fmt"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
)

type ResetResult struct {
	Deleted    int64
	Bookmarked int64
}

type ResetUsecase struct {
	articleRepo articlerepo.Repository
	userID      int64
}

func NewResetUsecase(articleRepo articlerepo.Repository, userID int64) *ResetUsecase {
	return &ResetUsecase{articleRepo: articleRepo, userID: userID}
}

func (uc *ResetUsecase) Count(ctx context.Context) (int64, error) {
	return uc.articleRepo.CountNonBookmarked(ctx, uc.userID)
}

func (uc *ResetUsecase) Execute(ctx context.Context) (ResetResult, error) {
	deleted, err := uc.articleRepo.DeleteNonBookmarked(ctx, uc.userID)
	if err != nil {
		return ResetResult{}, fmt.Errorf("delete failed: %w", err)
	}

	bookmarked, err := uc.articleRepo.CountBookmarked(ctx, uc.userID)
	if err != nil {
		return ResetResult{}, fmt.Errorf("count failed: %w", err)
	}

	return ResetResult{Deleted: deleted, Bookmarked: bookmarked}, nil
}
