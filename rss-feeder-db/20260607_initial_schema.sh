#!/usr/bin/env bash
# 初期スキーマ作成: feeds / articles / audit_log
set -euo pipefail

DB="${1:-rss-feeder-db/reader.db}"

sqlite3 "$DB" <<'SQL'
CREATE TABLE IF NOT EXISTS feeds (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    feed_url     TEXT     UNIQUE NOT NULL,
    title        TEXT,
    last_fetched DATETIME,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS articles (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    feed_id      INTEGER  NOT NULL,
    url          TEXT     UNIQUE NOT NULL,
    title        TEXT     NOT NULL,
    content      TEXT,
    published_at DATETIME,
    read         BOOLEAN  DEFAULT 0,
    bookmarked   BOOLEAN  DEFAULT 0,
    fetched_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (feed_id) REFERENCES feeds(id)
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
SQL
