#!/usr/bin/env bash
# articles テーブルへ記事メタデータカラムを追加する
# SQLite は ADD COLUMN IF NOT EXISTS 非対応のため、duplicate column name エラーは無視する
set -euo pipefail

DB="${1:-rss-feeder-db/reader.db}"

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
