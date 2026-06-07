package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// --- mocks ---

type mockArticleRepo struct {
	existing map[string]bool
}

func (m *mockArticleRepo) Save(_ context.Context, a domain.Article) error {
	if m.existing[a.URL] {
		return errors.New("duplicate")
	}
	return nil
}
func (m *mockArticleRepo) FindAll(_ context.Context) ([]domain.Article, error)          { return nil, nil }
func (m *mockArticleRepo) FindUnread(_ context.Context) ([]domain.Article, error)        { return nil, nil }
func (m *mockArticleRepo) FindBookmarked(_ context.Context) ([]domain.Article, error)    { return nil, nil }
func (m *mockArticleRepo) FindByID(_ context.Context, _ int64) (*domain.Article, error)  { return nil, nil }
func (m *mockArticleRepo) Update(_ context.Context, _ domain.Article) error              { return nil }
func (m *mockArticleRepo) DeleteNonBookmarked(_ context.Context) (int64, error)          { return 0, nil }
func (m *mockArticleRepo) CountNonBookmarked(_ context.Context) (int64, error)           { return 0, nil }

type mockFeedRepo struct{}

func (m *mockFeedRepo) Save(_ context.Context, _ domain.Feed) (int64, error)        { return 1, nil }
func (m *mockFeedRepo) FindByURL(_ context.Context, _ string) (*domain.Feed, error) { return nil, nil }

type mockRSSReader struct {
	articles []domain.Article
	err      error
}

func (m *mockRSSReader) Fetch(_ context.Context, _ string) (string, []domain.Article, error) {
	return "mock feed", m.articles, m.err
}

// --- tests ---

func TestFetchUsecase_SkipsDuplicates(t *testing.T) {
	repo := &mockArticleRepo{existing: map[string]bool{"https://example.com/1": true}}
	uc := usecase.NewFetchUsecase(repo, &mockFeedRepo{}, &mockRSSReader{
		articles: []domain.Article{{URL: "https://example.com/1", Title: "Test"}},
	})

	result, err := uc.Execute(context.Background(), []string{"https://feed.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Saved != 0 {
		t.Errorf("Saved: got %d, want 0", result.Saved)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped: got %d, want 1", result.Skipped)
	}
}

func TestFetchUsecase_SavesNewArticles(t *testing.T) {
	uc := usecase.NewFetchUsecase(&mockArticleRepo{}, &mockFeedRepo{}, &mockRSSReader{
		articles: []domain.Article{
			{URL: "https://example.com/1", Title: "Article 1"},
			{URL: "https://example.com/2", Title: "Article 2"},
		},
	})

	result, err := uc.Execute(context.Background(), []string{"https://feed.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Saved != 2 {
		t.Errorf("Saved: got %d, want 2", result.Saved)
	}
}

func TestFetchUsecase_HandlesReaderError(t *testing.T) {
	uc := usecase.NewFetchUsecase(&mockArticleRepo{}, &mockFeedRepo{}, &mockRSSReader{
		err: errors.New("connection failed"),
	})

	result, err := uc.Execute(context.Background(), []string{"https://feed.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Errors != 1 {
		t.Errorf("Errors: got %d, want 1", result.Errors)
	}
}
