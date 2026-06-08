# フェーズ 3：記事の SQLite 保存

## 概要

取得した記事を SQLite データベースに保存する

## 受け入れ条件

- 初回起動時にデータベースとテーブルを自動作成する
- 同一 URL の記事は重複して保存しない（URL を一意キーとする）
- 保存成功時に件数をログ出力する
- Claude Code が `rss-feeder fetch` を Bash ツールで実行した後、**PostToolUse Hook** が Go バイナリ経由で保存操作を audit_log に記録する
