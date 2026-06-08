# 設計：記事のリセット

## 対象コマンド

```bash
rss-feeder reset [-y]
```

## 動作

- `-y` なし：削除件数を表示して `[y/N]` 確認
- `-y` あり：確認なしで即時削除
- 削除対象は `bookmarked = 0` の記事のみ

## 出力イメージ

```
お気に入り以外の記事 38 件を削除します。よろしいですか？ [y/N]: y
削除完了: 38 件（お気に入り: 5 件 を保持）
```

## Hook 連携

```
[PreToolUse] validate-state.sh
  └─ rss-feeder check-bookmarked  # お気に入り件数を情報提供（ブロックなし）
```

## 関連ファイル

| ファイル | 役割 |
|---------|------|
| `internal/usecase/reset.go` | 削除ロジック |
| `internal/adapter/handler/reset.go` | cobra コマンド・確認プロンプト |
| `.claude/hooks/validate-state.sh` | PreToolUse Hook |
