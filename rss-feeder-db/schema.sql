-- rss-feeder SQLite schema
-- 新規 DB 作成時はこのファイルをそのまま適用する:
--   sqlite3 rss-feeder-db/reader.db < rss-feeder-db/schema.sql

CREATE TABLE IF NOT EXISTS users (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    name       TEXT     UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- name = 'default' は cmd/rss-feeder・cmd/web・cmd/agent が常に暗黙的に利用するユーザー
-- （internal/domain.DefaultUserName と一致させる）。
INSERT OR IGNORE INTO users (name) VALUES ('default');

CREATE TABLE IF NOT EXISTS feeds (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER  NOT NULL REFERENCES users(id),
    feed_url     TEXT     NOT NULL,
    title        TEXT,
    last_fetched DATETIME,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, feed_url)
);

CREATE INDEX IF NOT EXISTS idx_feeds_user_id ON feeds(user_id);

-- articles は feeds が「1ユーザー1購読=1行」の設計であるため、同じ外部フィードURLを
-- 複数ユーザーが購読すると記事も別行として重複保存される。そのため url 単体ではなく
-- feed_id との複合ユニーク制約にする（詳細は docs/steering/20260726_mcp_user_management/design.md）。
CREATE TABLE IF NOT EXISTS articles (
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

CREATE TABLE IF NOT EXISTS audit_log (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    action     TEXT     NOT NULL,
    article_id INTEGER,
    old_state  TEXT,
    new_state  TEXT,
    timestamp  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (article_id) REFERENCES articles(id)
);
