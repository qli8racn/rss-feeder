package feed

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
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

	if _, err := r.Save(ctx, domain.Feed{FeedURL: "https://example.com/feed", Title: "My Feed"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

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

func TestFeedRepository_Register_InsertsNew(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Register(ctx, "https://new.example.com/feed"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	found, err := r.FindByURL(ctx, "https://new.example.com/feed")
	if err != nil {
		t.Fatalf("FindByURL after Register: %v", err)
	}
	if found == nil {
		t.Fatal("expected feed, got nil")
	}
}

func TestFeedRepository_Register_ErrAlreadyExists(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Register(ctx, "https://example.com/feed"); err != nil {
		t.Fatalf("Register setup: %v", err)
	}

	err := r.Register(ctx, "https://example.com/feed")
	if !errors.Is(err, feedrepo.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestFeedRepository_ListAll_ReturnsAll(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Register(ctx, "https://feed1.example.com"); err != nil {
		t.Fatalf("Register setup: %v", err)
	}
	if err := r.Register(ctx, "https://feed2.example.com"); err != nil {
		t.Fatalf("Register setup: %v", err)
	}

	feeds, err := r.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(feeds) != 2 {
		t.Errorf("len: got %d, want 2", len(feeds))
	}
}

func TestFeedRepository_ListAll_Empty(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	feeds, err := r.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(feeds) != 0 {
		t.Errorf("expected 0 feeds, got %d", len(feeds))
	}
}

func TestFeedRepository_Remove_DeletesFeed(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Register(ctx, "https://example.com/feed"); err != nil {
		t.Fatalf("Register setup: %v", err)
	}
	found, err := r.FindByURL(ctx, "https://example.com/feed")
	if err != nil {
		t.Fatalf("FindByURL setup: %v", err)
	}
	if found == nil {
		t.Fatal("expected feed after Register, got nil")
	}

	if err := r.Remove(ctx, found.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	after, err := r.FindByURL(ctx, "https://example.com/feed")
	if err != nil {
		t.Fatalf("FindByURL after Remove: %v", err)
	}
	if after != nil {
		t.Error("expected nil after remove, got feed")
	}
}

func TestFeedRepository_Remove_ErrNotFound(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	err := r.Remove(ctx, 999)
	if !errors.Is(err, feedrepo.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
