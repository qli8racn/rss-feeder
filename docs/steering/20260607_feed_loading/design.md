# 設計：RSS リンクの読み込み

## 対象コマンド

`rss-feeder fetch`（フェーズ 1 担当部分）

## 処理フロー

```
feeds.txt 読み込み
  └─ driver/file/feeds_reader.go
       └─ コメント行（#）・空行スキップ
       └─ 有効 URL を []string で返す
```

## 関連ファイル

| ファイル | 役割 |
|---------|------|
| `feeds.txt` | RSS フィード URL リスト（1行1URL） |
| `internal/adapter/driver/file/feeds_reader.go` | FeedsReader interface |
| `internal/driver/file/feeds_reader.go` | FeedsReader 実装 |
