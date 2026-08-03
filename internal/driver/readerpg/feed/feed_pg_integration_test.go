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
	if _, err := db.Exec(`TRUNCATE TABLE audit_log, articles, feeds, users RESTART IDENTITY CASCADE`); err != nil {
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

// createUser はテスト用にユーザーを1件作成してIDを返す。
func createUser(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`INSERT INTO users (name) VALUES ($1) RETURNING id`, name).Scan(&id); err != nil {
		t.Fatalf("create user %q: %v", name, err)
	}
	return id
}

func TestFeedRepository_Save_InsertNew(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)
	userID := createUser(t, r.db, "alice")

	id, err := r.Save(ctx, domain.Feed{FeedURL: "https://example.com/feed", Title: "Example"}, userID)
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
	userID := createUser(t, r.db, "alice")

	id1, err := r.Save(ctx, domain.Feed{FeedURL: "https://example.com/feed", Title: "Old Title"}, userID)
	if err != nil {
		t.Fatalf("first Save: %v", err)
	}

	id2, err := r.Save(ctx, domain.Feed{FeedURL: "https://example.com/feed", Title: "New Title"}, userID)
	if err != nil {
		t.Fatalf("second Save: %v", err)
	}
	if id1 != id2 {
		t.Errorf("id should not change on upsert: got %d and %d", id1, id2)
	}

	found, err := r.FindByURL(ctx, "https://example.com/feed", userID)
	if err != nil || found == nil {
		t.Fatalf("FindByURL: %v", err)
	}
	if found.Title != "New Title" {
		t.Errorf("title: got %q, want %q", found.Title, "New Title")
	}
}

func TestFeedRepository_Save_SameURLDifferentUsers(t *testing.T) {
	// 同一の外部フィードURLを異なるユーザーがそれぞれ独立に購読できることを確認する。
	ctx := context.Background()
	r := newRepo(t)
	alice := createUser(t, r.db, "alice")
	bob := createUser(t, r.db, "bob")

	id1, err := r.Save(ctx, domain.Feed{FeedURL: "https://shared.example.com/feed", Title: "Alice's copy"}, alice)
	if err != nil {
		t.Fatalf("alice Save: %v", err)
	}
	id2, err := r.Save(ctx, domain.Feed{FeedURL: "https://shared.example.com/feed", Title: "Bob's copy"}, bob)
	if err != nil {
		t.Fatalf("bob Save: %v", err)
	}
	if id1 == id2 {
		t.Errorf("expected different feed rows for different users, got same id %d", id1)
	}
}

func TestFeedRepository_FindByURL_Exists(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)
	userID := createUser(t, r.db, "alice")

	if _, err := r.Save(ctx, domain.Feed{FeedURL: "https://example.com/feed", Title: "My Feed"}, userID); err != nil {
		t.Fatalf("Save: %v", err)
	}

	found, err := r.FindByURL(ctx, "https://example.com/feed", userID)
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
	userID := createUser(t, r.db, "alice")

	found, err := r.FindByURL(ctx, "https://notexist.example.com/feed", userID)
	if err != nil {
		t.Fatalf("FindByURL: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil, got %+v", found)
	}
}

func TestFeedRepository_FindByURL_ScopedToUser(t *testing.T) {
	// alice が登録したフィードは bob からは見えないことを確認する。
	ctx := context.Background()
	r := newRepo(t)
	alice := createUser(t, r.db, "alice")
	bob := createUser(t, r.db, "bob")

	if err := r.Register(ctx, "https://example.com/feed", alice); err != nil {
		t.Fatalf("Register: %v", err)
	}

	found, err := r.FindByURL(ctx, "https://example.com/feed", bob)
	if err != nil {
		t.Fatalf("FindByURL: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil (feed belongs to a different user), got %+v", found)
	}
}

func TestFeedRepository_Register_InsertsNew(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)
	userID := createUser(t, r.db, "alice")

	if err := r.Register(ctx, "https://new.example.com/feed", userID); err != nil {
		t.Fatalf("Register: %v", err)
	}

	found, err := r.FindByURL(ctx, "https://new.example.com/feed", userID)
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
	userID := createUser(t, r.db, "alice")

	if err := r.Register(ctx, "https://example.com/feed", userID); err != nil {
		t.Fatalf("Register setup: %v", err)
	}

	err := r.Register(ctx, "https://example.com/feed", userID)
	if !errors.Is(err, feedrepo.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestFeedRepository_Register_SameURLDifferentUsersBothSucceed(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)
	alice := createUser(t, r.db, "alice")
	bob := createUser(t, r.db, "bob")

	if err := r.Register(ctx, "https://shared.example.com/feed", alice); err != nil {
		t.Fatalf("alice Register: %v", err)
	}
	if err := r.Register(ctx, "https://shared.example.com/feed", bob); err != nil {
		t.Errorf("bob Register with same URL should succeed, got %v", err)
	}
}

