package migration

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	})
	return db
}

func createSQLFor(t *testing.T, db *sql.DB, table string) string {
	t.Helper()
	var s string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&s); err != nil {
		t.Fatalf("read sqlite_master for %s: %v", table, err)
	}
	return s
}

func TestRun_CreatesUsersTableWithDefaultUser(t *testing.T) {
	db := newTestDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var name string
	if err := db.QueryRow(`SELECT name FROM users WHERE name = ?`, domain.DefaultUserName).Scan(&name); err != nil {
		t.Fatalf("expected default user to exist: %v", err)
	}
	if name != domain.DefaultUserName {
		t.Errorf("name: got %q, want %q", name, domain.DefaultUserName)
	}
}

func TestRun_FeedsTableHasUserScopedUniqueConstraint(t *testing.T) {
	db := newTestDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("Run: %v", err)
	}

	createSQL := createSQLFor(t, db, "feeds")
	if !strings.Contains(createSQL, feedsUniqueConstraint) {
		t.Errorf("feeds table should have %q, got schema: %s", feedsUniqueConstraint, createSQL)
	}
}

// TestRun_PreservesAuditLogForeignKey は、feeds/articles のテーブル再作成手順が
// 「旧テーブルをリネームしてから新テーブルを作る」順序に戻っていないことを確認する回帰テスト。
// その順序だとSQLite 3.25以降ではRENAMEにより audit_log.article_id の REFERENCES 句が
// articles_old(id) 等へ書き換わってしまう（design.mdの「実装時の追記」7参照）。
func TestRun_PreservesAuditLogForeignKey(t *testing.T) {
	db := newTestDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("Run: %v", err)
	}

	createSQL := createSQLFor(t, db, "audit_log")
	if !strings.Contains(createSQL, "REFERENCES articles(id)") {
		t.Errorf("audit_log の FK が articles(id) を指していない（rename-old-first に戻った可能性）: %s", createSQL)
	}
}

func TestRun_ArticlesTableHasFeedScopedUniqueConstraint(t *testing.T) {
	db := newTestDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("Run: %v", err)
	}

	createSQL := createSQLFor(t, db, "articles")
	if !strings.Contains(createSQL, articlesUniqueConstraint) {
		t.Errorf("articles table should have %q, got schema: %s", articlesUniqueConstraint, createSQL)
	}
}

// TestRun_ExistingDataMigratesToDefaultUser は、users テーブル新設前に保存されていた
// 既存フィード・記事がすべてデフォルトユーザーに紐付き、データ損失が発生しないことを確認する
// （旧バージョンのDBに対して初めてマイグレーションを実行するケースを再現する）。
func TestRun_ExistingDataMigratesToDefaultUser(t *testing.T) {
	db := newTestDB(t)

	// users テーブル新設前の古いスキーマを再現する。
	if _, err := db.Exec(`
		CREATE TABLE feeds (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			feed_url     TEXT UNIQUE NOT NULL,
			title        TEXT,
			last_fetched DATETIME,
			created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE articles (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			feed_id      INTEGER NOT NULL,
			url          TEXT UNIQUE NOT NULL,
			title        TEXT NOT NULL,
			content      TEXT,
			published_at DATETIME,
			read         BOOLEAN DEFAULT 0,
			bookmarked   BOOLEAN DEFAULT 0,
			fetched_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(feed_id) REFERENCES feeds(id)
		);
	`); err != nil {
		t.Fatalf("setup old schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO feeds (id, feed_url, title) VALUES (1, 'https://example.com/feed', 'Example')`); err != nil {
		t.Fatalf("setup feed: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO articles (feed_id, url, title) VALUES (1, 'https://example.com/1', 'Article1')`); err != nil {
		t.Fatalf("setup article: %v", err)
	}

	if err := Run(db); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var defaultUserID int64
	if err := db.QueryRow(`SELECT id FROM users WHERE name = ?`, domain.DefaultUserName).Scan(&defaultUserID); err != nil {
		t.Fatalf("expected default user: %v", err)
	}

	var feedUserID int64
	var feedURL string
	if err := db.QueryRow(`SELECT user_id, feed_url FROM feeds WHERE id = 1`).Scan(&feedUserID, &feedURL); err != nil {
		t.Fatalf("query migrated feed: %v", err)
	}
	if feedUserID != defaultUserID {
		t.Errorf("feed.user_id: got %d, want %d (default user)", feedUserID, defaultUserID)
	}
	if feedURL != "https://example.com/feed" {
		t.Errorf("feed.feed_url: got %q, want unchanged", feedURL)
	}

	var articleCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM articles WHERE feed_id = 1`).Scan(&articleCount); err != nil {
		t.Fatalf("count articles: %v", err)
	}
	if articleCount != 1 {
		t.Errorf("articles for feed 1: got %d, want 1 (no data loss)", articleCount)
	}
}

func TestRun_IsIdempotent(t *testing.T) {
	db := newTestDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO feeds (user_id, feed_url, title) VALUES (
		(SELECT id FROM users WHERE name = ?), 'https://example.com/feed', 'Example')`, domain.DefaultUserName); err != nil {
		t.Fatalf("insert feed after first Run: %v", err)
	}

	if err := Run(db); err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if err := Run(db); err != nil {
		t.Fatalf("third Run: %v", err)
	}

	var userCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE name = ?`, domain.DefaultUserName).Scan(&userCount); err != nil {
		t.Fatalf("count default users: %v", err)
	}
	if userCount != 1 {
		t.Errorf("default user count: got %d, want 1 (repeated Run should not duplicate)", userCount)
	}

	var feedCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM feeds WHERE feed_url = 'https://example.com/feed'`).Scan(&feedCount); err != nil {
		t.Fatalf("count feeds: %v", err)
	}
	if feedCount != 1 {
		t.Errorf("feed count: got %d, want 1 (repeated Run should not lose or duplicate data)", feedCount)
	}
}

