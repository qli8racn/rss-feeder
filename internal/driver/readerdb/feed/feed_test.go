package feed

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
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
	t.Cleanup(func() { db.Close() })
	return db
}

func newRepo(t *testing.T) *repository {
	t.Helper()
	return &repository{db: newTestDB(t)}
}

func TestFeedRepository_Save_InsertNew(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	id, err := r.Save(ctx, domain.Feed{FeedURL: "https://example.com/feed", Title: "Example"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if id <= 0 {
		t.Errorf("id: got %d, want > 0", id)
	}
}

func TestFeedRepository_Save_UpdateExisting(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	id1, err := r.Save(ctx, domain.Feed{FeedURL: "https://example.com/feed", Title: "Old Title"})
	if err != nil {
		t.Fatalf("first Save: %v", err)
	}

	id2, err := r.Save(ctx, domain.Feed{FeedURL: "https://example.com/feed", Title: "New Title"})
	if err != nil {
		t.Fatalf("second Save: %v", err)
	}
	if id1 != id2 {
		t.Errorf("id should not change on upsert: got %d and %d", id1, id2)
	}

	found, err := r.FindByURL(ctx, "https://example.com/feed")
	if err != nil || found == nil {
		t.Fatalf("FindByURL: %v", err)
	}
	if found.Title != "New Title" {
		t.Errorf("title: got %q, want %q", found.Title, "New Title")
	}
}

func TestFeedRepository_FindByURL_Exists(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	r.Save(ctx, domain.Feed{FeedURL: "https://example.com/feed", Title: "My Feed"})

	found, err := r.FindByURL(ctx, "https://example.com/feed")
	if err != nil {
		t.Fatalf("FindByURL: %v", err)
	}
	if found == nil {
		t.Fatal("expected feed, got nil")
	}
	if found.FeedURL != "https://example.com/feed" {
		t.Errorf("FeedURL: got %q", found.FeedURL)
	}
}

func TestFeedRepository_FindByURL_NotExists(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	found, err := r.FindByURL(ctx, "https://notexist.example.com/feed")
	if err != nil {
		t.Fatalf("FindByURL: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil, got %+v", found)
	}
}
