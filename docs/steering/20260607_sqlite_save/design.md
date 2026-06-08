# 設計：記事の SQLite 保存

## 対象コマンド

```bash
rss-feeder fetch
```

## データフロー

```
Claude Code: Bash ツールで "rss-feeder fetch" 実行
  │
  ├─ [PreToolUse] validate-state.sh（fetch はスキップ）
  │
  ├─ rss-feeder fetch
  │    └─ driver/readerdb/article/article.go  # INSERT OR IGNORE
  │
  └─ [PostToolUse] audit-log.sh
       └─ rss-feeder audit --action=fetch
```

## サブコマンド

| サブコマンド | 用途 |
|------------|------|
| `rss-feeder audit --action=<action>` | audit_log に記録 |

## 関連ファイル

| ファイル | 役割 |
|---------|------|
| `internal/migration/migration.go` | DB・テーブル自動作成 |
| `internal/driver/readerdb/article/article.go` | INSERT OR IGNORE 実装 |
| `.claude/hooks/audit-log.sh` | PostToolUse Hook |
