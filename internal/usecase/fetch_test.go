package usecase

import (
	"context"
	"errors"
	"testing"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

// --- mocks ---

type mockArticleRepo struct {
	existing map[string]bool
}

func (m *mockArticleRepo) Save(_ context.Context, a domain.Article) error {
	if m.existing[a.URL] {
		return articlerepo.ErrDuplicate
	}
	return nil
}
func (m *mockArticleRepo) FindAll(_ context.Context) ([]domain.Article, error)    { return nil, nil }
func (m *mockArticleRepo) FindUnread(_ context.Context) ([]domain.Article, error) { return nil, nil }
func (m *mockArticleRepo) FindBookmarked(_ context.Context) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockArticleRepo) FindByID(_ context.Context, _ int64) (*domain.Article, error) {
	return nil, nil
}
func (m *mockArticleRepo) Update(_ context.Context, _ domain.Article) error     { return nil }
func (m *mockArticleRepo) MarkAsRead(_ context.Context, _ []int64) error        { return nil }
func (m *mockArticleRepo) DeleteNonBookmarked(_ context.Context) (int64, error) { return 0, nil }
func (m *mockArticleRepo) CountNonBookmarked(_ context.Context) (int64, error)  { return 0, nil }
func (m *mockArticleRepo) CountBookmarked(_ context.Context) (int64, error)     { return 0, nil }
func (m *mockArticleRepo) FetchLatest(_ context.Context, _ int, _ string) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockArticleRepo) Search(_ context.Context, _ string, _ bool) ([]domain.Article, error) {
	return nil, nil
}

type mockFeedRepo struct {
	registerFn func(ctx context.Context, url string) error
	listAllFn  func(ctx context.Context) ([]domain.Feed, error)
	removeFn   func(ctx context.Context, id int64) error
}

func (m *mockFeedRepo) Save(_ context.Context, _ domain.Feed) (int64, error)        { return 1, nil }
func (m *mockFeedRepo) FindByURL(_ context.Context, _ string) (*domain.Feed, error) { return nil, nil }
func (m *mockFeedRepo) Register(ctx context.Context, url string) error {
	if m.registerFn != nil {
		return m.registerFn(ctx, url)
	}
	return nil
}
func (m *mockFeedRepo) ListAll(ctx context.Context) ([]domain.Feed, error) {
	if m.listAllFn != nil {
		return m.listAllFn(ctx)
	}
	return nil, nil
}
func (m *mockFeedRepo) Remove(ctx context.Context, id int64) error {
	if m.removeFn != nil {
		return m.removeFn(ctx, id)
	}
	return nil
}

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
	uc := NewFetchUsecase(repo, &mockFeedRepo{}, &mockRSSReader{
		articles: []domain.Article{{URL: "https://example.com/1", Title: "Test"}},
	})

	result, err := uc.Execute(context.Background(), []string{"https://feed.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalSaved() != 0 {
		t.Errorf("Saved: got %d, want 0", result.TotalSaved())
	}
	if result.TotalSkipped() != 1 {
		t.Errorf("Skipped: got %d, want 1", result.TotalSkipped())
	}
}

func TestFetchUsecase_SavesNewArticles(t *testing.T) {
	uc := NewFetchUsecase(&mockArticleRepo{}, &mockFeedRepo{}, &mockRSSReader{
		articles: []domain.Article{
			{URL: "https://example.com/1", Title: "Article 1"},
			{URL: "https://example.com/2", Title: "Article 2"},
		},
	})

	result, err := uc.Execute(context.Background(), []string{"https://feed.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalSaved() != 2 {
		t.Errorf("Saved: got %d, want 2", result.TotalSaved())
	}
}

func TestFetchUsecase_HandlesReaderError(t *testing.T) {
	uc := NewFetchUsecase(&mockArticleRepo{}, &mockFeedRepo{}, &mockRSSReader{
		err: errors.New("connection failed"),
	})

	result, err := uc.Execute(context.Background(), []string{"https://feed.example.com"})
	if err == nil {
		t.Error("expected error when feed fetch fails, got nil")
	}
	if result.TotalErrors() != 1 {
		t.Errorf("Errors: got %d, want 1", result.TotalErrors())
	}
}

func TestFetchUsecase_NonDuplicateSaveError_CountsAsError(t *testing.T) {
	uc := NewFetchUsecase(
		&mockSaveErrRepo{},
		&mockFeedRepo{},
		&mockRSSReader{articles: []domain.Article{{URL: "https://example.com/1", Title: "A"}}},
	)

	result, err := uc.Execute(context.Background(), []string{"https://feed.example.com"})
	if err == nil {
		t.Error("expected error when article save fails, got nil")
	}
	if result.TotalErrors() != 1 {
		t.Errorf("Errors: got %d, want 1", result.TotalErrors())
	}
}

type mockSaveErrRepo struct{ mockArticleRepo }

func (m *mockSaveErrRepo) Save(_ context.Context, _ domain.Article) error {
	return errors.New("disk full")
}
