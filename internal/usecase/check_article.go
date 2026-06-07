package usecase

import (
	"context"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
)

type CheckArticleUsecase struct {
	articleRepo articlerepo.Repository
}

func NewCheckArticleUsecase(repo articlerepo.Repository) *CheckArticleUsecase {
	return &CheckArticleUsecase{articleRepo: repo}
}

func (uc *CheckArticleUsecase) Execute(ctx context.Context, id int64) error {
	_, err := findArticleByID(ctx, uc.articleRepo, id)
	return err
}
