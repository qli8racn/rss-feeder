package article

import (
	"context"
	"database/sql"
	"errors"
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
	db.Exec(`INSERT INTO feeds (id, feed_url, title) VALUES (1, 'https://example.com/feed', 'Test')`)
	t.Cleanup(func() { db.Close() })
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

	r.Save(ctx, makeArticle("https://example.com/1"))
	err := r.Save(ctx, makeArticle("https://example.com/1"))
	if !errors.Is(err, articlerepo.ErrDuplicate) {
		t.Errorf("expected ErrDuplicate, got %v", err)
	}
}

func TestArticleRepository_FindAll(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	r.Save(ctx, makeArticle("https://example.com/1"))
	r.Save(ctx, makeArticle("https://example.com/2"))

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

	r.Save(ctx, makeArticle("https://example.com/1"))
	r.Save(ctx, makeArticle("https://example.com/2"))
	r.db.Exec(`UPDATE articles SET read = 1 WHERE url = 'https://example.com/1'`)

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

	r.Save(ctx, makeArticle("https://example.com/1"))
	r.Save(ctx, makeArticle("https://example.com/2"))
	r.db.Exec(`UPDATE articles SET bookmarked = 1 WHERE url = 'https://example.com/2'`)

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

	r.Save(ctx, makeArticle("https://example.com/1"))
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

	r.Save(ctx, makeArticle("https://example.com/1"))
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

	r.Save(ctx, makeArticle("https://example.com/1"))
	r.Save(ctx, makeArticle("https://example.com/2"))
	r.db.Exec(`UPDATE articles SET bookmarked = 1 WHERE url = 'https://example.com/1'`)

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

	r.Save(ctx, makeArticle("https://example.com/1"))
	r.Save(ctx, makeArticle("https://example.com/2"))
	r.Save(ctx, makeArticle("https://example.com/3"))

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

	r.Save(ctx, makeArticle("https://example.com/1"))
	r.Save(ctx, makeArticle("https://example.com/2"))
	r.db.Exec(`UPDATE articles SET bookmarked = 1 WHERE url = 'https://example.com/1'`)

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

	r.Save(ctx, makeArticle("https://example.com/1"))
	r.Save(ctx, makeArticle("https://example.com/2"))
	r.db.Exec(`UPDATE articles SET bookmarked = 1 WHERE url = 'https://example.com/1'`)

	count, err := r.CountNonBookmarked(ctx)
	if err != nil {
		t.Fatalf("CountNonBookmarked: %v", err)
	}
	if count != 1 {
		t.Errorf("count: got %d, want 1", count)
	}
}
