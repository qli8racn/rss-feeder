package migration

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

func Run(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS feeds (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			feed_url     TEXT UNIQUE NOT NULL,
			title        TEXT,
			last_fetched DATETIME,
			created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS articles (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			feed_id      INTEGER NOT NULL,
			url          TEXT UNIQUE NOT NULL,
			title        TEXT NOT NULL,
			content      TEXT,
			published_at DATETIME,
			read         BOOLEAN DEFAULT 0,
			bookmarked   BOOLEAN DEFAULT 0,
			fetched_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			publisher     TEXT,
			thumbnail_url TEXT,
			summary       TEXT,
			category      TEXT,
			FOREIGN KEY(feed_id) REFERENCES feeds(id)
		);
		CREATE TABLE IF NOT EXISTS audit_log (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			action     TEXT NOT NULL,
			article_id INTEGER,
			old_state  TEXT,
			new_state  TEXT,
			timestamp  DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(article_id) REFERENCES articles(id)
		);
	`)
	if err != nil {
		return err
	}
	if err := addArticleColumns(db); err != nil {
		return err
	}
	return addUserManagement(db)
}

// addArticleColumns は既存DBに対して articles テーブルへ新規カラムを追加する。
// SQLite は ADD COLUMN IF NOT EXISTS をサポートしないため、
// 既にカラムが存在する場合のエラー（duplicate column name）は無視する。
func addArticleColumns(db *sql.DB) error {
	columns := []string{
		"ALTER TABLE articles ADD COLUMN publisher TEXT",
		"ALTER TABLE articles ADD COLUMN thumbnail_url TEXT",
		"ALTER TABLE articles ADD COLUMN summary TEXT",
		"ALTER TABLE articles ADD COLUMN category TEXT",
	}
	for _, stmt := range columns {
		if _, err := db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	return nil
}

// feedsUniqueConstraint / articlesUniqueConstraint は、テーブル再作成後のスキーマに
// 含まれるユニーク制約の文字列。sqlite_master.sql に含まれるかどうかで、
// 既にテーブル再作成済み（マイグレーション済み）かどうかを冪等に判定する。
const (
	feedsUniqueConstraint    = "UNIQUE(user_id, feed_url)"
	articlesUniqueConstraint = "UNIQUE(feed_id, url)"
)

// addUserManagement は users テーブルを新設し、feeds をユーザーに紐付ける。
// 既存の全フィード・記事はデフォルトユーザー（domain.DefaultUserName）に紐付ける形で
// マイグレーションし、データ損失を発生させない。
//
// articles.url は元々グローバルなUNIQUE制約だったが、feeds が「1ユーザー1購読=1行」
// （同じ外部フィードURLを複数ユーザーが購読すると、ユーザーごとに別のfeeds行になる）設計を
// 採るため、articles も UNIQUE(feed_id, url) に変更しないと、複数ユーザーが同じ記事URLを
// 保存しようとした際に2人目以降の記事が articles.url の重複としてサイレントに保存されなくなる
// （users追加に伴い必然的に必要になる変更のため、ここに含める）。
func addUserManagement(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id         INTEGER  PRIMARY KEY AUTOINCREMENT,
			name       TEXT     UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}

	defaultUserID, err := ensureDefaultUser(db)
	if err != nil {
		return err
	}

	if _, err := db.Exec(`ALTER TABLE feeds ADD COLUMN user_id INTEGER REFERENCES users(id)`); err != nil &&
		!strings.Contains(err.Error(), "duplicate column name") {
		return err
	}
	if _, err := db.Exec(`UPDATE feeds SET user_id = ? WHERE user_id IS NULL`, defaultUserID); err != nil {
		return err
	}

	if err := recreateFeedsTableWithUserScope(db); err != nil {
		return err
	}
	return recreateArticlesTableWithFeedScope(db)
}

// ensureDefaultUser は name = domain.DefaultUserName の行が無ければ作成し、そのIDを返す。
func ensureDefaultUser(db *sql.DB) (int64, error) {
	if _, err := db.Exec(`INSERT INTO users (name) VALUES (?) ON CONFLICT(name) DO NOTHING`, domain.DefaultUserName); err != nil {
		return 0, err
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM users WHERE name = ?`, domain.DefaultUserName).Scan(&id); err != nil {
		return 0, fmt.Errorf("デフォルトユーザーの取得に失敗しました: %w", err)
	}
	return id, nil
}