// TestRun_NewMigrationStepAppliesExactlyOnceAcrossRepeatedRuns は、PRAGMA user_version による
// バージョン管理が正しく機能していることを、実際に新しいステップを1つ追加した状態で検証する。
// migrationSteps の末尾に追加したステップが、複数回 Run を呼んでも常にちょうど1回だけ適用され、
// 既存の全ステップが完了済みのDBに対しても（早期returnで丸ごと無視されずに）確実に適用される
// ことを確認する。
func TestRun_NewMigrationStepAppliesExactlyOnceAcrossRepeatedRuns(t *testing.T) {
	db := newTestDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("initial Run (existing steps only): %v", err)
	}

	originalSteps := migrationSteps
	t.Cleanup(func() { migrationSteps = originalSteps })

	calls := 0
	newVersion := originalSteps[len(originalSteps)-1].version + 1
	migrationSteps = append(append([]migrationStep{}, originalSteps...), migrationStep{
		version: newVersion,
		apply: func(db *sql.DB) error {
			calls++
			return nil
		},
	})

	for i := 0; i < 3; i++ {
		if err := Run(db); err != nil {
			t.Fatalf("Run #%d after adding new step: %v", i+1, err)
		}
	}

	if calls != 1 {
		t.Errorf("new step apply called %d times, want 1 (already-completed steps must not rerun, but a genuinely new step must still run against an already-migrated DB)", calls)
	}

	version, err := getUserVersion(db)
	if err != nil {
		t.Fatalf("getUserVersion: %v", err)
	}
	if version != newVersion {
		t.Errorf("user_version after new step: got %d, want %d", version, newVersion)
	}
}

