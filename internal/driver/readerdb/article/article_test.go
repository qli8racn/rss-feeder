package article

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/qli8racn/rss-feeder/internal/migration"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	if err := migration.Run(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	// insert a feed so articles can reference it
	if _, err := db.Exec(`INSERT INTO feeds (id, feed_url, title) VALUES (1, 'https://example.com/feed', 'Test')`); err != nil {
		t.Fatalf("setup feed: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	})
	return db
}

func newRepo(t *testing.T) *repository {
	t.Helper()
	return &repository{db: newTestDB(t)}
}

func makeArticle(url string) domain.Article {
	return domain.Article{FeedID: 1, URL: url, Title: "Title " + url, Content: "body"}
}

func TestArticleRepository_Save_InsertNew(t *testing.T) {
	r := newRepo(t)
	err := r.Save(context.Background(), makeArticle("https://example.com/1"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func TestArticleRepository_Save_Duplicate(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	err := r.Save(ctx, makeArticle("https://example.com/1"))
	if !errors.Is(err, articlerepo.ErrDuplicate) {
		t.Errorf("expected ErrDuplicate, got %v", err)
	}
}

func TestArticleRepository_FindAll(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/2")); err != nil {
		t.Fatalf("Save: %v", err)
	}

	articles, err := r.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(articles) != 2 {
		t.Errorf("count: got %d, want 2", len(articles))
	}
}

func TestArticleRepository_FindUnread(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/2")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := r.db.Exec(`UPDATE articles SET read = 1 WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup read: %v", err)
	}

	articles, err := r.FindUnread(ctx)
	if err != nil {
		t.Fatalf("FindUnread: %v", err)
	}
	if len(articles) != 1 {
		t.Errorf("count: got %d, want 1", len(articles))
	}
	if articles[0].URL != "https://example.com/2" {
		t.Errorf("URL: got %q", articles[0].URL)
	}
}

func TestArticleRepository_FindBookmarked(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/2")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := r.db.Exec(`UPDATE articles SET bookmarked = 1 WHERE url = 'https://example.com/2'`); err != nil {
		t.Fatalf("setup bookmarked: %v", err)
	}

	articles, err := r.FindBookmarked(ctx)
	if err != nil {
		t.Fatalf("FindBookmarked: %v", err)
	}
	if len(articles) != 1 {
		t.Errorf("count: got %d, want 1", len(articles))
	}
	if articles[0].URL != "https://example.com/2" {
		t.Errorf("URL: got %q", articles[0].URL)
	}
}

func TestArticleRepository_FindByID(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	all, _ := r.FindAll(ctx)
	id := all[0].ID

	found, err := r.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found == nil {
		t.Fatal("expected article, got nil")
	}
	if found.ID != id {
		t.Errorf("ID: got %d, want %d", found.ID, id)
	}
}

func TestArticleRepository_FindByID_NotExists(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	found, err := r.FindByID(ctx, 9999)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil, got %+v", found)
	}
}

func TestArticleRepository_Update(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	all, _ := r.FindAll(ctx)
	a := all[0]

	a.Read = true
	a.Bookmarked = true
	if err := r.Update(ctx, a); err != nil {
		t.Fatalf("Update: %v", err)
	}

	found, _ := r.FindByID(ctx, a.ID)
	if !found.Read {
		t.Error("Read should be true after Update")
	}
	if !found.Bookmarked {
		t.Error("Bookmarked should be true after Update")
	}
}

func TestArticleRepository_DeleteNonBookmarked(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/2")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := r.db.Exec(`UPDATE articles SET bookmarked = 1 WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup bookmarked: %v", err)
	}

	n, err := r.DeleteNonBookmarked(ctx)
	if err != nil {
		t.Fatalf("DeleteNonBookmarked: %v", err)
	}
	if n != 1 {
		t.Errorf("deleted: got %d, want 1", n)
	}

	all, _ := r.FindAll(ctx)
	if len(all) != 1 {
		t.Errorf("remaining: got %d, want 1", len(all))
	}
}

