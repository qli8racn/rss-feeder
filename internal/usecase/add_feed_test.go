package usecase

import (
	"context"
	"errors"
	"testing"

	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
)

func TestAddFeedUsecase_RegistersNewFeed(t *testing.T) {
	var registered string
	uc := NewAddFeedUsecase(&mockFeedRepo{
		registerFn: func(_ context.Context, url string) error {
			registered = url
			return nil
		},
	})

	if err := uc.Execute(context.Background(), "https://example.com/feed"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registered != "https://example.com/feed" {
		t.Errorf("registered: got %q, want %q", registered, "https://example.com/feed")
	}
}

func TestAddFeedUsecase_AlreadyExists(t *testing.T) {
	uc := NewAddFeedUsecase(&mockFeedRepo{
		registerFn: func(_ context.Context, _ string) error {
			return feedrepo.ErrAlreadyExists
		},
	})

	err := uc.Execute(context.Background(), "https://example.com/feed")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, feedrepo.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestAddFeedUsecase_RepoError(t *testing.T) {
	repoErr := errors.New("db error")
	uc := NewAddFeedUsecase(&mockFeedRepo{
		registerFn: func(_ context.Context, _ string) error {
			return repoErr
		},
	})

	if err := uc.Execute(context.Background(), "https://example.com/feed"); !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}
