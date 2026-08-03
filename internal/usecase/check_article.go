package usecase

import (
	"context"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
)

type CheckArticleUsecase struct {
	articleRepo articlerepo.Repository
	userID      int64
}

func NewCheckArticleUsecase(repo articlerepo.Repository, userID int64) *CheckArticleUsecase {
	return &CheckArticleUsecase{articleRepo: repo, userID: userID}
}

func (uc *CheckArticleUsecase) Execute(ctx context.Context, id int64) error {
	_, err := findArticleByID(ctx, uc.articleRepo, id, uc.userID)
	return err
}
