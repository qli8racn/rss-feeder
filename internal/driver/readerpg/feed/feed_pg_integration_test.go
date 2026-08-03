//go:build pg_integration

package feed

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/qli8racn/rss-feeder/internal/migration"
)

// newTestDB は環境変数 TEST_POSTGRES_DSN が設定されている場合のみ実際のPostgresに接続する。
// 未設定の場合は t.Skip する（通常の `go test ./...`（ビルドタグなし）ではこのファイル自体が
// コンパイル対象外のため、CIコストは増えない）。
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set; skipping postgres integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open postgres db: %v", err)
	}
	if err := migration.RunPostgres(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	// テスト間で状態を共有しないよう、各テスト開始前に全テーブルを空にする。
	if _, err := db.Exec(`TRUNCATE TABLE audit_log, articles, feeds RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate tables: %v", err)
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

func TestFeedRepository_Remove_DeletesAssociatedArticles(t *testing.T) {
	// articles.feed_id はON DELETE CASCADEを付けていない（SQLite実装と方針を揃えている）ため、
	// RemoveがトランザクションでarticlesをDELETEしていることを確認する。
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Register(ctx, "https://example.com/feed"); err != nil {
		t.Fatalf("Register setup: %v", err)
	}
	found, err := r.FindByURL(ctx, "https://example.com/feed")
	if err != nil || found == nil {
		t.Fatalf("FindByURL setup: found=%v err=%v", found, err)
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO articles (feed_id, url, title) VALUES ($1, $2, $3)`,
		found.ID, "https://example.com/feed/article1", "記事1"); err != nil {
		t.Fatalf("insert article setup: %v", err)
	}

	if err := r.Remove(ctx, found.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM articles WHERE feed_id = $1`, found.ID).Scan(&count); err != nil {
		t.Fatalf("count articles after Remove: %v", err)
	}
	if count != 0 {
		t.Errorf("Remove: got %d articles remaining for removed feed, want 0", count)
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
