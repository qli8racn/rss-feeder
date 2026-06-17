package usecase

import (
	"context"
	"errors"
	"testing"
)

func TestListCategoriesUsecase_ReturnsCategories(t *testing.T) {
	repo := &mockListArticleRepo{categories: []string{"AI", "Tech"}}
	uc := NewListCategoriesUsecase(repo)

	categories, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(categories) != 2 || categories[0] != "AI" || categories[1] != "Tech" {
		t.Errorf("categories: got %+v", categories)
	}
}

func TestListCategoriesUsecase_RepoError(t *testing.T) {
	repo := &mockListArticleRepo{categoriesErr: errors.New("db error")}
	uc := NewListCategoriesUsecase(repo)

	if _, err := uc.Execute(context.Background()); err == nil {
		t.Error("expected error, got nil")
	}
}
