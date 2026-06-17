package usecase

import (
	"context"
	"errors"
	"testing"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type mockBookmarkArticleRepo struct {
	articles  map[int64]*domain.Article
	updated   []domain.Article
	findErr   error
	updateErr error
}

func (m *mockBookmarkArticleRepo) Save(_ context.Context, _ domain.Article) error { return nil }
func (m *mockBookmarkArticleRepo) FindAll(_ context.Context) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockBookmarkArticleRepo) FindUnread(_ context.Context) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockBookmarkArticleRepo) FindBookmarked(_ context.Context) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockBookmarkArticleRepo) FindByID(_ context.Context, id int64) (*domain.Article, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.articles[id], nil
}
func (m *mockBookmarkArticleRepo) Update(_ context.Context, a domain.Article) error {
	m.updated = append(m.updated, a)
	return m.updateErr
}
func (m *mockBookmarkArticleRepo) MarkAsRead(_ context.Context, _ []int64) error { return nil }
func (m *mockBookmarkArticleRepo) DeleteNonBookmarked(_ context.Context) (int64, error) {
	return 0, nil
}
func (m *mockBookmarkArticleRepo) CountNonBookmarked(_ context.Context) (int64, error) {
	return 0, nil
}
func (m *mockBookmarkArticleRepo) CountBookmarked(_ context.Context) (int64, error) {
	return 0, nil
}
func (m *mockBookmarkArticleRepo) FetchLatest(_ context.Context, _ int, _ string) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockBookmarkArticleRepo) UpdateEnrichment(_ context.Context, _ int64, _, _ string) error {
	return nil
}
func (m *mockBookmarkArticleRepo) FindWithoutSummary(_ context.Context, _ int) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockBookmarkArticleRepo) Search(_ context.Context, _ string, _ bool) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockBookmarkArticleRepo) FindFiltered(_ context.Context, _ articlerepo.ListFilter) ([]domain.Article, int64, error) {
	return nil, 0, nil
}
func (m *mockBookmarkArticleRepo) DistinctCategories(_ context.Context) ([]string, error) {
	return nil, nil
}

func TestBookmarkUsecase_Toggle_FalseToTrue(t *testing.T) {
	repo := &mockBookmarkArticleRepo{
		articles: map[int64]*domain.Article{
			1: {ID: 1, Title: "A", Bookmarked: false},
		},
	}
	uc := NewBookmarkUsecase(repo)

	article, err := uc.Execute(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !article.Bookmarked {
		t.Error("expected Bookmarked to be true after toggle")
	}
	if len(repo.updated) != 1 || !repo.updated[0].Bookmarked {
		t.Error("Update should have been called with Bookmarked=true")
	}
}

func TestBookmarkUsecase_Toggle_TrueToFalse(t *testing.T) {
	repo := &mockBookmarkArticleRepo{
		articles: map[int64]*domain.Article{
			1: {ID: 1, Title: "A", Bookmarked: true},
		},
	}
	uc := NewBookmarkUsecase(repo)

	article, err := uc.Execute(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if article.Bookmarked {
		t.Error("expected Bookmarked to be false after toggle")
	}
}

func TestBookmarkUsecase_NotFound(t *testing.T) {
	repo := &mockBookmarkArticleRepo{articles: map[int64]*domain.Article{}}
	uc := NewBookmarkUsecase(repo)

	_, err := uc.Execute(context.Background(), 99)
	if !errors.Is(err, ErrArticleNotFound) {
		t.Errorf("expected ErrArticleNotFound, got %v", err)
	}
}

func TestBookmarkUsecase_FindError(t *testing.T) {
	repo := &mockBookmarkArticleRepo{findErr: errors.New("db error")}
	uc := NewBookmarkUsecase(repo)

	_, err := uc.Execute(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
