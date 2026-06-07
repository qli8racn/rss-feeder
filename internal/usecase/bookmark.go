package usecase

import (
	"context"
	"errors"
	"fmt"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

var ErrArticleNotFound = errors.New("article not found")

type BookmarkUsecase struct {
	articleRepo articlerepo.Repository
}

func NewBookmarkUsecase(articleRepo articlerepo.Repository) *BookmarkUsecase {
	return &BookmarkUsecase{articleRepo: articleRepo}
}

func (uc *BookmarkUsecase) Execute(ctx context.Context, id int64) (*domain.Article, error) {
	article, err := uc.articleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("article lookup failed: %w", err)
	}
	if article == nil {
		return nil, fmt.Errorf("ID %d: %w", id, ErrArticleNotFound)
	}

	article.ToggleBookmark()

	if err := uc.articleRepo.Update(ctx, *article); err != nil {
		return nil, fmt.Errorf("update failed: %w", err)
	}

	return article, nil
}
