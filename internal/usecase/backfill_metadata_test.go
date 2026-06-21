package usecase

import (
	"context"
	"errors"
	"testing"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type mockMetadataArticleRepo struct {
	mockArticleRepo
	updateFn func(ctx context.Context, updates []articlerepo.MetadataUpdate) (int64, error)
	gotCalls [][]articlerepo.MetadataUpdate
}

func (m *mockMetadataArticleRepo) UpdateMetadataBatch(ctx context.Context, updates []articlerepo.MetadataUpdate) (int64, error) {
	m.gotCalls = append(m.gotCalls, updates)
	if m.updateFn != nil {
		return m.updateFn(ctx, updates)
	}
	return int64(len(updates)), nil
}

func TestBackfillMetadataUsecase_UpdatesEachFeedFromFreshFetch(t *testing.T) {
	repo := &mockMetadataArticleRepo{}
	uc := NewBackfillMetadataUsecase(repo, &mockFeedRepo{
		listAllFn: func(_ context.Context) ([]domain.Feed, error) {
			return []domain.Feed{{FeedURL: "https://feed.example.com/a"}, {FeedURL: "https://feed.example.com/b"}}, nil
		},
	}, &mockRSSReader{
		articles: []domain.Article{
			{URL: "https://example.com/1", Publisher: "Example", ThumbnailURL: "https://example.com/thumb.jpg"},
		},
	})

	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Feeds) != 2 {
		t.Fatalf("Feeds: got %d, want 2", len(result.Feeds))
	}
	if result.TotalUpdated() != 2 {
		t.Errorf("TotalUpdated: got %d, want 2", result.TotalUpdated())
	}
	if len(repo.gotCalls) != 2 {
		t.Fatalf("UpdateMetadataBatch calls: got %d, want 2", len(repo.gotCalls))
	}
	got := repo.gotCalls[0][0]
	if got.URL != "https://example.com/1" || got.Publisher != "Example" || got.ThumbnailURL != "https://example.com/thumb.jpg" {
		t.Errorf("MetadataUpdate not passed through correctly: %+v", got)
	}
}

func TestBackfillMetadataUsecase_HandlesFetchError(t *testing.T) {
	repo := &mockMetadataArticleRepo{}
	uc := NewBackfillMetadataUsecase(repo, &mockFeedRepo{
		listAllFn: func(_ context.Context) ([]domain.Feed, error) {
			return []domain.Feed{{FeedURL: "https://feed.example.com/a"}}, nil
		},
	}, &mockRSSReader{err: errors.New("connection failed")})

	result, err := uc.Execute(context.Background())
	if err == nil {
		t.Error("expected error when feed fetch fails, got nil")
	}
	if result.TotalErrors() != 1 {
		t.Errorf("TotalErrors: got %d, want 1", result.TotalErrors())
	}
	if len(repo.gotCalls) != 0 {
		t.Errorf("UpdateMetadataBatch should not be called when fetch fails, got %d calls", len(repo.gotCalls))
	}
}

func TestBackfillMetadataUsecase_HandlesUpdateError(t *testing.T) {
	repo := &mockMetadataArticleRepo{
		updateFn: func(_ context.Context, _ []articlerepo.MetadataUpdate) (int64, error) {
			return 0, errors.New("disk full")
		},
	}
	uc := NewBackfillMetadataUsecase(repo, &mockFeedRepo{
		listAllFn: func(_ context.Context) ([]domain.Feed, error) {
			return []domain.Feed{{FeedURL: "https://feed.example.com/a"}}, nil
		},
	}, &mockRSSReader{articles: []domain.Article{{URL: "https://example.com/1"}}})

	result, err := uc.Execute(context.Background())
	if err == nil {
		t.Error("expected error when UpdateMetadataBatch fails, got nil")
	}
	if result.TotalErrors() != 1 {
		t.Errorf("TotalErrors: got %d, want 1", result.TotalErrors())
	}
}

func TestBackfillMetadataUsecase_ListFeedsError(t *testing.T) {
	uc := NewBackfillMetadataUsecase(&mockMetadataArticleRepo{}, &mockFeedRepo{
		listAllFn: func(_ context.Context) ([]domain.Feed, error) {
			return nil, errors.New("db unavailable")
		},
	}, &mockRSSReader{})

	result, err := uc.Execute(context.Background())
	if err == nil {
		t.Error("expected error when listing feeds fails, got nil")
	}
	if len(result.Feeds) != 0 {
		t.Errorf("Feeds: got %d, want 0", len(result.Feeds))
	}
}

func TestBackfillMetadataUsecase_NoFeeds(t *testing.T) {
	uc := NewBackfillMetadataUsecase(&mockMetadataArticleRepo{}, &mockFeedRepo{}, &mockRSSReader{})

	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Feeds) != 0 {
		t.Errorf("Feeds: got %d, want 0", len(result.Feeds))
	}
}