// tableHasUniqueConstraint は sqlite_master に保存されているテーブル定義のSQLに
// substr が含まれるかどうかを返す（テーブル再作成済みかどうかの冪等性判定に使う）。
func tableHasUniqueConstraint(db *sql.DB, table, substr string) (bool, error) {
	var createSQL sql.NullString
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&createSQL)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return createSQL.Valid && strings.Contains(createSQL.String, substr), nil
}

// recreateFeedsTableWithUserScope は feeds テーブルのユニーク制約を
// feed_url単体からUNIQUE(user_id, feed_url)へ変更する（SQLiteはALTER TABLEで
// 直接ユニーク制約を変更できないため、テーブル再作成手順が必要）。
// articles.feed_id は数値IDをそのまま参照しているため、id を明示的に指定してコピーすることで
// articles側の変更なしに整合性を維持する。既に再作成済みの場合は何もしない（冪等）。
//
// 手順は「新テーブルを別名(feeds_new)で作成→コピー→旧テーブルをDROP→新テーブルを
// 本来の名前にRENAME」の順（SQLite公式推奨の12-step手順）で行う。
// 旧手順（feeds RENAME TO feeds_old → 新feeds作成）だと、SQLite 3.25以降では
// ALTER TABLE X RENAME TO Y が他テーブルのREFERENCES X(...)句もYへ自動書き換えてしまうため、
// articles.feed_id REFERENCES feeds(id) が REFERENCES feeds_old(id) に書き換わってしまう
// （直後にfeeds_oldをDROPして新feedsを作り直すため偶然自己修復していただけ）。
// 本手順ではRENAME対象が誰からも参照されていないfeeds_newであるため、articles側のFK句は
// 一切書き換わらない。
func recreateFeedsTableWithUserScope(db *sql.DB) error {
	migrated, err := tableHasUniqueConstraint(db, "feeds", feedsUniqueConstraint)
	if err != nil {
		return err
	}
	if migrated {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
		CREATE TABLE feeds_new (
			id           INTEGER  PRIMARY KEY AUTOINCREMENT,
			user_id      INTEGER  NOT NULL REFERENCES users(id),
			feed_url     TEXT     NOT NULL,
			title        TEXT,
			last_fetched DATETIME,
			created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			` + feedsUniqueConstraint + `
		)
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO feeds_new (id, user_id, feed_url, title, last_fetched, created_at)
		SELECT id, user_id, feed_url, title, last_fetched, created_at FROM feeds
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE feeds`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE feeds_new RENAME TO feeds`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_feeds_user_id ON feeds(user_id)`); err != nil {
		return err
	}
	return tx.Commit()
}

// recreateArticlesTableWithFeedScope は articles テーブルのユニーク制約を
// url単体からUNIQUE(feed_id, url)へ変更する。recreateFeedsTableWithUserScope と同じ理由
// （SQLiteはALTER TABLEでユニーク制約を変更できない）でテーブル再作成が必要。
// 既に再作成済みの場合は何もしない（冪等）。
//
// recreateFeedsTableWithUserScope と同じ理由（audit_log.article_id REFERENCES articles(id)
// が RENAME により書き換わってしまうのを防ぐため）で、articles_new を作成してから
// 旧articlesをDROP・articles_newをarticlesへRENAMEする順序を採る。
func recreateArticlesTableWithFeedScope(db *sql.DB) error {
	migrated, err := tableHasUniqueConstraint(db, "articles", articlesUniqueConstraint)
	if err != nil {
		return err
	}
	if migrated {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
		CREATE TABLE articles_new (
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
			` + articlesUniqueConstraint + `
		)
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO articles_new (id, feed_id, url, title, content, published_at, read, bookmarked,
			fetched_at, publisher, thumbnail_url, summary, category)
		SELECT id, feed_id, url, title, content, published_at, read, bookmarked,
			fetched_at, publisher, thumbnail_url, summary, category FROM articles
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE articles`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE articles_new RENAME TO articles`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id)`); err != nil {
		return err
	}
	return tx.Commit()
}
