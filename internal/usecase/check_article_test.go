package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

func TestCheckArticleUsecase_Found(t *testing.T) {
	repo := &mockBookmarkArticleRepo{
		articles: map[int64]*domain.Article{
			1: {ID: 1, Title: "Test"},
		},
	}
	uc := NewCheckArticleUsecase(repo)

	if err := uc.Execute(context.Background(), 1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckArticleUsecase_NotFound(t *testing.T) {
	repo := &mockBookmarkArticleRepo{articles: map[int64]*domain.Article{}}
	uc := NewCheckArticleUsecase(repo)

	err := uc.Execute(context.Background(), 99)
	if !errors.Is(err, ErrArticleNotFound) {
		t.Errorf("expected ErrArticleNotFound, got %v", err)
	}
}

func TestCheckArticleUsecase_RepoError(t *testing.T) {
	repo := &mockBookmarkArticleRepo{findErr: errors.New("db error")}
	uc := NewCheckArticleUsecase(repo)

	if err := uc.Execute(context.Background(), 1); err == nil {
		t.Error("expected error, got nil")
	}
}
