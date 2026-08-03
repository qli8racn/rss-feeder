package migration

import (
	"database/sql"
	"fmt"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

// RunPostgres は SQLite 版の Run 相当のスキーマ初期化を Postgres 向けに行う。
// Postgres は ADD COLUMN IF NOT EXISTS をネイティブサポートするため、SQLite版のような
// 「エラーメッセージ文字列を見て無視する」実装（addArticleColumns）は不要になる。
//
// 注意: 下記は CREATE TABLE IF NOT EXISTS のため、既にテーブルが作成済みの環境では
// ここでの制約定義（例: audit_log.article_id の ON DELETE 句）を変更しても反映されない。
// 既存環境に反映するには別途 ALTER TABLE が必要（本ブランチはまだ本番運用前のため未対応）。
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
		-- Postgresは常にFKを強制する（SQLite実装はforeign_keysプラグマを有効化しておらず
		-- articlesの削除を自由に行えている）ため、ON DELETE SET NULLを明示しないと
		-- reset（DeleteNonBookmarked）やremove-feed（Remove）でaudit_logから参照されている
		-- 記事を削除しようとした際にFK違反になる。監査ログの行自体は残しつつ、
		-- 参照先の記事が削除済みであることをarticle_id=NULLで表現する。
		CREATE TABLE IF NOT EXISTS audit_log (
			id         BIGSERIAL PRIMARY KEY,
			action     TEXT NOT NULL,
			article_id BIGINT,
			old_state  TEXT,
			new_state  TEXT,
			timestamp  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(article_id) REFERENCES articles(id) ON DELETE SET NULL
		);

		ALTER TABLE articles ADD COLUMN IF NOT EXISTS publisher TEXT;
		ALTER TABLE articles ADD COLUMN IF NOT EXISTS thumbnail_url TEXT;
		ALTER TABLE articles ADD COLUMN IF NOT EXISTS summary TEXT;
		ALTER TABLE articles ADD COLUMN IF NOT EXISTS category TEXT;
	`)
	if err != nil {
		return err
	}
	return addUserManagementPostgres(db)
}

// addUserManagementPostgres は users テーブルを新設し、feeds をユーザーに紐付ける
// （SQLite版 internal/migration/migration.go の addUserManagement に相当）。
// SQLiteはユニーク制約を ALTER TABLE で直接変更できずテーブル再作成が必要だったが、
// Postgresは ALTER TABLE ... DROP/ADD CONSTRAINT を直接サポートするため、制約の
// 付け替えだけで済む（テーブル再作成は不要）。全ステップが繰り返し実行しても安全
// （IF NOT EXISTS・ON CONFLICT DO NOTHING・pg_constraintでの存在確認）なため、
// SQLite版のようなバージョン管理（PRAGMA user_version）は導入していない。
func addUserManagementPostgres(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id         BIGSERIAL PRIMARY KEY,
			name       TEXT UNIQUE NOT NULL,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`INSERT INTO users (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`, domain.DefaultUserName); err != nil {
		return err
	}
	var defaultUserID int64
	if err := db.QueryRow(`SELECT id FROM users WHERE name = $1`, domain.DefaultUserName).Scan(&defaultUserID); err != nil {
		return fmt.Errorf("デフォルトユーザーの取得に失敗しました: %w", err)
	}

	if _, err := db.Exec(`ALTER TABLE feeds ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id)`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE feeds SET user_id = $1 WHERE user_id IS NULL`, defaultUserID); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE feeds ALTER COLUMN user_id SET NOT NULL`); err != nil {
		return err
	}

	// feed_url単体のUNIQUE制約をUNIQUE(user_id, feed_url)に、url単体のUNIQUE制約を
	// UNIQUE(feed_id, url)に、それぞれ付け替える（SQLite版と同じ理由。
	// docs/steering/20260726_mcp_user_management/design.md 参照）。
	// 制約名はCREATE TABLE時にPostgresが自動生成する既定の命名規則（<table>_<column>_key）に基づく。
	if err := swapUniqueConstraint(db, "feeds", "feeds_feed_url_key", "feeds_user_id_feed_url_key", "UNIQUE (user_id, feed_url)"); err != nil {
		return err
	}
	if err := swapUniqueConstraint(db, "articles", "articles_url_key", "articles_feed_id_url_key", "UNIQUE (feed_id, url)"); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_feeds_user_id ON feeds(user_id);
		CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id);
	`); err != nil {
		return err
	}
	return nil
}

// swapUniqueConstraint は table の oldConstraint（存在すれば）を削除し、newConstraint が
// 未作成であれば newDefinition で追加する。PostgresはADD CONSTRAINT IF NOT EXISTSを
// サポートしないため、pg_constraintを見て存在確認してから追加する（table・oldConstraint・
// newConstraint・newDefinitionは全て呼び出し元のGoコード上の定数でユーザー入力ではないため、
// 文字列埋め込みでもSQLインジェクションの余地はない）。
func swapUniqueConstraint(db *sql.DB, table, oldConstraint, newConstraint, newDefinition string) error {
	if _, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s`, table, oldConstraint)); err != nil {
		return err
	}

	var exists bool
	if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = $1)`, newConstraint).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD CONSTRAINT %s %s`, table, newConstraint, newDefinition))
	return err
}
