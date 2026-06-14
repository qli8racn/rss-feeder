package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

type mockResetArticleRepo struct {
	nonBookmarked int64
	bookmarked    int64
	deleteErr     error
}

func (m *mockResetArticleRepo) Save(_ context.Context, _ domain.Article) error      { return nil }
func (m *mockResetArticleRepo) FindAll(_ context.Context) ([]domain.Article, error) { return nil, nil }
func (m *mockResetArticleRepo) FindUnread(_ context.Context) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockResetArticleRepo) FindBookmarked(_ context.Context) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockResetArticleRepo) FindByID(_ context.Context, _ int64) (*domain.Article, error) {
	return nil, nil
}
func (m *mockResetArticleRepo) Update(_ context.Context, _ domain.Article) error { return nil }
func (m *mockResetArticleRepo) MarkAsRead(_ context.Context, _ []int64) error    { return nil }
func (m *mockResetArticleRepo) DeleteNonBookmarked(_ context.Context) (int64, error) {
	if m.deleteErr != nil {
		return 0, m.deleteErr
	}
	n := m.nonBookmarked
	m.nonBookmarked = 0
	return n, nil
}
func (m *mockResetArticleRepo) CountNonBookmarked(_ context.Context) (int64, error) {
	return m.nonBookmarked, nil
}
func (m *mockResetArticleRepo) CountBookmarked(_ context.Context) (int64, error) {
	return m.bookmarked, nil
}
func (m *mockResetArticleRepo) FetchLatest(_ context.Context, _ int, _ string) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockResetArticleRepo) Search(_ context.Context, _ string, _ bool) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockResetArticleRepo) UpdateEnrichment(_ context.Context, _ int64, _, _ string) error {
	return nil
}
func (m *mockResetArticleRepo) FindWithoutSummary(_ context.Context, _ int) ([]domain.Article, error) {
	return nil, nil
}

func TestResetUsecase_Count(t *testing.T) {
	repo := &mockResetArticleRepo{nonBookmarked: 38}
	uc := NewResetUsecase(repo)

	n, err := uc.Count(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 38 {
		t.Errorf("Count: got %d, want 38", n)
	}
}

func TestResetUsecase_Execute(t *testing.T) {
	repo := &mockResetArticleRepo{nonBookmarked: 38, bookmarked: 5}
	uc := NewResetUsecase(repo)

	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Deleted != 38 {
		t.Errorf("Deleted: got %d, want 38", result.Deleted)
	}
	if result.Bookmarked != 5 {
		t.Errorf("Bookmarked: got %d, want 5", result.Bookmarked)
	}
}

func TestResetUsecase_Execute_DeleteError(t *testing.T) {
	repo := &mockResetArticleRepo{deleteErr: errors.New("db error")}
	uc := NewResetUsecase(repo)

	_, err := uc.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResetUsecase_Execute_NothingToDelete(t *testing.T) {
	repo := &mockResetArticleRepo{nonBookmarked: 0, bookmarked: 3}
	uc := NewResetUsecase(repo)

	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Deleted != 0 {
		t.Errorf("Deleted: got %d, want 0", result.Deleted)
	}
}
