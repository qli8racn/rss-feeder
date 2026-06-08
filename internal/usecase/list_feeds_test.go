package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

func TestListFeedsUsecase_ReturnsFeeds(t *testing.T) {
	expected := []domain.Feed{
		{ID: 1, FeedURL: "https://example.com/feed"},
		{ID: 2, FeedURL: "https://other.com/rss"},
	}
	uc := NewListFeedsUsecase(&mockFeedRepo{
		listAllFn: func(_ context.Context) ([]domain.Feed, error) {
			return expected, nil
		},
	})

	feeds, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(feeds) != len(expected) {
		t.Errorf("len: got %d, want %d", len(feeds), len(expected))
	}
}

func TestListFeedsUsecase_ReturnsEmpty(t *testing.T) {
	uc := NewListFeedsUsecase(&mockFeedRepo{})

	feeds, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(feeds) != 0 {
		t.Errorf("expected empty, got %d feeds", len(feeds))
	}
}

func TestListFeedsUsecase_RepoError(t *testing.T) {
	repoErr := errors.New("db error")
	uc := NewListFeedsUsecase(&mockFeedRepo{
		listAllFn: func(_ context.Context) ([]domain.Feed, error) {
			return nil, repoErr
		},
	})

	if _, err := uc.Execute(context.Background()); !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}
