# 設計：お気に入り登録

## 対象コマンド

```bash
rss-feeder bookmark <id>
```

## 動作

| 現在の状態 | 実行後 | メッセージ |
|----------|-------|-----------|
| `bookmarked = 0` | `bookmarked = 1` | お気に入りに登録しました |
| `bookmarked = 1` | `bookmarked = 0` | お気に入りを解除しました |
| ID 不存在 | 変更なし | エラー・終了コード 1 |

## Hook 連携

```
[PreToolUse] validate-state.sh
  └─ rss-feeder check-article <id>  # 不在なら exit 2 でブロック

[PostToolUse] audit-log.sh
  └─ rss-feeder audit --action=bookmark --article-id=<id>
```

## 関連ファイル

| ファイル | 役割 |
|---------|------|
| `internal/domain/article.go` | ToggleBookmark() |
| `internal/usecase/bookmark.go` | トグルロジック |
| `internal/adapter/handler/bookmark.go` | cobra コマンド |
| `.claude/hooks/validate-state.sh` | PreToolUse Hook |
