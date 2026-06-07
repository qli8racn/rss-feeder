#!/bin/bash
# PostToolUse: 書き込み操作後に audit_log を記録

INPUT=$(cat)
COMMAND=$(printf '%s' "$INPUT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('tool_input',{}).get('command',''))" 2>/dev/null || echo "")

# 書き込みコマンド以外はスキップ
if ! echo "$COMMAND" | grep -qE 'rss-feeder (fetch|bookmark|reset)'; then
    exit 0
fi

# バイナリ存在確認
if [ ! -x "bin/rss-feeder" ]; then
    exit 0
fi

# アクション種別の特定
ACTION=$(echo "$COMMAND" | grep -oE '(fetch|bookmark|reset)' | head -1)

# bookmark の場合は記事 ID を付与
if [ "$ACTION" = "bookmark" ]; then
    ARTICLE_ID=$(echo "$COMMAND" | grep -oE 'bookmark ([0-9]+)' | awk '{print $2}')
    if [ -n "$ARTICLE_ID" ]; then
        bin/rss-feeder audit --action="$ACTION" --article-id="$ARTICLE_ID" > /dev/null 2>&1 || true
        exit 0
    fi
fi

bin/rss-feeder audit --action="$ACTION" > /dev/null 2>&1 || true
exit 0
