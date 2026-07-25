#!/bin/bash
# devcontainer内のcron（.devcontainer/poll-feeds.cron）から1時間ごとに起動され、RSSフィードの取得・要約を行う。
# rss-feeder-db/reader.db / internal/config/config.yml への参照が相対パスのため、
# 必ずリポジトリルートをカレントディレクトリにして実行する。
set -uo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="$REPO_DIR/logs"
LOG_FILE="$LOG_DIR/poll-feeds.log"

mkdir -p "$LOG_DIR"
cd "$REPO_DIR" || exit 1

log() {
    printf '%s %s\n' "$(TZ='Asia/Tokyo' date '+%Y-%m-%d %H:%M:%S %Z')" "$1" >> "$LOG_FILE"
}

if [ ! -x bin/rss-feeder ] || [ ! -x bin/rss-agent ]; then
    log "ERROR: bin/rss-feeder または bin/rss-agent が見つかりません。go build してください。"
    exit 1
fi

log "START fetch"
FETCH_OUTPUT="$(./bin/rss-feeder fetch 2>&1)"
FETCH_STATUS=$?
log "$FETCH_OUTPUT"
log "fetch exit=$FETCH_STATUS"

log "START enrich"
ENRICH_OUTPUT="$(./bin/rss-agent enrich 2>&1)"
ENRICH_STATUS=$?
log "$ENRICH_OUTPUT"
log "enrich exit=$ENRICH_STATUS"

log "DONE"