func TestArticleRepository_MarkAsRead(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/2")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/3")); err != nil {
		t.Fatalf("Save: %v", err)
	}

	all, _ := r.FindAll(ctx)
	ids := []int64{all[0].ID, all[1].ID}

	if err := r.MarkAsRead(ctx, ids); err != nil {
		t.Fatalf("MarkAsRead: %v", err)
	}

	unread, _ := r.FindUnread(ctx)
	if len(unread) != 1 {
		t.Errorf("unread count: got %d, want 1", len(unread))
	}
}

func TestArticleRepository_MarkAsRead_Empty(t *testing.T) {
	r := newRepo(t)
	if err := r.MarkAsRead(context.Background(), nil); err != nil {
		t.Fatalf("MarkAsRead with empty ids should be no-op: %v", err)
	}
}

func TestArticleRepository_CountBookmarked(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/2")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := r.db.Exec(`UPDATE articles SET bookmarked = 1 WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup bookmarked: %v", err)
	}

	count, err := r.CountBookmarked(ctx)
	if err != nil {
		t.Fatalf("CountBookmarked: %v", err)
	}
	if count != 1 {
		t.Errorf("count: got %d, want 1", count)
	}
}

func TestArticleRepository_CountNonBookmarked(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/2")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := r.db.Exec(`UPDATE articles SET bookmarked = 1 WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup bookmarked: %v", err)
	}

	count, err := r.CountNonBookmarked(ctx)
	if err != nil {
		t.Fatalf("CountNonBookmarked: %v", err)
	}
	if count != 1 {
		t.Errorf("count: got %d, want 1", count)
	}
}

func TestArticleRepository_Search_TitleMatch(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	a1 := makeArticle("https://example.com/1")
	a1.Title = "Go言語入門"
	a2 := makeArticle("https://example.com/2")
	a2.Title = "Python基礎"
	if err := r.Save(ctx, a1); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, a2); err != nil {
		t.Fatalf("Save: %v", err)
	}

	results, err := r.Search(ctx, "Go", false)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("count: got %d, want 1", len(results))
	}
	if results[0].Title != "Go言語入門" {
		t.Errorf("Title: got %q", results[0].Title)
	}
}

func TestArticleRepository_Search_ContentMatch(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	a := makeArticle("https://example.com/1")
	a.Content = "goroutine を使った並行処理"
	if err := r.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	results, err := r.Search(ctx, "goroutine", false)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("count: got %d, want 1", len(results))
	}
}

func TestArticleRepository_Search_NoMatch(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}

	results, err := r.Search(ctx, "nomatch_xyz", false)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty, got %d", len(results))
	}
}

func TestArticleRepository_Save_PublisherAndThumbnail(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	a := makeArticle("https://example.com/1")
	a.Publisher = "Example Tech Blog"
	a.ThumbnailURL = "https://example.com/thumb.jpg"
	if err := r.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	all, _ := r.FindAll(ctx)
	if all[0].Publisher != "Example Tech Blog" {
		t.Errorf("Publisher: got %q", all[0].Publisher)
	}
	if all[0].ThumbnailURL != "https://example.com/thumb.jpg" {
		t.Errorf("ThumbnailURL: got %q", all[0].ThumbnailURL)
	}
}

func TestArticleRepository_UpdateEnrichment(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	all, _ := r.FindAll(ctx)
	id := all[0].ID

	if err := r.UpdateEnrichment(ctx, id, "要約テキスト", "Tech"); err != nil {
		t.Fatalf("UpdateEnrichment: %v", err)
	}

	found, _ := r.FindByID(ctx, id)
	if found.Summary != "要約テキスト" {
		t.Errorf("Summary: got %q", found.Summary)
	}
	if found.Category != "Tech" {
		t.Errorf("Category: got %q", found.Category)
	}
}

func TestArticleRepository_FindWithoutSummary(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)
	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/2")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	all, _ := r.FindAll(ctx)
	if err := r.UpdateEnrichment(ctx, all[0].ID, "要約", "Tech"); err != nil {
		t.Fatalf("UpdateEnrichment: %v", err)
	}

	results, err := r.FindWithoutSummary(ctx, 10)
	if err != nil {
		t.Fatalf("FindWithoutSummary: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("count: got %d, want 1", len(results))
	}
	if results[0].Summary != "" {
		t.Errorf("Summary should be empty, got %q", results[0].Summary)
	}
}

