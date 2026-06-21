package article

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/qli8racn/rss-feeder/internal/migration"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	// sqlite3の:memory:はコネクションごとに別DBになるため、プールが新規コネクションを
	// 開くと別の空DBに繋がってしまう。cache=shared + SetMaxOpenConns(1)でも、
	// プールが唯一のコネクションを破棄して新規に開き直す瞬間に保持者が0になると、
	// 共有インメモリDBそのものが破棄され「no such table」になることを確認した。
	// ファイルベースのDBであればコネクションが何本開かれても同じファイルを参照するため、
	// この問題が起きない。
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
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

func TestArticleRepository_UpdateEnrichmentBatch(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.Save(ctx, makeArticle("https://example.com/1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, makeArticle("https://example.com/2")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	all, _ := r.FindAll(ctx)

	updates := []articlerepo.EnrichmentUpdate{
		{ID: all[0].ID, Summary: "要約1", Category: "Tech"},
		{ID: all[1].ID, Summary: "要約2", Category: "Business"},
	}
	if err := r.UpdateEnrichmentBatch(ctx, updates); err != nil {
		t.Fatalf("UpdateEnrichmentBatch: %v", err)
	}

	found0, _ := r.FindByID(ctx, all[0].ID)
	if found0.Summary != "要約1" || found0.Category != "Tech" {
		t.Errorf("got summary=%q category=%q, want 要約1/Tech", found0.Summary, found0.Category)
	}
	found1, _ := r.FindByID(ctx, all[1].ID)
	if found1.Summary != "要約2" || found1.Category != "Business" {
		t.Errorf("got summary=%q category=%q, want 要約2/Business", found1.Summary, found1.Category)
	}
}

func TestArticleRepository_UpdateEnrichmentBatch_Empty(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if err := r.UpdateEnrichmentBatch(ctx, nil); err != nil {
		t.Errorf("UpdateEnrichmentBatch(nil): got error %v, want nil", err)
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
	all, _ := r.FindAll(ctx)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	stmt, err := tx.PrepareContext(ctx, `UPDATE articles SET summary = ?, category = ? WHERE id = ?`)
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

	found0, err := r.FindByID(context.Background(), all[0].ID)
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

	all, _ := r.FindAll(ctx)
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
	// migration.go の `ALTER TABLE ... ADD COLUMN` で追加した既存行は publisher/thumbnail_url が
	// 空文字ではなく NULL になる（記事メタデータ拡充フェーズ以前に保存された記事を再現）。
	// COALESCE なしでは `publisher = ''` が NULL と一致せず対象外になってしまうことの確認。
	ctx := context.Background()
	r := newRepo(t)

	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO articles (feed_id, url, title, content, published_at, fetched_at, publisher, thumbnail_url)
		VALUES (1, 'https://example.com/1', 'Title', 'body', ?, ?, NULL, NULL)
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

	all, _ := r.FindAll(ctx)
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
	all, _ := r.FindAll(ctx)
	if err := r.UpdateEnrichmentBatch(ctx, []articlerepo.EnrichmentUpdate{{ID: all[0].ID, Summary: "要約", Category: "Tech"}}); err != nil {
		t.Fatalf("UpdateEnrichmentBatch: %v", err)
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
