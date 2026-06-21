package migration

import (
	"database/sql"
	"strings"
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
	return addArticleColumns(db)
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