func TestArticleRepository_FindFiltered_Category(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	a1 := makeArticle("https://example.com/1")
	a2 := makeArticle("https://example.com/2")
	if err := r.Save(ctx, a1); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, a2); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE articles SET category = 'Tech' WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup category: %v", err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE articles SET category = 'AI' WHERE url = 'https://example.com/2'`); err != nil {
		t.Fatalf("setup category: %v", err)
	}

	articles, total, err := r.FindFiltered(ctx, articlerepo.ListFilter{Category: "Tech", PerPage: 25})
	if err != nil {
		t.Fatalf("FindFiltered: %v", err)
	}
	if total != 1 {
		t.Errorf("total: got %d, want 1", total)
	}
	if len(articles) != 1 || articles[0].Category != "Tech" {
		t.Errorf("articles: got %+v", articles)
	}
}

func TestArticleRepository_FindFiltered_SortAndOrder(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	a1 := makeArticle("https://example.com/1")
	a1.Title = "B"
	a2 := makeArticle("https://example.com/2")
	a2.Title = "A"
	if err := r.Save(ctx, a1); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, a2); err != nil {
		t.Fatalf("Save: %v", err)
	}

	articles, _, err := r.FindFiltered(ctx, articlerepo.ListFilter{Sort: "title", Order: "asc", PerPage: 25})
	if err != nil {
		t.Fatalf("FindFiltered: %v", err)
	}
	if len(articles) != 2 || articles[0].Title != "A" || articles[1].Title != "B" {
		t.Errorf("expected [A, B] order, got %+v", articles)
	}
}

func TestArticleRepository_FindFiltered_Pagination(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	for i := 0; i < 5; i++ {
		if err := r.Save(ctx, makeArticle(fmt.Sprintf("https://example.com/%d", i))); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	articles, total, err := r.FindFiltered(ctx, articlerepo.ListFilter{Page: 2, PerPage: 2})
	if err != nil {
		t.Fatalf("FindFiltered: %v", err)
	}
	if total != 5 {
		t.Errorf("total: got %d, want 5", total)
	}
	if len(articles) != 2 {
		t.Errorf("page size: got %d, want 2", len(articles))
	}
}

func TestArticleRepository_FindFiltered_UnreadAndBookmarked(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/2")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE articles SET read = 1, bookmarked = 1 WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup: %v", err)
	}

	unread, _, err := r.FindFiltered(ctx, articlerepo.ListFilter{Unread: true, PerPage: 25})
	if err != nil {
		t.Fatalf("FindFiltered: %v", err)
	}
	if len(unread) != 1 || unread[0].URL != "https://example.com/2" {
		t.Errorf("unread: got %+v", unread)
	}

	bookmarked, _, err := r.FindFiltered(ctx, articlerepo.ListFilter{BookmarkedOnly: true, PerPage: 25})
	if err != nil {
		t.Fatalf("FindFiltered: %v", err)
	}
	if len(bookmarked) != 1 || bookmarked[0].URL != "https://example.com/1" {
		t.Errorf("bookmarked: got %+v", bookmarked)
	}
}

func TestArticleRepository_DistinctCategories(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/2")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/3")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE articles SET category = 'Tech' WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup category: %v", err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE articles SET category = 'AI' WHERE url = 'https://example.com/2'`); err != nil {
		t.Fatalf("setup category: %v", err)
	}
	// url 3 はカテゴリ未設定（空文字）のまま

	categories, err := r.DistinctCategories(ctx)
	if err != nil {
		t.Fatalf("DistinctCategories: %v", err)
	}
	if len(categories) != 2 {
		t.Fatalf("count: got %d, want 2 (got %+v)", len(categories), categories)
	}
	if categories[0] != "AI" || categories[1] != "Tech" {
		t.Errorf("expected [AI, Tech] (alphabetical), got %+v", categories)
	}
}

func TestArticleRepository_Search_BookmarkedOnly(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	a1 := makeArticle("https://example.com/1")
	a1.Title = "Go入門"
	a2 := makeArticle("https://example.com/2")
	a2.Title = "Go応用"
	if err := r.Save(ctx, a1); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, a2); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE articles SET bookmarked = 1 WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup bookmarked: %v", err)
	}

	results, err := r.Search(ctx, "Go", true)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("count: got %d, want 1", len(results))
	}
	if !results[0].Bookmarked {
		t.Error("result should be bookmarked")
	}
}
