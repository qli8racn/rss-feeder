package usecase

import (
	"context"
	"testing"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

type mockListArticleRepo struct {
	all        []domain.Article
	unread     []domain.Article
	bookmarked []domain.Article
	updated    []domain.Article
	findErr    error
	updateErr  error
}

func (m *mockListArticleRepo) Save(_ context.Context, _ domain.Article) error        { return nil }
func (m *mockListArticleRepo) FindAll(_ context.Context) ([]domain.Article, error)   { return m.all, m.findErr }
func (m *mockListArticleRepo) FindUnread(_ context.Context) ([]domain.Article, error) {
	return m.unread, m.findErr
}
func (m *mockListArticleRepo) FindBookmarked(_ context.Context) ([]domain.Article, error) {
	return m.bookmarked, m.findErr
}
func (m *mockListArticleRepo) FindByID(_ context.Context, _ int64) (*domain.Article, error) {
	return nil, nil
}
func (m *mockListArticleRepo) Update(_ context.Context, a domain.Article) error {
	m.updated = append(m.updated, a)
	return m.updateErr
}
func (m *mockListArticleRepo) DeleteNonBookmarked(_ context.Context) (int64, error) { return 0, nil }
func (m *mockListArticleRepo) CountNonBookmarked(_ context.Context) (int64, error)  { return 0, nil }

func TestListUsecase_DefaultMode_ReturnsUnread(t *testing.T) {
	repo := &mockListArticleRepo{
		unread: []domain.Article{
			{ID: 1, Title: "A", Read: false},
			{ID: 2, Title: "B", Read: false},
		},
	}
	uc := NewListUsecase(repo)

	articles, err := uc.Execute(context.Background(), ListModeUnread)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 2 {
		t.Errorf("count: got %d, want 2", len(articles))
	}
}

func TestListUsecase_AllMode(t *testing.T) {
	repo := &mockListArticleRepo{
		all: []domain.Article{
			{ID: 1, Title: "A", Read: true},
			{ID: 2, Title: "B", Read: false},
		},
	}
	uc := NewListUsecase(repo)

	articles, err := uc.Execute(context.Background(), ListModeAll)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 2 {
		t.Errorf("count: got %d, want 2", len(articles))
	}
}

func TestListUsecase_BookmarkedMode(t *testing.T) {
	repo := &mockListArticleRepo{
		bookmarked: []domain.Article{
			{ID: 3, Title: "C", Bookmarked: true},
		},
	}
	uc := NewListUsecase(repo)

	articles, err := uc.Execute(context.Background(), ListModeBookmarked)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 1 {
		t.Errorf("count: got %d, want 1", len(articles))
	}
}

func TestListUsecase_MarksUnreadAsRead(t *testing.T) {
	repo := &mockListArticleRepo{
		unread: []domain.Article{
			{ID: 1, Title: "A", Read: false},
			{ID: 2, Title: "B", Read: false},
		},
	}
	uc := NewListUsecase(repo)

	uc.Execute(context.Background(), ListModeUnread)

	if len(repo.updated) != 2 {
		t.Errorf("updated count: got %d, want 2", len(repo.updated))
	}
	for _, a := range repo.updated {
		if !a.Read {
			t.Errorf("article %d should be marked as read", a.ID)
		}
	}
}

func TestListUsecase_SkipsAlreadyRead(t *testing.T) {
	repo := &mockListArticleRepo{
		all: []domain.Article{
			{ID: 1, Title: "A", Read: true},
			{ID: 2, Title: "B", Read: false},
		},
	}
	uc := NewListUsecase(repo)

	uc.Execute(context.Background(), ListModeAll)

	if len(repo.updated) != 1 {
		t.Errorf("updated count: got %d, want 1 (already-read should be skipped)", len(repo.updated))
	}
	if repo.updated[0].ID != 2 {
		t.Errorf("expected article 2 to be updated, got %d", repo.updated[0].ID)
	}
}

func TestListUsecase_EmptyList(t *testing.T) {
	repo := &mockListArticleRepo{unread: []domain.Article{}}
	uc := NewListUsecase(repo)

	articles, err := uc.Execute(context.Background(), ListModeUnread)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 0 {
		t.Errorf("expected empty list, got %d", len(articles))
	}
}
