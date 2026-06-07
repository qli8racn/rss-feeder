#!/bin/bash
# PreToolUse: rss-feeder bookmark/reset 実行前の状態検証

INPUT=$(cat)
COMMAND=$(printf '%s' "$INPUT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('tool_input',{}).get('command',''))" 2>/dev/null || echo "")

# 対象外コマンドはスキップ
if ! echo "$COMMAND" | grep -qE 'rss-feeder (bookmark|reset)'; then
    exit 0
fi

# バイナリ存在確認（未ビルドの場合はスキップ）
if [ ! -x "bin/rss-feeder" ]; then
    exit 0
fi

# bookmark: 記事 ID の存在確認
if echo "$COMMAND" | grep -qE 'rss-feeder bookmark ([0-9]+)'; then
    ARTICLE_ID=$(echo "$COMMAND" | grep -oE 'bookmark ([0-9]+)' | awk '{print $2}')
    if ! bin/rss-feeder check-article "$ARTICLE_ID" > /dev/null 2>&1; then
        echo "記事 ID $ARTICLE_ID は存在しません。操作を中止します。"
        exit 2
    fi
fi

# reset: お気に入り記事の保護確認（情報提供）
if echo "$COMMAND" | grep -qE 'rss-feeder reset'; then
    bin/rss-feeder check-bookmarked 2>/dev/null || true
fi

exit 0
