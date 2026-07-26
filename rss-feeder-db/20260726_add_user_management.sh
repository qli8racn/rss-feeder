#!/usr/bin/env bash
# users テーブルを新設し、feeds をユーザーに紐付ける（MCPサーバー利用者単位のフィード管理）。
# 既存の全フィード・記事は「default」ユーザーに紐付ける形でマイグレーションし、データ損失を発生させない。
# feeds は UNIQUE(feed_url) から UNIQUE(user_id, feed_url) へ、articles は UNIQUE(url) から
# UNIQUE(feed_id, url) へ、それぞれユニーク制約を変更する（SQLiteはALTER TABLEでユニーク制約を
# 直接変更できないため、テーブル再作成手順を用いる）。
# 複数回実行しても安全（冪等）。詳細は docs/steering/20260726_mcp_user_management/design.md 参照。
set -euo pipefail

DB="${1:-rss-feeder-db/reader.db}"

sqlite3 "$DB" "CREATE TABLE IF NOT EXISTS users (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    name       TEXT     UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)"

sqlite3 "$DB" "INSERT INTO users (name) VALUES ('default') ON CONFLICT(name) DO NOTHING"
DEFAULT_USER_ID="$(sqlite3 "$DB" "SELECT id FROM users WHERE name = 'default'")"

err="$(sqlite3 "$DB" "ALTER TABLE feeds ADD COLUMN user_id INTEGER REFERENCES users(id)" 2>&1)" || {
    echo "$err" | grep -q "duplicate column name" || { echo "$err" >&2; exit 1; }
}

sqlite3 "$DB" "UPDATE feeds SET user_id = $DEFAULT_USER_ID WHERE user_id IS NULL"

# feeds: UNIQUE(feed_url) -> UNIQUE(user_id, feed_url) へのテーブル再作成（既に再作成済みならスキップ）
FEEDS_SQL="$(sqlite3 "$DB" "SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'feeds'")"
if [[ "$FEEDS_SQL" != *"UNIQUE(user_id, feed_url)"* ]]; then
    sqlite3 "$DB" <<'EOF'
BEGIN TRANSACTION;

ALTER TABLE feeds RENAME TO feeds_old;

CREATE TABLE feeds (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER  NOT NULL REFERENCES users(id),
    feed_url     TEXT     NOT NULL,
    title        TEXT,
    last_fetched DATETIME,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, feed_url)
);

INSERT INTO feeds (id, user_id, feed_url, title, last_fetched, created_at)
SELECT id, user_id, feed_url, title, last_fetched, created_at FROM feeds_old;

DROP TABLE feeds_old;

CREATE INDEX IF NOT EXISTS idx_feeds_user_id ON feeds(user_id);

COMMIT;
EOF
fi

# articles: UNIQUE(url) -> UNIQUE(feed_id, url) へのテーブル再作成（既に再作成済みならスキップ）
# feeds が「1ユーザー1購読=1行」になったことで、同じ外部フィードURLを複数ユーザーが購読すると
# 記事も別行として重複保存される設計になるため、articles.url のグローバルなユニーク制約は
# 2人目以降のユーザーの記事保存をブロックしてしまう。これを避けるための変更。
ARTICLES_SQL="$(sqlite3 "$DB" "SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'articles'")"
if [[ "$ARTICLES_SQL" != *"UNIQUE(feed_id, url)"* ]]; then
    sqlite3 "$DB" <<'EOF'
BEGIN TRANSACTION;

ALTER TABLE articles RENAME TO articles_old;

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
    FOREIGN KEY (feed_id) REFERENCES feeds(id),
    UNIQUE(feed_id, url)
);

INSERT INTO articles (id, feed_id, url, title, content, published_at, read, bookmarked,
    fetched_at, publisher, thumbnail_url, summary, category)
SELECT id, feed_id, url, title, content, published_at, read, bookmarked,
    fetched_at, publisher, thumbnail_url, summary, category FROM articles_old;

DROP TABLE articles_old;

COMMIT;
EOF
fi
