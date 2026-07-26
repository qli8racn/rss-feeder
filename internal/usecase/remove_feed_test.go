package usecase

import (
	"context"
	"errors"
	"testing"

	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
)

func TestRemoveFeedUsecase_RemovesSuccessfully(t *testing.T) {
	var removedID int64
	uc := NewRemoveFeedUsecase(&mockFeedRepo{
		removeFn: func(_ context.Context, id int64) error {
			removedID = id
			return nil
		},
	}, testUserID)

	if err := uc.Execute(context.Background(), 42); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removedID != 42 {
		t.Errorf("removedID: got %d, want 42", removedID)
	}
}

func TestRemoveFeedUsecase_NotFound(t *testing.T) {
	uc := NewRemoveFeedUsecase(&mockFeedRepo{
		removeFn: func(_ context.Context, _ int64) error {
			return feedrepo.ErrNotFound
		},
	}, testUserID)

	if err := uc.Execute(context.Background(), 99); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRemoveFeedUsecase_RepoError(t *testing.T) {
	repoErr := errors.New("db error")
	uc := NewRemoveFeedUsecase(&mockFeedRepo{
		removeFn: func(_ context.Context, _ int64) error {
			return repoErr
		},
	}, testUserID)

	if err := uc.Execute(context.Background(), 1); !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}
