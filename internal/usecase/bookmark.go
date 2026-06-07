package usecase

import (
	"context"
	"errors"
	"fmt"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

var ErrArticleNotFound = errors.New("article not found")

func findArticleByID(ctx context.Context, repo articlerepo.Repository, id int64) (*domain.Article, error) {
	article, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, fmt.Errorf("ID %d: %w", id, ErrArticleNotFound)
	}
	return article, nil
}

type BookmarkUsecase struct {
	articleRepo articlerepo.Repository
}

func NewBookmarkUsecase(articleRepo articlerepo.Repository) *BookmarkUsecase {
	return &BookmarkUsecase{articleRepo: articleRepo}
}

func (uc *BookmarkUsecase) Execute(ctx context.Context, id int64) (*domain.Article, error) {
	article, err := findArticleByID(ctx, uc.articleRepo, id)
	if err != nil {
		return nil, err
	}

	article.ToggleBookmark()

	if err := uc.articleRepo.Update(ctx, *article); err != nil {
		return nil, fmt.Errorf("update failed: %w", err)
	}

	return article, nil
}
