# フェーズ 8：フィード管理の SQLite 移行

## 概要

RSS フィードの登録先を `feeds.txt` から SQLite（`reader.db`）に移行し、
フィードの追加・一覧・削除を行う CLI コマンドを提供する。

## 受け入れ条件

- `add-feed <url>` でフィード URL を `feeds` テーブルに登録できる
- 既に登録済みの URL を `add-feed` した場合はエラーメッセージを表示して終了コード 1 で終了する
- `list-feeds` で登録済みフィードを ID・URL・タイトル・最終取得日時つきで一覧表示する
- 登録済みフィードが 0 件の場合はガイドメッセージを表示する
- `remove-feed <id>` で指定 ID のフィードを削除する（関連記事も連動削除）
- 存在しない ID を `remove-feed` した場合はエラーメッセージを表示して終了コード 1 で終了する
- `fetch` コマンドはフィードの取得元を `feeds.txt` ではなく DB から読み込む
- `feeds.txt` および file ドライバは削除する
- Claude Code の `add-feed` スキルは `bin/rss-feeder add-feed <url>` を実行する
