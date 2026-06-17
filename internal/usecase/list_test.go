package usecase

import (
	"context"
	"testing"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type mockListArticleRepo struct {
	all           []domain.Article
	unread        []domain.Article
	bookmarked    []domain.Article
	markedIDs     []int64 // IDs passed to MarkAsRead
	findErr       error
	markErr       error
	filtered      []domain.Article
	filteredTotal int64
	filteredErr   error
	gotFilter     articlerepo.ListFilter
	categories    []string
	categoriesErr error
}

func (m *mockListArticleRepo) Save(_ context.Context, _ domain.Article) error { return nil }
func (m *mockListArticleRepo) FindAll(_ context.Context) ([]domain.Article, error) {
	return m.all, m.findErr
}
func (m *mockListArticleRepo) FindUnread(_ context.Context) ([]domain.Article, error) {
	return m.unread, m.findErr
}
func (m *mockListArticleRepo) FindBookmarked(_ context.Context) ([]domain.Article, error) {
	return m.bookmarked, m.findErr
}
func (m *mockListArticleRepo) FindByID(_ context.Context, _ int64) (*domain.Article, error) {
	return nil, nil
}
func (m *mockListArticleRepo) Update(_ context.Context, _ domain.Article) error { return nil }
func (m *mockListArticleRepo) MarkAsRead(_ context.Context, ids []int64) error {
	m.markedIDs = append(m.markedIDs, ids...)
	return m.markErr
}
func (m *mockListArticleRepo) DeleteNonBookmarked(_ context.Context) (int64, error) { return 0, nil }
func (m *mockListArticleRepo) CountNonBookmarked(_ context.Context) (int64, error)  { return 0, nil }
func (m *mockListArticleRepo) CountBookmarked(_ context.Context) (int64, error)     { return 0, nil }
func (m *mockListArticleRepo) FetchLatest(_ context.Context, _ int, _ string) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockListArticleRepo) Search(_ context.Context, _ string, _ bool) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockListArticleRepo) UpdateEnrichment(_ context.Context, _ int64, _, _ string) error {
	return nil
}
func (m *mockListArticleRepo) FindWithoutSummary(_ context.Context, _ int) ([]domain.Article, error) {
	return nil, nil
}
func (m *mockListArticleRepo) FindFiltered(_ context.Context, filter articlerepo.ListFilter) ([]domain.Article, int64, error) {
	m.gotFilter = filter
	return m.filtered, m.filteredTotal, m.filteredErr
}
func (m *mockListArticleRepo) DistinctCategories(_ context.Context) ([]string, error) {
	return m.categories, m.categoriesErr
}

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

	if _, err := uc.Execute(context.Background(), ListModeUnread); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(repo.markedIDs) != 2 {
		t.Errorf("markedIDs count: got %d, want 2", len(repo.markedIDs))
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

	if _, err := uc.Execute(context.Background(), ListModeAll); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(repo.markedIDs) != 1 {
		t.Errorf("markedIDs count: got %d, want 1 (already-read should be skipped)", len(repo.markedIDs))
	}
	if repo.markedIDs[0] != 2 {
		t.Errorf("expected article ID 2 to be marked, got %d", repo.markedIDs[0])
	}
}

func TestListUsecase_ExecuteFiltered_ModeMapping(t *testing.T) {
	repo := &mockListArticleRepo{
		filtered:      []domain.Article{{ID: 1, Title: "A", Read: true}},
		filteredTotal: 1,
	}
	uc := NewListUsecase(repo)

	if _, _, err := uc.ExecuteFiltered(context.Background(), ListFilterOptions{Mode: ListModeBookmarked}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.gotFilter.BookmarkedOnly {
		t.Error("expected BookmarkedOnly to be true for ListModeBookmarked")
	}
	if repo.gotFilter.Unread {
		t.Error("expected Unread to be false for ListModeBookmarked")
	}

	if _, _, err := uc.ExecuteFiltered(context.Background(), ListFilterOptions{Mode: ListModeUnread}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.gotFilter.Unread {
		t.Error("expected Unread to be true for ListModeUnread")
	}

	if _, _, err := uc.ExecuteFiltered(context.Background(), ListFilterOptions{Mode: ListModeAll}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotFilter.Unread || repo.gotFilter.BookmarkedOnly {
		t.Error("expected no read/bookmarked filter for ListModeAll")
	}
}

func TestListUsecase_ExecuteFiltered_PassesThroughOptionsAndTotal(t *testing.T) {
	repo := &mockListArticleRepo{
		filtered:      []domain.Article{{ID: 1, Title: "A", Read: true}},
		filteredTotal: 42,
	}
	uc := NewListUsecase(repo)

	articles, total, err := uc.ExecuteFiltered(context.Background(), ListFilterOptions{
		Category: "Tech",
		Sort:     "title",
		Order:    "asc",
		Page:     2,
		PerPage:  10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 42 {
		t.Errorf("total: got %d, want 42", total)
	}
	if len(articles) != 1 {
		t.Errorf("articles: got %d, want 1", len(articles))
	}
	if repo.gotFilter.Category != "Tech" || repo.gotFilter.Sort != "title" || repo.gotFilter.Order != "asc" || repo.gotFilter.Page != 2 || repo.gotFilter.PerPage != 10 {
		t.Errorf("filter not passed through correctly: %+v", repo.gotFilter)
	}
}

func TestListUsecase_ExecuteFiltered_MarksUnreadAsRead(t *testing.T) {
	repo := &mockListArticleRepo{
		filtered: []domain.Article{
			{ID: 1, Title: "A", Read: false},
			{ID: 2, Title: "B", Read: true},
		},
	}
	uc := NewListUsecase(repo)

	if _, _, err := uc.ExecuteFiltered(context.Background(), ListFilterOptions{Mode: ListModeAll}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.markedIDs) != 1 || repo.markedIDs[0] != 1 {
		t.Errorf("markedIDs: got %+v, want [1]", repo.markedIDs)
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
