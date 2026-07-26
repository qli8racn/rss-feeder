package migration

import "database/sql"

// RunPostgres は SQLite 版の Run 相当のスキーマ初期化を Postgres 向けに行う。
// Postgres は ADD COLUMN IF NOT EXISTS をネイティブサポートするため、SQLite版のような
// 「エラーメッセージ文字列を見て無視する」実装（addArticleColumns）は不要になる。
func RunPostgres(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS feeds (
			id           BIGSERIAL PRIMARY KEY,
			feed_url     TEXT UNIQUE NOT NULL,
			title        TEXT,
			last_fetched TIMESTAMPTZ,
			created_at   TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS articles (
			id           BIGSERIAL PRIMARY KEY,
			feed_id      BIGINT NOT NULL,
			url          TEXT UNIQUE NOT NULL,
			title        TEXT NOT NULL,
			content      TEXT,
			published_at TIMESTAMPTZ,
			read         BOOLEAN DEFAULT FALSE,
			bookmarked   BOOLEAN DEFAULT FALSE,
			fetched_at   TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			publisher     TEXT,
			thumbnail_url TEXT,
			summary       TEXT,
			category      TEXT,
			FOREIGN KEY(feed_id) REFERENCES feeds(id)
		);
		CREATE TABLE IF NOT EXISTS audit_log (
			id         BIGSERIAL PRIMARY KEY,
			action     TEXT NOT NULL,
			article_id BIGINT,
			old_state  TEXT,
			new_state  TEXT,
			timestamp  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(article_id) REFERENCES articles(id)
		);

		ALTER TABLE articles ADD COLUMN IF NOT EXISTS publisher TEXT;
		ALTER TABLE articles ADD COLUMN IF NOT EXISTS thumbnail_url TEXT;
		ALTER TABLE articles ADD COLUMN IF NOT EXISTS summary TEXT;
		ALTER TABLE articles ADD COLUMN IF NOT EXISTS category TEXT;
	`)
	return err
}
