package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

type mockSearchArticleRepo struct {
	results []domain.Article
	err     error
	// captured args
	gotKeyword        string
	gotBookmarkedOnly bool
}

func (m *mockSearchArticleRepo) Save(_ context.Context, _ domain.Article) error { return nil }
func (m *mockSearchArticleRepo) FindAll(_ context.Context) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockSearchArticleRepo) FindUnread(_ context.Context) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockSearchArticleRepo) FindBookmarked(_ context.Context) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockSearchArticleRepo) FindByID(_ context.Context, _ int64) (*domain.Article, error) {
	return nil, nil
}
func (m *mockSearchArticleRepo) Update(_ context.Context, _ domain.Article) error { return nil }
func (m *mockSearchArticleRepo) MarkAsRead(_ context.Context, _ []int64) error    { return nil }
func (m *mockSearchArticleRepo) DeleteNonBookmarked(_ context.Context) (int64, error) {
	return 0, nil
}
func (m *mockSearchArticleRepo) CountNonBookmarked(_ context.Context) (int64, error) { return 0, nil }
func (m *mockSearchArticleRepo) CountBookmarked(_ context.Context) (int64, error)    { return 0, nil }
func (m *mockSearchArticleRepo) FetchLatest(_ context.Context, _ int, _ string) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockSearchArticleRepo) Search(_ context.Context, keyword string, bookmarkedOnly bool) ([]domain.Article, error) {
	m.gotKeyword = keyword
	m.gotBookmarkedOnly = bookmarkedOnly
	return m.results, m.err
}
func (m *mockSearchArticleRepo) UpdateEnrichment(_ context.Context, _ int64, _, _ string) error {
	return nil
}
func (m *mockSearchArticleRepo) FindWithoutSummary(_ context.Context, _ int) ([]domain.Article, error) {
	return nil, nil
}

func TestSearchUsecase_ReturnsMatchingArticles(t *testing.T) {
	repo := &mockSearchArticleRepo{
		results: []domain.Article{
			{ID: 1, Title: "Go言語の記事"},
			{ID: 2, Title: "Go入門"},
		},
	}
	uc := NewSearchUsecase(repo)

	articles, err := uc.Execute(context.Background(), "Go", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 2 {
		t.Errorf("count: got %d, want 2", len(articles))
	}
	if repo.gotKeyword != "Go" {
		t.Errorf("keyword: got %q, want %q", repo.gotKeyword, "Go")
	}
	if repo.gotBookmarkedOnly != false {
		t.Error("bookmarkedOnly should be false")
	}
}

func TestSearchUsecase_ZeroResults(t *testing.T) {
	repo := &mockSearchArticleRepo{results: []domain.Article{}}
	uc := NewSearchUsecase(repo)

	articles, err := uc.Execute(context.Background(), "nomatch", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 0 {
		t.Errorf("expected empty result, got %d", len(articles))
	}
}

func TestSearchUsecase_BookmarkedOnlyFilter(t *testing.T) {
	repo := &mockSearchArticleRepo{
		results: []domain.Article{
			{ID: 3, Title: "お気に入り記事", Bookmarked: true},
		},
	}
	uc := NewSearchUsecase(repo)

	articles, err := uc.Execute(context.Background(), "記事", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 1 {
		t.Errorf("count: got %d, want 1", len(articles))
	}
	if repo.gotBookmarkedOnly != true {
		t.Error("bookmarkedOnly should be true")
	}
}

func TestSearchUsecase_RepoError(t *testing.T) {
	repo := &mockSearchArticleRepo{err: errors.New("db error")}
	uc := NewSearchUsecase(repo)

	_, err := uc.Execute(context.Background(), "keyword", false)
	if err == nil {
		t.Error("expected error, got nil")
	}
}
