#!/usr/bin/env bash
# users テーブルを新設し、feeds をユーザーに紐付ける（MCPサーバー利用者単位のフィード管理）。
# 既存の全フィード・記事は「default」ユーザーに紐付ける形でマイグレーションし、データ損失を発生させない。
# feeds は UNIQUE(feed_url) から UNIQUE(user_id, feed_url) へ、articles は UNIQUE(url) から
# UNIQUE(feed_id, url) へ、それぞれユニーク制約を変更する（SQLiteはALTER TABLEでユニーク制約を
# 直接変更できないため、テーブル再作成手順を用いる）。
# 複数回実行しても安全（冪等）。詳細は docs/steering/20260726_mcp_user_management/design.md 参照。
set -euo pipefail

DB="${1:-rss-feeder-db/reader.db}"

# articles 再作成時に publisher/thumbnail_url/summary/category 列を参照するため、
# 20260614_add_article_metadata.sh 未適用のDBに対して単独実行しても失敗しないよう、
# ここで先に列追加を済ませておく（列が既にある場合は duplicate column name を無視、冪等）。
for col in \
    "ALTER TABLE articles ADD COLUMN publisher     TEXT" \
    "ALTER TABLE articles ADD COLUMN thumbnail_url TEXT" \
    "ALTER TABLE articles ADD COLUMN summary       TEXT" \
    "ALTER TABLE articles ADD COLUMN category      TEXT"
do
    err="$(sqlite3 "$DB" "$col" 2>&1)" || {
        echo "$err" | grep -q "duplicate column name" || { echo "$err" >&2; exit 1; }
    }
done

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
# 「feeds RENAME TO feeds_old」を先に行うと、SQLite 3.25以降ではALTER TABLE X RENAME TO Y が
# 他テーブルのREFERENCES X(...)句もYへ自動書き換えてしまう（articles.feed_id REFERENCES feeds(id)が
# REFERENCES feeds_old(id)になる）。articlesは直後に再作成されるため偶然自己修復するが、
# 順序依存で危ういため、誰からも参照されていないfeeds_newをRENAME先にする手順に統一する。
FEEDS_SQL="$(sqlite3 "$DB" "SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'feeds'")"
if [[ "$FEEDS_SQL" != *"UNIQUE(user_id, feed_url)"* ]]; then
    sqlite3 "$DB" <<'EOF'
BEGIN TRANSACTION;

CREATE TABLE feeds_new (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER  NOT NULL REFERENCES users(id),
    feed_url     TEXT     NOT NULL,
    title        TEXT,
    last_fetched DATETIME,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, feed_url)
);

INSERT INTO feeds_new (id, user_id, feed_url, title, last_fetched, created_at)
SELECT id, user_id, feed_url, title, last_fetched, created_at FROM feeds;

DROP TABLE feeds;

ALTER TABLE feeds_new RENAME TO feeds;

CREATE INDEX IF NOT EXISTS idx_feeds_user_id ON feeds(user_id);

COMMIT;
EOF
fi

# articles: UNIQUE(url) -> UNIQUE(feed_id, url) へのテーブル再作成（既に再作成済みならスキップ）
# feeds が「1ユーザー1購読=1行」になったことで、同じ外部フィードURLを複数ユーザーが購読すると
# 記事も別行として重複保存される設計になるため、articles.url のグローバルなユニーク制約は
# 2人目以降のユーザーの記事保存をブロックしてしまう。これを避けるための変更。
# feeds と同じ理由（audit_log.article_id REFERENCES articles(id) が RENAME により
# 書き換わってしまうのを防ぐため）で、articles_new を作成してから旧articlesをDROP・
# articles_newをarticlesへRENAMEする順序を採る。
ARTICLES_SQL="$(sqlite3 "$DB" "SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'articles'")"
if [[ "$ARTICLES_SQL" != *"UNIQUE(feed_id, url)"* ]]; then
    sqlite3 "$DB" <<'EOF'
BEGIN TRANSACTION;

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
    FOREIGN KEY (feed_id) REFERENCES feeds(id),
    UNIQUE(feed_id, url)
);

INSERT INTO articles_new (id, feed_id, url, title, content, published_at, read, bookmarked,
    fetched_at, publisher, thumbnail_url, summary, category)
SELECT id, feed_id, url, title, content, published_at, read, bookmarked,
    fetched_at, publisher, thumbnail_url, summary, category FROM articles;

DROP TABLE articles;

ALTER TABLE articles_new RENAME TO articles;

CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id);

COMMIT;
EOF
fi
