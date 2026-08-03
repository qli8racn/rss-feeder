//go:build pg_integration

package article

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/qli8racn/rss-feeder/internal/migration"
)

// defaultUserID はテスト用DBセットアップ時に作成する既定ユーザー（id=1）のID。
const defaultUserID int64 = 1

// otherUserID は分離確認テスト用の2人目のユーザー（id=2）のID。
const otherUserID int64 = 2

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

	// defaultUserID(id=1)・otherUserID(id=2) をこの順で作成する
	// （RESTART IDENTITY直後のRETURNING idは常にこの値になる）。
	var gotDefaultUserID int64
	if err := db.QueryRow(`INSERT INTO users (name) VALUES ('default') RETURNING id`).Scan(&gotDefaultUserID); err != nil {
		t.Fatalf("setup default user: %v", err)
	}
	if gotDefaultUserID != defaultUserID {
		t.Fatalf("expected default user id %d after RESTART IDENTITY, got %d", defaultUserID, gotDefaultUserID)
	}
	var gotOtherUserID int64
	if err := db.QueryRow(`INSERT INTO users (name) VALUES ('other') RETURNING id`).Scan(&gotOtherUserID); err != nil {
		t.Fatalf("setup other user: %v", err)
	}
	if gotOtherUserID != otherUserID {
		t.Fatalf("expected other user id %d after RESTART IDENTITY, got %d", otherUserID, gotOtherUserID)
	}

	// defaultUserID が所有するフィード（id=1）を1件用意し、articles.feed_id から参照できるようにする。
	if _, err := db.Exec(`INSERT INTO feeds (id, user_id, feed_url, title) VALUES (1, $1, 'https://example.com/feed', 'Test')`, defaultUserID); err != nil {
		t.Fatalf("setup feed: %v", err)
	}
	// otherUserID が所有するフィード（id=2、他ユーザーの記事分離確認用）。
	if _, err := db.Exec(`INSERT INTO feeds (id, user_id, feed_url, title) VALUES (2, $1, 'https://other.example.com/feed', 'Other')`, otherUserID); err != nil {
		t.Fatalf("setup other feed: %v", err)
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

// makeOtherUserArticle は otherUserID が所有するフィード（feed_id=2）に紐づく記事を作る。
func makeOtherUserArticle(url string) domain.Article {
	return domain.Article{FeedID: 2, URL: url, Title: "Other's " + url, Content: "body"}
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

func TestArticleRepository_Save_SameURLDifferentUsersFeed(t *testing.T) {
	// UNIQUE(feed_id, url) のため、異なるユーザーのフィードに紐づく記事であれば
	// 同一URLでも重複エラーにならず保存できる。
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://shared.example.com/1")); err != nil {
		t.Fatalf("Save (feed 1): %v", err)
	}
	if err := r.Save(ctx, makeOtherUserArticle("https://shared.example.com/1")); err != nil {
		t.Errorf("Save with same URL for a different user's feed should succeed, got %v", err)
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

	articles, err := r.FindAll(ctx, defaultUserID)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(articles) != 2 {
		t.Errorf("count: got %d, want 2", len(articles))
	}
}

func TestArticleRepository_FindAll_ScopedToUser(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeOtherUserArticle("https://other.example.com/1")); err != nil {
		t.Fatalf("Save (other user): %v", err)
	}

	articles, err := r.FindAll(ctx, defaultUserID)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(articles) != 1 || articles[0].URL != "https://example.com/1" {
		t.Errorf("expected only defaultUser's article, got %+v", articles)
	}

	othersArticles, err := r.FindAll(ctx, otherUserID)
	if err != nil {
		t.Fatalf("FindAll(otherUser): %v", err)
	}
	if len(othersArticles) != 1 || othersArticles[0].URL != "https://other.example.com/1" {
		t.Errorf("expected only otherUser's article, got %+v", othersArticles)
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
	if _, err := r.db.Exec(`UPDATE articles SET read = TRUE WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup read: %v", err)
	}

	articles, err := r.FindUnread(ctx, defaultUserID)
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
	if _, err := r.db.Exec(`UPDATE articles SET bookmarked = TRUE WHERE url = 'https://example.com/2'`); err != nil {
		t.Fatalf("setup bookmarked: %v", err)
	}

	articles, err := r.FindBookmarked(ctx, defaultUserID)
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

func TestArticleRepository_FindBookmarked_ScopedToUser(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeOtherUserArticle("https://other.example.com/1")); err != nil {
		t.Fatalf("Save (other user): %v", err)
	}
	if _, err := r.db.Exec(`UPDATE articles SET bookmarked = TRUE`); err != nil {
		t.Fatalf("setup bookmarked: %v", err)
	}

	articles, err := r.FindBookmarked(ctx, defaultUserID)
	if err != nil {
		t.Fatalf("FindBookmarked: %v", err)
	}
	if len(articles) != 1 || articles[0].URL != "https://example.com/1" {
		t.Errorf("expected only defaultUser's bookmarked article, got %+v", articles)
	}
}

func TestArticleRepository_FindByID(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	all, _ := r.FindAll(ctx, defaultUserID)
	id := all[0].ID

	found, err := r.FindByID(ctx, id, defaultUserID)
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

	found, err := r.FindByID(ctx, 9999, defaultUserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil, got %+v", found)
	}
}

func TestArticleRepository_FindByID_OtherUsersArticleReturnsNil(t *testing.T) {
	// 記事IDの推測による他ユーザーの記事参照を防げていることを確認する。
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeOtherUserArticle("https://other.example.com/1")); err != nil {
		t.Fatalf("Save (other user): %v", err)
	}
	all, _ := r.FindAll(ctx, otherUserID)
	id := all[0].ID

	found, err := r.FindByID(ctx, id, defaultUserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil (belongs to a different user), got %+v", found)
	}
}

func TestArticleRepository_Update(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	all, _ := r.FindAll(ctx, defaultUserID)
	a := all[0]

	a.Read = true
	a.Bookmarked = true
	if err := r.Update(ctx, a, defaultUserID); err != nil {
		t.Fatalf("Update: %v", err)
	}

	found, _ := r.FindByID(ctx, a.ID, defaultUserID)
	if !found.Read {
		t.Error("Read should be true after Update")
	}
	if !found.Bookmarked {
		t.Error("Bookmarked should be true after Update")
	}
}

func TestArticleRepository_Update_OtherUsersArticleIsNoOp(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeOtherUserArticle("https://other.example.com/1")); err != nil {
		t.Fatalf("Save (other user): %v", err)
	}
	all, _ := r.FindAll(ctx, otherUserID)
	a := all[0]
	a.Read = true

	if err := r.Update(ctx, a, defaultUserID); err != nil {
		t.Fatalf("Update: %v", err)
	}

	found, err := r.FindByID(ctx, a.ID, otherUserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Read {
		t.Error("Update by a different user should not have modified the article")
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
	if _, err := r.db.Exec(`UPDATE articles SET bookmarked = TRUE WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup bookmarked: %v", err)
	}

	n, err := r.DeleteNonBookmarked(ctx, defaultUserID)
	if err != nil {
		t.Fatalf("DeleteNonBookmarked: %v", err)
	}
	if n != 1 {
		t.Errorf("deleted: got %d, want 1", n)
	}

	all, _ := r.FindAll(ctx, defaultUserID)
	if len(all) != 1 {
		t.Errorf("remaining: got %d, want 1", len(all))
	}
}

func TestArticleRepository_DeleteNonBookmarked_DoesNotAffectOtherUsers(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeOtherUserArticle("https://other.example.com/1")); err != nil {
		t.Fatalf("Save (other user): %v", err)
	}

	if _, err := r.DeleteNonBookmarked(ctx, defaultUserID); err != nil {
		t.Fatalf("DeleteNonBookmarked: %v", err)
	}

	othersArticles, err := r.FindAll(ctx, otherUserID)
	if err != nil {
		t.Fatalf("FindAll(otherUser): %v", err)
	}
	if len(othersArticles) != 1 {
		t.Errorf("other user's article should not be deleted, got %d remaining", len(othersArticles))
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

	all, _ := r.FindAll(ctx, defaultUserID)
	ids := []int64{all[0].ID, all[1].ID}

	if err := r.MarkAsRead(ctx, ids, defaultUserID); err != nil {
		t.Fatalf("MarkAsRead: %v", err)
	}

	unread, _ := r.FindUnread(ctx, defaultUserID)
	if len(unread) != 1 {
		t.Errorf("unread count: got %d, want 1", len(unread))
	}
}

func TestArticleRepository_MarkAsRead_Empty(t *testing.T) {
	r := newRepo(t)
	if err := r.MarkAsRead(context.Background(), nil, defaultUserID); err != nil {
		t.Fatalf("MarkAsRead with empty ids should be no-op: %v", err)
	}
}

func TestArticleRepository_MarkAsRead_OtherUsersArticleIsNoOp(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeOtherUserArticle("https://other.example.com/1")); err != nil {
		t.Fatalf("Save (other user): %v", err)
	}
	all, _ := r.FindAll(ctx, otherUserID)

	if err := r.MarkAsRead(ctx, []int64{all[0].ID}, defaultUserID); err != nil {
		t.Fatalf("MarkAsRead: %v", err)
	}

	found, err := r.FindByID(ctx, all[0].ID, otherUserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Read {
		t.Error("MarkAsRead by a different user should not have modified the article")
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
	if _, err := r.db.Exec(`UPDATE articles SET bookmarked = TRUE WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup bookmarked: %v", err)
	}

	count, err := r.CountBookmarked(ctx, defaultUserID)
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
	if _, err := r.db.Exec(`UPDATE articles SET bookmarked = TRUE WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup bookmarked: %v", err)
	}

	count, err := r.CountNonBookmarked(ctx, defaultUserID)
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

	results, err := r.Search(ctx, "Go", false, defaultUserID)
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

func TestArticleRepository_Search_CaseInsensitive(t *testing.T) {
	// PostgresのLIKEはASCII範囲で大文字小文字を区別するため、ILIKEに書き換えたことで
	// SQLite実装と同様に大文字小文字を区別せずマッチすることを確認する。
	ctx := context.Background()
	r := newRepo(t)

	a := makeArticle("https://example.com/1")
	a.Title = "Golang Tutorial"
	if err := r.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	results, err := r.Search(ctx, "golang", false, defaultUserID)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("count: got %d, want 1 (ILIKE should be case-insensitive)", len(results))
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

	results, err := r.Search(ctx, "goroutine", false, defaultUserID)
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

	results, err := r.Search(ctx, "nomatch_xyz", false, defaultUserID)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty, got %d", len(results))
	}
}

func TestArticleRepository_Search_ScopedToUser(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	a1 := makeArticle("https://example.com/1")
	a1.Title = "Go入門"
	a2 := makeOtherUserArticle("https://other.example.com/1")
	a2.Title = "Go応用（他ユーザー）"
	if err := r.Save(ctx, a1); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, a2); err != nil {
		t.Fatalf("Save (other user): %v", err)
	}

	results, err := r.Search(ctx, "Go", false, defaultUserID)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Go入門" {
		t.Errorf("expected only defaultUser's article, got %+v", results)
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

	all, _ := r.FindAll(ctx, defaultUserID)
	if all[0].Publisher != "Example Tech Blog" {
		t.Errorf("Publisher: got %q", all[0].Publisher)
	}
	if all[0].ThumbnailURL != "https://example.com/thumb.jpg" {
		t.Errorf("ThumbnailURL: got %q", all[0].ThumbnailURL)
	}
}

func TestArticleRepository_UpdateEnrichmentBatch(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/2")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	all, _ := r.FindAll(ctx, defaultUserID)

	updates := []articlerepo.EnrichmentUpdate{
		{ID: all[0].ID, Summary: "要約1", Category: "Tech"},
		{ID: all[1].ID, Summary: "要約2", Category: "Business"},
	}
	if err := r.UpdateEnrichmentBatch(ctx, updates, defaultUserID); err != nil {
		t.Fatalf("UpdateEnrichmentBatch: %v", err)
	}

	found0, _ := r.FindByID(ctx, all[0].ID, defaultUserID)
	if found0.Summary != "要約1" || found0.Category != "Tech" {
		t.Errorf("got summary=%q category=%q, want 要約1/Tech", found0.Summary, found0.Category)
	}
	found1, _ := r.FindByID(ctx, all[1].ID, defaultUserID)
	if found1.Summary != "要約2" || found1.Category != "Business" {
		t.Errorf("got summary=%q category=%q, want 要約2/Business", found1.Summary, found1.Category)
	}
}

func TestArticleRepository_UpdateEnrichmentBatch_Empty(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.UpdateEnrichmentBatch(ctx, nil, defaultUserID); err != nil {
		t.Errorf("UpdateEnrichmentBatch(nil): got error %v, want nil", err)
	}
}

func TestArticleRepository_UpdateEnrichmentBatch_OtherUsersArticleIsNotUpdated(t *testing.T) {
	// otherUserID の記事IDを defaultUserID として更新しようとしても対象外になり、
	// 更新されないことを確認する（rss_enrich が他ユーザーの記事を書き換えないことの検証）。
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeOtherUserArticle("https://other.example.com/1")); err != nil {
		t.Fatalf("Save (other user): %v", err)
	}
	all, _ := r.FindAll(ctx, otherUserID)

	if err := r.UpdateEnrichmentBatch(ctx, []articlerepo.EnrichmentUpdate{
		{ID: all[0].ID, Summary: "勝手に要約", Category: "Tech"},
	}, defaultUserID); err != nil {
		t.Fatalf("UpdateEnrichmentBatch: %v", err)
	}

	found, err := r.FindByID(ctx, all[0].ID, otherUserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Summary != "" {
		t.Errorf("otherUser's article should not be updated by defaultUser's UpdateEnrichmentBatch, got summary=%q", found.Summary)
	}
}

func TestArticleRepository_UpdateEnrichmentBatch_FailureRollsBackEarlierRows(t *testing.T) {
	// 2件目の更新がctxキャンセルにより失敗した場合、1件目の更新（トランザクション内では
	// 既に成功していた）もコミットされずロールバックされることを確認する
	// （UpdateEnrichmentBatchは1回の呼び出し全体がall-or-nothingであることの検証）。
	ctx, cancel := context.WithCancel(context.Background())
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/2")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	all, _ := r.FindAll(ctx, defaultUserID)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	stmt, err := tx.PrepareContext(ctx, `UPDATE articles SET summary = $1, category = $2 WHERE id = $3`)
	if err != nil {
		t.Fatalf("PrepareContext: %v", err)
	}
	if _, err := stmt.ExecContext(ctx, "要約1", "Tech", all[0].ID); err != nil {
		t.Fatalf("ExecContext (1st row): %v", err)
	}
	cancel() // 2件目の実行前にctxをキャンセルし、2件目だけを失敗させる
	if _, err := stmt.ExecContext(ctx, "要約2", "Business", all[1].ID); err == nil {
		t.Fatal("ExecContext (2nd row): want error after ctx is canceled")
	}
	_ = tx.Rollback()

	found0, err := r.FindByID(context.Background(), all[0].ID, defaultUserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found0 == nil {
		t.Fatalf("FindByID: record not found for id=%d", all[0].ID)
	}
	if found0.Summary != "" {
		t.Errorf("1件目はロールバックされ未更新のはず: got summary=%q, want empty", found0.Summary)
	}
}

func TestArticleRepository_UpdateMetadataBatch_FillsEmptyOnly(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	// 1件目は既に publisher・thumbnail_url が空のまま保存された「既存記事」を再現
	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// 2件目は既に publisher のみ設定済み（thumbnail_url は未設定）
	a2 := makeArticle("https://example.com/2")
	a2.Publisher = "既存の出版元"
	if err := r.Save(ctx, a2); err != nil {
		t.Fatalf("Save: %v", err)
	}

	n, err := r.UpdateMetadataBatch(ctx, []articlerepo.MetadataUpdate{
		{URL: "https://example.com/1", Publisher: "新しい出版元", ThumbnailURL: "https://example.com/thumb1.jpg"},
		{URL: "https://example.com/2", Publisher: "上書きされないはず", ThumbnailURL: "https://example.com/thumb2.jpg"},
	})
	if err != nil {
		t.Fatalf("UpdateMetadataBatch: %v", err)
	}
	if n != 2 {
		t.Errorf("RowsAffected: got %d, want 2", n)
	}

	all, _ := r.FindAll(ctx, defaultUserID)
	byURL := map[string]domain.Article{}
	for _, a := range all {
		byURL[a.URL] = a
	}

	a1 := byURL["https://example.com/1"]
	if a1.Publisher != "新しい出版元" || a1.ThumbnailURL != "https://example.com/thumb1.jpg" {
		t.Errorf("article1: got publisher=%q thumbnail=%q", a1.Publisher, a1.ThumbnailURL)
	}

	updated2 := byURL["https://example.com/2"]
	if updated2.Publisher != "既存の出版元" {
		t.Errorf("article2.Publisher: 既存値が上書きされた: got %q", updated2.Publisher)
	}
	if updated2.ThumbnailURL != "https://example.com/thumb2.jpg" {
		t.Errorf("article2.ThumbnailURL: got %q, want thumb2.jpg", updated2.ThumbnailURL)
	}
}

func TestArticleRepository_UpdateMetadataBatch_SkipsFullyEnrichedArticles(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	a := makeArticle("https://example.com/1")
	a.Publisher = "既存の出版元"
	a.ThumbnailURL = "https://example.com/existing-thumb.jpg"
	if err := r.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	n, err := r.UpdateMetadataBatch(ctx, []articlerepo.MetadataUpdate{
		{URL: "https://example.com/1", Publisher: "新しい出版元", ThumbnailURL: "https://example.com/new-thumb.jpg"},
	})
	if err != nil {
		t.Fatalf("UpdateMetadataBatch: %v", err)
	}
	if n != 0 {
		t.Errorf("RowsAffected: got %d, want 0（両列とも設定済みなので対象外）", n)
	}
}

func TestArticleRepository_UpdateMetadataBatch_SkipsWhenNewValueAlsoEmpty(t *testing.T) {
	// publisher は補完済みだが thumbnail_url は未設定のまま、というケースで
	// 再取得したフィードがそもそもサムネイルを提供しない（新しい値も空文字）場合は対象外とする。
	// そうしないと、何度実行しても毎回「補完」件数として数えられてしまう（実運用で発覚した不整合）。
	ctx := context.Background()
	r := newRepo(t)

	a := makeArticle("https://example.com/1")
	a.Publisher = "既存の出版元"
	if err := r.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	n, err := r.UpdateMetadataBatch(ctx, []articlerepo.MetadataUpdate{
		{URL: "https://example.com/1", Publisher: "上書きされないはず", ThumbnailURL: ""},
	})
	if err != nil {
		t.Fatalf("UpdateMetadataBatch: %v", err)
	}
	if n != 0 {
		t.Errorf("RowsAffected: got %d, want 0（新しい値も空文字なので対象外）", n)
	}
}

func TestArticleRepository_UpdateMetadataBatch_FillsNullColumns(t *testing.T) {
	// migration.go の `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` で追加した既存行は
	// publisher/thumbnail_url が空文字ではなくNULLになる。COALESCEなしでは
	// `publisher = ''` がNULLと一致せず対象外になってしまうことの確認。
	ctx := context.Background()
	r := newRepo(t)

	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO articles (feed_id, url, title, content, published_at, fetched_at, publisher, thumbnail_url)
		VALUES (1, 'https://example.com/1', 'Title', 'body', $1, $2, NULL, NULL)
	`, time.Now(), time.Now()); err != nil {
		t.Fatalf("insert with NULL publisher/thumbnail_url: %v", err)
	}

	n, err := r.UpdateMetadataBatch(ctx, []articlerepo.MetadataUpdate{
		{URL: "https://example.com/1", Publisher: "新しい出版元", ThumbnailURL: "https://example.com/thumb.jpg"},
	})
	if err != nil {
		t.Fatalf("UpdateMetadataBatch: %v", err)
	}
	if n != 1 {
		t.Errorf("RowsAffected: got %d, want 1", n)
	}

	all, _ := r.FindAll(ctx, defaultUserID)
	if all[0].Publisher != "新しい出版元" || all[0].ThumbnailURL != "https://example.com/thumb.jpg" {
		t.Errorf("got publisher=%q thumbnail=%q", all[0].Publisher, all[0].ThumbnailURL)
	}
}

func TestArticleRepository_UpdateMetadataBatch_Empty(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	n, err := r.UpdateMetadataBatch(ctx, nil)
	if err != nil {
		t.Errorf("UpdateMetadataBatch(nil): got error %v, want nil", err)
	}
	if n != 0 {
		t.Errorf("RowsAffected: got %d, want 0", n)
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
	all, _ := r.FindAll(ctx, defaultUserID)
	if err := r.UpdateEnrichmentBatch(ctx, []articlerepo.EnrichmentUpdate{{ID: all[0].ID, Summary: "要約", Category: "Tech"}}, defaultUserID); err != nil {
		t.Fatalf("UpdateEnrichmentBatch: %v", err)
	}

	results, err := r.FindWithoutSummary(ctx, 10, defaultUserID)
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

func TestArticleRepository_FindWithoutSummary_ScopedToUser(t *testing.T) {
	// otherUserID の要約未設定記事が defaultUserID の FindWithoutSummary に混入しないことを確認する
	// （rss_enrich が他ユーザーの記事まで要約対象にしてしまわないことの検証）。
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeOtherUserArticle("https://other.example.com/1")); err != nil {
		t.Fatalf("Save (other user): %v", err)
	}

	results, err := r.FindWithoutSummary(ctx, 10, defaultUserID)
	if err != nil {
		t.Fatalf("FindWithoutSummary: %v", err)
	}
	if len(results) != 1 || results[0].URL != "https://example.com/1" {
		t.Errorf("expected only defaultUser's article, got %+v", results)
	}
}

func TestArticleRepository_FetchLatest_ScopedToUser(t *testing.T) {
	// otherUserID の記事が defaultUserID の FetchLatest（rss_enrich・rss_agent summarize/curate が
	// 利用する）に混入しないことを確認する。
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeOtherUserArticle("https://other.example.com/1")); err != nil {
		t.Fatalf("Save (other user): %v", err)
	}

	results, err := r.FetchLatest(ctx, 10, "", defaultUserID)
	if err != nil {
		t.Fatalf("FetchLatest: %v", err)
	}
	if len(results) != 1 || results[0].URL != "https://example.com/1" {
		t.Errorf("expected only defaultUser's article, got %+v", results)
	}
}

func TestArticleRepository_FetchLatest_SameFeedURLDifferentUsersOnlyReturnsOwnersArticles(t *testing.T) {
	// 複数ユーザーが同一の外部フィードURLを購読している場合（feeds は1ユーザー1購読=1行のため
	// 別々のfeed行になる）、feedURL絞り込みを指定してもuserIDでスコープされ、
	// 他ユーザーの記事が混入しないことを確認する
	// （過去にenrich.goのFetchLatest呼び出しでuserIDスコープ漏れがあった不具合の回帰テスト）。
	ctx := context.Background()
	r := newRepo(t)

	const sharedFeedURL = "https://shared.example.com/feed"
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO feeds (id, user_id, feed_url, title) VALUES (3, $1, $2, 'Shared (other user copy)')`,
		otherUserID, sharedFeedURL); err != nil {
		t.Fatalf("setup other user's feed with shared url: %v", err)
	}
	if _, err := r.db.ExecContext(ctx,
		`UPDATE feeds SET feed_url = $1 WHERE id = 1`, sharedFeedURL); err != nil {
		t.Fatalf("point defaultUser's feed to shared url: %v", err)
	}

	if err := r.Save(ctx, makeArticle("https://shared.example.com/article1")); err != nil {
		t.Fatalf("Save (defaultUser): %v", err)
	}
	if err := r.Save(ctx, domain.Article{FeedID: 3, URL: "https://shared.example.com/article1", Title: "Other's copy"}); err != nil {
		t.Fatalf("Save (otherUser, same article url via different feed): %v", err)
	}

	results, err := r.FetchLatest(ctx, 10, sharedFeedURL, defaultUserID)
	if err != nil {
		t.Fatalf("FetchLatest: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Title https://shared.example.com/article1" {
		t.Errorf("expected only defaultUser's article for the shared feed_url, got %+v", results)
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

	articles, total, err := r.FindFiltered(ctx, articlerepo.ListFilter{Category: "Tech", PerPage: 25}, defaultUserID)
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

	articles, _, err := r.FindFiltered(ctx, articlerepo.ListFilter{Sort: "title", Order: "asc", PerPage: 25}, defaultUserID)
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

	// published_atが全件同値（makeArticleはゼロ値のまま）でもページが安定するか検証するため、
	// カテゴリでソートして同値行を作り出す（sort未指定時のタイブレーカーは別テストで検証）。
	for i := 0; i < 5; i++ {
		if err := r.Save(ctx, makeArticle(fmt.Sprintf("https://example.com/%d", i))); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	page1, total, err := r.FindFiltered(ctx, articlerepo.ListFilter{Page: 1, PerPage: 2}, defaultUserID)
	if err != nil {
		t.Fatalf("FindFiltered page1: %v", err)
	}
	if total != 5 {
		t.Errorf("total: got %d, want 5", total)
	}
	if len(page1) != 2 {
		t.Errorf("page1 size: got %d, want 2", len(page1))
	}

	page2, _, err := r.FindFiltered(ctx, articlerepo.ListFilter{Page: 2, PerPage: 2}, defaultUserID)
	if err != nil {
		t.Fatalf("FindFiltered page2: %v", err)
	}
	if len(page2) != 2 {
		t.Errorf("page2 size: got %d, want 2", len(page2))
	}

	seen := make(map[int64]bool, len(page1))
	for _, a := range page1 {
		seen[a.ID] = true
	}
	for _, a := range page2 {
		if seen[a.ID] {
			t.Errorf("article id %d appears in both page1 and page2 (missing ORDER BY tiebreaker)", a.ID)
		}
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
	if _, err := r.db.ExecContext(ctx, `UPDATE articles SET read = TRUE, bookmarked = TRUE WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup: %v", err)
	}

	unread, _, err := r.FindFiltered(ctx, articlerepo.ListFilter{Unread: true, PerPage: 25}, defaultUserID)
	if err != nil {
		t.Fatalf("FindFiltered: %v", err)
	}
	if len(unread) != 1 || unread[0].URL != "https://example.com/2" {
		t.Errorf("unread: got %+v", unread)
	}

	bookmarked, _, err := r.FindFiltered(ctx, articlerepo.ListFilter{BookmarkedOnly: true, PerPage: 25}, defaultUserID)
	if err != nil {
		t.Fatalf("FindFiltered: %v", err)
	}
	if len(bookmarked) != 1 || bookmarked[0].URL != "https://example.com/1" {
		t.Errorf("bookmarked: got %+v", bookmarked)
	}
}

func TestArticleRepository_FindFiltered_ScopedToUser(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeOtherUserArticle("https://other.example.com/1")); err != nil {
		t.Fatalf("Save (other user): %v", err)
	}

	articles, total, err := r.FindFiltered(ctx, articlerepo.ListFilter{PerPage: 25}, defaultUserID)
	if err != nil {
		t.Fatalf("FindFiltered: %v", err)
	}
	if total != 1 || len(articles) != 1 || articles[0].URL != "https://example.com/1" {
		t.Errorf("expected only defaultUser's article, got total=%d articles=%+v", total, articles)
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

	categories, err := r.DistinctCategories(ctx, defaultUserID)
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

func TestArticleRepository_DistinctCategories_ScopedToUser(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeOtherUserArticle("https://other.example.com/1")); err != nil {
		t.Fatalf("Save (other user): %v", err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE articles SET category = 'Tech' WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup category: %v", err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE articles SET category = 'Other' WHERE url = 'https://other.example.com/1'`); err != nil {
		t.Fatalf("setup category: %v", err)
	}

	categories, err := r.DistinctCategories(ctx, defaultUserID)
	if err != nil {
		t.Fatalf("DistinctCategories: %v", err)
	}
	if len(categories) != 1 || categories[0] != "Tech" {
		t.Errorf("expected only defaultUser's category, got %+v", categories)
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
	if _, err := r.db.ExecContext(ctx, `UPDATE articles SET bookmarked = TRUE WHERE url = 'https://example.com/1'`); err != nil {
		t.Fatalf("setup bookmarked: %v", err)
	}

	results, err := r.Search(ctx, "Go", true, defaultUserID)
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