// TestRun_BackfillsVersionForPreVersioningMigratedDB は、PRAGMA user_version 導入前の
// isFullyMigrated のみでの判定によって既に「移行済み」だったDB（PRAGMA user_version は
// 0 のまま）に対して、Run が version 1 の実処理を再実行せずに user_version だけを
// backfill することを確認する。
func TestRun_BackfillsVersionForPreVersioningMigratedDB(t *testing.T) {
	db := newTestDB(t)
	// Run を経由せず、version 1 到達後の最終スキーマ相当を直接構築する
	// （PRAGMA user_version 導入前の既存DBを模す。user_version はデフォルトの0のまま）。
	if _, err := db.Exec(`
		CREATE TABLE users (
			id         INTEGER  PRIMARY KEY AUTOINCREMENT,
			name       TEXT     UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO users (name) VALUES ('` + domain.DefaultUserName + `');
		CREATE TABLE feeds (
			id           INTEGER  PRIMARY KEY AUTOINCREMENT,
			user_id      INTEGER  NOT NULL REFERENCES users(id),
			feed_url     TEXT     NOT NULL,
			title        TEXT,
			last_fetched DATETIME,
			created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, feed_url)
		);
		CREATE TABLE articles (
			id            INTEGER  PRIMARY KEY AUTOINCREMENT,
			feed_id       INTEGER  NOT NULL,
			url           TEXT     NOT NULL,
			title         TEXT     NOT NULL,
			content       TEXT,
			published_at  DATETIME,
			read          BOOLEAN  DEFAULT 0,
			bookmarked    BOOLEAN  DEFAULT 0,
			fetched_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
			publisher     TEXT,
			thumbnail_url TEXT,
			summary       TEXT,
			category      TEXT,
			FOREIGN KEY(feed_id) REFERENCES feeds(id),
			UNIQUE(feed_id, url)
		);
	`); err != nil {
		t.Fatalf("seed pre-versioning migrated schema: %v", err)
	}

	if err := Run(db); err != nil {
		t.Fatalf("Run: %v", err)
	}

	version, err := getUserVersion(db)
	if err != nil {
		t.Fatalf("getUserVersion: %v", err)
	}
	if version != 1 {
		t.Errorf("user_version after backfill: got %d, want 1", version)
	}

	var userCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE name = ?`, domain.DefaultUserName).Scan(&userCount); err != nil {
		t.Fatalf("count default users: %v", err)
	}
	if userCount != 1 {
		t.Errorf("default user count: got %d, want 1 (backfill must not rerun applyBaseMigration)", userCount)
	}
}

func TestRun_MultipleUsersCanSubscribeToSameFeedURL(t *testing.T) {
	// feeds.UNIQUE(user_id, feed_url) により、異なるユーザーは同一の外部フィードURLを
	// それぞれ独立に購読できることを確認する。
	db := newTestDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO users (name) VALUES ('alice'), ('bob')`); err != nil {
		t.Fatalf("insert users: %v", err)
	}

	var aliceID, bobID int64
	if err := db.QueryRow(`SELECT id FROM users WHERE name = 'alice'`).Scan(&aliceID); err != nil {
		t.Fatalf("find alice: %v", err)
	}
	if err := db.QueryRow(`SELECT id FROM users WHERE name = 'bob'`).Scan(&bobID); err != nil {
		t.Fatalf("find bob: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO feeds (user_id, feed_url) VALUES (?, ?)`, aliceID, "https://shared.example.com/feed"); err != nil {
		t.Fatalf("alice subscribe: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO feeds (user_id, feed_url) VALUES (?, ?)`, bobID, "https://shared.example.com/feed"); err != nil {
		t.Fatalf("bob subscribe to same feed_url should succeed: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM feeds WHERE feed_url = ?`, "https://shared.example.com/feed").Scan(&count); err != nil {
		t.Fatalf("count feeds: %v", err)
	}
	if count != 2 {
		t.Errorf("feed rows for shared feed_url: got %d, want 2 (one per user)", count)
	}
}

func TestRun_MultipleUsersCanSaveArticleWithSameURL(t *testing.T) {
	// articles.UNIQUE(feed_id, url) により、異なるユーザーの feed_id に紐づく記事であれば
	// 同一の記事URLを重複なく保存できることを確認する
	// （articles.UNIQUE(url) 単体のままだと2人目のユーザーの記事保存がブロックされてしまう）。
	db := newTestDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO users (name) VALUES ('alice'), ('bob')`); err != nil {
		t.Fatalf("insert users: %v", err)
	}

	res1, err := db.Exec(`INSERT INTO feeds (user_id, feed_url) VALUES ((SELECT id FROM users WHERE name = 'alice'), 'https://shared.example.com/feed')`)
	if err != nil {
		t.Fatalf("alice subscribe: %v", err)
	}
	aliceFeedID, _ := res1.LastInsertId()

	res2, err := db.Exec(`INSERT INTO feeds (user_id, feed_url) VALUES ((SELECT id FROM users WHERE name = 'bob'), 'https://shared.example.com/feed')`)
	if err != nil {
		t.Fatalf("bob subscribe: %v", err)
	}
	bobFeedID, _ := res2.LastInsertId()

	if _, err := db.Exec(`INSERT INTO articles (feed_id, url, title) VALUES (?, ?, ?)`, aliceFeedID, "https://shared.example.com/article1", "Article1"); err != nil {
		t.Fatalf("alice save article: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO articles (feed_id, url, title) VALUES (?, ?, ?)`, bobFeedID, "https://shared.example.com/article1", "Article1"); err != nil {
		t.Fatalf("bob save article with same url should succeed: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM articles WHERE url = ?`, "https://shared.example.com/article1").Scan(&count); err != nil {
		t.Fatalf("count articles: %v", err)
	}
	if count != 2 {
		t.Errorf("article rows for shared url: got %d, want 2 (one per user's feed)", count)
	}
}