func TestFeedRepository_ListAll_ReturnsAll(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)
	userID := createUser(t, r.db, "alice")

	if err := r.Register(ctx, "https://feed1.example.com", userID); err != nil {
		t.Fatalf("Register setup: %v", err)
	}
	if err := r.Register(ctx, "https://feed2.example.com", userID); err != nil {
		t.Fatalf("Register setup: %v", err)
	}

	feeds, err := r.ListAll(ctx, userID)
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
	userID := createUser(t, r.db, "alice")

	feeds, err := r.ListAll(ctx, userID)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(feeds) != 0 {
		t.Errorf("expected 0 feeds, got %d", len(feeds))
	}
}

func TestFeedRepository_ListAll_ScopedToUser(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)
	alice := createUser(t, r.db, "alice")
	bob := createUser(t, r.db, "bob")

	if err := r.Register(ctx, "https://alice-feed.example.com", alice); err != nil {
		t.Fatalf("Register alice feed: %v", err)
	}
	if err := r.Register(ctx, "https://bob-feed.example.com", bob); err != nil {
		t.Fatalf("Register bob feed: %v", err)
	}

	aliceFeeds, err := r.ListAll(ctx, alice)
	if err != nil {
		t.Fatalf("ListAll(alice): %v", err)
	}
	if len(aliceFeeds) != 1 || aliceFeeds[0].FeedURL != "https://alice-feed.example.com" {
		t.Errorf("alice's feeds: got %+v", aliceFeeds)
	}

	bobFeeds, err := r.ListAll(ctx, bob)
	if err != nil {
		t.Fatalf("ListAll(bob): %v", err)
	}
	if len(bobFeeds) != 1 || bobFeeds[0].FeedURL != "https://bob-feed.example.com" {
		t.Errorf("bob's feeds: got %+v", bobFeeds)
	}
}

func TestFeedRepository_Remove_DeletesFeed(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)
	userID := createUser(t, r.db, "alice")

	if err := r.Register(ctx, "https://example.com/feed", userID); err != nil {
		t.Fatalf("Register setup: %v", err)
	}
	found, err := r.FindByURL(ctx, "https://example.com/feed", userID)
	if err != nil {
		t.Fatalf("FindByURL setup: %v", err)
	}
	if found == nil {
		t.Fatal("expected feed after Register, got nil")
	}

	if err := r.Remove(ctx, found.ID, userID); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	after, err := r.FindByURL(ctx, "https://example.com/feed", userID)
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
	userID := createUser(t, r.db, "alice")

	if err := r.Register(ctx, "https://example.com/feed", userID); err != nil {
		t.Fatalf("Register setup: %v", err)
	}
	found, err := r.FindByURL(ctx, "https://example.com/feed", userID)
	if err != nil || found == nil {
		t.Fatalf("FindByURL setup: found=%v err=%v", found, err)
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO articles (feed_id, url, title) VALUES ($1, $2, $3)`,
		found.ID, "https://example.com/feed/article1", "記事1"); err != nil {
		t.Fatalf("insert article setup: %v", err)
	}

	if err := r.Remove(ctx, found.ID, userID); err != nil {
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
	userID := createUser(t, r.db, "alice")

	err := r.Remove(ctx, 999, userID)
	if !errors.Is(err, feedrepo.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFeedRepository_Remove_OtherUsersFeedReturnsErrNotFound(t *testing.T) {
	// bob が alice のフィードIDを指定してもErrNotFoundになり、削除できないことを確認する
	// （フィードIDの推測による他ユーザーのフィード操作を防ぐ）。
	ctx := context.Background()
	r := newRepo(t)
	alice := createUser(t, r.db, "alice")
	bob := createUser(t, r.db, "bob")

	if err := r.Register(ctx, "https://alice-only.example.com", alice); err != nil {
		t.Fatalf("Register: %v", err)
	}
	aliceFeed, err := r.FindByURL(ctx, "https://alice-only.example.com", alice)
	if err != nil || aliceFeed == nil {
		t.Fatalf("FindByURL: found=%v err=%v", aliceFeed, err)
	}

	// Remove は所有権チェックを DELETE FROM articles より前に行うことで、他ユーザーの
	// フィードIDを指定された場合に記事も削除されないことを保証している。この安全性を
	// 検証するため、記事を1件挿入しておく。
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO articles (feed_id, url, title) VALUES ($1, $2, $3)`,
		aliceFeed.ID, "https://alice-only.example.com/article-1", "Article 1"); err != nil {
		t.Fatalf("insert article for alice's feed: %v", err)
	}

	if err := r.Remove(ctx, aliceFeed.ID, bob); !errors.Is(err, feedrepo.ErrNotFound) {
		t.Errorf("expected ErrNotFound when bob removes alice's feed, got %v", err)
	}

	// alice のフィードは削除されずに残っていることを確認する。
	stillExists, err := r.FindByURL(ctx, "https://alice-only.example.com", alice)
	if err != nil {
		t.Fatalf("FindByURL after failed Remove: %v", err)
	}
	if stillExists == nil {
		t.Error("alice's feed should still exist after bob's failed Remove attempt")
	}

	// alice の記事も削除されずに残っていることを確認する。
	var articleCount int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM articles WHERE feed_id = $1`, aliceFeed.ID).Scan(&articleCount); err != nil {
		t.Fatalf("count articles after failed Remove: %v", err)
	}
	if articleCount != 1 {
		t.Errorf("alice's article count after bob's failed Remove attempt: got %d, want 1", articleCount)
	}
}
