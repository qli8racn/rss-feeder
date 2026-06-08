# タスクリスト：記事の SQLite 保存

## 実装タスク

- [x] migration.go による DB・テーブル自動作成（`internal/migration/migration.go`）
- [x] ArticleRepository interface 定義
- [x] INSERT OR IGNORE による重複スキップ実装（`internal/driver/readerdb/article/article.go`）
- [x] `rss-feeder audit --action=<action>` サブコマンド実装
- [x] PostToolUse Hook スクリプト（`.claude/hooks/audit-log.sh`）

## テスト

- [x] driver 統合テスト（インメモリ SQLite 使用）
  - INSERT・重複スキップ・件数確認
