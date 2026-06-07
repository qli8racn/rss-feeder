#!/bin/bash
# Stop: セッション終了時の DB メンテナンス（VACUUM + 整合性チェック）

# バイナリ存在確認
if [ ! -x "bin/rss-feeder" ]; then
    exit 0
fi

bin/rss-feeder maintenance 2>/dev/null || true
exit 0
