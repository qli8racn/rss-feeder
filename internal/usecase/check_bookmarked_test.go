package usecase

import (
	"context"
	"testing"
)

func TestCheckBookmarkedUsecase_ReturnsCount(t *testing.T) {
	repo := &mockResetArticleRepo{bookmarked: 3}
	uc := NewCheckBookmarkedUsecase(repo, testUserID)

	count, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("count: got %d, want 3", count)
	}
}

func TestCheckBookmarkedUsecase_ZeroBookmarked(t *testing.T) {
	repo := &mockResetArticleRepo{bookmarked: 0}
	uc := NewCheckBookmarkedUsecase(repo, testUserID)

	count, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("count: got %d, want 0", count)
	}
}
