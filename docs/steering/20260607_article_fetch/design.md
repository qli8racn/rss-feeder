# 設計：記事の取得と標準出力

## 対象コマンド

```bash
rss-feeder fetch
```

## 出力イメージ

```
[1/3] https://example.com/feed.xml
  + Go 1.23 リリースノート  https://example.com/go123
  スキップ: 5 件

[2/3] https://blog.example.org/rss
  エラー: タイムアウト (10s)

完了: 新規 1 件 / スキップ 5 件 / エラー 1 件
```

## 処理フロー

```
usecase/fetch.go
  └─ driver/rss/reader.go   # gofeed で HTTP GET・RSS/Atom パース
       └─ 各記事: タイトル・URL・公開日時を返す
```

## 関連ファイル

| ファイル | 役割 |
|---------|------|
| `internal/adapter/driver/rss/rss_reader.go` | RSSReader interface |
| `internal/driver/rss/reader.go` | RSSReader 実装（gofeed） |
| `internal/usecase/fetch.go` | fetch orchestration |
| `internal/adapter/handler/fetch.go` | cobra コマンド・出力フォーマット |
